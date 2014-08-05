package lock

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

type LockClient interface {
	Init() error
	Get() (*Semaphore, error)
	Set(*Semaphore) error
}

func GetMachineID(root string) string {
	fullPath := filepath.Join(root, "/etc/machine-id")
	id, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(id))
}
