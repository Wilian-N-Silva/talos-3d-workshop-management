package credentials

import (
	"errors"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	credentialTypeGeneric      = 1
	credentialPersistLocalMach = 2
	maximumCredentialBlobSize  = 2560
)

var (
	advapi32       = windows.NewLazySystemDLL("advapi32.dll")
	procCredWrite  = advapi32.NewProc("CredWriteW")
	procCredRead   = advapi32.NewProc("CredReadW")
	procCredDelete = advapi32.NewProc("CredDeleteW")
	procCredFree   = advapi32.NewProc("CredFree")
)

type windowsCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

type windowsCredentialBackend struct{}

func (windowsCredentialBackend) Write(target string, secret []byte) error {
	if len(secret) == 0 || len(secret) > maximumCredentialBlobSize {
		return errors.New("credential secret has invalid size")
	}
	targetName, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	username, err := windows.UTF16PtrFromString("Talos desktop session")
	if err != nil {
		return err
	}
	credential := windowsCredential{
		Type:               credentialTypeGeneric,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(secret)),
		CredentialBlob:     &secret[0],
		Persist:            credentialPersistLocalMach,
		UserName:           username,
	}
	result, _, callErr := procCredWrite.Call(uintptr(unsafe.Pointer(&credential)), 0)
	runtime.KeepAlive(secret)
	if result == 0 {
		return callErr
	}
	return nil
}

func (windowsCredentialBackend) Read(target string) ([]byte, error) {
	targetName, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return nil, err
	}
	var credential *windowsCredential
	result, _, callErr := procCredRead.Call(
		uintptr(unsafe.Pointer(targetName)),
		credentialTypeGeneric,
		0,
		uintptr(unsafe.Pointer(&credential)),
	)
	if result == 0 {
		if errors.Is(callErr, windows.ERROR_NOT_FOUND) {
			return nil, ErrNotFound
		}
		return nil, callErr
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(credential)))
	if credential.CredentialBlobSize == 0 || credential.CredentialBlob == nil || credential.CredentialBlobSize > maximumCredentialBlobSize {
		return nil, errors.New("credential secret has invalid size")
	}
	secret := make([]byte, credential.CredentialBlobSize)
	copy(secret, unsafe.Slice(credential.CredentialBlob, credential.CredentialBlobSize))
	return secret, nil
}

func (windowsCredentialBackend) Delete(target string) error {
	targetName, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	result, _, callErr := procCredDelete.Call(uintptr(unsafe.Pointer(targetName)), credentialTypeGeneric, 0)
	if result == 0 {
		if errors.Is(callErr, windows.ERROR_NOT_FOUND) {
			return ErrNotFound
		}
		return callErr
	}
	return nil
}
