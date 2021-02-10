// Copyright (c) 2021 Pierce Bartine. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

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
