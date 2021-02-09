package main

import (
	"fmt"
	"log"
	"os"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	flag "github.com/spf13/pflag"
)

var (
	version = "unknown"

	flagVersion               = flag.BoolP("version", "v", false, "Print version information")
	flagTimeout               = flag.Duration("timeout", 10*time.Minute, "Time to wait before failing the Vault init process")
	flagRecoveryShares        = flag.Int("recovery-shares", 1, "Recovery shares")
	flagRecoveryThreshold     = flag.Int("recovery-threshold", 1, "Recovery threshold")
	flagStoreType             = flag.StringP("store-type", "t", "file", "Where to store the Vault init result. One of: kube-secret|file")
	flagFilePath              = flag.String("file-path", "vault-init.json", "Path on disk to store the Vault init result")
	flagKubeSecretName        = flag.String("kube-secret-name", "vault-init", "Name of the Kubernetes secret to store Vault init result")
	flagKubeSecretNamespace   = flag.String("kube-secret-namespace", "", "Namespace to create the Kubernetes secret in. Defaults to the current namespace.")
	flagKubeSecretLabels      = flag.StringToString("kube-secret-labels", nil, "Labels to add to the Kubernetes secret")
	flagKubeSecretAnnotations = flag.StringToString("kube-secret-annotations", nil, "Labels to add to the Kubernetes secret")

	storeFuncs = map[string]func(*vaultapi.InitResponse) error{
		StoreFile:       storeFile,
		StoreKubeSecret: storeKubeSecret,
	}
)

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	storeFunc, exists := storeFuncs[*flagStoreType]
	if !exists {
		log.Fatalf("unsupported result store: %s\n", *flagStoreType)
	}

	vault, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		log.Fatalf("unable to create vault client: %v\n", err)
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
		log.Println("vault initialization successful")
		if err := storeFunc(initResp); err != nil && *flagStoreType != StoreFile {
			log.Printf("warning: store-type=%s failed: %v\n", *flagStoreType, err)
			log.Printf("falling back to store-type=file %s\n", *flagFilePath)
			if err := storeFile(initResp); err != nil {
				log.Printf("fallback store-type=file %s failed: %v\n", *flagFilePath, err)
				log.Fatalln("init results unable to be stored: vault has been initialized, but root token and keys have been lost permanently")
			}
		}
		log.Println("vault init results stored successfully")
		os.Exit(0)
	}

	log.Fatalf("timeout reached: unable to initialize vault after %s\n", *flagTimeout)
}
