package main

import "embed"

//go:embed static/js/*.js static/css
var static embed.FS
