// Copyright (c) 2021 Pierce Bartine. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/pbar1/vault-init/internal/pkg/save"
	"github.com/pbar1/vault-init/internal/pkg/sealops"
	"github.com/pbar1/vault-init/internal/pkg/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"
)

var (
	version = "unknown"

	flagHelp              = flag.BoolP("help", "h", false, "Prints program usage information")
	flagVersion           = flag.BoolP("version", "v", false, "Print version information")
	flagLogLevel          = flag.String("log-level", "info", "Log level")
	flagLogFormat         = flag.String("log-format", "json", "Log output format")
	flagTimeout           = flag.Duration("timeout", 10*time.Minute, "Time to wait before failing the Vault init process")
	flagRecoveryShares    = flag.Int("recovery-shares", 1, "Recovery shares")
	flagRecoveryThreshold = flag.Int("recovery-threshold", 1, "Recovery threshold")
	flagRotate            = flag.Bool("rotate", false, "Rotates recovery keys and root token instead of initializing Vault")
	flagSave              = flag.StringP("save", "s", "file", "How to save the Vault init result. One of: file|kube-secret|vaultkv")
	flagOverwrite         = flag.Bool("overwrite", false, "Overwrite existing values at save method location")

	// File
	flagFilePath = flag.String("file-path", save.DefaultInitResponseKey, "Path on disk to save the Vault init result")

	// KubeSecret
	flagKubeconfig            = flag.String("kubeconfig", "", "Path to Kubeconfig to use when saving the Kubernetes secret. If unset, will use inClusterConfig.")
	flagKubeSecretName        = flag.String("kube-secret-name", "vault-init", "Name of the Kubernetes secret to save Vault init result")
	flagKubeSecretNamespace   = flag.String("kube-secret-namespace", "", "Namespace to create the Kubernetes secret in. Defaults to the current namespace.")
	flagKubeSecretLabels      = flag.StringToString("kube-secret-labels", nil, "Labels to add to the Kubernetes secret")
	flagKubeSecretAnnotations = flag.StringToString("kube-secret-annotations", nil, "Labels to add to the Kubernetes secret")

	// VaultKV
	flagVaultKVSecretPath      = flag.String("vaultkv-secret-path", "vault-init", "Vault KV secret path where the secret will be created. Excludes the mount path.")
	flagVaultKVMountPath       = flag.String("vaultkv-mount-path", "secret", "Vault KV secret engine mount path where the secret will be created.")
	flagVaultKVInitResponseKey = flag.String("vaultkv-init-response-key", save.DefaultInitResponseKey, "Key in the Vault KV secret where the Vault init response JSON payload will be stored.")
	flagVaultKVRootTokenKey    = flag.String("vaultkv-root-token-key", save.DefaultRootTokenKey, "Key in the Vault KV secret where the Vault root token will be stored.")
)

func main() {
	flag.Parse()

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

	// create and configure the save method (ie, persistence)
	var sm save.SaveMethod
	switch *flagSave {
	case save.SaveFile:
		sm, err = save.NewFileSaveMethod(*flagFilePath)
	case save.SaveKubeSecret:
		sm, err = save.NewKubeSecretSaveMethod(*flagKubeSecretName, func(m *save.KubeSecretSaveMethod) {
			m.KubeconfigPath = *flagKubeconfig
			m.SecretNamespace = *flagKubeSecretNamespace
			m.SecretLabels = *flagKubeSecretLabels
			m.SecretAnnotations = *flagKubeSecretAnnotations
		})
	case save.SaveVaultKV:
		util.EnvBackup("VAULTINIT_BACKUP_", "VAULT_")
		util.EnvRestore("VAULTINIT_VAULTKV_")
		sm, err = save.NewVaultKVSaveMethod(*flagVaultKVSecretPath, func(m *save.VaultKVSaveMethod) {
			m.MountPath = *flagVaultKVMountPath
			m.InitResponseKey = *flagVaultKVInitResponseKey
			m.RootTokenKey = *flagVaultKVRootTokenKey
		})
		util.EnvClear("VAULT_")
		util.EnvRestore("VAULTINIT_BACKUP_")
	default:
		log.Fatal().Str("save", *flagSave).Msg("unsupported save method")
	}
	if err != nil {
		log.Fatal().Err(err).Msg("unable to construct save method")
	}

	// create vault client
	vault, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create vault api client")
	}
	log.Info().Str("vaultAddr", vault.Address()).Msg("configured vault api client")

	// create the initializer
	vi := sealops.NewVaultInitializer(vault, sm, func(i *sealops.VaultInitializer) {
		i.RecoveryShares = *flagRecoveryShares
		i.RecoveryThreshold = *flagRecoveryThreshold
	})

	// either init or rotate
	if *flagRotate {
		if err := vi.RekeyRecoveryKey(); err != nil {
			log.Fatal().Err(err).Msg("unable to rekey recovery keys")
		}
		log.Info().Msg("successfully rekeyed recovery keys")
		if err := vi.RotateRoot(); err != nil {
			log.Fatal().Err(err).Msg("unable to rotate root token")
		}
		log.Info().Msg("successfully rotated root token")
	} else {
		if err := vi.Init(*flagTimeout); err != nil {
			log.Fatal().Err(err).Dur("timeout", *flagTimeout).Msg("unable to initialize vault within timeout duration")
		}
		log.Info().Msg("successfully initialized vault")
	}
}
