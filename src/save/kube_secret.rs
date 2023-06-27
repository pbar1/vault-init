use std::collections::BTreeMap;

use anyhow::Context;
use k8s_openapi::api::core::v1::Secret;
use k8s_openapi::apimachinery::pkg::apis::meta::v1::ObjectMeta;
use kube::ResourceExt;
use serde::Deserialize;
use serde::Serialize;
use serde_json::json;
use tracing::debug;
use tracing::warn;

use super::Load;
use super::Save;
use crate::vault::models::sys::init::PostInitResponse;

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

#[async_trait::async_trait]
impl Save for KubeSecret {
    async fn save_init(&self, data: &PostInitResponse) -> anyhow::Result<()> {
        debug!(save_method = "kube_secret", "Saving init data");

        // Create K8s client
        let client = kube::Client::try_default().await?;
        let secrets: kube::Api<Secret> = match &self.namespace {
            Some(ns) => kube::Api::namespaced(client, ns),
            None => kube::Api::default_namespaced(client),
        };

        let key = self.key.clone().unwrap_or(DEFAULT_SECRET_KEY.to_owned());
        let mut string_data: BTreeMap<String, String> = BTreeMap::new();
        string_data.insert(key, json!(data).to_string());

        let name = self.name.clone().unwrap_or(DEFAULT_SECRET_NAME.to_owned());
        let mut secret = Secret {
            metadata: ObjectMeta {
                name: Some(name.clone()),
                labels: self.labels.clone(),
                annotations: self.annotations.clone(),
                ..Default::default()
            },
            string_data: Some(string_data),
            ..Default::default()
        };

        if let Ok(existing) = secrets.get(&name).await {
            if !self.overwrite.unwrap_or(false) {
                return Err(anyhow::anyhow!(
                    "Kube secret already exists, but not configured to overwrite"
                ));
            }

            warn!(
                save_method = "kube_secret",
                secret = name,
                "Existing secret found, overwriting"
            );
            secret.metadata.resource_version = existing.resource_version();
            secrets
                .replace(&name, &kube::api::PostParams::default(), &secret)
                .await?;
        } else {
            secrets
                .create(&kube::api::PostParams::default(), &secret)
                .await?;
        }

        Ok(())
    }
}

#[async_trait::async_trait]
impl Load for KubeSecret {
    async fn load_init(&self) -> anyhow::Result<PostInitResponse> {
        debug!(save_method = "kube_secret", "Loading init data");

        let client = kube::Client::try_default().await?;
        let secrets: kube::Api<Secret> = match &self.namespace {
            Some(ns) => kube::Api::namespaced(client, ns),
            None => kube::Api::default_namespaced(client),
        };

        let name = self.name.clone().unwrap_or(DEFAULT_SECRET_NAME.to_owned());
        let secret = secrets.get(&name).await?;

        let data = secret.data.context("Kubernetes secret contained no data")?;

        let key = self.key.clone().unwrap_or(DEFAULT_SECRET_KEY.to_owned());
        let byte_string = data
            .get(&key)
            .context("Kubernetes secret did not contain expected key")?;

        let out = String::from_utf8(byte_string.0.clone())?;

        let init_response: PostInitResponse = serde_json::from_str(&out)?;

        Ok(init_response)
    }
}
