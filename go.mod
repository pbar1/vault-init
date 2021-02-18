module github.com/pbar1/vault-init

go 1.15

require (
	github.com/googleapis/gnostic v0.5.4 // indirect
	github.com/hashicorp/vault/api v1.0.4
	github.com/rs/zerolog v1.20.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
)

replace k8s.io/api => k8s.io/api v0.20.2

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.2

replace k8s.io/apimachinery => k8s.io/apimachinery v0.21.0-alpha.0

replace k8s.io/apiserver => k8s.io/apiserver v0.20.2

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.2

replace k8s.io/client-go => k8s.io/client-go v0.20.2

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.2

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.3-0.20210113223636-14b48a912564

replace k8s.io/code-generator => k8s.io/code-generator v0.20.3-rc.0

replace k8s.io/component-base => k8s.io/component-base v0.20.2

replace k8s.io/component-helpers => k8s.io/component-helpers v0.20.3-0.20210113212619-366422c2e4de

replace k8s.io/controller-manager => k8s.io/controller-manager v0.20.3-0.20210113222514-4179eafc027c
