# vault-init

Initializes HashiCorp Vault and saves the root token and keys in a provider of your choice.

## Usage

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
