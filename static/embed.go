package static

import "embed"

// Files contains the embedded static assets (index.html, etc.).
//
//go:embed index.html
var Files embed.FS
