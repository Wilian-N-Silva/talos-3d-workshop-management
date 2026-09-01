package httpplatform

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"net/http"
)

const (
	RequestIDHeader    = "X-Request-ID"
	maxRequestIDLength = 128
)

type requestIDContextKey struct{}
type requestIDGenerator func() (string, error)

// RequestIDFromContext returns the validated correlation ID assigned to a
// versioned API request.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func requestIDMiddleware(generator requestIDGenerator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			requestID := request.Header.Get(RequestIDHeader)
			if !validRequestID(requestID) {
				generated, err := generator()
				if err != nil {
					WriteError(
						response,
						http.StatusInternalServerError,
						"internal_error",
						"Internal server error",
						nil,
					)
					return
				}
				requestID = generated
			}

			response.Header().Set(RequestIDHeader, requestID)
			ctx := context.WithValue(request.Context(), requestIDContextKey{}, requestID)
			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

func generateRequestID() (string, error) {
	var value [16]byte
	if _, err := cryptorand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate request ID: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		value[0:4],
		value[4:6],
		value[6:8],
		value[8:10],
		value[10:16],
	), nil
}

func validRequestID(value string) bool {
	if len(value) == 0 || len(value) > maxRequestIDLength {
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
