package rolevalidator

import (
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

func TestIsWhitelisted(t *testing.T) {
	testCases := []struct {
		role, annotation string
		expectAllowed    bool
	}{
		{
			role:          "role1",
			annotation:    "[\"role1\"]",
			expectAllowed: true,
		},
		{
			role:          "role2",
			annotation:    "[\"role1\", \"role2\"]",
			expectAllowed: true,
		},
		{
			role:          "role1",
			annotation:    "[\"role2\"]",
			expectAllowed: false,
		},
	}

	mng := &mockNSGetter{}
	rv := NewRoleValidator(&mockARNGetter{}, mng)
	for _, test := range testCases {
		mng.annotation = test.annotation
		isAllowed, err := rv.IsWhitelisted(test.role, test.annotation)
		if err != nil {
			t.Errorf("got error with annotation %s, role %s: %s", test.annotation, test.role, err)
		}
		if isAllowed != test.expectAllowed {
			t.Errorf("annotation %s, role %s, expected %t, got %t", test.annotation, test.role, test.expectAllowed, isAllowed)
		}
	}
}
