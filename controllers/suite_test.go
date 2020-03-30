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
	"github.com/aws/aws-sdk-go/aws"
	awsclient "github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	secretsv1 "github.com/contentful-labs/k8s-secret-syncer/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var k8sManager ctrl.Manager
var testEnv *envtest.Environment

const TEST_NAMESPACE = "secret-sync-test"

var time_now = time.Now()

var Secretsoutput *secretsmanager.ListSecretsOutput

var MockSecretsOutput = mockSecretsOutput{}

type mockSecretsOutput struct {
	SecretsPageOutput  *secretsmanager.ListSecretsOutput
	SecretsValueOutput *secretsmanager.GetSecretValueOutput
}

type mockSecretsManagerClient struct {
	secretsmanageriface.SecretsManagerAPI
}

func _s(A string) *string {
	return &A
}

func _t(A time.Time) *time.Time {
	return &A
}

type mockRoleValidator struct{}

func (m *mockRoleValidator) IsWhitelisted(string, string) (bool, error) {
	return true, nil
}

// TODO this needs to be more dynamic when an update comes by
func (m *mockSecretsManagerClient) ListSecretsPages(input *secretsmanager.ListSecretsInput, fn func(*secretsmanager.ListSecretsOutput, bool) bool) error {
	fn(MockSecretsOutput.SecretsPageOutput, true)
	return nil
}

func (m *mockSecretsManagerClient) GetSecretValue(*secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	return MockSecretsOutput.SecretsValueOutput, nil
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	os.Setenv("POLL_INTERVAL_SEC", "3")

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = secretsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	syncPeriod := 2 * time.Second
	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:     scheme.Scheme,
		SyncPeriod: &syncPeriod,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sManager).ToNot(BeNil())

	smSvc := mockSecretsManagerClient{}

	// create the secret for testing
	MockSecretsOutput.SecretsPageOutput = &secretsmanager.ListSecretsOutput{
		SecretList: []*secretsmanager.SecretListEntry{
			{
				Name:            _s("random/aws/secret002"),
				LastChangedDate: _t(time_now.AddDate(0, 0, -2)),
				SecretVersionsToStages: map[string][]*string{
					"002": {
						_s("AWSCURRENT"),
					},
				},
			}, {
				Name:            _s("random/aws/secret003"),
				LastChangedDate: _t(time_now.AddDate(0, 0, -3)),
				SecretVersionsToStages: map[string][]*string{
					"005": {
						_s("AWSCURRENT"),
					},
					"003": {
						_s("AWSPREVIOUS"),
					},
				},
			},
		},
	}

	MockSecretsOutput.SecretsValueOutput = &secretsmanager.GetSecretValueOutput{
		SecretString: _s(`{"database_name":"secretDB","database_pass":"cupofcoffee", "database_name1":"secretDB02"}`),
		VersionId:    _s(`005`),
	}

	// mock the manager setup
	Retry5Cfg := request.WithRetryer(aws.NewConfig(), awsclient.DefaultRetryer{NumMaxRetries: 5})
	err = (&SyncedSecretReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SyncedSecret"),
		Sess:   session.New(Retry5Cfg),
		GetSMClient: func(IAMRole string) (secretsmanageriface.SecretsManagerAPI, error) {
			return &smSvc, nil
		},
		RoleValidator: &mockRoleValidator{},
		gauges:        map[string]prometheus.Gauge{},
		sync_state:    map[string]bool{},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// start the reconcilers
	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	// create a namespace for running our tests in
	toCreate := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: TEST_NAMESPACE,
		},
	}

	err = k8sClient.Create(context.Background(), toCreate)
	Expect(err).To(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())

	os.Unsetenv("POLL_INTERVAL_SEC")
})
