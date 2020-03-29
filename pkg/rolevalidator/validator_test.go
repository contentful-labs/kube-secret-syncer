package rolevalidator

import (
	"github.com/contentful-labs/k8s-secret-syncer/pkg/k8snamespace"
	v1 "k8s.io/api/core/v1"
	"testing"
)

type mockARNGetter struct{}

func (*mockARNGetter) GetARN(role string) (string, error) { return role, nil }

type mockNSGetter struct {
	annotation string
}

func (m *mockNSGetter) Get(nsName string) (*v1.Namespace, error) {
	ns := &v1.Namespace{}
	ns.Annotations = map[string]string{
		annotationName: m.annotation,
	}

	return ns, nil
}

type mockUnannottatedNSGetter struct {
	annotation string
}

func (m *mockUnannottatedNSGetter) Get(nsName string) (*v1.Namespace, error) {
	ns := &v1.Namespace{}
	ns.Annotations = map[string]string{}

	return ns, nil
}

func TestIsWhitelisted(t *testing.T) {
	testCases := []struct {
		role          string
		expectAllowed bool
		getter        k8snamespace.NamespaceGetter
	}{
		{
			role:          "role1",
			expectAllowed: true,
			getter: &mockNSGetter{
				annotation: "[\"role1\"]",
			},
		},
		{
			role:          "role2",
			expectAllowed: true,
			getter: &mockNSGetter{
				annotation: "[\"role1\", \"role2\"]",
			},
		},
		{
			role:          "role1",
			expectAllowed: false,
			getter: &mockNSGetter{
				annotation: "[\"role2\"]",
			},
		},
		{
			role:          "role1",
			expectAllowed: true,
			getter:        &mockUnannottatedNSGetter{},
		},
		{
			role:          "",
			expectAllowed: false,
			getter: &mockNSGetter{
				annotation: "[\"role2\"]",
			},
		},
	}

	for _, test := range testCases {
		rv := NewRoleValidator(&mockARNGetter{}, test.getter, "iam.amazonaws.com/allowed-roles")
		isAllowed, err := rv.IsWhitelisted(test.role, "test")
		if err != nil {
			t.Errorf("got error with role %s: %s", test.role, err)
		}
		if isAllowed != test.expectAllowed {
			t.Errorf("role %s, expected %t, got %t", test.role, test.expectAllowed, isAllowed)
		}
	}
}
