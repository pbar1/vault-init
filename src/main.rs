#![warn(clippy::pedantic)]
#![allow(clippy::struct_excessive_bools)]
#![allow(clippy::module_name_repetitions)]

mod config;
mod save;
mod vault;

use std::path::PathBuf;

use anyhow::bail;
use clap::Parser;
use tracing::debug;
use tracing::error;
use tracing::info;
use tracing_subscriber::prelude::*;

use crate::config::Config;
use crate::vault::models::auth::token::PostRevokeRequest;
use crate::vault::models::sys::generate_root::PostGenerateRootAttemptRequest;
use crate::vault::models::sys::generate_root::PostGenerateRootUpdateRequest;
use crate::vault::models::sys::init::PostInitRequest;
use crate::vault::models::sys::unseal::PostUnsealRequest;
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

    /// Config file.
    #[clap(long, short, default_value = "vault-init.hcl")]
    config: PathBuf,

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

    let config = args.config.clone();
    debug!(phase = "start", ?config, "Reading config file");
    let buf = tokio::fs::read(config).await?;
    let config: Config = hcl::from_slice(&buf)?;
    debug!(phase = "start", ?config, "Read config file");

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
        init_and_save(&vault, args.clone(), &config).await?;
    }

    // Ensure unseal ----------------------------------------------------------

    info!(phase = "unseal", "Checking status");
    let seal_status = vault.get_seal_status().await.map_err(|err| {
        error!(phase = "unseal", "Failed checking status");
        err
    })?;
    if seal_status.sealed {
        info!(phase = "unseal", "Vault is sealed");
        load_and_unseal(&vault, &config).await?;
    } else {
        info!(phase = "unseal", "Vault is already unsealed");
    }

    // Rotate root ------------------------------------------------------------

    rotate_root(&vault, &config).await?;

    Ok(())
}

async fn init_and_save(vault: &VaultClient, args: Args, config: &Config) -> anyhow::Result<()> {
    info!(phase = "init", "Performing initialization");
    let init_request = PostInitRequest::from(args);
    let init_response = vault.start_init(&init_request).await.map_err(|err| {
        error!(phase = "init", "Failed performing initialization");
        err
    })?;
    info!(phase = "init", "Successfully initialized Vault");

    info!(phase = "init", "Writing init data to save methods");
    config
        .save_method
        .save_init_all(&init_response)
        .await
        .map_err(|err| {
            error!(phase = "init", "Failed writing init data to save methods");
            err
        })?;
    info!(
        phase = "init",
        "Successfully wrote init data to save methods"
    );

    Ok(())
}

async fn load_and_unseal(vault: &VaultClient, config: &Config) -> anyhow::Result<()> {
    info!(phase = "unseal", "Reading init data from save methods");
    let init_response = config.save_method.load_init_all().await.map_err(|err| {
        error!(
            phase = "unseal",
            "Failed reading init data from save methods"
        );
        err
    })?;
    info!(
        phase = "unseal",
        "Successfully read init data from save methods"
    );

    info!(phase = "unseal", "Starting key submission process");
    for (i, key) in init_response.keys.iter().enumerate() {
        info!(phase = "unseal", "Submitting key #{i}");
        let unseal_request = PostUnsealRequest {
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

async fn rotate_root(vault: &VaultClient, config: &Config) -> anyhow::Result<()> {
    let phase = "rotate_root";

    // TODO: Consider if an in-progress genroot is actually a failure condition or
    // if it could be resumed Check if generate root is already in progress and
    // fail if so
    info!(phase = "rotate_root", "Checking generate root status");
    let genroot_status = vault.get_generate_root_attempt().await.map_err(|err| {
        error!(phase, "Failed checking generate root status");
        err
    })?;
    if genroot_status.started {
        let msg = "Generate root process is already in progress";
        error!(phase, msg);
        bail!(msg);
    }
    info!(phase, "Generate root process is not in progress");

    // Load init response (containing root token)
    info!(phase, "Reading init data from save methods");
    let init_response = config.save_method.load_init_all().await.map_err(|err| {
        error!(phase, "Failed reading init data from save methods");
        err
    })?;
    info!(phase, "Successfully read init data from save methods");

    // Start generate root process
    info!(phase, "Starting generate root process");
    let genroot_start_request = PostGenerateRootAttemptRequest { pgp_key: None };
    let genroot_start_response = vault
        .post_generate_root_attempt(&genroot_start_request)
        .await
        .map_err(|err| {
            error!(phase, "Failed starting generate root process");
            err
        })?;
    info!(phase, "Successfully started generate root process");

    let nonce = genroot_start_response.nonce;
    let otp = genroot_start_response.otp;

    for (i, key) in init_response.keys.iter().enumerate() {
        info!(phase, "Submitting key #{i}");
        let genroot_update_request = PostGenerateRootUpdateRequest {
            key: key.clone(),
            nonce: nonce.clone(),
        };

        let Ok(genroot_update_response) = vault.post_generate_root_update(&genroot_update_request).await else {
            error!(phase, "Failed submitting key #{i}");
            continue;
        };
        if genroot_update_response.complete {
            info!(phase, "Successfully generated root");

            // FIXME: Save

            // FIXME:
            info!(phase, "Revoking previous root token");
            let vault = vault.with_token(init_response.root_token.into());
            vault.post_auth_token_revoke_self().await?;
            info!(phase, "Successfully revoked previous root token");

            return Ok(());
        }
    }

    Ok(())
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
