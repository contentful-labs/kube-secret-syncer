/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	awssecretsmanager "github.com/aws/aws-sdk-go/service/secretsmanager"
	secretsv1 "github.com/contentful-labs/kube-secret-syncer/api/v1"
	"github.com/contentful-labs/kube-secret-syncer/pkg/k8snamespace"
	"github.com/contentful-labs/kube-secret-syncer/pkg/k8ssecret"
	"github.com/contentful-labs/kube-secret-syncer/pkg/secretsmanager"
)

type RoleValidator interface {
	IsWhitelisted(role, namespace string) (bool, error)
}

type NamespaceValidator interface {
	HasNamespaceType(secret awssecretsmanager.DescribeSecretOutput, namespace string) (bool, error)
}

// SyncedSecretReconciler reconciles a SyncedSecret object
type SyncedSecretReconciler struct {
	client.Client
	Sess               *session.Session
	GetSMClient        func(string) (secretsmanageriface.SecretsManagerAPI, error)
	poller             *secretsmanager.Poller
	getNamespace       k8snamespace.NamespaceGetter
	RoleValidator      RoleValidator
	NamespaceValidator NamespaceValidator
	PollInterval       time.Duration
	Log                logr.Logger
	wg                 sync.WaitGroup

	DefaultSearchRole string

	gauges     map[string]prometheus.Gauge
	sync_state map[string]bool
}

const (
	LogFieldSyncedSecret = "SyncedSecret"
	LogFieldK8SSecret    = "KubernetesSecret"
)

// +kubebuilder:rbac:groups=secrets.contentful.com,resources=syncedsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secrets.contentful.com,resources=syncedsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

func (r *SyncedSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	var cs secretsv1.SyncedSecret

	defer r.updatePrometheus(r.sync_state)

	log := r.Log.WithValues(LogFieldSyncedSecret, req.NamespacedName.String())
	if err = r.Get(ctx, req.NamespacedName, &cs); err != nil {
		log.Info("unable to fetch SyncedSecret, was maybe deleted")
		return ctrl.Result{}, nil
	}

	// even though the SyncedSecret can contain name and namespace for the k8s secret to be created, we are disregarding it.
	// the generated secret will have the same name/namesapce as the CRD
	K8SSecretName := types.NamespacedName{
		Name:      cs.ObjectMeta.Name,
		Namespace: cs.ObjectMeta.Namespace,
	}
	log = log.WithValues(LogFieldK8SSecret, K8SSecretName.String())

	if cs.Spec.AWSAccountID != nil {
		IAMRole := fmt.Sprintf("arn:aws:iam::%s:role/secret-syncer", *cs.Spec.AWSAccountID)
		var secretRef *string // secretID of the secret in secret Manager

		// We need to check each secret in Data and DataFrom to see if they are allowed in the namespace
		if cs.Spec.DataFrom != nil {
			if cs.Spec.DataFrom.SecretRef != nil {
				secretRef = cs.Spec.DataFrom.SecretRef.Name
				if secretRef == nil {
					return ctrl.Result{}, errors.WithMessagef(err, "secretRef name is invalid %s", *secretRef)
				}

				allowed, err := r.secretAllowedInNamespace(*secretRef, IAMRole, cs.Namespace, cs.Name)

				if !allowed || err != nil {
					return ctrl.Result{}, errors.WithMessagef(err, "failed to validate if secret %s with role %s is allowed in namespace %s", *secretRef, IAMRole, cs.Namespace)
				}
			}

		}

		if cs.Spec.Data != nil {
			for _, field := range cs.Spec.Data {
				if field.ValueFrom.SecretRef != nil {
					secretRef = field.ValueFrom.SecretKeyRef.Name
					if secretRef == nil {
						return ctrl.Result{}, errors.WithMessagef(err, "secretRef name is invalid %s", *secretRef)
					}

					allowed, err := r.secretAllowedInNamespace(*secretRef, IAMRole, cs.Namespace, cs.Name)

					if !allowed || err != nil {
						return ctrl.Result{}, errors.WithMessagef(err, "failed to validate if secret %s with role %s is allowed in namespace %s", *secretRef, IAMRole, cs.Namespace)
					}
				}
			}
		}

	} else {
		allowed, err := r.RoleValidator.IsWhitelisted(*cs.Spec.IAMRole, cs.Namespace)
		if !allowed {
			r.sync_state[cs.Name] = false
			log.Error(err, "role not allowed by namespace", "role", *cs.Spec.IAMRole, "namespace", cs.Namespace)
			return ctrl.Result{}, errors.WithMessagef(err, "role %s not allowed in namespace %s", *cs.Spec.IAMRole, cs.Namespace)
		}
		if err != nil {
			r.sync_state[cs.Name] = false
			log.Error(err, "failed verifying if IAMRole is whitelisted", "role", *cs.Spec.IAMRole, "namespace", cs.Namespace)
			return ctrl.Result{}, errors.WithMessagef(err, "failed verifying role %s: %s", *cs.Spec.IAMRole, err)
		}
	}

	var k8sSecret corev1.Secret = corev1.Secret{}
	err = r.Get(ctx, K8SSecretName, &k8sSecret)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			r.sync_state[cs.Name] = false
			return ctrl.Result{}, errors.WithMessagef(err, "error retrieving k8s secret %s", K8SSecretName)
		}

		// Create the k8S secret if it was not found
		createdSecret, err := r.createK8SSecret(ctx, &cs)
		if err != nil {
			r.sync_state[cs.Name] = false
			return ctrl.Result{}, errors.WithMessagef(err, "failed creating K8S Secret %s", K8SSecretName)
		}
		log.Info("created k8s secret", "K8SSecret", createdSecret)
	} else {
		// Update the K8S Secret if it already exists
		updatedSecret, err := r.updateK8SSecret(ctx, &cs)
		if err != nil {
			r.sync_state[cs.Name] = false
			return ctrl.Result{}, errors.WithMessagef(err, "failed updating k8s secret %s", K8SSecretName)
		}
		if !k8ssecret.K8SSecretsEqual(k8sSecret, *updatedSecret) {
			log.Info("updated secret", "K8SSecret", updatedSecret.ObjectMeta, "secretSize", k8ssecret.SecretLength(updatedSecret))
		}
	}

	if err = r.updateCSStatus(ctx, &cs); err != nil {
		r.sync_state[cs.Name] = false
		log.Error(err, "failed to update SyncedSecret status")
		return ctrl.Result{}, errors.WithMessagef(err, "failed to update SyncedSecret status for %s", K8SSecretName)
	}

	r.sync_state[cs.Name] = true

	return ctrl.Result{}, nil
}

