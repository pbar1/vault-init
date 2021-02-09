package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	vaultapi "github.com/hashicorp/vault/api"
	flag "github.com/spf13/pflag"
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

	saveFuncs = map[string]func(*vaultapi.InitResponse) (string, error){
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
	storeFunc, exists := saveFuncs[*flagSave]
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
			RecoveryShares:    *flagRecoveryShares,
			RecoveryThreshold: *flagRecoveryThreshold,
		})
		if err != nil {
			log.Warn().Err(err).Msg("error during vault init, retrying in 10s")
			continue
		}
		log.Info().Msg("vault init succeeded")
		location, err := storeFunc(initResp)
		if err != nil {
			log.Warn().Err(err).Str("save", *flagSave).Msg("save failed, falling back to file to avoid data loss")
			*flagSave = "file"
			location, err = saveFile(initResp)
			if err != nil {
				log.Fatal().Err(err).Str("filePath", *flagFilePath).Msg("fallback file save failed, root token and keys have been lost")
			}
		}
		log.Info().Str("save", *flagSave).Str("location", location).Msg("save succeeded")
		os.Exit(0)
	}
	log.Fatal().Dur("timeout", *flagTimeout).Msg("unable to initialize vault within timeout")
}
