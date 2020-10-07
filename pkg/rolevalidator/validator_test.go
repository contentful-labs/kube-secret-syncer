package rolevalidator

import (
	"github.com/contentful-labs/kube-secret-syncer/pkg/k8snamespace"
	v1 "k8s.io/api/core/v1"
	"testing"
)

type mockARNGetter struct{}

func (*mockARNGetter) GetARN(role string) (string, error) { return role, nil }

type mockNSGetter struct {
	annotationName string
	annotation     string
}

func (m *mockNSGetter) Get(string) (*v1.Namespace, error) {
	ns := &v1.Namespace{}
	ns.Annotations = map[string]string{
		m.annotationName: m.annotation,
	}

	return ns, nil
}

type mockUnannottatedNSGetter struct {
	annotation string
}

func (m *mockUnannottatedNSGetter) Get(string) (*v1.Namespace, error) {
	ns := &v1.Namespace{}
	ns.Annotations = map[string]string{}

	return ns, nil
}

func TestIsWhitelisted(t *testing.T) {
	const annotationName = "iam.amazonaws.com/allowed-roles"

	testCases := []struct {
		name          string
		role          string
		expectAllowed bool
		getter        k8snamespace.NamespaceGetter
	}{
		{
			name:          "namespace has annotation whitelisting specified role",
			role:          "role1",
			expectAllowed: true,
			getter: &mockNSGetter{
				annotationName: annotationName,
				annotation:     "[\"role1\"]",
			},
		},
		{

			name:          "namespace has several annotations and is whitelisting specified role",
			role:          "role2",
			expectAllowed: true,
			getter: &mockNSGetter{
				annotationName: annotationName,
				annotation:     "[\"role1\", \"role2\"]",
			},
		},
		{
			name:          "namespace has an annotations but is not whitelisting specified role",
			role:          "role1",
			expectAllowed: false,
			getter: &mockNSGetter{
				annotationName: annotationName,
				annotation:     "[\"role2\"]",
			},
		},
		{
			name:          "namespace has no annotation, role is specified",
			role:          "role1",
			expectAllowed: true,
			getter:        &mockUnannottatedNSGetter{},
		},
		{
			name:          "namespace has an annotation, role is not specified",
			role:          "",
			expectAllowed: false,
			getter: &mockNSGetter{
				annotationName: annotationName,
				annotation:     "[\"role2\"]",
			},
		},
	}

	for _, test := range testCases {
		rv := NewRoleValidator(&mockARNGetter{}, test.getter, annotationName)
		isAllowed, err := rv.IsWhitelisted(test.role, "test")
		if err != nil {
			t.Errorf("got error with role %s: %s", test.role, err)
		}
		if isAllowed != test.expectAllowed {
			t.Errorf("failed %s", test.name)
		}
	}
}
