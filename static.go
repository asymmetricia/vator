package main

import (
	_ "embed"
)

//go:embed templates/preamble.tmpl
var preambleHtml string

//go:embed templates/signup.tmpl
var signupTemplate string

//go:embed templates/login.tmpl
var loginHtml string

//go:embed templates/postamble.tmpl
var postambleHtml string

//go:embed templates/index.tmpl
var indexTemplate string