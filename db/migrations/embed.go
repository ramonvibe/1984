package migrations

import "embed"

// Files contains database migrations bundled into the server binary.
//
//go:embed *.sql
var Files embed.FS

