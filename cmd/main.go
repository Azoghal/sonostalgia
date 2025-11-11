package main

import (
	"html/template"
	"os"

	sonostalgia "github.com/azoghal/sonostalgia/src"
)

func main() {
	tmpl, err := template.ParseGlob("src/templates/*.html")
	if err != nil {
		panic(err)
	}

	err = tmpl.ExecuteTemplate(os.Stdout, "memory.template.html", sonostalgia.ExampleMemory)
	if err != nil {
		panic(err)
	}
}
