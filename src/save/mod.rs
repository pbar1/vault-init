mod file;
mod kube_secret;

pub use file::File;
pub use kube_secret::KubeSecret;

use crate::vault::models::sys::init::PostInitResponse;

#[async_trait::async_trait]
pub trait Save {
    async fn save_init(&self, data: &PostInitResponse) -> anyhow::Result<()>;
}

#[async_trait::async_trait]
pub trait Load {
    async fn load_init(&self) -> anyhow::Result<PostInitResponse>;
}
