# vault-init

[![Build](https://github.com/pbar1/vault-init/actions/workflows/build.yml/badge.svg)](https://github.com/pbar1/vault-init/actions/workflows/build.yml)

Initializes HashiCorp Vault and saves the root token and keys in a provider of your choice.

```sh
docker pull ghcr.io/pbar1/vault-init
```

## Usage

```
Initialize an instance of `HashiCorp` Vault and persist the keys

Usage: vault-init [OPTIONS]

Options:
      --vault-addr <VAULT_ADDR>
          Address of the Vault server expressed as a URL and port [env: VAULT_ADDR=] [default: http://127.0.0.1:8200]
      --pgp-keys <PGP_KEYS>
          Specifies an array of PGP public keys used to encrypt the output unseal keys. Ordering is preserved. The keys must be base64-encoded from their original binary representation. The size of this array must be the same as `secret_shares`
      --root-token-pgp-key <ROOT_TOKEN_PGP_KEY>
          Specifies a PGP public key used to encrypt the initial root token. The key must be base64-encoded from its original binary representations
      --secret-shares <SECRET_SHARES>
          Specifies the number of shares to split the root key into [default: 1]
      --secret-threshold <SECRET_THRESHOLD>
          Specifies the number of shares required to reconstruct the root key. This must be less than or equal `secret_shares` [default: 1]
      --stored-shares <STORED_SHARES>
          Specifies the number of shares that should be encrypted by the HSM and stored for auto-unsealing. Currently must be the same as `secret_shares`
      --recovery-shares <RECOVERY_SHARES>
          Specifies the number of shares to split the recovery key into. This is only available when using Auto Unseal
      --recovery-threshold <RECOVERY_THRESHOLD>
          Specifies the number of shares required to reconstruct the recovery key. This must be less than or equal to recovery_shares. This is only available when using Auto Unseal
      --recovery-pgp-keys <RECOVERY_PGP_KEYS>
          Specifies an array of PGP public keys used to encrypt the output recovery keys. Ordering is preserved. The keys must be base64-encoded from their original binary representation. The size of this array must be the same as `recovery_shares`. This is only available when using Auto Unseal
  -h, --help
          Print help
```

<!-- Links -->

[1]: https://www.vaultproject.io/docs/commands#environment-variables
