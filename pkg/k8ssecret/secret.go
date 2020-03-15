package k8ssecret

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/contentful-labs/k8s-secret-syncer/pkg/secretsmanager"
	"github.com/pkg/errors"
	"reflect"
	"text/template"

	secretsv1 "github.com/contentful-labs/k8s-secret-syncer/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func K8SSecretsEqual(secret1, secret2 corev1.Secret) bool {
	if !reflect.DeepEqual(secret1.Data, secret2.Data) {
		return false
	}

	if !reflect.DeepEqual(secret1.ObjectMeta.Annotations, secret2.ObjectMeta.Annotations) {
		return false
	}

	return true
}

func GenerateK8SSecret(
	cs secretsv1.SyncedSecret,
	secrets secretsmanager.Secrets,
	secretValueGetter func(string, string) (map[string]interface{}, error),
	secretFilterByTagKey func(secretsmanager.Secrets, string) secretsmanager.Secrets,
) (*corev1.Secret, error) {
	annotations := map[string]string{}
	if cs.Spec.SecretMetadata.Annotations != nil {
		for key, val := range cs.Spec.SecretMetadata.Annotations {
			annotations[key] = val
		}
	}

	labels := map[string]string{}
	if cs.Spec.SecretMetadata.Labels != nil {
		for key, val := range cs.Spec.SecretMetadata.Labels {
			labels[key] = val
		}
	}

	secretMeta := metav1.ObjectMeta{
		Name:      cs.ObjectMeta.Name,
		Namespace: cs.ObjectMeta.Namespace,
	}
	if len(annotations) > 0 {
		secretMeta.Annotations = annotations
	}
	if len(labels) > 0 {
		secretMeta.Labels = labels
	}

	// Now to the data...
	data := make(map[string][]byte)

	if cs.Spec.DataFrom != nil {
		var secretMapRef *string // secretID of the secret in secret Manager
		if cs.Spec.DataFrom.SecretMapRef != nil {
			secretMapRef = cs.Spec.DataFrom.SecretMapRef.Name
		}

		AWSSecretValues := map[string]interface{}{}
		var err error
		if secretMapRef != nil {
			if AWSSecretValues, err = secretValueGetter(*secretMapRef, *cs.Spec.IAMRole); err != nil {
				return nil, err
			}
			for secretKey, secretValue := range AWSSecretValues {
				data[secretKey] = []byte(fmt.Sprintf("%v", secretValue))
			}
		}
	}

	if cs.Spec.Data != nil {
		for _, field := range cs.Spec.Data {
			if field.Value != nil {
				data[*field.Name] = []byte(*field.Value)
			}

			if field.ValueFrom != nil {
				if field.ValueFrom.SecretKeyRef != nil {
					AWSSecretValues := map[string]interface{}{}
					var err error
					if AWSSecretValues, err = secretValueGetter(*field.ValueFrom.SecretKeyRef.Name, *cs.Spec.IAMRole); err != nil {
						return nil, err
					}
					data[*field.Name] = []byte(fmt.Sprintf("%v", AWSSecretValues[*field.ValueFrom.SecretKeyRef.Key]))
				}

				if field.ValueFrom.Template != nil {
					tpl := template.New(cs.Name)
					tpl = tpl.Funcs(template.FuncMap{
						"getSecretValue": func(secretID string) (map[string]interface{}, error) {
							return secretValueGetter(secretID, *cs.Spec.IAMRole)
						},
						"filterByTagKey": secretFilterByTagKey,
						"base64": func(value interface{}) string {
							return base64.StdEncoding.EncodeToString([]byte(value.(string)))
						},
						"indent": sprig.FuncMap()["indent"],
					})

					var err error
					if tpl, err = tpl.Parse(*field.ValueFrom.Template); err != nil {
						return nil, errors.Wrap(err, "error parsing template from secret")
					}

					buf := new(bytes.Buffer)
					type templateParams struct {
						Secrets secretsmanager.Secrets
					}
					if err = tpl.Execute(buf, templateParams{Secrets: secrets}); err != nil {
						return nil, errors.Wrap(err, "error executing template from SyncedSecret")
					}

					data[*field.Name] = buf.Bytes()
				}
			}
		}
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: secretMeta,
		Type:       "Opaque",
		Data:       data,
	}

	return secret, nil
}
