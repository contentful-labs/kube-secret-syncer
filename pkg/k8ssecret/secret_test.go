package k8ssecret

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/contentful-labs/k8s-secret-syncer/pkg/secretsmanager"
	"reflect"
	"strings"
	"testing"

	secretsv1 "github.com/contentful-labs/k8s-secret-syncer/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func _s(A string) *string {
	return &A
}

func mockgetSecretValue(secretID string, role string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}, nil
}

func mockgetDBSecretValue(secretID string, role string) (map[string]interface{}, error) {
	user := "contentful"
	if strings.Contains(secretID, "graphapi") {
		user = "graphapi"
	}

	return map[string]interface{}{
		"shardid":  secretID,
		"host":     fmt.Sprintf("%s-host", secretID),
		"user":     user,
		"password": fmt.Sprintf("%s-password", secretID),
	}, nil
}

func mockFailinggetSecretValue(secretID string, role string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("failed getting secret value")
}

func TestGenerateSecret(t *testing.T) {
	type have struct {
		secretsv1.SyncedSecret
		secretVersion     string
		err               error
		cachedSecrets     secretsmanager.Secrets
		secretValueGetter func(string, string) (map[string]interface{}, error)
	}
	testCases := []struct {
		name string
		have have
		want *corev1.Secret
	}{
		{
			name: "it should copy all fields from a K8S Secret given a DataFrom field",
			have: have{
				SyncedSecret: secretsv1.SyncedSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-name",
						Namespace: "secret-namespace",
					},
					Spec: secretsv1.SyncedSecretSpec{
						SecretMetadata: metav1.ObjectMeta{
							Name:      "secret-name",
							Namespace: "secret-namespace",
							Annotations: map[string]string{
								"randomkey": "random/string",
							},
						},
						DataFrom: &secretsv1.DataFrom{SecretMapRef: &secretsv1.SecretMapRef{Name: aws.String("cf/secret/test")}},
						IAMRole: _s("iam_role"),
					},
				},
				err:               nil,
				cachedSecrets:     secretsmanager.Secrets{"cachedSecret1": {}, "cachedSecret2": {}},
				secretValueGetter: mockgetSecretValue,
			},
			want: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-name",
					Namespace: "secret-namespace",
					Annotations: map[string]string{
						"randomkey": "random/string",
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"key1": []byte("value1"),
					"key2": []byte("value2"),
				},
			},
		},
		{
			name: "it should support fields with a hardcoded value",
			have: have{
				SyncedSecret: secretsv1.SyncedSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-name",
						Namespace: "secret-namespace",
					},
					Spec: secretsv1.SyncedSecretSpec{
						SecretMetadata: metav1.ObjectMeta{
							Name:      "secret-name",
							Namespace: "secret-namespace",
							Annotations: map[string]string{
								"randomkey": "random/string",
							},
						},
						Data: []*secretsv1.SecretField{
							{
								Name:  _s("foo"),
								Value: _s("bar"),
							},
							{
								Name:  _s("field2"),
								Value: _s("value2"),
							},
						},
						IAMRole: _s("iam_role"),
					},
				},
				err:               nil,
				cachedSecrets:     secretsmanager.Secrets{"cachedSecret1": {}, "cachedSecret2": {}},
				secretValueGetter: mockgetSecretValue,
			},
			want: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-name",
					Namespace: "secret-namespace",
					Annotations: map[string]string{
						"randomkey": "random/string",
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"foo":    []byte("bar"),
					"field2": []byte("value2"),
				},
			},
		},
		{
			name: "it should support references to a single field in an AWS Secret",
			have: have{
				SyncedSecret: secretsv1.SyncedSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-name",
						Namespace: "secret-namespace",
					},
					Spec: secretsv1.SyncedSecretSpec{
						SecretMetadata: metav1.ObjectMeta{
							Name:      "secret-name",
							Namespace: "secret-namespace",
							Annotations: map[string]string{
								"randomkey": "random/string",
							},
						},
						Data: []*secretsv1.SecretField{
							{
								Name: _s("foo"),
								ValueFrom: &secretsv1.ValueFrom{
									SecretKeyRef: &secretsv1.SecretKeyRef{
										Name: _s("cf/secret/test"),
										Key:  _s("key2"),
									},
								},
							},
						},
						IAMRole: _s("iam_role"),
					},
				},
				err:               nil,
				cachedSecrets:     secretsmanager.Secrets{"cachedSecret1": {}, "cachedSecret2": {}},
				secretValueGetter: mockgetSecretValue,
			},
			want: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-name",
					Namespace: "secret-namespace",
					Annotations: map[string]string{
						"randomkey": "random/string",
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"foo": []byte("value2"),
				},
			},
		},
		{
			name: "it should support templated fields",
			have: have{
				SyncedSecret: secretsv1.SyncedSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-name",
						Namespace: "secret-namespace",
					},
					Spec: secretsv1.SyncedSecretSpec{
						SecretMetadata: metav1.ObjectMeta{
							Name:      "secret-name",
							Namespace: "secret-namespace",
							Annotations: map[string]string{
								"randomkey": "random/string",
							},
						},
						Data: []*secretsv1.SecretField{
							{
								Name: _s("foo"),
								ValueFrom: &secretsv1.ValueFrom{
									Template: _s(`{{- with getSecretValue "cachedSecret1" }}{{ .key2 }}{{ end -}}`),
								},
							},
						},
						IAMRole: _s("iam_role"),
					},
				},
				err:               nil,
				cachedSecrets:     secretsmanager.Secrets{"cachedSecret1": {}, "cachedSecret2": {}},
				secretValueGetter: mockgetSecretValue,
			},
			want: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-name",
					Namespace: "secret-namespace",
					Annotations: map[string]string{
						"randomkey": "random/string",
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"foo": []byte("value2"),
				},
			},
		},
		{
			name: "it should be able to iterate through the available secrets",
			have: have{
				SyncedSecret: secretsv1.SyncedSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-name",
						Namespace: "secret-namespace",
					},
					Spec: secretsv1.SyncedSecretSpec{
						SecretMetadata: metav1.ObjectMeta{
							Name:      "secret-name",
							Namespace: "secret-namespace",
							Annotations: map[string]string{
								"randomkey": "random/string",
							},
						},
						Data: []*secretsv1.SecretField{
							{
								Name: _s("foo"),
								ValueFrom: &secretsv1.ValueFrom{
									Template: _s(`
{{- $cfg := "" -}}
{{- range $secretName, $_ := filterByTagKey .Secrets "tag1" -}}
  {{- $secretValue := getSecretValue $secretName -}}
  {{- $cfg = printf "%shost=%s user=%s password=%s\n" $cfg $secretValue.host $secretValue.user $secretValue.password -}}
{{- end -}}
{{- $cfg -}}
`),
								},
							},
						},
						IAMRole: _s("iam_role"),
					},
				},
				err: nil,
				cachedSecrets: secretsmanager.Secrets{
					"cachedSecret1": {
						Tags: map[string]string{
							"unknownTag": "true",
						},
					},
					"cachedSecret2": {
						Tags: map[string]string{
							"tag1": "true",
						},
					},
					"cachedSecret3": {
						Tags: map[string]string{
							"tag1": "true",
						},
					},
				},
				secretValueGetter: mockgetDBSecretValue,
			},
			want: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-name",
					Namespace: "secret-namespace",
					Annotations: map[string]string{
						"randomkey": "random/string",
					},
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"foo": []byte("host=cachedSecret2-host user=contentful password=cachedSecret2-password\nhost=cachedSecret3-host user=contentful password=cachedSecret3-password\n"),
				},
			},
		},
		{
			name: "AwsSecret should fail if getSecretvalue Fails",
			have: have{
				SyncedSecret: secretsv1.SyncedSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-name",
						Namespace: "secret-namespace",
					},
					Spec: secretsv1.SyncedSecretSpec{
						Data: []*secretsv1.SecretField{
							{
								Name: _s("foo"),
								ValueFrom: &secretsv1.ValueFrom{
									Template: _s(`
{{- $cfg := "" -}}
{{- range $secretName, $_ := filterByTagKey .Secrets "tag1" -}}
  {{- $secretValue := getSecretValue $secretName -}}
  {{- $cfg = printf "%shost=%s user=%s password=%s\n" $cfg $secretValue.host $secretValue.user $secretValue.password -}}
{{- end -}}
{{- $cfg -}}
`),
								},
							},
						},
						IAMRole: _s("iam_role"),
					},
				},
				err:               nil,
				cachedSecrets:     secretsmanager.Secrets{"cachedSecret1": {Tags: map[string]string{"tag1": ""}}, "cachedSecret2": {}},
				secretValueGetter: mockFailinggetSecretValue,
			},
			want: nil,
		},
	}

	for _, test := range testCases {
		k8sSecret, err := GenerateK8SSecret(test.have.SyncedSecret, test.have.cachedSecrets, test.have.secretValueGetter, secretsmanager.FilterByTagKey)
		if !reflect.DeepEqual(k8sSecret, test.want) {
			if k8sSecret != nil && k8sSecret.Data != nil {
				for k, v := range k8sSecret.Data {
					fmt.Printf("%s: %s\n", k, string(v))
				}
			}
			want, _ := json.MarshalIndent(test.want, "", " ")
			got, _ := json.MarshalIndent(k8sSecret, "", " ")
			t.Errorf("Failed: %s\nwanted:\t%s\ngenerated:\t%s \nerror: %s", test.name, want, got, err)
		}
	}
}
func TestK8SSecretsEqual(t *testing.T) {
	testEqualCases := []struct {
		secret1, secret2 corev1.Secret
	}{
		{
			corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testName",
					Namespace:   "testNamespace",
					Annotations: make(map[string]string),
					Labels:      make(map[string]string),
				},
				Type: "Opaque",
				Data: map[string][]byte{},
			},
			corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "testName",
					Namespace:   "testNamespace",
					Annotations: make(map[string]string),
					Labels:      make(map[string]string),
				},
				Type: "Opaque",
				Data: map[string][]byte{},
			},
		},
	}

	for _, testCase := range testEqualCases {
		if !K8SSecretsEqual(testCase.secret1, testCase.secret2) {
			t.Errorf("secrets not equal, but should be")
		}
	}
}
