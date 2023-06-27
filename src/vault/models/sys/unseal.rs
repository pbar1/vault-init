use serde::Deserialize;
use serde::Serialize;

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PostUnsealRequest {
    pub key: Option<String>,
    pub reset: bool,
    pub migrate: bool,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PostUnsealResponse {
    pub sealed: bool,
    pub t: i64,
    pub n: i64,
    pub progress: i64,
    pub version: String,
    pub cluster_name: Option<String>,
    pub cluster_id: Option<String>,
}
