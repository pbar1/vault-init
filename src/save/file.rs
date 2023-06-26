use std::path::PathBuf;

use serde::Deserialize;
use serde::Serialize;
use tracing::debug;

use super::Load;
use super::Save;
use crate::vault::StartInitResponse;

const DEFAULT_PATH: &str = "vault-init.json";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct File {
    pub path: Option<PathBuf>,
    pub overwrite: Option<bool>,
}

#[async_trait::async_trait]
impl Save for File {
    async fn save_init(&self, data: &StartInitResponse) -> anyhow::Result<()> {
        debug!(save_method = "file", "Saving init data");
        let path = self.path.clone().unwrap_or(PathBuf::from(DEFAULT_PATH));
        let contents = serde_json::to_vec(data)?;
        tokio::fs::write(path, &contents).await?;
        Ok(())
    }
}

#[async_trait::async_trait]
impl Load for File {
    async fn load_init(&self) -> anyhow::Result<StartInitResponse> {
        debug!(save_method = "file", "Loading init data");
        let path = self.path.clone().unwrap_or(PathBuf::from(DEFAULT_PATH));
        let contents = tokio::fs::read(path).await?;
        let data: StartInitResponse = serde_json::from_slice(&contents)?;
        Ok(data)
    }
}
