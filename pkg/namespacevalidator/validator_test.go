package namespacevalidator

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awssecretsmanager "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/contentful-labs/kube-secret-syncer/pkg/k8snamespace"
	v1 "k8s.io/api/core/v1"
)

type mockNSGetter struct {
	namespaceLabel string
}

func (m *mockNSGetter) Get(string) (*v1.Namespace, error) {
	ns := &v1.Namespace{}
	ns.Labels = map[string]string{
		"k8s.contentful.com/namespace-type": m.namespaceLabel,
	}

	return ns, nil
}

func TestHasNamespaceType(t *testing.T) {
	testCases := []struct {
		name                   string
		secret                 awssecretsmanager.DescribeSecretOutput
		namespace              string
		getter                 k8snamespace.NamespaceGetter
		expectHasNamespaceType bool
	}{
		{
			name: "secret has correct namespace tag",
			secret: awssecretsmanager.DescribeSecretOutput{
				Tags: []*awssecretsmanager.Tag{
					{
						Key:   aws.String("k8s.contentful.com/namespace_type/some-namespace"),
						Value: aws.String("1"),
					},
				},
			},
			namespace: "some-namespace",
			getter: &mockNSGetter{
				namespaceLabel: "some-namespace",
			},
			expectHasNamespaceType: true,
		},
		{
			name: "secret has incorrect namespace tag",
			secret: awssecretsmanager.DescribeSecretOutput{
				Tags: []*awssecretsmanager.Tag{
					{
						Key:   aws.String("k8s.contentful.com/namespace_type/some-other-namespace"),
						Value: aws.String("1"),
					},
				},
			},
			namespace: "some-namespace",
			getter: &mockNSGetter{
				namespaceLabel: "some-namespace",
			},
			expectHasNamespaceType: false,
		},
		{
			name: "secret has incorrect namespace tag",
			secret: awssecretsmanager.DescribeSecretOutput{
				Tags: []*awssecretsmanager.Tag{},
			},
			namespace: "some-namespace",
			getter: &mockNSGetter{
				namespaceLabel: "some-namespace",
			},
			expectHasNamespaceType: false,
		},
	}

	for _, test := range testCases {
		rv := NewNamespaceValidator(test.getter)
		isAllowed, err := rv.HasNamespaceType(test.secret, test.namespace)
		if err != nil {
			t.Errorf("got error with namespace %s: %s", test.namespace, err)
		}
		if isAllowed != test.expectHasNamespaceType {
			t.Errorf("failed %s", test.name)
		}
	}
}
