package main

import (
	"encoding/json"
	"io/ioutil"

	vaultapi "github.com/hashicorp/vault/api"
)

const StoreFile = "file"

func storeFile(response *vaultapi.InitResponse) error {
	initJSON, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(*flagFilePath, initJSON, 0644)
}
