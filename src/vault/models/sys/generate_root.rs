use serde::Deserialize;
use serde::Serialize;

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct GetGenerateRootAttemptResponse {
    pub started: bool,
    pub nonce: String,
    pub progress: i64,
    pub required: i64,
    pub encoded_token: String,
    pub pgp_fingerprint: String,
    pub otp_length: i64,
    pub complete: bool,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PostGenerateRootAttemptRequest {
    /// Specifies a base64-encoded PGP public key. The raw bytes of the token
    /// will be encrypted with this value before being returned to the final
    /// unseal key provider.
    pub pgp_key: Option<String>,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PostGenerateRootAttemptResponse {
    pub started: bool,
    pub nonce: String,
    pub progress: i64,
    pub required: i64,
    pub encoded_token: String,
    pub otp: String,
    pub otp_length: i64,
    pub complete: bool,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PostGenerateRootUpdateRequest {
    /// Specifies a single root key share.
    pub key: String,
    /// Specifies the nonce of the attempt.
    pub nonce: String,
}

#[derive(Default, Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PostGenerateRootUpdateResponse {
    pub started: bool,
    pub nonce: String,
    pub progress: i64,
    pub required: i64,
    pub pgp_fingerprint: String,
    pub complete: bool,
    pub encoded_token: String,
}
