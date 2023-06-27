pub mod models;

use crate::vault::models::sys::init::*;
use crate::vault::models::sys::seal_status::*;
use crate::vault::models::sys::unseal::*;

pub struct VaultClient {
    pub addr: url::Url,
    pub http: reqwest::Client,
}

impl VaultClient {
    pub fn new(addr: url::Url) -> Self {
        let http = reqwest::Client::new();
        Self { addr, http }
    }

    pub async fn read_init_status(&self) -> anyhow::Result<GetInitResponse> {
        let endpoint = self.addr.join("v1/sys/init")?;

        let response: GetInitResponse = self
            .http
            .get(endpoint)
            .send()
            .await?
            .error_for_status()?
            .json()
            .await?;

        Ok(response)
    }

    pub async fn start_init(&self, request: &PostInitRequest) -> anyhow::Result<PostInitResponse> {
        let endpoint = self.addr.join("v1/sys/init")?;

        let response: PostInitResponse = self
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

    pub async fn get_seal_status(&self) -> anyhow::Result<GetSealStatusResponse> {
        let endpoint = self.addr.join("v1/sys/seal-status")?;

        let response: GetSealStatusResponse = self
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
        request: &PostUnsealRequest,
    ) -> anyhow::Result<PostUnsealResponse> {
        let endpoint = self.addr.join("v1/sys/unseal")?;

        let response: PostUnsealResponse = self
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
