//go:build !windows

package credentials

import "errors"

type windowsCredentialBackend struct{}

func (windowsCredentialBackend) Write(string, []byte) error {
	return errors.New("Windows Credential Manager is unavailable on this platform")
}

func (windowsCredentialBackend) Read(string) ([]byte, error) {
	return nil, errors.New("Windows Credential Manager is unavailable on this platform")
}

func (windowsCredentialBackend) Delete(string) error {
	return errors.New("Windows Credential Manager is unavailable on this platform")
}
