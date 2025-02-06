package k8ssecret

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/contentful-labs/kube-secret-syncer/pkg/secretsmanager"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	secretsv1 "github.com/contentful-labs/kube-secret-syncer/api/v1"
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
	secretValueGetter func(string, string) (string, error),
	secretFilterByTagKey func(secretsmanager.Secrets, string) secretsmanager.Secrets,
	log logr.Logger,
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
		var secretRef *string // secretID of the secret in secret Manager
		if cs.Spec.DataFrom.SecretRef != nil {
			secretRef = cs.Spec.DataFrom.SecretRef.Name
		}

		if secretRef != nil {
			var iamrole string
			if cs.Spec.AWSAccountID != nil {
				iamrole = fmt.Sprintf("arn:aws:iam::%s:role/secret-syncer", *cs.Spec.AWSAccountID)
			} else {
				iamrole = *cs.Spec.IAMRole
			}
			AWSSecretValue, err := secretValueGetter(*secretRef, iamrole)
			if err != nil {
				return nil, err
			}
			var AWSSecretValuesMap map[string]interface{}
			err = json.Unmarshal([]byte(AWSSecretValue), &AWSSecretValuesMap)
			if err != nil {
				return nil, fmt.Errorf("secret %s is not a valid JSON", *secretRef)
			}
			for secretKey, secretValue := range AWSSecretValuesMap {
				data[secretKey] = []byte(fmt.Sprintf("%v", secretValue))
			}
		}
	}

	if cs.Spec.Data != nil {
		var iamrole string
		if cs.Spec.AWSAccountID != nil {
			iamrole = fmt.Sprintf("arn:aws:iam::%s:role/secret-syncer", *cs.Spec.AWSAccountID)
		} else {
			iamrole = *cs.Spec.IAMRole
		}
		for _, field := range cs.Spec.Data {
			if field.Value != nil {
				data[*field.Name] = []byte(*field.Value)
			}

			if field.ValueFrom != nil {
				if field.ValueFrom.SecretRef != nil {
					AWSSecretValue, err := secretValueGetter(*field.ValueFrom.SecretRef.Name, iamrole)
					if err != nil {
						return nil, err
					}
					data[*field.Name] = []byte(AWSSecretValue)
				}

				if field.ValueFrom.SecretKeyRef != nil {
					AWSSecretValue, err := secretValueGetter(*field.ValueFrom.SecretKeyRef.Name, iamrole)
					if err != nil {
						return nil, err
					}
					AWSSecretValuesMap := map[string]interface{}{}
					if err := json.Unmarshal([]byte(AWSSecretValue), &AWSSecretValuesMap); err != nil {
						return nil, err
					}
					data[*field.Name] = []byte(fmt.Sprintf("%v", AWSSecretValuesMap[*field.ValueFrom.SecretKeyRef.Key]))
				}

				if field.ValueFrom.Template != nil {
					tpl := template.New(cs.Name)
					tpl = tpl.Funcs(template.FuncMap{
						"getSecretValue": func(secretID string) (string, error) {
							return secretValueGetter(secretID, iamrole)
						},
						"getSecretValueMap": func(secretID string) (map[string]interface{}, error) {
							raw, err := secretValueGetter(secretID, iamrole)
							if err != nil {
								return nil, fmt.Errorf("failed retrieving value for secret %s", secretID)
							}
							var asMap map[string]interface{}
							if err := json.Unmarshal([]byte(raw), &asMap); err != nil {
								return nil, fmt.Errorf("secret %s does not contain a valid JSON", secretID)
							}
							return asMap, err
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

func SecretLength(secret *corev1.Secret) int {
	length := 0

	for _, v := range secret.Data {
		length += len(v)
	}

	return length
}
