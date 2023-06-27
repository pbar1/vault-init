use serde::Deserialize;
use serde::Serialize;

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
