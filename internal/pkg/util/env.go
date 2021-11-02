package util

import (
	"os"
	"strings"
)

func EnvBackup(backupPrefix, filterPrefix string) {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, filterPrefix) {
			s := strings.SplitN(e, "=", 2)
			k := s[0]
			v := s[1]

			os.Setenv(backupPrefix+k, v)
			os.Unsetenv(k)
		}
	}
}

func EnvRestore(backupPrefix string) {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, backupPrefix) {
			s := strings.SplitN(e, "=", 2)
			k := s[0]
			v := s[1]
			rst := strings.TrimPrefix(k, backupPrefix)

			os.Setenv(rst, v)
			os.Unsetenv(k)
		}
	}
}

func EnvClear(backupPrefix string) {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, backupPrefix) {
			s := strings.SplitN(e, "=", 2)
			k := s[0]

			os.Unsetenv(k)
		}
	}
}
