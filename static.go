package main

import "embed"

//go:embed static/js/*.js static/js/*.js.map static/css
var static embed.FS
