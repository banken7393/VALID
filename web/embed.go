// Package web embeds the VALID dashboard static assets.
package web

import "embed"

// FS contains the dashboard HTML and client script (layout via Tailwind CDN).
//
//go:embed index.html app.js
var FS embed.FS
