// Copyright (c) 2021 Pierce Bartine. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package main

import (
	"io/ioutil"

	flag "github.com/spf13/pflag"
)

const saveMethodFile = "file"

var flagFilePath = flag.String("file-path", "vault-init.json", "Path on disk to save the Vault init result")

func saveFnFile(r *initResult) (string, error) {
	err := ioutil.WriteFile(*flagFilePath, r.ResponseJSON, 0644)
	if err != nil {
		return "", err
	}

	return *flagFilePath, nil
}
