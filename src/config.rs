use serde::Deserialize;
use serde::Serialize;

use crate::save::File;
use crate::save::KubeSecret;
use crate::save::Load;
use crate::save::Save;
use crate::vault::models::sys::init::StartInitResponse;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub save_method: SaveMethod,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SaveMethod {
    pub file: Option<File>,
    pub kube_secret: Option<KubeSecret>,
}

impl SaveMethod {
    pub async fn save_init_all(&self, data: &StartInitResponse) -> anyhow::Result<()> {
        if let Some(file) = self.file.clone() {
            file.save_init(data).await?;
        }
        if let Some(kube_secret) = self.kube_secret.clone() {
            kube_secret.save_init(data).await?;
        }

        Ok(())
    }

    pub async fn load_init_all(&self) -> anyhow::Result<StartInitResponse> {
        if let Some(file) = self.file.clone() {
            if let Ok(data) = file.load_init().await {
                return Ok(data);
            }
        }
        if let Some(kube_secret) = self.kube_secret.clone() {
            if let Ok(data) = kube_secret.load_init().await {
                return Ok(data);
            }
        }

        Err(anyhow::anyhow!(
            "Failed loading init data from all save methods"
        ))
    }
}
