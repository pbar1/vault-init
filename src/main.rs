#![warn(clippy::pedantic)]

use clap::Parser;
use serde::Deserialize;
use serde::Serialize;
use tracing::error;
use tracing::info;
use tracing::trace;

/// Initialize an instance of HashiCorp Vault and persist the keys
#[derive(Parser, Debug)]
struct Args {
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

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
struct ReadInitStatusResponse {
    pub initialized: bool,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
struct StartInitRequest {
    pgp_keys: Option<Vec<String>>,
    root_token_pgp_key: Option<String>,
    secret_shares: u8,
    secret_threshold: u8,
    stored_shares: Option<u8>,
    recovery_shares: Option<u8>,
    recovery_threshold: Option<u8>,
    recovery_pgp_keys: Option<Vec<String>>,
}

impl From<Args> for StartInitRequest {
    fn from(args: Args) -> Self {
        Self {
            pgp_keys: args.pgp_keys,
            root_token_pgp_key: args.root_token_pgp_key,
            secret_shares: args.secret_shares,
            secret_threshold: args.secret_threshold,
            stored_shares: args.stored_shares,
            recovery_shares: args.recovery_shares,
            recovery_threshold: args.recovery_threshold,
            recovery_pgp_keys: args.recovery_pgp_keys,
        }
    }
}

/// An object including the (possibly encrypted, if `pgp_keys` was provided)
/// root keys, base 64 encoded root keys and initial root token
#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
struct StartInitResponse {
    pub keys: Vec<String>,
    pub keys_base64: Vec<String>,
    pub root_token: String,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
struct SealStatusResponse {
    pub r#type: String,
    pub initialized: bool,
    pub sealed: bool,
    pub t: i64,
    pub n: i64,
    pub progress: i64,
    pub nonce: String,
    pub version: String,
    pub build_date: String,
    pub migration: bool,
    pub cluster_name: Option<String>,
    pub cluster_id: Option<String>,
    pub recovery_seal: bool,
    pub storage_type: String,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
struct UnsealRequest {
    key: Option<String>,
    reset: bool,
    migrate: bool,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct UnsealResponse {
    pub sealed: bool,
    pub t: i64,
    pub n: i64,
    pub progress: i64,
    pub version: String,
    pub cluster_name: Option<String>,
    pub cluster_id: Option<String>,
}

struct VaultClient {
    addr: url::Url,
    http: reqwest::blocking::Client,
}

impl VaultClient {
    fn new(addr: url::Url) -> Self {
        let http = reqwest::blocking::Client::new();
        Self { addr, http }
    }

    fn read_init_status(&self) -> anyhow::Result<ReadInitStatusResponse> {
        let endpoint = self.addr.join("v1/sys/init")?;

        let response: ReadInitStatusResponse =
            self.http.get(endpoint).send()?.error_for_status()?.json()?;

        Ok(response)
    }

    fn start_init(&self, request: &StartInitRequest) -> anyhow::Result<StartInitResponse> {
        let endpoint = self.addr.join("v1/sys/init")?;

        let response: StartInitResponse = self
            .http
            .post(endpoint)
            .json(request)
            .send()?
            .error_for_status()?
            .json()?;

        Ok(response)
    }

    fn get_seal_status(&self) -> anyhow::Result<SealStatusResponse> {
        let endpoint = self.addr.join("v1/sys/seal-status")?;

        let response: SealStatusResponse =
            self.http.get(endpoint).send()?.error_for_status()?.json()?;

        Ok(response)
    }

    fn submit_unseal_key(&self, request: &UnsealRequest) -> anyhow::Result<UnsealResponse> {
        let endpoint = self.addr.join("v1/sys/unseal")?;

        let response: UnsealResponse = self
            .http
            .post(endpoint)
            .json(request)
            .send()?
            .error_for_status()?
            .json()?;

        Ok(response)
    }
}

fn main() -> anyhow::Result<()> {
    let args = Args::parse();

    setup_logging();

    info!("Started vault-init");

    let vault = VaultClient::new(args.vault_addr.clone());

    info!("Checking initializtion status");
    let init_status = vault.read_init_status().map_err(|err| {
        error!("Failed checking initializtion status");
        err
    })?;
    trace!(?init_status);

    if init_status.initialized {
        info!("Vault is already initialized");
        return Ok(());
    }
    info!("Vault is not yet initialized");

    info!("Performing initialization");
    let init_request = StartInitRequest::from(args);
    let init_response = vault.start_init(&init_request).map_err(|err| {
        error!("Failed performing initialization");
        err
    })?;
    info!("Successfully initialized Vault");
    trace!(?init_response);

    info!("Checking seal status");
    let seal_status = vault.get_seal_status().map_err(|err| {
        error!("Failed checking seal status");
        err
    })?;
    trace!(?seal_status);

    if !seal_status.sealed {
        info!("Vault is already unsealed");
        return Ok(());
    }
    info!("Vault is sealed");

    // FIXME: Clowntown
    info!("Performing unseal");
    let unseal_request = UnsealRequest {
        key: Some(init_response.keys.get(0).unwrap().to_owned()),
        reset: false,
        migrate: false,
    };
    let unseal_response = vault.submit_unseal_key(&unseal_request).map_err(|err| {
        error!("Failed submitting unseal key");
        err
    })?;
    trace!(?unseal_response);

    if !unseal_response.sealed {
        info!("Successfully unsealed Vault");
        return Ok(());
    }
    // TODO: Continue the unseal flow

    error!("Unable to completely unseal Vault");
    Err(anyhow::anyhow!("Unable to completely unseal Vault"))
}

fn setup_logging() {
    let format = tracing_subscriber::fmt::format();
    tracing_subscriber::fmt().event_format(format).init();
}
