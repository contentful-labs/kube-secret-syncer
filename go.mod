module github.com/contentful-labs/k8s-secret-syncer

go 1.13

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/aws/aws-sdk-go v1.25.5
	github.com/go-logr/logr v0.1.0
	github.com/go-yaml/yaml v2.1.0+incompatible
	github.com/hashicorp/golang-lru v0.5.1
	github.com/huandu/xstrings v1.3.0 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0 // indirect
	go.uber.org/zap v1.9.1
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/controller-tools v0.2.4 // indirect
)