func (r *SyncedSecretReconciler) secretAllowedInNamespace(secretID string, IAMRole string, namespace string, name string) (bool, error) {
	log := r.Log.WithValues(LogFieldSyncedSecret, namespace)
	secret, err := r.poller.DescribeSecret(aws.String(secretID), IAMRole)
	if err != nil {
		log.Error(err, "failed to describe secret", "role", IAMRole, "namespace", namespace)
		return false, errors.WithMessagef(err, "failed to fetch secret %s with role %s in namespace %s", secretID, IAMRole, namespace)
	}

	allowed, err := r.NamespaceValidator.HasNamespaceType(secret, namespace)
	if !allowed {
		r.sync_state[name] = false
		log.Error(err, "namespace not allowed in secret", "namespace", namespace, "secret", secretID)
		return false, errors.WithMessagef(err, "namespace %s not allowed in secret %s", namespace, secretID)
	}
	if err != nil {
		r.sync_state[name] = false
		log.Error(err, "failed verifying if namespace is allowed in secret", "namespace", namespace, "secret", secretID)
		return false, errors.WithMessagef(err, "failed verifying secret %s: %s", secretID, err)
	}
	return true, nil
}

func (r *SyncedSecretReconciler) templateSecretGetter(secretID string, IAMRole string) (string, error) {
	secretString, _, err := r.poller.GetSecret(aws.String(secretID), IAMRole)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("error retrieving secret %s", secretID))
	}

	return secretString, err
}

// createSecret creates a k8s Secret from a SyncedSecret
func (r *SyncedSecretReconciler) createK8SSecret(ctx context.Context, cs *secretsv1.SyncedSecret) (*corev1.Secret, error) {

	secret, err := k8ssecret.GenerateK8SSecret(*cs, r.poller.PolledSecrets, r.templateSecretGetter, secretsmanager.FilterByTagKey, r.Log)
	if err != nil {
		return nil, err
	}

	if err = r.Create(ctx, secret); err != nil {
		return nil, err
	}

	r.Log.Info("Created K8S Secret", "K8SSecret", secret.ObjectMeta, "secretSize", k8ssecret.SecretLength(secret))

	return secret, nil
}

func (r *SyncedSecretReconciler) updateK8SSecret(ctx context.Context, cs *secretsv1.SyncedSecret) (*corev1.Secret, error) {
	var secret *corev1.Secret
	var err error

	secret, err = k8ssecret.GenerateK8SSecret(*cs, r.poller.PolledSecrets, r.templateSecretGetter, secretsmanager.FilterByTagKey, r.Log)
	if err != nil {
		return nil, err
	}

	if err = r.Update(ctx, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

// updateCSStatus updates the SyncedSecret.Status versionId (from aws SSM) seen
func (r *SyncedSecretReconciler) updateCSStatus(ctx context.Context, cs *secretsv1.SyncedSecret) error {
	//cs.Status.CurrentVersionID = r.poller.PolledSecrets[cs.Spec.SecretID].CurrentVersionID
	return r.Status().Update(ctx, cs)
}

func (r *SyncedSecretReconciler) Quit() {
	r.poller.Stop()
	r.wg.Wait()
}

func (r *SyncedSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error

	errs := make(chan error)
	go func() {
		r.wg.Add(1)
		defer r.wg.Done()

		for err := range errs {
			r.Log.Error(err, "polling error")
		}
	}()

	r.sync_state = map[string]bool{}
	r.gauges = map[string]prometheus.Gauge{
		"secret_sync_success": prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "secret_sync_success",
				Help: "Number of SyncedSecrets successfully syncing",
			},
		),
		"secret_sync_failures": prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "secret_sync_failures",
				Help: "Number of SyncedSecrers failing to sync",
			},
		),
	}

	for _, metric := range r.gauges {
		metrics.Registry.MustRegister(metric)
	}

	if r.poller, err = secretsmanager.New(r.PollInterval, errs, r.GetSMClient, r.DefaultSearchRole, r.Log); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1.SyncedSecret{}).
		Complete(r)
}

func (r *SyncedSecretReconciler) updatePrometheus(syncState map[string]bool) {
	success := 0
	failures := 0

	for _, state := range syncState {
		if state == true {
			success++
		} else {
			failures++
		}
	}

	if _, ok := r.gauges["secret_sync_success"]; ok {
		r.gauges["secret_sync_success"].Set(float64(success))
	}
	if _, ok := r.gauges["secret_sync_failures"]; ok {
		r.gauges["secret_sync_failures"].Set(float64(success))
	}
}
