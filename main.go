// Copyright (c) 2021 Pierce Bartine. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/helper/xor"
	"github.com/hashicorp/vault/sdk/helper/base62"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"
)

type (
	saveFunc func(*vaultapi.InitResponse) (string, error)

	tokenAccessorList struct {
	}
	tokenAccessor struct {
		Accessor       string      `json:"accessor"`
		CreationTime   int         `json:"creation_time"`
		CreationTTL    int         `json:"creation_ttl"`
		DisplayName    string      `json:"display_name"`
		EntityID       string      `json:"entity_id"`
		ExpireTime     interface{} `json:"expire_time"`
		ExplicitMaxTTL int         `json:"explicit_max_ttl"`
		ID             string      `json:"id"`
		Meta           interface{} `json:"meta"`
		NumUses        int         `json:"num_uses"`
		Orphan         bool        `json:"orphan"`
		Path           string      `json:"path"`
		Policies       []string    `json:"policies"`
		TTL            int         `json:"ttl"`
		Type           string      `json:"type"`
	}
)

var (
	version = "unknown"

	flagVersion               = flag.BoolP("version", "v", false, "Print version information")
	flagLogLevel              = flag.String("log-level", "info", "Log level")
	flagLogFormat             = flag.String("log-format", "json", "Log output format")
	flagTimeout               = flag.Duration("timeout", 10*time.Minute, "Time to wait before failing the Vault init process")
	flagRecoveryShares        = flag.Int("recovery-shares", 1, "Recovery shares")
	flagRecoveryThreshold     = flag.Int("recovery-threshold", 1, "Recovery threshold")
	flagSave                  = flag.StringP("save", "s", "file", "How to save the Vault init result. One of: file|kube-secret")
	flagFilePath              = flag.String("file-path", "vault-init.json", "Path on disk to save the Vault init result")
	flagKubeconfig            = flag.String("kubeconfig", "", "Path to Kubeconfig to use when saving the Kubernetes secret. If unset, will use inClusterConfig.")
	flagKubeSecretName        = flag.String("kube-secret-name", "vault-init", "Name of the Kubernetes secret to save Vault init result")
	flagKubeSecretNamespace   = flag.String("kube-secret-namespace", "", "Namespace to create the Kubernetes secret in. Defaults to the current namespace.")
	flagKubeSecretLabels      = flag.StringToString("kube-secret-labels", nil, "Labels to add to the Kubernetes secret")
	flagKubeSecretAnnotations = flag.StringToString("kube-secret-annotations", nil, "Labels to add to the Kubernetes secret")
	flagOverwrite             = flag.Bool("overwrite", false, "Overwrite existing values at save method location")
	flagRotate                = flag.Bool("rotate", false, "Rotates recovery keys and root token instead of initializing Vault")

	saveFuncs = map[string]saveFunc{
		SaveFile:       saveFile,
		SaveKubeSecret: saveKubeSecret,
	}
)

func main() {
	flag.Parse()
	if *flagVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	// configure logging
	logLevel, err := zerolog.ParseLevel(strings.ToLower(*flagLogLevel))
	if err != nil {
		log.Warn().Err(err).Msg("unable to parse log level, using default: info")
	} else {
		zerolog.SetGlobalLevel(logLevel)
	}
	if strings.ToLower(*flagLogFormat) != "json" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
	log.Logger = log.With().Caller().Logger()

	// lookup result store function
	saveFunc, exists := saveFuncs[*flagSave]
	if !exists {
		log.Fatal().Str("save", *flagSave).Msg("unsupported save type")
	}
	log.Info().Str("save", *flagSave).Msg("using save type")

	// create vault client
	vault, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create vault api client")
	}
	log.Info().Str("vaultAddr", vault.Address()).Msg("configured vault api client")

	if *flagRotate {
		// TODO: find a better way to intake this value, like a simple save/load persistence layer
		log.Info().Msg("loading vault init data from environment variable VAULT_INIT_JSON")
		initJSON := os.Getenv("VAULT_INIT_JSON")
		if initJSON == "" {
			log.Fatal().Msg("environment variable VAULT_INIT_JSON empty or unset")
		}
		log.Trace().Str("initJSON", initJSON).Msg("loaded data from VAULT_INIT_JSON")
		initResp := new(vaultapi.InitResponse)
		if err := json.Unmarshal([]byte(initJSON), initResp); err != nil {
			log.Fatal().Err(err).Msg("unable to unmarshal as VAULT_INIT_JSON")
		}
		log.Trace().Interface("initResp", initResp).Msg("unmarshalled vault init response from json")

		rekey(vault, saveFunc, initResp)
		rotateRoot(vault, saveFunc, initResp)
	} else {
		initialize(vault, saveFunc)
	}
}

