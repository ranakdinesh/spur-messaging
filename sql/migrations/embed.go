package migrations

import "embed"

// FS embeds module-owned migration files so host applications can run the
// messaging schema without depending on a filesystem layout.
//
//go:embed *.up.sql
var FS embed.FS
