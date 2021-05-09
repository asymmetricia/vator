package main

import (
	"embed"
	"html/template"
)

type TemplateContext struct {
	Error string
	Toast string
	Phone string
	Kgs   bool

	Withings bool

	User  string
	Page  string
	Share bool
}

//go:embed templates/*
var templateFs embed.FS
var templates *template.Template

func init() {
	var err error
	templates, err = templates.ParseFS(templateFs, "templates/*.tmpl")
	if err != nil {
		panic(err)
	}
}
