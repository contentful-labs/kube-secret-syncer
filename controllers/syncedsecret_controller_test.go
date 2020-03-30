package controllers

import (
	"context"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"os"
	"reflect"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	secretsv1 "github.com/contentful-labs/k8s-secret-syncer/api/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetPollInterval(t *testing.T) {
	defer os.Unsetenv("POLL_INTERVAL_SEC")

	for _, test := range []struct {
		have string
		want time.Duration
	}{
		{
			have: "",
			want: defaultPollInterval,
		},
		{
			have: "1000",
			want: time.Second * time.Duration(1000),
		},
	} {
		if test.have != "" {
			os.Setenv("POLL_INTERVAL_SEC", test.have)
		}
		got, err := getPollInterval()
		if err != nil {
			t.Errorf("error getting poll interval: %s", err)
		}
		if got != test.want {
			t.Errorf("poller interval: wanted %s got %s", test.want, got)
		}

	}
}

var _ = Describe("SyncedSecret Controller", func() {
	const timeout = time.Minute * 3
	const interval = time.Second * 2

	BeforeEach(func() {
	})

	AfterEach(func() {
	})

	// Add Tests for OpenAPI validation (or additonal CRD features) specified in
	// your API definition.
	// Avoid adding tests for vanilla CRUD operations because they would
	// test Kubernetes API server, which isn't the goal here.
	Context("For a single SyncedSecret", func() {
		secretKey := types.NamespacedName{
			Name:      "secret-name",
			Namespace: TEST_NAMESPACE,
		}

		resourceVersion := ""
		It("Should Create K8S Secrets for SyncedSecret CRD", func() {
			toCreate := &secretsv1.SyncedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretKey.Name,
					Namespace:       secretKey.Namespace,
					ResourceVersion: resourceVersion,
				},
				Spec: secretsv1.SyncedSecretSpec{
					SecretMetadata: metav1.ObjectMeta{
						Name:      secretKey.Name,
						Namespace: secretKey.Namespace,
					},
					IAMRole: _s("test"),
					Data: []*secretsv1.SecretField{
						{
							Name: _s("DB_NAME"),
							ValueFrom: &secretsv1.ValueFrom{
								SecretKeyRef: &secretsv1.SecretKeyRef{
									Name: _s("random/aws/secret003"),
									Key:  _s("database_name"),
								},
							},
						},
						{
							Name: _s("DB_PASS"),
							ValueFrom: &secretsv1.ValueFrom{
								SecretKeyRef: &secretsv1.SecretKeyRef{
									Name: _s("random/aws/secret003"),
									Key:  _s("database_pass"),
								},
							},
						},
					},
				},
			}

			secretExpect := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretKey.Name,
					Namespace: secretKey.Namespace,
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"DB_NAME": []byte("secretDB"),
					"DB_PASS": []byte("cupofcoffee"),
				},
			}

			Expect(k8sClient.Create(context.Background(), toCreate)).Should(Succeed())

			fetchedSecret := &corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretKey, fetchedSecret)
				return k8serrors.IsNotFound(err)
			}, timeout, interval).Should(BeFalse())

			// we need to ensure that that secretExpect.Data is a subset of fetchedSecret.Data
			// the kubernetes client.go doesn't base64 values this is something that kubectl maybe does
			Expect(reflect.DeepEqual(fetchedSecret.Data, secretExpect.Data)).To(BeTrue())

			fetchedCfSecret := &secretsv1.SyncedSecret{}
			err := k8sClient.Get(context.Background(), secretKey, fetchedCfSecret)
			Expect(err).ToNot(HaveOccurred())
			resourceVersion = fetchedCfSecret.ResourceVersion
		})

		It("Should update k8s secret object if there is change in AwsSecret CRD", func() {
			toUpdate := &secretsv1.SyncedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretKey.Name,
					Namespace:       secretKey.Namespace,
					ResourceVersion: resourceVersion,
				},
				Spec: secretsv1.SyncedSecretSpec{
					SecretMetadata: metav1.ObjectMeta{
						Name:      secretKey.Name,
						Namespace: secretKey.Namespace,
					},
					IAMRole: _s("test"),
					Data: []*secretsv1.SecretField{
						{
							Name: _s("DB_NAME"),
							ValueFrom: &secretsv1.ValueFrom{
								SecretKeyRef: &secretsv1.SecretKeyRef{
									Name: _s("random/aws/secret003"),
									Key:  _s("database_name1"),
								},
							},
						},
						{
							Name: _s("DB_PASS"),
							ValueFrom: &secretsv1.ValueFrom{
								SecretKeyRef: &secretsv1.SecretKeyRef{
									Name: _s("random/aws/secret003"),
									Key:  _s("database_pass"),
								},
							},
						},
					},
				},
			}

			secretExpect := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretKey.Name,
					Namespace: secretKey.Namespace,
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"DB_NAME": []byte("secretDB02"),
					"DB_PASS": []byte("cupofcoffee"),
				},
			}

			Expect(k8sClient.Update(context.Background(), toUpdate)).Should(Succeed())

			fetchedSecret := &corev1.Secret{}
			Eventually(func() bool {
				k8sClient.Get(context.Background(), secretKey, fetchedSecret)
				return reflect.DeepEqual(fetchedSecret.Data, secretExpect.Data)
			}, timeout, interval).Should(BeTrue())

			fetchedCfSecret := &secretsv1.SyncedSecret{}
			err := k8sClient.Get(context.Background(), secretKey, fetchedCfSecret)
			Expect(err).ToNot(HaveOccurred())
			resourceVersion = fetchedCfSecret.ResourceVersion
		})

		It("Should update the k8s secret object if the mapped AWS Secret changes", func() {
			MockSecretsOutput.SecretsValueOutput = &secretsmanager.GetSecretValueOutput{
				SecretString: _s(`{"database_pass":"cupoftea", "database_name1":"secretDB02"}`),
				VersionId:    _s(`006`),
			}

			MockSecretsOutput.SecretsPageOutput = &secretsmanager.ListSecretsOutput{
				SecretList: []*secretsmanager.SecretListEntry{
					{
						Name:            _s("random/aws/secret003"),
						LastChangedDate: _t(time_now.AddDate(0, 0, -2)),
						SecretVersionsToStages: map[string][]*string{
							"002": []*string{
								_s("AWSCURRENT"),
							},
						},
					}, {
						Name:            _s("random/aws/secret003"),
						LastChangedDate: _t(time_now.AddDate(0, 0, -1)),
						SecretVersionsToStages: map[string][]*string{
							"005": {
								_s("AWSPREVIOUS"),
							},
							"006": {
								_s("AWSCURRENT"),
							},
						},
					},
				},
			}

			secretExpect := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretKey.Name,
					Namespace: secretKey.Namespace,
				},
				Type: "Opaque",
				Data: map[string][]byte{
					"DB_NAME": []byte("secretDB02"),
					"DB_PASS": []byte("cupoftea"),
				},
			}

			fetchedSecret := &corev1.Secret{}
			Eventually(func() bool {
				k8sClient.Get(context.Background(), secretKey, fetchedSecret)
				return reflect.DeepEqual(fetchedSecret.Data, secretExpect.Data)
			}, timeout, interval).Should(BeTrue())
		})
	})
})
