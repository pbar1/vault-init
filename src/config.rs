use serde::Deserialize;
use serde::Serialize;

use crate::save::File;
use crate::save::KubeSecret;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub save_method: SaveMethod,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SaveMethod {
    pub file: Option<File>,
    pub kube_secret: Option<KubeSecret>,
}
