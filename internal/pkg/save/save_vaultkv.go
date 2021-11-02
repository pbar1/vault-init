package save

import (
	"encoding/json"
	"path"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

// VaultKVSaveMethod saves and loads a Vault init response to a Vault KV secret. It is expected to be the KV v2 engine.
type VaultKVSaveMethod struct {
	// VaultClient is the Vault API client used to interact with Vault.
	VaultClient *vaultapi.Client

	// SecretPath is the Vault KV secret path where the Vault init response exists.
	SecretPath string

	// MountPath is the Vault KV engine mount path.
	MountPath string

	// InitResponseKey is the key in the Vault KV secret where the Vault init response JSON payload will be stored.
	InitResponseKey string

	// RootTokenKey is the key in the Vault KV secret where the Vault root token will be stored.
	RootTokenKey string
}

const SaveVaultKV = "vaultkv"

// NewVaultKVSaveMethod constructs a VaultKVSaveMethod from a secret path.
func NewVaultKVSaveMethod(secretPath string, options ...func(*VaultKVSaveMethod)) (*VaultKVSaveMethod, error) {
	m := VaultKVSaveMethod{SecretPath: secretPath}

	// apply functional options
	for _, option := range options {
		option(&m)
	}

	var err error
	if m.VaultClient == nil {
		m.VaultClient, err = vaultapi.NewClient(vaultapi.DefaultConfig())
		if err != nil {
			log.Error().Err(err).Msg("failed to create default vault api client")
			return nil, err
		}
	}

	if m.MountPath == "" {
		m.MountPath = "secret"
	}
	if m.InitResponseKey == "" {
		m.InitResponseKey = DefaultInitResponseKey
	}
	if m.RootTokenKey == "" {
		m.RootTokenKey = DefaultRootTokenKey
	}

	return &m, nil
}

// Save writes a Vault init response to its configured Vault KV secret, and returns the secret's location in the
// string form `{{ mountPath }}/data/{{ secretPath }}`.
func (m *VaultKVSaveMethod) Save(response *vaultapi.InitResponse) (string, error) {
	l := log.With().Str("secretPath", m.SecretPath).Str("mountPath", m.MountPath).Logger()

	initJSON, err := json.Marshal(response)
	if err != nil {
		l.Error().Err(err).Msg("failed to json marshal vault init response")
		return "", err
	}
	l.Trace().Interface("response", response).Msg("json marshalled vault init response")

	fullPath := path.Join(m.MountPath, "data", m.SecretPath)
	if _, err := m.VaultClient.Logical().Write(fullPath, map[string]interface{}{
		"data": map[string]string{
			m.InitResponseKey: string(initJSON),
			m.RootTokenKey:    response.RootToken,
		},
	}); err != nil {
		l.Error().Err(err).Msg("failed to write vault kv secret")
		return "", err
	}
	l.Trace().Bytes("initJSON", initJSON).Msg("wrote vault kv secret contents")

	return fullPath, nil
}

// Load reads a Vault init response from its configured Vault KV secret.
func (m *VaultKVSaveMethod) Load() (*vaultapi.InitResponse, error) {
	l := log.With().Str("secretPath", m.SecretPath).Str("mountPath", m.MountPath).Logger()

	fullPath := path.Join(m.MountPath, "data", m.SecretPath)
	secret, err := m.VaultClient.Logical().Read(fullPath)
	if err != nil {
		l.Error().Err(err).Msg("failed to read vault kv secret")
	}
	l.Trace().Interface("secret", secret).Msg("loaded secret")
	secretDataIface, found := secret.Data["data"]
	if !found {
		l.Error().Msg("failed to find 'data' key in kv secret response")
		return nil, err
	}
	secretData, ok := secretDataIface.(map[string]interface{})
	if !ok {
		l.Error().Msg("failed to cast secret data interface to map[string]interface{}")
		return nil, err
	}

	initJSON, found := secretData[m.InitResponseKey]
	if !found {
		l.Error().Msg("failed to find vault init response key in vault kv secret")
		return nil, err
	}
	l.Trace().Interface("initJSON", initJSON).Msg("read vault kv secret contents")

	initJSONStr, ok := initJSON.(string)
	if !ok {
		l.Error().Msg("failed to cast initJSON interface to string")
		return nil, err
	}

	response := new(vaultapi.InitResponse)
	if err := json.Unmarshal([]byte(initJSONStr), response); err != nil {
		l.Error().Err(err).Msg("failed to json unmarshal vault init response")
		return nil, err
	}
	l.Trace().Interface("response", response).Msg("json unmarshalled vault init response")

	return response, nil
}
