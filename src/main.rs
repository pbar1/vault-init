#![warn(clippy::pedantic)]
#![allow(clippy::struct_excessive_bools)]
#![allow(clippy::module_name_repetitions)]

mod k8s;
mod vault;

use clap::Parser;
use tracing::error;
use tracing::info;

use crate::k8s::read_kube_secret;
use crate::k8s::write_kube_secret;
use crate::vault::StartInitRequest;
use crate::vault::UnsealRequest;
use crate::vault::VaultClient;

/// Initialize an instance of `HashiCorp` Vault and persist the keys
#[derive(Parser, Debug, Clone)]
pub struct Args {
    /// Address of the Vault server expressed as a URL and port
    #[clap(long, env = "VAULT_ADDR", default_value = "http://127.0.0.1:8200")]
    vault_addr: url::Url,

    /// Specifies an array of PGP public keys used to encrypt the output unseal
    /// keys. Ordering is preserved. The keys must be base64-encoded from their
    /// original binary representation. The size of this array must be the same
    /// as `secret_shares`
    #[clap(long)]
    pgp_keys: Option<Vec<String>>,

    /// Specifies a PGP public key used to encrypt the initial root token. The
    /// key must be base64-encoded from its original binary representations
    #[clap(long)]
    root_token_pgp_key: Option<String>,

    /// Specifies the number of shares to split the root key into
    #[clap(long, default_value_t = 1)]
    secret_shares: u8,

    /// Specifies the number of shares required to reconstruct the root key.
    /// This must be less than or equal `secret_shares`
    #[clap(long, default_value_t = 1)]
    secret_threshold: u8,

    /// Specifies the number of shares that should be encrypted by the HSM and
    /// stored for auto-unsealing. Currently must be the same as `secret_shares`
    #[clap(long)]
    stored_shares: Option<u8>,

    /// Specifies the number of shares to split the recovery key into. This is
    /// only available when using Auto Unseal
    #[clap(long)]
    recovery_shares: Option<u8>,

    /// Specifies the number of shares required to reconstruct the recovery key.
    /// This must be less than or equal to recovery_shares. This is only
    /// available when using Auto Unseal
    #[clap(long)]
    recovery_threshold: Option<u8>,

    /// Specifies an array of PGP public keys used to encrypt the output
    /// recovery keys. Ordering is preserved. The keys must be base64-encoded
    /// from their original binary representation. The size of this array must
    /// be the same as `recovery_shares`. This is only available when using Auto
    /// Unseal.
    #[clap(long)]
    recovery_pgp_keys: Option<Vec<String>>,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();
    setup_logging();
    let vault = VaultClient::new(args.vault_addr.clone());
    info!("Started vault-init");

    // Ensure init ------------------------------------------------------------

    info!("Checking initializtion status");
    let init_status = vault.read_init_status().await.map_err(|err| {
        error!("Failed checking initializtion status");
        err
    })?;
    if !init_status.initialized {
        info!("Vault is uninitialized");
        init_and_write_kube_secret(&vault, args.clone()).await?;
    }
    info!("Vault is already initialized");

    // Ensure unseal ----------------------------------------------------------

    info!("Checking seal status");
    let seal_status = vault.get_seal_status().await.map_err(|err| {
        error!("Failed checking seal status");
        err
    })?;
    if seal_status.sealed {
        info!("Vault is sealed");
        read_kube_secret_and_unseal(&vault).await?;
    }
    info!("Vault is already unsealed");

    Ok(())
}

async fn init_and_write_kube_secret(vault: &VaultClient, args: Args) -> anyhow::Result<()> {
    info!("Performing initialization");
    let init_request = StartInitRequest::from(args);
    let init_response = vault.start_init(&init_request).await.map_err(|err| {
        error!("Failed performing initialization");
        err
    })?;
    info!("Successfully initialized Vault");

    info!("Writing init response to Kubernetes secret");
    write_kube_secret("vault-init", &init_response)
        .await
        .map_err(|err| {
            error!("Failed writing init response to Kubernetes secret");
            err
        })?;
    info!("Successfully wrote init response to Kubernetes secret");

    Ok(())
}

async fn read_kube_secret_and_unseal(vault: &VaultClient) -> anyhow::Result<()> {
    info!("Reading init response from Kubernetes secret");
    let init_respose = read_kube_secret("vault-init").await.map_err(|err| {
        error!("Failed reading init response from Kubernetes secret");
        err
    })?;
    info!("Successfully read init response from Kubernetes secret");

    info!("Starting unseal key submission process");
    for (i, key) in init_respose.keys.iter().enumerate() {
        info!("Submitting unseal key #{i}");
        let unseal_request = UnsealRequest {
            key: Some(key.clone()),
            reset: false,
            migrate: false,
        };

        // TODO: Consider allowing continue instead of early return failure
        let unseal_response = vault
            .submit_unseal_key(&unseal_request)
            .await
            .map_err(|err| {
                error!("Failed submitting unseal key #{i}");
                err
            })?;
        if !unseal_response.sealed {
            info!("Successfully unsealed Vault");
            return Ok(());
        }
    }

    Err(anyhow::anyhow!("Unable to completely unseal Vault"))
}

fn setup_logging() {
    let format = tracing_subscriber::fmt::format();
    tracing_subscriber::fmt().event_format(format).init();
}
