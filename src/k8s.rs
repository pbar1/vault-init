use std::collections::BTreeMap;

use anyhow::Context;
use k8s_openapi::api::core::v1::Secret;
use k8s_openapi::apimachinery::pkg::apis::meta::v1::ObjectMeta;
use serde_json::json;

use crate::vault::StartInitResponse;

const SECRET_KEY: &str = "init.json";

pub async fn write_kube_secret(
    name: &str,
    init_response: &StartInitResponse,
) -> anyhow::Result<()> {
    let client = kube::Client::try_default().await?;

    let secrets: kube::Api<Secret> = kube::Api::default_namespaced(client);

    let mut string_data: BTreeMap<String, String> = BTreeMap::new();
    string_data.insert(SECRET_KEY.to_owned(), json!(init_response).to_string());

    let secret = Secret {
        metadata: ObjectMeta {
            name: Some(name.to_owned()),
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

pub async fn read_kube_secret(name: &str) -> anyhow::Result<StartInitResponse> {
    let client = kube::Client::try_default().await?;

    let secrets: kube::Api<Secret> = kube::Api::default_namespaced(client);

    let secret = secrets.get(name).await?;

    let data = secret.data.context("Kubernetes secret contained no data")?;

    let byte_string = data
        .get(SECRET_KEY)
        .context("Kubernetes secret did not contain expected key")?;

    let out = String::from_utf8(byte_string.0.clone())?;

    let init_response: StartInitResponse = serde_json::from_str(&out)?;

    Ok(init_response)
}
