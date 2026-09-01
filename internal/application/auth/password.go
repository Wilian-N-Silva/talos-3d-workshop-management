// Package auth contains authentication application services.
package auth

import (
	cryptorand "crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	minimumMemoryKiB     uint32 = 19 * 1024
	maximumMemoryKiB     uint32 = 64 * 1024
	minimumIterations    uint32 = 2
	maximumIterations    uint32 = 5
	minimumParallelism   uint8  = 1
	maximumParallelism   uint8  = 4
	minimumSaltLength    uint32 = 16
	maximumSaltLength    uint32 = 64
	minimumKeyLength     uint32 = 32
	maximumKeyLength     uint32 = 64
	maximumEncodedLength        = 512
)

var (
	// ErrInvalidPasswordHash indicates a malformed, unsupported, or unsafe hash.
	ErrInvalidPasswordHash = errors.New("invalid password hash")
	// ErrInvalidPasswordParameters indicates an unsupported Argon2id cost profile.
	ErrInvalidPasswordParameters = errors.New("invalid password parameters")
)

// PasswordParameters controls new Argon2id hashes. Stored PHC hashes retain
// their own parameters so defaults can be raised without invalidating them.
type PasswordParameters struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultPasswordParameters returns the centralized password hashing profile.
func DefaultPasswordParameters() PasswordParameters {
	return PasswordParameters{
		MemoryKiB:   minimumMemoryKiB,
		Iterations:  minimumIterations,
		Parallelism: minimumParallelism,
		SaltLength:  minimumSaltLength,
		KeyLength:   minimumKeyLength,
	}
}

// PasswordService hashes and verifies passwords using Argon2id PHC strings.
type PasswordService struct {
	parameters PasswordParameters
	random     io.Reader
}

// NewPasswordService creates a service with an explicitly selected profile.
func NewPasswordService(parameters PasswordParameters) (*PasswordService, error) {
	return newPasswordService(parameters, cryptorand.Reader)
}

func newPasswordService(parameters PasswordParameters, random io.Reader) (*PasswordService, error) {
	if err := validatePasswordParameters(parameters); err != nil {
		return nil, err
	}
	if random == nil {
		return nil, fmt.Errorf("%w: random source is required", ErrInvalidPasswordParameters)
	}
	return &PasswordService{parameters: parameters, random: random}, nil
}

// Hash creates a salted Argon2id PHC string. The plaintext is never included
// in the returned value.
func (service *PasswordService) Hash(password []byte) (string, error) {
	salt := make([]byte, service.parameters.SaltLength)
	if _, err := io.ReadFull(service.random, salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	derivedKey := argon2.IDKey(
		password,
		salt,
		service.parameters.Iterations,
		service.parameters.MemoryKiB,
		service.parameters.Parallelism,
		service.parameters.KeyLength,
	)
	defer clear(derivedKey)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		service.parameters.MemoryKiB,
		service.parameters.Iterations,
		service.parameters.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(derivedKey),
	), nil
}

// Verify compares a password with an Argon2id PHC string in constant time.
// Malformed or resource-unsafe hashes return ErrInvalidPasswordHash without
// invoking Argon2.
func (service *PasswordService) Verify(password []byte, encodedHash string) (bool, error) {
	parameters, salt, expectedKey, err := parsePasswordHash(encodedHash)
	if err != nil {
		return false, err
	}

	actualKey := argon2.IDKey(
		password,
		salt,
		parameters.Iterations,
		parameters.MemoryKiB,
		parameters.Parallelism,
		parameters.KeyLength,
	)
	defer clear(actualKey)

	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}

func parsePasswordHash(encodedHash string) (PasswordParameters, []byte, []byte, error) {
	if len(encodedHash) == 0 || len(encodedHash) > maximumEncodedLength {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	version, err := parseUint(parts[2], "v=", 8)
	if err != nil || version != uint64(argon2.Version) {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	parameterParts := strings.Split(parts[3], ",")
	if len(parameterParts) != 3 {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	memory, err := parseUint(parameterParts[0], "m=", 32)
	if err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	iterations, err := parseUint(parameterParts[1], "t=", 32)
	if err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	parallelism, err := parseUint(parameterParts[2], "p=", 8)
	if err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	expectedKey, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	parameters := PasswordParameters{
		MemoryKiB:   uint32(memory),
		Iterations:  uint32(iterations),
		Parallelism: uint8(parallelism),
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(expectedKey)),
	}
	if err := validatePasswordParameters(parameters); err != nil {
		return PasswordParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	return parameters, salt, expectedKey, nil
}

func parseUint(value string, prefix string, bitSize int) (uint64, error) {
	raw, ok := strings.CutPrefix(value, prefix)
	if !ok || raw == "" {
		return 0, ErrInvalidPasswordHash
	}
	parsed, err := strconv.ParseUint(raw, 10, bitSize)
	if err != nil {
		return 0, ErrInvalidPasswordHash
	}
	return parsed, nil
}

func validatePasswordParameters(parameters PasswordParameters) error {
	if parameters.MemoryKiB < minimumMemoryKiB || parameters.MemoryKiB > maximumMemoryKiB {
		return fmt.Errorf("%w: memory must be between %d and %d KiB", ErrInvalidPasswordParameters, minimumMemoryKiB, maximumMemoryKiB)
	}
	if parameters.Iterations < minimumIterations || parameters.Iterations > maximumIterations {
		return fmt.Errorf("%w: iterations must be between %d and %d", ErrInvalidPasswordParameters, minimumIterations, maximumIterations)
	}
	if parameters.Parallelism < minimumParallelism || parameters.Parallelism > maximumParallelism {
		return fmt.Errorf("%w: parallelism must be between %d and %d", ErrInvalidPasswordParameters, minimumParallelism, maximumParallelism)
	}
	if parameters.MemoryKiB < 8*uint32(parameters.Parallelism) {
		return fmt.Errorf("%w: memory is too low for parallelism", ErrInvalidPasswordParameters)
	}
	if parameters.SaltLength < minimumSaltLength || parameters.SaltLength > maximumSaltLength {
		return fmt.Errorf("%w: salt length must be between %d and %d bytes", ErrInvalidPasswordParameters, minimumSaltLength, maximumSaltLength)
	}
	if parameters.KeyLength < minimumKeyLength || parameters.KeyLength > maximumKeyLength {
		return fmt.Errorf("%w: key length must be between %d and %d bytes", ErrInvalidPasswordParameters, minimumKeyLength, maximumKeyLength)
	}
	return nil
}
