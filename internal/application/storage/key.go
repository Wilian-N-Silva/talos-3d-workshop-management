package storage

import "errors"

const maxObjectKeyLength = 128

// ErrInvalidObjectKey is returned when serialized key data is not an opaque,
// path-independent token.
var ErrInvalidObjectKey = errors.New("invalid object key")

// ObjectKey is an opaque storage identifier. Its value cannot contain path
// separators, dots, drive prefixes, escaping syntax, or Unicode lookalikes.
// The zero value is invalid.
type ObjectKey struct {
	value string
}

// ParseObjectKey validates a key loaded from persistent metadata.
func ParseObjectKey(value string) (ObjectKey, error) {
	if !validObjectKey(value) {
		return ObjectKey{}, ErrInvalidObjectKey
	}

	return ObjectKey{value: value}, nil
}

// String returns the serialized opaque token.
func (key ObjectKey) String() string {
	return key.value
}

// Valid reports whether the key can be passed to a Store implementation.
func (key ObjectKey) Valid() bool {
	return validObjectKey(key.value)
}

func validObjectKey(value string) bool {
	if len(value) == 0 || len(value) > maxObjectKeyLength {
		return false
	}

	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' {
			continue
		}
		return false
	}

	return true
}
