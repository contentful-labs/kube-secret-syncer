# Local Development

## Requires

* Kubebuilder ([install instructions](https://book.kubebuilder.io/quick-start.html#installation))
* Kustomize ([install instructions](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md))
* Kind ([install instructions](https://kind.sigs.k8s.io/docs/user/quick-start/#installation))
* Docker
* go

Make sure you install these listed tools via go and not your local package manager, this can lead to path issues.

## Local Development

### Development flow

To develop locally you'll need a kubernetes cluster installed. *kind* can setup a local cluster within docker.
After you have *kind* installed, follow the *kind*
[Creating a Cluster](https://kind.sigs.k8s.io/docs/user/quick-start/#creating-a-cluster) and
[Interacting With Your Cluster](https://kind.sigs.k8s.io/docs/user/quick-start/#interacting-with-your-cluster)
instructions to get setup with a working local cluster.

You should now be able to run `make docker-build kind` and have the controller run in your local cluster. The
namespace and CRD for use with kind is available in
[config/samples/secrets_v1_syncedsecret.yaml](../config/samples/secrets_v1_awssecret.yaml). You can apply these using
`kubectl apply -f config/samples/secrets_v1_syncedsecret.yaml --context kubernetes-admin@kind`.

Additionaly, to ensure all tests pass, run `make tests`.

Here's what your flow would look like after 
1. do code changes
2. `make docker-build kind`

#### AWS Credentials

When you run `make kind` above, a config map is added to your local kind cluster which contains the aws credentials
from the `preview` profile on the host system. To use a different profile set the `AWS_KIND_PROFILE` make variable.
Eg `make AWS_KIND_PROFILE=staging docker-build kind`
