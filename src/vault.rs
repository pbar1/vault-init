use serde::Deserialize;
use serde::Serialize;

use crate::Args;

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ReadInitStatusResponse {
    pub initialized: bool,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct StartInitRequest {
    pub pgp_keys: Option<Vec<String>>,
    pub root_token_pgp_key: Option<String>,
    pub secret_shares: u8,
    pub secret_threshold: u8,
    pub stored_shares: Option<u8>,
    pub recovery_shares: Option<u8>,
    pub recovery_threshold: Option<u8>,
    pub recovery_pgp_keys: Option<Vec<String>>,
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
pub struct StartInitResponse {
    pub keys: Vec<String>,
    pub keys_base64: Vec<String>,
    pub root_token: String,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct SealStatusResponse {
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
pub struct UnsealRequest {
    pub key: Option<String>,
    pub reset: bool,
    pub migrate: bool,
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

pub struct VaultClient {
    pub addr: url::Url,
    pub http: reqwest::Client,
}

impl VaultClient {
    pub fn new(addr: url::Url) -> Self {
        let http = reqwest::Client::new();
        Self { addr, http }
    }

    pub async fn read_init_status(&self) -> anyhow::Result<ReadInitStatusResponse> {
        let endpoint = self.addr.join("v1/sys/init")?;

        let response: ReadInitStatusResponse = self
            .http
            .get(endpoint)
            .send()
            .await?
            .error_for_status()?
            .json()
            .await?;

        Ok(response)
    }

    pub async fn start_init(
        &self,
        request: &StartInitRequest,
    ) -> anyhow::Result<StartInitResponse> {
        let endpoint = self.addr.join("v1/sys/init")?;

        let response: StartInitResponse = self
            .http
            .post(endpoint)
            .json(request)
            .send()
            .await?
            .error_for_status()?
            .json()
            .await?;

        Ok(response)
    }

    pub async fn get_seal_status(&self) -> anyhow::Result<SealStatusResponse> {
        let endpoint = self.addr.join("v1/sys/seal-status")?;

        let response: SealStatusResponse = self
            .http
            .get(endpoint)
            .send()
            .await?
            .error_for_status()?
            .json()
            .await?;

        Ok(response)
    }

    pub async fn submit_unseal_key(
        &self,
        request: &UnsealRequest,
    ) -> anyhow::Result<UnsealResponse> {
        let endpoint = self.addr.join("v1/sys/unseal")?;

        let response: UnsealResponse = self
            .http
            .post(endpoint)
            .json(request)
            .send()
            .await?
            .error_for_status()?
            .json()
            .await?;

        Ok(response)
    }
}
