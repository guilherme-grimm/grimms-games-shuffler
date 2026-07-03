// Package web carries the compiled frontend. dist/ is populated by
// `pnpm build`; only .gitkeep is committed, so dev builds embed an
// almost-empty FS and the HTTP handler falls back to a placeholder page.
package web

import "embed"

// DistFS holds the built frontend; serve it via fs.Sub(DistFS, "dist").
//
//go:embed all:dist
var DistFS embed.FS
