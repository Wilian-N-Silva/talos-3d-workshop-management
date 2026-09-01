package auth

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestPasswordServiceHashesAndVerifiesPassword(t *testing.T) {
	service := defaultPasswordService(t)
	password := []byte("correct horse battery staple")

	encodedHash, err := service.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if strings.Contains(encodedHash, string(password)) {
		t.Fatal("encoded hash contains plaintext password")
	}
	if !strings.HasPrefix(encodedHash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("encoded hash = %q, want centralized Argon2id parameters", encodedHash)
	}

	match, err := service.Verify(password, encodedHash)
	if err != nil {
		t.Fatalf("Verify() correct password error = %v", err)
	}
	if !match {
		t.Fatal("Verify() correct password = false")
	}

	match, err = service.Verify([]byte("wrong password"), encodedHash)
	if err != nil {
		t.Fatalf("Verify() wrong password error = %v", err)
	}
	if match {
		t.Fatal("Verify() wrong password = true")
	}
}

func TestPasswordServiceUsesUniqueRandomSalts(t *testing.T) {
	service := defaultPasswordService(t)
	password := []byte("same password")

	first, err := service.Hash(password)
	if err != nil {
		t.Fatalf("first Hash() error = %v", err)
	}
	second, err := service.Hash(password)
	if err != nil {
		t.Fatalf("second Hash() error = %v", err)
	}
	if first == second {
		t.Fatal("equal passwords produced equal salted hashes")
	}
}

func TestPasswordServiceVerifiesHashesAfterDefaultParametersChange(t *testing.T) {
	original := defaultPasswordService(t)
	encodedHash, err := original.Hash([]byte("upgradeable password"))
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	updatedParameters := DefaultPasswordParameters()
	updatedParameters.Iterations++
	updated, err := NewPasswordService(updatedParameters)
	if err != nil {
		t.Fatalf("NewPasswordService() error = %v", err)
	}
	match, err := updated.Verify([]byte("upgradeable password"), encodedHash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !match {
		t.Fatal("Verify() existing hash after parameter change = false")
	}
}

func TestPasswordServiceRejectsMalformedHashesSafely(t *testing.T) {
	service := defaultPasswordService(t)
	malformed := []string{
		"",
		"not-a-password-hash",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0MTIzNA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"$argon2id$v=16$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0MTIzNA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"$argon2id$v=19$m=999999999,t=2,p=1$c2FsdHNhbHRzYWx0MTIzNA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"$argon2id$v=19$m=19456,t=0,p=1$c2FsdHNhbHRzYWx0MTIzNA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"$argon2id$v=19$m=19456,t=2,p=0$c2FsdHNhbHRzYWx0MTIzNA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"$argon2id$v=19$m=19456,t=2,p=1$c2hvcnQ$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0MTIzNA$c2hvcnQ",
		"$argon2id$v=19$m=19456,t=2,p=1$invalid!$invalid!",
		strings.Repeat("x", maximumEncodedLength+1),
	}

	for _, encodedHash := range malformed {
		match, err := service.Verify([]byte("password"), encodedHash)
		if match {
			t.Fatalf("Verify(%q) = true", encodedHash)
		}
		if !errors.Is(err, ErrInvalidPasswordHash) {
			t.Fatalf("Verify(%q) error = %v, want ErrInvalidPasswordHash", encodedHash, err)
		}
	}
}

func TestNewPasswordServiceRejectsUnsafeParameters(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PasswordParameters)
	}{
		{name: "memory too low", mutate: func(parameters *PasswordParameters) { parameters.MemoryKiB = minimumMemoryKiB - 1 }},
		{name: "memory too high", mutate: func(parameters *PasswordParameters) { parameters.MemoryKiB = maximumMemoryKiB + 1 }},
		{name: "iterations too low", mutate: func(parameters *PasswordParameters) { parameters.Iterations = 1 }},
		{name: "parallelism zero", mutate: func(parameters *PasswordParameters) { parameters.Parallelism = 0 }},
		{name: "salt too short", mutate: func(parameters *PasswordParameters) { parameters.SaltLength = minimumSaltLength - 1 }},
		{name: "key too short", mutate: func(parameters *PasswordParameters) { parameters.KeyLength = minimumKeyLength - 1 }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parameters := DefaultPasswordParameters()
			test.mutate(&parameters)
			if _, err := NewPasswordService(parameters); !errors.Is(err, ErrInvalidPasswordParameters) {
				t.Fatalf("NewPasswordService() error = %v, want ErrInvalidPasswordParameters", err)
			}
		})
	}
}

func TestPasswordServiceHandlesRandomSourceFailure(t *testing.T) {
	service, err := newPasswordService(DefaultPasswordParameters(), errorReader{})
	if err != nil {
		t.Fatalf("newPasswordService() error = %v", err)
	}
	if encodedHash, err := service.Hash([]byte("password")); err == nil || encodedHash != "" {
		t.Fatalf("Hash() = %q, %v, want empty hash and error", encodedHash, err)
	}
}

func defaultPasswordService(t *testing.T) *PasswordService {
	t.Helper()
	service, err := NewPasswordService(DefaultPasswordParameters())
	if err != nil {
		t.Fatalf("NewPasswordService() error = %v", err)
	}
	return service
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
