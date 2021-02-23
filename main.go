// Copyright (c) 2021 Pierce Bartine. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"
)

type (
	saveFn     func(*initResult) (string, error)
	saveFnMap  map[string]saveFn
	initResult struct {
		ResponseJSON []byte
		RootToken    string
	}
)

const helpText = `Initializes HashiCorp Vault and saves the root token and keys in providers of your choice

Usage:
  vault-init [flags]

Flags:`

var (
	version = "unknown"
	saveFns = saveFnMap{
		saveMethodFile:       saveFnFile,
		saveMethodKubeSecret: saveFnKubeSecret,
		saveMethodVaultKV:    saveFnVaultKV,
		saveMethodAWSSSM:     saveFnAWSSSM,
	}

	// general flags
	flagHelp       = flag.BoolP("help", "h", false, "Print help and usage information")
	flagVersion    = flag.BoolP("version", "v", false, "Print version information")
	flagLogLevel   = flag.String("log-level", "info", "Log level")
	flagLogFormat  = flag.String("log-format", "json", "Log output format")
	flagTimeout    = flag.Duration("timeout", 20*time.Minute, "Time to wait before failing the Vault init process")
	flagForce      = flag.Bool("force", false, "Overwrites the contents of save method locations, if they exist")
	flagSaveMethod = flag.StringP("save", "s", "file", fmt.Sprintf("How to save the Vault init result. One of: %s", strings.Join(saveFns.methods(), "|")))

	// vault init flags
	flagPGPKeys           = flag.StringSlice("pgp-keys", nil, "Comma-separated list of paths to files on disk containing public GPG keys OR a comma-separated list of Keybase usernames using the format \"keybase:<username>\". When supplied, the generated unseal keys will be encrypted and base64-encoded in the order specified in this list. The number of entries must match --key-shares.")
	flagKeyShares         = flag.IntP("key-shares", "n", 5, "Number of key shares to split the generated master key into. This is the number of \"unseal keys\" to generate.")
	flagKeyThreshold      = flag.IntP("key-threshold", "t", 3, "Number of key shares required to reconstruct the master key. This must be less than or equal to --key-shares.")
	flagRootTokenPGPKey   = flag.String("root-token-pgp-key", "", "Path to a file on disk containing a binary or base64-encoded public GPG key. This can also be specified as a Keybase username using the format \"keybase:<username>\". When supplied, the generated root token will be encrypted and base64-encoded with the given public key.")
	flagRecoveryPGPKeys   = flag.StringSlice("recovery-pgp-keys", nil, "Behaves like --pgp-keys, but for the recovery key shares. Only used in Auto-Unseal mode.")
	flagRecoveryShares    = flag.Int("recovery-shares", 5, "Number of key shares to split the recovery key into. Only used in Auto-Unseal mode.")
	flagRecoveryThreshold = flag.Int("recovery-threshold", 3, "Number of key shares required to reconstruct the recovery key. Only used in Auto-Unseal mode.")
)

func init() {
	flag.CommandLine.SortFlags = false
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, helpText)
		fmt.Fprintln(os.Stderr, flag.CommandLine.FlagUsagesWrapped(200))
	}
	flag.Parse()
}

func main() {
	if *flagHelp {
		flag.Usage()
		os.Exit(0)
	}
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
	saveFn, exists := saveFns[*flagSaveMethod]
	if !exists {
		log.Fatal().Str("save", *flagSaveMethod).Msg("unsupported save type")
	}
	log.Info().Str("save", *flagSaveMethod).Msg("using save type")

	// create vault client
	vault, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create vault api client")
	}
	log.Info().Str("vaultAddr", vault.Address()).Msg("configured vault api client")

	// loop until timeout, checking init status every 10 seconds
	for start := time.Now(); time.Now().Before(start.Add(*flagTimeout)); time.Sleep(10 * time.Second) {
		log.Info().Msg("checking vault init status")
		initialized, err := vault.Sys().InitStatus()
		if err != nil {
			log.Warn().Err(err).Msg("error checking vault init status, retrying in 10s")
			continue
		}
		if initialized {
			log.Info().Msg("vault is already initialized")
			os.Exit(0)
		}
		log.Info().Msg("vault is not initialized")
		log.Info().Int("recoveryShares", *flagRecoveryShares).Int("recoveryThreshold", *flagRecoveryThreshold).Msg("initializing vault")
		initResp, err := vault.Sys().Init(&vaultapi.InitRequest{
			PGPKeys:           *flagPGPKeys,
			SecretShares:      *flagKeyShares,
			SecretThreshold:   *flagKeyThreshold,
			RootTokenPGPKey:   *flagRootTokenPGPKey,
			RecoveryPGPKeys:   *flagRecoveryPGPKeys,
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
		initRespJSON, err := json.Marshal(initResp)
		if err != nil {
			log.Error().Err(err).Msg("unable to serialize init response to json")
		}
		initResult := &initResult{
			ResponseJSON: initRespJSON,
			RootToken:    initResp.RootToken,
		}
		location, err := saveFn(initResult)
		if err != nil {
			log.Warn().Err(err).Str("save", *flagSaveMethod).Msg("save failed, falling back to file to avoid data loss")
			*flagSaveMethod = "file"
			location, err = saveFnFile(initResult)
			if err != nil {
				log.Fatal().Err(err).Str("filePath", *flagFilePath).Msg("fallback file save failed, root token and keys have been lost")
			}
		}
		log.Info().Str("save", *flagSaveMethod).Str("location", location).Msg("save succeeded")
		os.Exit(0)
	}
	log.Fatal().Dur("timeout", *flagTimeout).Msg("unable to initialize vault within timeout")
}

func (m *saveFnMap) methods() []string {
	keys := make([]string, 0)
	if m == nil {
		return keys
	}
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}
