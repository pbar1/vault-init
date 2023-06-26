use std::collections::BTreeMap;

use anyhow::Context;
use k8s_openapi::api::core::v1::Secret;
use k8s_openapi::apimachinery::pkg::apis::meta::v1::ObjectMeta;
use serde::Deserialize;
use serde::Serialize;
use serde_json::json;
use tracing::debug;

use super::Load;
use super::Save;
use crate::vault::StartInitResponse;

const DEFAULT_SECRET_NAME: &str = "vault-init";
const DEFAULT_SECRET_KEY: &str = "init.json";

#[derive(Default, Debug, Clone, Serialize, Deserialize)]
pub struct KubeSecret {
    pub name: Option<String>,
    pub namespace: Option<String>,
    pub labels: Option<BTreeMap<String, String>>,
    pub annotations: Option<BTreeMap<String, String>>,
    pub key: Option<String>,
    pub overwrite: Option<bool>,
}

// FIXME: Enforce overwrite setting
#[async_trait::async_trait]
impl Save for KubeSecret {
    async fn save_init(&self, data: &StartInitResponse) -> anyhow::Result<()> {
        debug!(save_method = "kube_secret", "Saving init data");

        let client = kube::Client::try_default().await?;

        // FIXME: Allow setting namespace
        let secrets: kube::Api<Secret> = kube::Api::default_namespaced(client);

        let key = self.key.clone().unwrap_or(DEFAULT_SECRET_KEY.to_owned());
        let mut string_data: BTreeMap<String, String> = BTreeMap::new();
        string_data.insert(key, json!(data).to_string());

        let name = self.name.clone().unwrap_or(DEFAULT_SECRET_NAME.to_owned());
        let secret = Secret {
            metadata: ObjectMeta {
                name: Some(name),
                labels: self.labels.clone(),
                annotations: self.annotations.clone(),
                ..Default::default()
            },
            string_data: Some(string_data),
            ..Default::default()
        };

        secrets
            .create(&kube::api::PostParams::default(), &secret)
            .await?;

        Ok(())
    }
}

#[async_trait::async_trait]
impl Load for KubeSecret {
    async fn load_init(&self) -> anyhow::Result<StartInitResponse> {
        debug!(save_method = "kube_secret", "Loading init data");

        let client = kube::Client::try_default().await?;

        let secrets: kube::Api<Secret> = kube::Api::default_namespaced(client);

        let name = self.name.clone().unwrap_or(DEFAULT_SECRET_NAME.to_owned());
        let secret = secrets.get(&name).await?;

        let data = secret.data.context("Kubernetes secret contained no data")?;

        let key = self.key.clone().unwrap_or(DEFAULT_SECRET_KEY.to_owned());
        let byte_string = data
            .get(&key)
            .context("Kubernetes secret did not contain expected key")?;

        let out = String::from_utf8(byte_string.0.clone())?;

        let init_response: StartInitResponse = serde_json::from_str(&out)?;

        Ok(init_response)
    }
}
