# vault-init

[![Go Report Card](https://goreportcard.com/badge/github.com/pbar1/vault-init)](https://goreportcard.com/report/github.com/pbar1/vault-init)

Initializes HashiCorp Vault and saves the root token and keys in a provider of your choice.

```sh
docker pull ghcr.io/pbar1/vault-init
```

## Usage

[Vault environment variables][1] (such as `VAULT_ADDR`, `VAULT_CACERT`, etc) are
recognized.

```
Usage of vault-init:
      --file-path string                         Path on disk to save the Vault init result (default "vault-init.json")
      --kube-secret-annotations stringToString   Labels to add to the Kubernetes secret (default [])
      --kube-secret-labels stringToString        Labels to add to the Kubernetes secret (default [])
      --kube-secret-name string                  Name of the Kubernetes secret to save Vault init result (default "vault-init")
      --kube-secret-namespace string             Namespace to create the Kubernetes secret in. Defaults to the current namespace.
      --kubeconfig string                        Path to Kubeconfig to use when saving the Kubernetes secret. If unset, will use inClusterConfig.
      --log-format string                        Log output format (default "json")
      --log-level string                         Log level (default "info")
      --recovery-shares int                      Recovery shares (default 1)
      --recovery-threshold int                   Recovery threshold (default 1)
  -s, --save string                              How to save the Vault init result. One of: file|kube-secret (default "file")
      --timeout duration                         Time to wait before failing the Vault init process (default 10m0s)
  -v, --version                                  Print version information
```

### Save to a file

By default, vault-init saves to a file in the current directory called
`vault-init.json`. This can be overridden with a flag.

```sh
vault-init --file-path="/vault-init.json"
```

### Save to Kubernetes secret

By default, vault-init will save to a Kubernetes secret called `vault-init`.
This can be overridden with a flag. If a secret with this name already exists,
vault-init will _not_ overwrite it, but rather save to a new secret with the
name as a prefix. If running in Kubernetes, the secret will be created in the
same namespace as the pod; the namespace may also be specified with a flag. If
neither of these are found, the secret will attempt to be created in the default
namespace. Labels and annotations may be added.

```sh
vault-init                                              \
  --save=kube-secret                                    \
  --kubeconfig="${HOME}/.kube/config"                   \
  --kube-secret-name=my-secret                          \
  --kube-secret-namespace=my-namespace                  \
  --kube-secret-labels="my-label-1=foo,my-label-2=bar"
```

[1]: https://www.vaultproject.io/docs/commands#environment-variables
