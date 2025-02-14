package namespacevalidator

import (
	"fmt"

	awssecretsmanager "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/contentful-labs/kube-secret-syncer/pkg/k8snamespace"
)

type NamespaceValidator struct {
	nsCache k8snamespace.NamespaceGetter
}

func NewNamespaceValidator(nsCache k8snamespace.NamespaceGetter) *NamespaceValidator {
	return &NamespaceValidator{
		nsCache: nsCache,
	}
}

func (rv *NamespaceValidator) HasNamespaceType(secret awssecretsmanager.DescribeSecretOutput, namespace string) (bool, error) {
	ns, err := rv.nsCache.Get(namespace)
	if err != nil {
		return false, err
	}

	const nsTypeLabel = "k8s.contentful.com/namespace-type"
	const nsTypeTag = "k8s.contentful.com/namespace_type"

	label, labelFound := ns.Labels[nsTypeLabel]
	if !labelFound {
		return false, nil
	}

	for _, tag := range secret.Tags {
		if *tag.Key == fmt.Sprintf("%s/%s", nsTypeTag, label) && *tag.Value == "1" {
			return true, nil
		}
	}

	return false, nil
}
