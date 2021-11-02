package sealops

import (
	"encoding/base64"
	"fmt"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/helper/xor"
	"github.com/hashicorp/vault/sdk/helper/base62"
	"github.com/pbar1/vault-init/internal/pkg/save"
	"github.com/rs/zerolog/log"
)

// VaultInitializer initializes Vault.
// [Init parameters are detailed here](https://www.vaultproject.io/api/system/init#start-initialization).
type VaultInitializer struct {
	vault *vaultapi.Client
	s     save.SaveMethod

	/* Init parameters */

	// Specifies an array of PGP public keys used to encrypt the output unseal keys. Ordering is preserved. The keys
	// must be base64-encoded from their original binary representation. The size of this array must be the same as
	// `secret_shares`.
	PGPKeys []string

	// Specifies a PGP public key used to encrypt the initial root token. The key must be base64-encoded from its
	// original binary representation
	RootTokenPGPKey string

	// Specifies the number of shares to split the master key into.
	SecretShares int

	// Specifies the number of shares required to reconstruct the master key. This must be less than or equal
	// `secret_shares`. If using Vault HSM with auto-unsealing, this value must be the same as `secret_shares`.
	SecretThreshold int

	// Specifies the number of shares that should be encrypted by the HSM and stored for auto-unsealing.
	// Currently must be the same as `secret_shares`.
	StoredShares int

	// Specifies the number of shares to split the recovery key into.
	RecoveryShares int

	// Specifies the number of shares required to reconstruct the recovery key. This must be less than or equal to
	// `recovery_shares`.
	RecoveryThreshold int

	// Specifies an array of PGP public keys used to encrypt the output recovery keys. Ordering is preserved. The keys
	// must be base64-encoded from their original binary representation. The size of this array must be the same as
	// `recovery_shares`.
	RecoveryPGPKeys []string

	/* Rekey & RekeyRecoveryKey Parameters */

	// Specifies if using PGP-encrypted keys, whether Vault should also store a plaintext backup of the PGP-encrypted
	// keys at `core/unseal-keys-backup` (or `core/recovery-keys-backup` for recovery keys) in the physical storage
	// backend. These can then be retrieved and removed via the `sys/rekey/backup` (or `sys/rekey-recovery-key/backup`
	// for recovery keys) endpoint.
	Backup bool

	// This turns on verification functionality. When verification is turned on, after successful authorization with the
	// current unseal keys, the new unseal keys are returned but the master key is not actually rotated. The new keys
	// must be provided to authorize the actual rotation of the master key. This ensures that the new keys have been
	// successfully saved and protects against a risk of the keys being lost after rotation but before they can be
	// persisted. This can be used with or without `pgp_keys` (or `recovery_pgp_keys` for recovery keys), and when used
	// with it, it allows ensuring that the returned keys can be successfully decrypted before committing to the new
	// shares, which the backup functionality does not provide.
	RequireVerification bool
}

func NewVaultInitializer(client *vaultapi.Client, saveMethod save.SaveMethod, options ...func(*VaultInitializer)) *VaultInitializer {
	i := &VaultInitializer{
		vault: client,
		s:     saveMethod,
	}

	// apply functional options
	for _, option := range options {
		option(i)
	}

	return i
}

// Init checks Vault's init status every 10s and performs the init if possible. Result is saved to the configured
// save backend, with fallback to the `file` backend to avoid data loss.
func (i *VaultInitializer) Init(timeout time.Duration) error {
	for start := time.Now(); time.Now().Before(start.Add(timeout)); time.Sleep(10 * time.Second) {
		log.Info().Msg("checking vault init status")
		initialized, err := i.vault.Sys().InitStatus()
		if err != nil {
			log.Warn().Err(err).Msg("error checking vault init status, retrying in 10s")
			continue
		}
		if initialized {
			log.Info().Msg("vault is already initialized")
			return nil
		}
		log.Info().Msg("vault is not initialized")

		initResp, err := i.vault.Sys().Init(&vaultapi.InitRequest{
			SecretShares:      i.SecretShares,
			SecretThreshold:   i.SecretThreshold,
			StoredShares:      i.StoredShares,
			PGPKeys:           i.PGPKeys,
			RecoveryShares:    i.RecoveryShares,
			RecoveryThreshold: i.RecoveryThreshold,
			RecoveryPGPKeys:   i.RecoveryPGPKeys,
			RootTokenPGPKey:   i.RootTokenPGPKey,
		})
		if err != nil {
			log.Warn().Err(err).Msg("failed to initialize vault, retrying in 10s")
			continue
		}
		log.Info().Msg("vault init succeeded")

		// TODO: allow saving the result in multiple locations
		// TODO: retry the save until a timeout, to avoid losing the init result due to transient failure
		location, err := i.s.Save(initResp)
		if err != nil {
			log.Error().Err(err).Msg("failed to save vault init response, data has been lost")
			return err
		}
		log.Info().Str("location", location).Msg("save succeeded")
		return nil
	}
	log.Error().Dur("timeout", timeout).Msg("failed to initialize vault within timeout")
	return fmt.Errorf("vault init failed")
}

