package save

import (
	"os"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

type SaveMethod interface {
	Save(*vaultapi.InitResponse) (string, error)
	Load() (*vaultapi.InitResponse, error)
}

const (
	DefaultInitResponseKey = "vault-init.json"
	DefaultRootTokenKey    = "root_token"
)

// unsetAndSetEnv unsets one map of environment variables then sets another one.
func unsetAndSetEnv(unset, set map[string]string) error {
	for key := range unset {
		if err := os.Unsetenv(key); err != nil {
			log.Error().Err(err).Str("key", key).Msg("failed to unset environment variable")
			return err
		}
	}
	for key, value := range set {
		if err := os.Setenv(key, value); err != nil {
			log.Error().Err(err).Str("key", key).Msg("failed to set environment variable")
			return err
		}
	}
	return nil
}