// initialize checks Vault's init status every 10s and performs the init if possible. Result is saved to the configured
// save backend, with fallback to the `file` backend to avoid data loss.
func initialize(vault *vaultapi.Client, saveFunc saveFunc) {
	for start := time.Now(); time.Now().Before(start.Add(*flagTimeout)); time.Sleep(10 * time.Second) {
		log.Info().Msg("checking vault init status")
		initialized, err := vault.Sys().InitStatus()
		if err != nil {
			log.Warn().Err(err).Msg("error checking vault init status, retrying in 10s")
			continue
		}
		if initialized {
			log.Info().Msg("vault is already initialized")
			return
		}
		log.Info().Msg("vault is not initialized")
		// TODO: allow the other init parameters
		log.Info().Int("recoveryShares", *flagRecoveryShares).Int("recoveryThreshold", *flagRecoveryThreshold).Msg("initializing vault")
		initResp, err := vault.Sys().Init(&vaultapi.InitRequest{
			RecoveryShares:    *flagRecoveryShares,
			RecoveryThreshold: *flagRecoveryThreshold,
		})
		if err != nil {
			log.Warn().Err(err).Msg("error during vault init, retrying in 10s")
			continue
		}
		log.Info().Msg("vault init succeeded")
		// TODO: allow saving the result in multiple locations simultaneously
		// TODO: retry the save until a timeout, to avoid losing the init result due to transient failure
		location, err := saveFunc(initResp)
		if err != nil {
			log.Warn().Err(err).Str("save", *flagSave).Msg("save failed, falling back to file to avoid data loss")
			*flagSave = "file"
			location, err = saveFile(initResp)
			if err != nil {
				log.Fatal().Err(err).Str("filePath", *flagFilePath).Msg("fallback file save failed, root token and keys have been lost")
			}
		}
		log.Info().Str("save", *flagSave).Str("location", location).Msg("save succeeded")
		return
	}
	log.Fatal().Dur("timeout", *flagTimeout).Msg("unable to initialize vault within timeout")
}

