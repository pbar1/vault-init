#![warn(clippy::pedantic)]
#![allow(clippy::struct_excessive_bools)]
#![allow(clippy::module_name_repetitions)]

mod k8s;
mod vault;

use clap::Parser;
use tracing::error;
use tracing::info;
use tracing_subscriber::prelude::*;

use crate::k8s::read_kube_secret;
use crate::k8s::write_kube_secret;
use crate::vault::StartInitRequest;
use crate::vault::UnsealRequest;
use crate::vault::VaultClient;

#[allow(clippy::doc_markdown)]
/// Initialize an instance of HashiCorp Vault and persist the keys
#[derive(Parser, Debug, Clone)]
#[clap(author, version, about)]
pub struct Args {
    /// Address of the Vault server expressed as a URL and port.
    #[clap(long, env = "VAULT_ADDR", default_value = "http://127.0.0.1:8200")]
    vault_addr: url::Url,

    /// Level directive for stdout logging.
    #[clap(long, env = "RUST_LOG", default_value = "info")]
    log_level: String,

    /// Array of PGP public keys used to encrypt the output unseal keys.
    ///
    /// Ordering is preserved. The keys must be base64-encoded from their
    /// original binary representation. The size of this array must be the same
    /// as `secret_shares`.
    #[clap(long, hide = true)]
    pgp_keys: Option<Vec<String>>,

    /// PGP public key used to encrypt the initial root token.
    ///
    /// The key must be base64-encoded from its original binary representations.
    #[clap(long, hide = true)]
    root_token_pgp_key: Option<String>,

    /// Number of shares to split the root key into.
    #[clap(long, default_value_t = 1)]
    secret_shares: u8,

    /// Number of shares required to reconstruct the root key.
    ///
    /// This must be less than or equal `secret_shares`.
    #[clap(long, default_value_t = 1)]
    secret_threshold: u8,

    /// Number of shares that should be encrypted by the HSM and stored for
    /// auto-unsealing.
    ///
    /// Currently must be the same as `secret_shares`.
    #[clap(long)]
    stored_shares: Option<u8>,

    /// Number of shares to split the recovery key into.
    ///
    /// This is only available when using Auto Unseal.
    #[clap(long)]
    recovery_shares: Option<u8>,

    /// Number of shares required to reconstruct the recovery key.
    ///
    /// This must be less than or equal to recovery_shares. This is only
    /// available when using Auto Unseal.
    #[clap(long)]
    recovery_threshold: Option<u8>,

    /// Array of PGP public keys used to encrypt the output recovery keys.
    ///
    /// Ordering is preserved. The keys must be base64-encoded from their
    /// original binary representation. The size of this array must be the same
    /// as `recovery_shares`. This is only available when using Auto Unseal.
    #[clap(long, hide = true)]
    recovery_pgp_keys: Option<Vec<String>>,
}

#[tokio::main(flavor = "current_thread")]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();
    setup_logging(&args.log_level)?;
    let vault = VaultClient::new(args.vault_addr.clone());
    info!(phase = "start", "Started process");

    // Ensure init ------------------------------------------------------------

    info!(phase = "init", "Checking status");
    let init_status = vault.read_init_status().await.map_err(|err| {
        error!(phase = "init", "Failed checking status");
        err
    })?;
    if init_status.initialized {
        info!(phase = "init", "Vault is already initialized");
    } else {
        info!(phase = "init", "Vault is uninitialized");
        init_and_write_kube_secret(&vault, args.clone()).await?;
    }

    // Ensure unseal ----------------------------------------------------------

    info!(phase = "unseal", "Checking status");
    let seal_status = vault.get_seal_status().await.map_err(|err| {
        error!(phase = "unseal", "Failed checking status");
        err
    })?;
    if seal_status.sealed {
        info!(phase = "unseal", "Vault is sealed");
        read_kube_secret_and_unseal(&vault).await?;
    } else {
        info!(phase = "unseal", "Vault is already unsealed");
    }

    Ok(())
}

async fn init_and_write_kube_secret(vault: &VaultClient, args: Args) -> anyhow::Result<()> {
    info!(phase = "init", "Performing initialization");
    let init_request = StartInitRequest::from(args);
    let init_response = vault.start_init(&init_request).await.map_err(|err| {
        error!(phase = "init", "Failed performing initialization");
        err
    })?;
    info!(phase = "init", "Successfully initialized Vault");

    info!(phase = "init", "Writing init data to K8s secret");
    write_kube_secret("vault-init", &init_response)
        .await
        .map_err(|err| {
            error!(phase = "init", "Failed writing init data to K8s secret");
            err
        })?;
    info!(phase = "init", "Successfully wrote init data to K8s secret");

    Ok(())
}

async fn read_kube_secret_and_unseal(vault: &VaultClient) -> anyhow::Result<()> {
    info!(phase = "unseal", "Reading init data from K8s secret");
    let init_response = read_kube_secret("vault-init").await.map_err(|err| {
        error!(phase = "unseal", "Failed reading init data from K8s secret");
        err
    })?;
    info!(
        phase = "unseal",
        "Successfully read init data from K8s secret"
    );

    info!(phase = "unseal", "Starting key submission process");
    for (i, key) in init_response.keys.iter().enumerate() {
        info!(phase = "unseal", "Submitting key #{i}");
        let unseal_request = UnsealRequest {
            key: Some(key.clone()),
            reset: false,
            migrate: false,
        };

        let Ok(unseal_response) = vault.submit_unseal_key(&unseal_request).await else {
            error!(phase = "unseal", "Failed submitting key #{i}");
            continue;
        };
        if !unseal_response.sealed {
            info!(phase = "unseal", "Successfully unsealed Vault");
            return Ok(());
        }
    }

    Err(anyhow::anyhow!("Unable to completely unseal Vault"))
}

fn setup_logging(log_level: &str) -> anyhow::Result<()> {
    let fmt_filter = tracing_subscriber::filter::EnvFilter::builder()
        .with_default_directive(tracing_subscriber::filter::LevelFilter::INFO.into())
        .parse_lossy(log_level);
    let fmt_layer = tracing_subscriber::fmt::layer().with_filter(fmt_filter);

    let subscriber = tracing_subscriber::Registry::default().with(fmt_layer);

    tracing::subscriber::set_global_default(subscriber)?;

    Ok(())
}
