package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	flag "github.com/spf13/pflag"
)

var (
	version                 = "unknown"
	flagVersion             = flag.BoolP("version", "v", false, "Print version information")
	flagTimeout             = flag.Duration("timeout", 10*time.Minute, "Time to wait before failing the Vault init process")
	flagRecoveryShares      = flag.Int("recovery-shares", 1, "Recovery shares")
	flagRecoveryThreshold   = flag.Int("recovery-threshold", 1, "Recovery threshold")
	flagResultStore         = flag.String("result-store", "kube-secret", "Where to store the Vault init result. One of: kube-secret|file")
	flagKubeSecretName      = flag.String("kube-secret-name", "vault-init", "Name of the Kubernetes secret to store Vault init result")
	flagKubeSecretNamespace = flag.String("kube-secret-namespace", "", "Namespace to create the Kubernetes secret in. Defaults to the current namespace.")
)

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Println(version)
	}

	vault, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	// loop until timeout, checking init status every 10 seconds
	for start := time.Now(); time.Now().Before(start.Add(*flagTimeout)); time.Sleep(10 * time.Second) {
		initialized, err := vault.Sys().InitStatus()
		if err != nil {
			log.Printf("error checking vault init status, waiting 10s: %v\n", err)
			continue
		}
		if initialized {
			log.Println("vault is already initialized")
			os.Exit(0)
		}
		initResp, err := vault.Sys().Init(&vaultapi.InitRequest{
			RecoveryShares:    *flagRecoveryShares,
			RecoveryThreshold: *flagRecoveryThreshold,
		})
		if err != nil {
			log.Printf("error during vault init operation: %v\n", err)
		}
		initJSON, err := json.Marshal(initResp)
		if err != nil {
			log.Printf("warning: unable to marshal init response to json: %v\n", err)
		}
		fmt.Println(string(initJSON))
		os.Exit(0)
	}

	log.Fatalf("timeout reached: unable to initialize vault after %s\n", *flagTimeout)
}
