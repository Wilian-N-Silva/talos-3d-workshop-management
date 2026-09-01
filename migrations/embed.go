// Package migrations exposes the immutable SQL migration set embedded in the
// server binary.
package migrations

import "embed"

// Files contains all numbered SQL migrations.
//
//go:embed *.sql
var Files embed.FS
