use serde::Deserialize;
use serde::Serialize;

use crate::Args;

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct GetInitResponse {
    pub initialized: bool,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PostInitRequest {
    pub pgp_keys: Option<Vec<String>>,
    pub root_token_pgp_key: Option<String>,
    pub secret_shares: u8,
    pub secret_threshold: u8,
    pub stored_shares: Option<u8>,
    pub recovery_shares: Option<u8>,
    pub recovery_threshold: Option<u8>,
    pub recovery_pgp_keys: Option<Vec<String>>,
}

impl From<Args> for PostInitRequest {
    fn from(args: Args) -> Self {
        Self {
            pgp_keys: args.pgp_keys,
            root_token_pgp_key: args.root_token_pgp_key,
            secret_shares: args.secret_shares,
            secret_threshold: args.secret_threshold,
            stored_shares: args.stored_shares,
            recovery_shares: args.recovery_shares,
            recovery_threshold: args.recovery_threshold,
            recovery_pgp_keys: args.recovery_pgp_keys,
        }
    }
}

/// An object including the (possibly encrypted, if `pgp_keys` was provided)
/// root keys, base 64 encoded root keys and initial root token
#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PostInitResponse {
    pub keys: Vec<String>,
    pub keys_base64: Vec<String>,
    pub root_token: String,
}
