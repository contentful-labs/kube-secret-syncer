package rolevalidator

import (
	"encoding/json"
	"github.com/contentful-labs/k8s-secret-syncer/pkg/iam"
	"github.com/contentful-labs/k8s-secret-syncer/pkg/k8snamespace"
	"github.com/pkg/errors"
)

type RoleValidator struct {
	arnGetter      iam.ARNGetter
	nsCache        k8snamespace.NamespaceGetter
	annotationName string
}

func NewRoleValidator(getter iam.ARNGetter, nsCache k8snamespace.NamespaceGetter, annotationName string) *RoleValidator {
	return &RoleValidator{
		arnGetter: getter,
		nsCache:   nsCache,
	}
}

func (rv *RoleValidator) IsWhitelisted(role, namespace string) (bool, error) {
	ns, err := rv.nsCache.Get(namespace)
	if err != nil {
		return false, err
	}

	annotation, annotationFound := ns.Annotations[rv.annotationName]
	if !annotationFound { // The namespace does not use kube2iam. We should not specify an IAMRole, but we dont prevent it
		return true, nil
	}

	if role == "" { // Secrets must have a role defined if an annotation is found
		return false, nil
	}

	return rv.isRoleAllowed(role, annotation)
}

func (rv *RoleValidator) isRoleAllowed(role, kube2iamAnnotation string) (bool, error) {
	roleArn, err := rv.arnGetter.GetARN(role)
	if err != nil {
		return false, errors.WithMessagef(err, "failed getting ARN for role %s", role)
	}

	var allowedRoles []string
	if err := json.Unmarshal([]byte(kube2iamAnnotation), &allowedRoles); err != nil {
		return false, err
	}

	for _, allowedRole := range allowedRoles {
		allowedRoleArn, err := rv.arnGetter.GetARN(allowedRole)
		if err != nil {
			return false, errors.WithMessagef(err, "failed getting ARN for role %s", allowedRole)
		}

		if roleArn == allowedRoleArn {
			return true, nil
		}
	}

	return false, nil
}
