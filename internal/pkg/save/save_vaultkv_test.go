package save

import (
	"net"
	"reflect"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/vault"
)

func TestVaultKVSaveMethod(t *testing.T) {
	_, client := createTestVault(t)

	type fields struct {
		Client     *vaultapi.Client
		MountPath  string
		SecretPath string
	}
	tests := []struct {
		name    string
		fields  fields
		payload *vaultapi.InitResponse
		wantErr bool
	}{
		{
			name: "New->Save->Load works",
			fields: fields{
				Client:     client,
				MountPath:  "secret",
				SecretPath: "test-path",
			},
			payload: TestInitResponse,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewVaultKVSaveMethod(tt.fields.SecretPath, func(m *VaultKVSaveMethod) {
				m.VaultClient = tt.fields.Client
				m.MountPath = tt.fields.MountPath
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileSaveMethod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			_, err = m.Save(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := m.Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.payload) {
				t.Errorf("Load() got = %v, payload %v", got, tt.payload)
			}
		})
	}
}

func createTestVault(t *testing.T) (net.Listener, *vaultapi.Client) {
	t.Helper()

	// Create an in-memory, unsealed core (the "backend", if you will).
	core, keyShares, rootToken := vault.TestCoreUnsealed(t)
	_ = keyShares

	// Start an HTTP server for the core.
	ln, addr := http.TestServer(t, core)

	// Create a client that talks to the server, initially authenticating with
	// the root token.
	conf := vaultapi.DefaultConfig()
	conf.Address = addr

	client, err := vaultapi.NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}
	client.SetToken(rootToken)

	// Setup required secrets, policies, etc.
	_, err = client.Logical().Write("secret/data/foo", map[string]interface{}{
		"secret": "bar",
	})
	if err != nil {
		t.Fatal(err)
	}

	return ln, client
}
