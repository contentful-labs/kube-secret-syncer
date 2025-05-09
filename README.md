# Kube-secret-syncer

Kube-secret-syncer is a [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) developed
using [the Kubebuilder framework](https://github.com/kubernetes-sigs/kubebuilder) that keeps the values of Kubernetes
Secrets synchronised to secrets in AWS Secrets Manager.

This mapping is described in a Kubernetes custom resource called SyncedSecret. The operator polls AWS for changes in
secret values at regular intervals, and upon detecting changes, updates the Kubernetes Secrets.

__WARNING__: updating the value of a secret in AWS SecretsManager will override secrets in Kubernetes, therefore
 can be a destructive action.

## Comparison to existing projects

Kube-secret-syncer is similar to other projects such as:
 * [Kubernetes external secrets](https://github.com/godaddy/kubernetes-external-secrets)
 * [AWS secret operator](https://github.com/mumoshu/aws-secret-operator)

Kube-secret-syncer improves on this approach: 
 * uses [caching](#caching) to only retrieve the value of secrets when they have changed, substantially reducing
 [costs](https://aws.amazon.com/secrets-manager/pricing/) when syncing a large number of secrets.
 * enables sophisticated access control to secrets in AWS SecretsManager using IAM roles - see our
 [security model](#security-model)
 * supports [templated fields](#templated-fields) for Kubernetes secrets - enabling the use of values from multiple AWS
 SecretsManager secrets in one Kubernetes Secret

## Defining mapping between an AWS SecretsManager secret and a Kubernetes Secret

The following resource will map the AWS Secret `secretsyncer/secret/sample` to the Kubernetes Secret
`demo-service-secret`, and copy all key-value pairs from the AWS SecretsManager secret to the  Kubernetes secret For
 this example, the AWS SecretsManager secret needs to be a valid JSON consisting only of key-value pairs.

To access the secrets, kube-secret-syncer will assume the role `iam_role` to poll the secret. Note: that role must be
 assumed by the Kubernetes cluster/node where the operator runs, eg part of the kube2iam annotation on the namespace.

```yaml
apiVersion: secrets.contentful.com/v1
kind: SyncedSecret
metadata:
  name: demo-service-secret
  namespace: kube-secret-syncer
spec:
  IAMRole: iam_role
  dataFrom:
    secretRef:
      name: secretsyncer/secret/sample
```

If you only need to retrieve select keys in a single AWS secret, or multiple keys from different AWS secrets, you
can use the following syntax:

```yaml
apiVersion: secrets.contentful.com/v1
kind: SyncedSecret
metadata:
  name: demo-service-secret
  namespace: kube-secret-syncer
spec:
  IAMRole: iam_role
  data:
    # Sets the key mysql_user for the Kubernetes Secret "demo-service-secret" to "contentful"
    - name: mysql_user
      value: "contentful"
    # Takes the value for key "password" from the Secrets Manager secret "mysql", assign to the
    # key "mysql_pw" of the Kubernetes secret "demo-service-secret"
    - name: mysql_pw
      valueFrom:
        secretKeyRef:
          name: mysql
          key: password
    - name: datadog_access_key
      valueFrom:
        secretKeyRef:
          name: datadog
          key: access_key
```

You can also chose to store non-JSON values in AWS Secret Manager, which might be more convenient for data such
as certificates.

```yaml
apiVersion: secrets.contentful.com/v1
kind: SyncedSecret
metadata:
  name: demo-service-secret
  namespace: kube-secret-syncer
spec:
  IAMRole: iam_role
  data:
    # Sets the key ssl-certificate for the Kubernetes Secret "demo-service-secret"
    # to the value of the secret "apache/ssl-cert"
    - name: ssl-certificate
      valueFrom:
        secretRef:
          name: apache/ssl-cert
```

## [Templated fields](#templated-fields)

Kube-secret-syncer supports templated fields. This allows, for example, to iterate over a list of secrets that
share the same tag, to output a configuration file, such as in the following example:

```yaml
apiVersion: secrets.contentful.com/v1
kind: SyncedSecret
metadata:
  name: pgbouncer.txt
  namespace: kube-secret-syncer
spec:
  IAMRole: iam_role
  data:
    - name: pgbouncer-hosts
      valueFrom:
        template: |
          {{- $cfg := "" -}}
          {{- range $secretID, $_ := filterByTagKey .Secrets "tag1" -}}
            {{- $secretValue := getSecretValueMap $secretID -}}
            {{- $cfg = printf "%shost=%s user=%s password=%s\n" $cfg $secretValue.host $secretValue.user $secretValue.password -}}
          {{- end -}}
          {{- $cfg -}}
```

This iterates over all secrets kube-secret-syncer has access to, select those that have the tag "tag1" set,
and for each of these, add a configuration line to $cfg. $cfg is then assigned to the key "pgbouncer-hosts" of
the Kubernetes secret pgbouncer.txt.

The template is a [Go template](https://golang.org/pkg/text/template/) with the following elements defined:
 * `.Secrets` - a map containing all listed secrets (without their value)
 * `filterByTagKey` - a helper function to filter the secrets by tag
 * `getSecretValue` - will retrieve the raw value of a Secret in SecretsManager, given its secret ID
 * `getSecretValueMap` - will retrieve the value of a Secret in SecretsManager that contains a JSON, given its secret ID -
 as a map

## [Caching](#caching)

Kube-secret-syncer maintains both the list of AWS Secrets as well as their values in cache. The list is updated every
`POLL_INTERVAL_SEC`, and values are retrieved whenever their VersionID changed.

## [Security model](#security-model)

Kube-secret-syncer relies on the AWS Go SDK to communicate with AWS - and supports the different ways of
authenticating to AWS described in the [AWS Go SDK documentation](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials).

When synced secrets have an IAMRole field defined, kube-secret-syncer will assume that role before retrieving the
secret. This implies that the role specified by IAMRole can be assumed by the default role kube-secret-syncer uses.

To ensure a specific namespace only has access to the secrets it needs to, kube-secret-syncer will use the
"iam.amazonaws.com/allowed-roles" annotation on the namespace (originally used by kube2iam) to validate that this
role can be assumed for that namespace.

The secret synchronisation will be allowed if:
 * the annotation is set on the namespace and contains the secrets IAMRole
 * no annotation is set on the namespace and the secret has a IAMRole set
 * no annotation is set on the namespace and the secret has no IAMRole set

The secret sync will be denied if:
 * the annotation is set on the namespace and does not contains the secrets IAMRole
 * the annotation is set on the namespace and the secret has no IAMRole set

## Configuration

Kube-secret-syncer supports the following environment variables:

 * `POLL_INTERVAL_SEC`: how often the list of secrets in cache is refreshed (default: `300`)
 * `SYNC_INTERVAL_SEC`: how often we will write to a Kubernetes secret (default: `120`)
 * `NS_ANNOTATION`: the annotation on the namespace that contains a list of IAM roles kube-secret-syncer is allowed
  to assume (default: `iam.amazonaws.com/allowed-roles`)
 * `METRICS_LISTEN`: what interface/port the metrics server shoult listen on (default: `:8080`)

Note  - when a secret in Secrets Manager is updated, the secret in Kubernetes will not be updated
until both the list of secrets is refreshed AND the sync_interval expires - therefore it might take up
to POLL_INTERVAL_SEC + SYNC_INTERVAL_SEC.

## Local development

Please refer to the [local development documentation](docs/development.md).

## Examples

See [sample configurations](config/samples) and [deployment example](examples).
