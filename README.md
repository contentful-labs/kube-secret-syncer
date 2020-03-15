# K8s-secret-syncer

K8s-Secrets-syncer is a [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) developed
using [the Kubebuilder framework](https://github.com/kubernetes-sigs/kubebuilder) that maps secrets stored in AWS Secret
Manager to secrets in Kubernetes.

When a secret is updated in Secrets Manager, k8s-secret-syncer will automatically update the value of the mapped secret
in Kubernetes. This might take a few minutes, depending on the configured polling interval.

__WARNING__: updating the value of a secret can be a destructive action.

## Comparison to existing projects

This is similar in concept to other projects such as:
 * [Kubernetes external secrets](https://github.com/godaddy/kubernetes-external-secrets)
 * [AWS secret operator](https://github.com/mumoshu/aws-secret-operator)
 
however both these projects poll the value of each mapped secret at a regular interval. When mapping a large number
of secrets, with a large number of namespaces / clusters, this can get quite expensive.

K8s-secret-syncer: 
 * only retrieves the value of secrets when they have been updated. This is done by maintaining a local
cache, and using the AWS Secret versionID to verify if the secret has changed.
 * restricts what secrets can be accessed using IAM roles
 * can use templates to generate Kubernetes secrets fields - enabling for example looping over multiple AWS Secrets

## Defining a SecretManager secret to map

The following resource will map the AWS Secret ```secretsyncer/secret/sample``` to the Kubernetes Secret
```demo-service-secret```, and copy all key/values of the secrets-manager to Kubernetes.

k8s-secret-syncer will assume the role ```iam_role``` to poll the secret. Note: that role must be part of the kube2iam
annotation on the namespace.

```
apiVersion: secrets.contentful.com/v1
kind: SyncedSecret
metadata:
  name: demo-service-secret
  namespace: secret-sync
spec:
  IAMRole: iam_role
  dataFrom:
    secretMapRef:
      name: secretsyncer/secret/sample
```

If you only need to retrieve a few keys in an AWS secret, or multiple keys from different AWS secrets, you
can use the following syntax:


```
apiVersion: secrets.contentful.com/v1
kind: SyncedSecret
metadata:
  name: demo-service-secret
  namespace: secret-sync
spec:
  IAMRole: iam_role
  data:
    # Sets the key mysql_user for the Kubernetes Secret "demo-service-secret" to "contentful"
    - name: mysql_user
      value: "contentful"

    # Takes the value for key "password" from the Secrets Manager secret "mysql", assign to the
    # key "mysql_pw" of the Kubernetes secret "demo-service-secret"
    - name: mysql_pw
      secretKeyRef:
        name: mysql
        key: password
    - name: datadog_access_key
      secretKeyRef:
        name: datadog
        key: access_key
```

Finally, k8s-secret-syncer supports complex templating:

```
apiVersion: secrets.contentful.com/v1
kind: SyncedSecret
metadata:
  name: pgbouncer.txt
  namespace: secret-sync
spec:
  IAMRole: iam_role
  data:
    - name: pgbouncer-hosts
      valueFrom:
        template: |
          {{- $cfg := "" -}}
          {{- range $secretName, $_ := filterByTagKey .Secrets "tag1" -}}
            {{- $secretValue := getSecretValue $secretName -}}
            {{- $cfg = printf "%shost=%s user=%s password=%s\n" $cfg $secretValue.host $secretValue.user $secretValue.password -}}
          {{- end -}}
          {{- $cfg -}}
```

This will iterate over all secrets k8s-secret-syncer has access to, select those that have the tag "tag1" set,
and for each of these, add a configuration line to $cfg. $cfg is then assigned to the key "pgbouncer-hosts" of
the Kubernetes secret pgbouncer.txt.

### Local development

Please refer to [local-development documentation](docs/development.md)

### Examples

See [sample configurations](config/samples)
