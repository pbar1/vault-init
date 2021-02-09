package main

import (
	"encoding/json"
	"io/ioutil"

	vaultapi "github.com/hashicorp/vault/api"
)

const SaveFile = "file"

func saveFile(response *vaultapi.InitResponse) (string, error) {
	initJSON, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(*flagFilePath, initJSON, 0644)
	if err != nil {
		return "", err
	}

	return *flagFilePath, nil
}
