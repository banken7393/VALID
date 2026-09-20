// Package web embeds the VALID dashboard static assets.
package web

import "embed"

// FS contains index.html, style.css, and app.js.
//
//go:embed index.html style.css app.js
var FS embed.FS