// rekey rotates Vault recovery keys and persists them using the configured save backend.
// See: [How-to rekey vault (recovery-keys) when using auto-unseal](https://support.hashicorp.com/hc/en-us/articles/4404364271763-How-to-rekey-vault-recovery-keys-when-using-auto-unseal)
func rekey(vault *vaultapi.Client, saveFunc saveFunc, initResp *vaultapi.InitResponse) {
	log.Info().Msg("checking rekey recovery key status")
	rekeyStatusResp, err := vault.Sys().RekeyRecoveryKeyStatus()
	if err != nil {
		log.Fatal().Err(err).Msg("error checking rekey recovery key status")
	}
	if rekeyStatusResp.Started {
		log.Fatal().Msg("rekey recovery key process is already in progress")
	}

	log.Info().Msg("beginning rekey recovery key process")
	rekeyInitResp, err := vault.Sys().RekeyRecoveryKeyInit(&vaultapi.RekeyInitRequest{
		SecretShares:        *flagRecoveryShares,
		SecretThreshold:     *flagRecoveryThreshold,
		RequireVerification: false,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error beginning rekey recovery key process")
	}

	for _, recoveryKey := range initResp.RecoveryKeysB64 {
		log.Trace().Str("recoveryKey", recoveryKey).Msg("sending recovery key")
		rekeyUpdateResp, err := vault.Sys().RekeyRecoveryKeyUpdate(recoveryKey, rekeyInitResp.Nonce)
		if err != nil {
			log.Error().Err(err).Msg("rekey recovery key update failed, cancelling process")
			if err := vault.Sys().RekeyRecoveryKeyCancel(); err != nil {
				log.Fatal().Err(err).Msg("error cancelling rekey recovery key process")
			}
			log.Fatal().Msg("cancelled rekey recovery key process")
		}
		if rekeyUpdateResp.Complete {
			log.Info().Msg("rekey recovery key success")
			initResp.RecoveryKeys = rekeyUpdateResp.Keys
			initResp.RecoveryKeysB64 = rekeyUpdateResp.KeysB64
			log.Trace().Interface("initResp", initResp).Msg("updated in-memory vault init response")
			break
		}
	}

	// seems wasteful that we're performing this logic within both `rekey` and `rotateRoot`,
	// but it does checkpoint against data loss
	location, err := saveFunc(initResp)
	if err != nil {
		log.Warn().Err(err).Str("save", *flagSave).Msg("save failed, falling back to file to avoid data loss")
		*flagSave = "file"
		location, err = saveFile(initResp)
		if err != nil {
			log.Fatal().Err(err).Str("filePath", *flagFilePath).Msg("fallback file save failed, new recovery keys have been lost")
		}
	}
	log.Info().Str("save", *flagSave).Str("location", location).Msg("save succeeded")
}

// rotateRoot rotates Vault root token and persists it using the configured save backend.
// See: [Generate Root Tokens Using Unseal Keys](https://learn.hashicorp.com/tutorials/vault/generate-root)
func rotateRoot(vault *vaultapi.Client, saveFunc saveFunc, initResp *vaultapi.InitResponse) {
	genrootStatusResp, err := vault.Sys().GenerateRootStatus()
	if err != nil {
		log.Fatal().Err(err).Msg("error checking generate root status before beginning")
	}
	if genrootStatusResp.Started {
		log.Fatal().Msg("generate root process is already in progress")
	}

	otp, err := base62.Random(26)
	if err != nil {
		log.Fatal().Err(err).Msg("error generating otp")
	}

	log.Info().Msg("beginning generate root process")
	genrootInitResp, err := vault.Sys().GenerateRootInit(otp, "")
	if err != nil {
		log.Fatal().Err(err).Msg("error beginning generate root process")
	}

	for _, recoveryKey := range initResp.RecoveryKeysB64 {
		genrootUpdateResp, err := vault.Sys().GenerateRootUpdate(recoveryKey, genrootInitResp.Nonce)
		if err != nil {
			log.Error().Err(err).Msg("generate root update failed, cancelling process")
			if err := vault.Sys().GenerateRootCancel(); err != nil {
				log.Fatal().Err(err).Msg("error cancelling generate root process")
			}
			log.Fatal().Msg("cancelled generate root process")
		}
		if genrootUpdateResp.Complete {
			log.Info().Msg("generate root success")
			decoded, err := decodeRoot(genrootUpdateResp.EncodedRootToken, otp)
			if err != nil {
				log.Fatal().Err(err).Msg("error decoding root token, new root token has been lost")
			}
			initResp.RootToken = *decoded
			break
		}
	}

	// seems wasteful that we're performing this logic within both `rekey` and `rotateRoot`,
	// but it does checkpoint against data loss
	location, err := saveFunc(initResp)
	if err != nil {
		log.Warn().Err(err).Str("save", *flagSave).Msg("save failed, falling back to file to avoid data loss")
		*flagSave = "file"
		location, err = saveFile(initResp)
		if err != nil {
			log.Fatal().Err(err).Str("filePath", *flagFilePath).Msg("fallback file save failed, new root token has been lost")
		}
	}
	log.Info().Str("save", *flagSave).Str("location", location).Msg("save succeeded")

	if err := vault.Auth().Token().RevokeSelf(""); err != nil {
		log.Fatal().Err(err).Msg("error revoking current root token")
	}
	log.Info().Msg("revoked current root token")
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
