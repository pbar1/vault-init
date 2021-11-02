package save

import (
	"encoding/json"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

// FileSaveMethod saves and loads a Vault init response to a file.
type FileSaveMethod struct {
	// FilePath is the filesystem path to read and write the Vault init response.
	FilePath string
}

const SaveFile = "file"

// NewFileSaveMethod constructs a FileSaveMethod from a file path.
func NewFileSaveMethod(filePath string, options ...func(*FileSaveMethod)) (*FileSaveMethod, error) {
	m := FileSaveMethod{FilePath: filePath}

	// apply functional options
	for _, option := range options {
		option(&m)
	}

	return &m, nil
}

// Save writes a Vault init response to its configured file path, and returns the secret's location in the
// string form `{{ path }}`.
func (m *FileSaveMethod) Save(response *vaultapi.InitResponse) (string, error) {
	l := log.With().Str("filePath", m.FilePath).Logger()

	initJSON, err := json.Marshal(response)
	if err != nil {
		l.Error().Err(err).Msg("failed to json marshal VaultClient init response")
		return "", err
	}
	l.Trace().Interface("response", response).Msg("json marshalled VaultClient init response")

	if err := os.WriteFile(m.FilePath, initJSON, 0644); err != nil {
		l.Error().Err(err).Msg("failed to write file")
		return "", err
	}
	l.Trace().Bytes("initJSON", initJSON).Msg("wrote file contents")

	return m.FilePath, nil
}

// Load reads a Vault init response from its configured file path.
func (m *FileSaveMethod) Load() (*vaultapi.InitResponse, error) {
	l := log.With().Str("filePath", m.FilePath).Logger()

	initJSON, err := os.ReadFile(m.FilePath)
	if err != nil {
		l.Error().Err(err).Msg("failed to read file")
		return nil, err
	}
	l.Trace().Bytes("initJSON", initJSON).Msg("read file contents")

	response := new(vaultapi.InitResponse)
	if err := json.Unmarshal(initJSON, response); err != nil {
		l.Error().Err(err).Msg("failed to json unmarshal VaultClient init response")
		return nil, err
	}
	l.Trace().Interface("response", response).Msg("json unmarshalled VaultClient init response")

	return response, nil
}
