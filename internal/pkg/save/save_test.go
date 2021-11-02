package save

import vaultapi "github.com/hashicorp/vault/api"

var (
	TestInitResponse = &vaultapi.InitResponse{
		Keys:            []string{"key1", "key2"},
		KeysB64:         []string{"key1b64", "key2b64"},
		RecoveryKeys:    []string{"recoverykey1", "recoverykey2"},
		RecoveryKeysB64: []string{"recoverykey1b64", "recoverykey2b64"},
		RootToken:       "roottoken",
	}

	TestInitResponseString = `{"keys":["key1","key2"],"keys_base64":["key1b64","key2b64"],"recovery_keys":["recoverykey1","recoverykey2"],"recovery_keys_base64":["recoverykey1b64","recoverykey2b64"],"root_token":"roottoken"}`
)
