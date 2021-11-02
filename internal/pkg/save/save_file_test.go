package save

import (
	"os"
	"reflect"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
)

func TestFileSaveMethod(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "")
	defer os.Remove(tmpFile.Name())

	type fields struct {
		FilePath string
	}
	tests := []struct {
		name    string
		fields  fields
		payload *vaultapi.InitResponse
		wantErr bool
	}{
		{
			name:    "New->Save->Load works",
			fields:  fields{FilePath: tmpFile.Name()},
			payload: TestInitResponse,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewFileSaveMethod(tt.fields.FilePath)
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