// Rekey rotates Vault unseal keys and persists them using the configured save backend.
// Note: Verification cannot be performed when using PGP encryption.
func (i *VaultInitializer) Rekey() error {
	if i.RequireVerification && len(i.PGPKeys) > 0 {
		return fmt.Errorf("rekey verification with PGP not supported")
	}

	log.Info().Msg("checking rekey status")
	rekeyStatusResp, err := i.vault.Sys().RekeyStatus()
	if err != nil {
		log.Error().Err(err).Msg("failed to check rekey status")
		return err
	}
	if rekeyStatusResp.Started {
		return fmt.Errorf("rekey process is already in progress")
	}

	initResp, err := i.s.Load()
	if err != nil {
		log.Error().Err(err).Msg("failed to load vault init response")
		return err
	}

	log.Info().Msg("starting rekey process")
	rekeyInitResp, err := i.vault.Sys().RekeyInit(&vaultapi.RekeyInitRequest{
		SecretShares:        i.SecretShares,
		SecretThreshold:     i.SecretThreshold,
		StoredShares:        i.StoredShares,
		PGPKeys:             i.PGPKeys,
		Backup:              i.Backup,
		RequireVerification: i.RequireVerification,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to start rekey process")
		return err
	}

	for _, key := range initResp.KeysB64 {
		log.Trace().Str("key", key).Msg("sending key")
		rekeyUpdateResp, err := i.vault.Sys().RekeyUpdate(key, rekeyInitResp.Nonce)
		if err != nil {
			log.Error().Err(err).Msg("rekey update failed, cancelling process")
			if err := i.vault.Sys().RekeyCancel(); err != nil {
				log.Error().Err(err).Msg("failed to cancel rekey process")
				return err
			}
			log.Info().Msg("cancelled rekey process")
			return err
		}
		if rekeyUpdateResp.Complete {
			log.Info().Msg("rekey success")
			initResp.Keys = rekeyUpdateResp.Keys
			initResp.KeysB64 = rekeyUpdateResp.KeysB64
			log.Trace().Interface("initResp", initResp).Msg("updated in-memory vault init response")
			break
		}
	}

	location, err := i.s.Save(initResp)
	if err != nil {
		log.Error().Err(err).Msg("failed to save vault init response after rekey")
		return err
	}
	log.Info().Str("location", location).Msg("save succeeded")

	if i.RequireVerification {
		for _, key := range initResp.RecoveryKeysB64 {
			log.Trace().Str("key", key).Msg("sending  for verification")
			verifyUpdateResp, err := i.vault.Sys().RekeyVerificationUpdate(key, rekeyInitResp.Nonce)
			if err != nil {
				log.Error().Err(err).Msg("rekey verification failed, cancelling process")
				if err := i.vault.Sys().RekeyVerificationCancel(); err != nil {
					log.Error().Err(err).Msg("failed to cancel rekey verification")
					return err
				}
				log.Info().Msg("cancelled rekey verification")
				return err
			}
			if verifyUpdateResp.Complete {
				log.Info().Msg("rekey verification success")
				break
			}
		}
	}

	return nil
}

// RekeyRecoveryKey rotates Vault recovery keys and persists them using the configured save backend.
// See: [How-to rekey vault (recovery-keys) when using auto-unseal](https://support.hashicorp.com/hc/en-us/articles/4404364271763-How-to-rekey-vault-recovery-keys-when-using-auto-unseal)
// Note: Verification cannot be performed when using PGP encryption.
func (i *VaultInitializer) RekeyRecoveryKey() error {
	if i.RequireVerification && len(i.RecoveryPGPKeys) > 0 {
		return fmt.Errorf("rekey recovery key verification with PGP not supported")
	}

	log.Info().Msg("checking rekey recovery key status")
	rekeyStatusResp, err := i.vault.Sys().RekeyRecoveryKeyStatus()
	if err != nil {
		log.Error().Err(err).Msg("failed to check rekey recovery key status")
		return err
	}
	if rekeyStatusResp.Started {
		return fmt.Errorf("rekey recovery key process is already in progress")
	}

	initResp, err := i.s.Load()
	if err != nil {
		log.Error().Err(err).Msg("failed to load vault init response")
		return err
	}

	log.Info().Msg("starting rekey recovery key process")
	rekeyInitResp, err := i.vault.Sys().RekeyRecoveryKeyInit(&vaultapi.RekeyInitRequest{
		SecretShares:        i.RecoveryShares,
		SecretThreshold:     i.RecoveryThreshold,
		StoredShares:        i.StoredShares,
		PGPKeys:             i.RecoveryPGPKeys,
		Backup:              i.Backup,
		RequireVerification: i.RequireVerification,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to start rekey recovery key process")
		return err
	}

	for _, recoveryKey := range initResp.RecoveryKeysB64 {
		log.Trace().Str("recoveryKey", recoveryKey).Msg("sending recovery key")
		rekeyUpdateResp, err := i.vault.Sys().RekeyRecoveryKeyUpdate(recoveryKey, rekeyInitResp.Nonce)
		if err != nil {
			log.Error().Err(err).Msg("rekey recovery key update failed, cancelling process")
			if err := i.vault.Sys().RekeyRecoveryKeyCancel(); err != nil {
				log.Error().Err(err).Msg("failed to cancel rekey recovery key process")
				return err
			}
			log.Info().Msg("cancelled rekey recovery key process")
			return err
		}
		if rekeyUpdateResp.Complete {
			log.Info().Msg("rekey recovery key success")
			initResp.RecoveryKeys = rekeyUpdateResp.Keys
			initResp.RecoveryKeysB64 = rekeyUpdateResp.KeysB64
			log.Trace().Interface("initResp", initResp).Msg("updated in-memory vault init response")
			break
		}
	}

	location, err := i.s.Save(initResp)
	if err != nil {
		log.Error().Err(err).Msg("failed to save vault init response after rekey recovery keys")
		return err
	}
	log.Info().Str("location", location).Msg("save succeeded")

	if i.RequireVerification {
		for _, recoveryKey := range initResp.RecoveryKeysB64 {
			log.Trace().Str("recoveryKey", recoveryKey).Msg("sending recovery key for verification")
			verifyUpdateResp, err := i.vault.Sys().RekeyRecoveryKeyVerificationUpdate(recoveryKey, rekeyInitResp.Nonce)
			if err != nil {
				log.Error().Err(err).Msg("rekey recovery key verification failed, cancelling process")
				if err := i.vault.Sys().RekeyRecoveryKeyVerificationCancel(); err != nil {
					log.Error().Err(err).Msg("failed to cancel rekey recovery key verification")
					return err
				}
				log.Info().Msg("cancelled rekey recovery key verification")
				return err
			}
			if verifyUpdateResp.Complete {
				log.Info().Msg("rekey recovery key verification success")
				break
			}
		}
	}

	return nil
}

// RotateRoot rotates Vault root token and persists it using the configured save backend.
// See: [Generate Root Tokens Using Unseal Keys](https://learn.hashicorp.com/tutorials/vault/generate-root)
func (i *VaultInitializer) RotateRoot() error {
	genrootStatusResp, err := i.vault.Sys().GenerateRootStatus()
	if err != nil {
		log.Error().Err(err).Msg("error checking generate root status before beginning")
		return err
	}
	if genrootStatusResp.Started {
		return fmt.Errorf("generate root process is already in progress")
	}

	initResp, err := i.s.Load()
	if err != nil {
		log.Error().Err(err).Msg("failed to load vault init response")
		return err
	}

	otp, err := base62.Random(26)
	if err != nil {
		log.Error().Err(err).Msg("error generating otp")
		return err
	}

	log.Info().Msg("beginning generate root process")
	genrootInitResp, err := i.vault.Sys().GenerateRootInit(otp, i.RootTokenPGPKey)
	if err != nil {
		log.Error().Err(err).Msg("error beginning generate root process")
		return err
	}

	keys := initResp.KeysB64
	if keys == nil || len(keys) < 1 {
		log.Warn().Msg("keys list was nil or empty, using recovery keys instead")
		keys = initResp.RecoveryKeysB64
	}

	for _, key := range keys {
		genrootUpdateResp, err := i.vault.Sys().GenerateRootUpdate(key, genrootInitResp.Nonce)
		if err != nil {
			log.Error().Err(err).Msg("generate root update failed, cancelling process")
			if err := i.vault.Sys().GenerateRootCancel(); err != nil {
				log.Error().Err(err).Msg("error cancelling generate root process")
				return err
			}
			log.Error().Msg("cancelled generate root process")
			return err
		}
		if genrootUpdateResp.Complete {
			log.Info().Msg("generate root success")
			decoded, err := decodeRoot(genrootUpdateResp.EncodedRootToken, otp)
			if err != nil {
				log.Error().Err(err).Msg("error decoding root token, new root token has been lost")
				return err
			}
			initResp.RootToken = *decoded
			break
		}
	}

	location, err := i.s.Save(initResp)
	if err != nil {
		log.Error().Err(err).Msg("failed to save vault init response after rotate root token")
		return err
	}
	log.Info().Str("location", location).Msg("save succeeded")

	if err := i.vault.Auth().Token().RevokeSelf(""); err != nil {
		log.Error().Err(err).Msg("error revoking current root token")
		return err
	}
	log.Info().Msg("revoked current root token")

	return nil
}

func decodeRoot(encoded, otp string) (*string, error) {
	tokenBytes, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	tokenBytes, err = xor.XORBytes(tokenBytes, []byte(otp))
	if err != nil {
		return nil, err
	}
	token := string(tokenBytes)

	return &token, nil
}
