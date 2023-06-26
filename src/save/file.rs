use std::path::PathBuf;

use serde::Deserialize;
use serde::Serialize;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct File {
    pub path: Option<PathBuf>,
    pub overwrite: Option<bool>,
}
