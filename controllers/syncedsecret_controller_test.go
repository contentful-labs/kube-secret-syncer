package controllers

import (
	"context"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/service/secretsmanager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	secretsv1 "github.com/contentful-labs/kube-secret-syncer/api/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

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
					SecretMetadata: secretsv1.SecretMetadata{
						Name:      secretKey.Name,
						Namespace: secretKey.Namespace,
						CreationTimestamp: metav1.Time{
							Time: time_now,
						},
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
					SecretMetadata: secretsv1.SecretMetadata{
						Name:      secretKey.Name,
						Namespace: secretKey.Namespace,
						CreationTimestamp: metav1.Time{
							Time: time_now,
						},
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

	Context("For a single SyncedSecret (using Data) with AWSAccountID", func() {
		secretKey := types.NamespacedName{
			Name:      "another-secret-name",
			Namespace: TEST_NAMESPACE2,
		}

		resourceVersion := ""

		It("Should Create K8S Secrets for SyncedSecret CRD with AWSAccountID", func() {
			MockSecretsOutput.SecretsValueOutput = &secretsmanager.GetSecretValueOutput{
				SecretString: _s(`{"database_name":"secretDB","database_pass":"cupofcoffee", "database_name1":"secretDB02"}`),
				VersionId:    _s(`005`),
			}

			toCreate := &secretsv1.SyncedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretKey.Name,
					Namespace:       secretKey.Namespace,
					ResourceVersion: resourceVersion,
				},
				Spec: secretsv1.SyncedSecretSpec{
					SecretMetadata: secretsv1.SecretMetadata{
						Name:      secretKey.Name,
						Namespace: secretKey.Namespace,
						CreationTimestamp: metav1.Time{
							Time: time_now,
						},
					},
					AWSAccountID: _s("12345678910"),
					IAMRole:      _s("test"),
					Data: []*secretsv1.SecretField{
						{
							Name: _s("DB_NAME"),
							ValueFrom: &secretsv1.ValueFrom{
								SecretKeyRef: &secretsv1.SecretKeyRef{
									Name: _s("random/aws/secret004"),
									Key:  _s("database_name"),
								},
							},
						},
						{
							Name: _s("DB_PASS"),
							ValueFrom: &secretsv1.ValueFrom{
								SecretKeyRef: &secretsv1.SecretKeyRef{
									Name: _s("random/aws/secret004"),
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
			err := k8sClient.Create(context.Background(), toCreate)
			Expect(err).ToNot(HaveOccurred())

			fetchedSecret := &corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretKey, fetchedSecret)
				return k8serrors.IsNotFound(err)
			}, timeout, interval).Should(BeFalse())

			// we need to ensure that that secretExpect.Data is a subset of fetchedSecret.Data
			// the kubernetes client.go doesn't base64 values this is something that kubectl maybe does
			Expect(reflect.DeepEqual(fetchedSecret.Data, secretExpect.Data)).To(BeTrue())

			fetchedCfSecret := &secretsv1.SyncedSecret{}
			err = k8sClient.Get(context.Background(), secretKey, fetchedCfSecret)
			Expect(err).ToNot(HaveOccurred())
			resourceVersion = fetchedCfSecret.ResourceVersion

		})

		It("Should update k8s secret object if there is change in AwsSecret CRD with AWSAccountID", func() {
			MockSecretsOutput.SecretsValueOutput = &secretsmanager.GetSecretValueOutput{
				SecretString: _s(`{"database_name":"secretDB","database_pass":"cupofcoffee", "database_name1":"secretDB02"}`),
				VersionId:    _s(`005`),
			}
			toUpdate := &secretsv1.SyncedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretKey.Name,
					Namespace:       secretKey.Namespace,
					ResourceVersion: resourceVersion,
				},
				Spec: secretsv1.SyncedSecretSpec{
					SecretMetadata: secretsv1.SecretMetadata{
						Name:      secretKey.Name,
						Namespace: secretKey.Namespace,
						CreationTimestamp: metav1.Time{
							Time: time_now,
						},
					},
					IAMRole:      _s("test"),
					AWSAccountID: _s("12345678910"),
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

		It("Should update the k8s secret object if the mapped AWS Secret changes with AWSAccountID", func() {
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

	Context("For a single SyncedSecret (using DataFrom) with AWSAccountID", func() {
		secretKey := types.NamespacedName{
			Name:      "secret-name-from-data",
			Namespace: TEST_NAMESPACE3,
		}

		resourceVersion := ""

		It("Should Create K8S Secrets for SyncedSecret (using Data) CRD with AWSAccountID", func() {
			MockSecretsOutput.SecretsValueOutput = &secretsmanager.GetSecretValueOutput{
				SecretString: _s(`{"DB_NAME":"secretDB","DB_PASS":"cupofcoffee"}`),
				VersionId:    _s(`006`),
			}

			toCreate := &secretsv1.SyncedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretKey.Name,
					Namespace:       secretKey.Namespace,
					ResourceVersion: resourceVersion,
				},
				Spec: secretsv1.SyncedSecretSpec{
					SecretMetadata: secretsv1.SecretMetadata{
						Name:      secretKey.Name,
						Namespace: secretKey.Namespace,
						CreationTimestamp: metav1.Time{
							Time: time_now,
						},
					},
					AWSAccountID: _s("12345678910"),
					IAMRole:      _s("test"),
					DataFrom: &secretsv1.DataFrom{
						SecretRef: &secretsv1.SecretRef{
							Name: _s("random/aws/secret005"),
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
			err := k8sClient.Create(context.Background(), toCreate)
			Expect(err).ToNot(HaveOccurred())

			fetchedSecret := &corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), secretKey, fetchedSecret)
				return k8serrors.IsNotFound(err)
			}, timeout, interval).Should(BeFalse())

			// we need to ensure that that secretExpect.Data is a subset of fetchedSecret.Data
			// the kubernetes client.go doesn't base64 values this is something that kubectl maybe does
			Expect(reflect.DeepEqual(fetchedSecret.Data, secretExpect.Data)).To(BeTrue())

			fetchedCfSecret := &secretsv1.SyncedSecret{}
			err = k8sClient.Get(context.Background(), secretKey, fetchedCfSecret)
			Expect(err).ToNot(HaveOccurred())
			resourceVersion = fetchedCfSecret.ResourceVersion

		})

		It("Should update k8s secret object if there is change in AwsSecret CRD with AWSAccountID", func() {
			MockSecretsOutput.SecretsValueOutput = &secretsmanager.GetSecretValueOutput{
				SecretString: _s(`{"DB_NAME":"secretDB02","DB_PASS":"cupofcoffee"}`),
				VersionId:    _s(`006`),
			}

			toUpdate := &secretsv1.SyncedSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            secretKey.Name,
					Namespace:       secretKey.Namespace,
					ResourceVersion: resourceVersion,
				},
				Spec: secretsv1.SyncedSecretSpec{
					SecretMetadata: secretsv1.SecretMetadata{
						Name:      secretKey.Name,
						Namespace: secretKey.Namespace,
						CreationTimestamp: metav1.Time{
							Time: time_now,
						},
					},
					IAMRole:      _s("test"),
					AWSAccountID: _s("12345678910"),
					DataFrom: &secretsv1.DataFrom{
						SecretRef: &secretsv1.SecretRef{
							Name: _s("random/aws/secret006"),
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

		It("Should update the k8s secret object if the mapped AWS Secret changes with AWSAccountID", func() {
			MockSecretsOutput.SecretsValueOutput = &secretsmanager.GetSecretValueOutput{
				SecretString: _s(`{"DB_PASS":"cupoftea3", "DB_NAME":"secretDB03"}`),
				VersionId:    _s(`007`),
			}

			MockSecretsOutput.SecretsPageOutput = &secretsmanager.ListSecretsOutput{
				SecretList: []*secretsmanager.SecretListEntry{
					{
						Name:            _s("random/aws/secret006"),
						LastChangedDate: _t(time_now.AddDate(0, 0, -1)),
						SecretVersionsToStages: map[string][]*string{
							"006": {
								_s("AWSPREVIOUS"),
							},
							"007": {
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
					"DB_NAME": []byte("secretDB03"),
					"DB_PASS": []byte("cupoftea3"),
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
