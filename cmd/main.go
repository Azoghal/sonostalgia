package main

import (
	"bytes"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"

	sonostalgia "github.com/azoghal/sonostalgia/src"
)

// For now we're hardcoding these but we'll actually parse these from files
var templateParamMap = map[string]any{
	"style.css":              struct{}{},
	"index.template.html":    sonostalgia.ExampleIndex,
	"about.template.html":    sonostalgia.ExampleAbout,
	"memories.template.html": sonostalgia.ExampleMemories,
	"years.template.html":    sonostalgia.ExampleYears,
	"memory.template.html":   sonostalgia.ExampleMemory,
}

func main() {

	funcMap := template.FuncMap{
		"markdown": func(md string) template.HTML {
			var buf bytes.Buffer
			goldmark.Convert([]byte(md), &buf)
			return template.HTML(buf.String())
		},
	}

	htmlTemplates, err := template.New("").Funcs(funcMap).ParseGlob("src/templates/*")
	if err != nil {
		log.Fatal("Error parsing templates: ", err)
	}

	outputDir := "output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Error creating output directory:", err)
	}

	err = doTemplate(htmlTemplates, outputDir)
	if err != nil {
		log.Fatal("error doing templates")
	}
}

func doTemplate(htmlTemplates *template.Template, outputDir string) error {

	// Execute each template and save to a file
	for _, t := range htmlTemplates.Templates() {
		templateName := t.Name()

		// skip the empty template
		if templateName == "" {
			continue
		}

		destinationName := strings.Replace(templateName, ".template", "", 1)
		log.Printf("Rendering template: %s", templateName)

		// Create output file
		outputPath := filepath.Join(outputDir, destinationName)
		f, err := os.Create(outputPath)
		if err != nil {
			log.Printf("Error creating file %s: %v", outputPath, err)
			return err
		}

		templateParams, ok := templateParamMap[templateName]
		if !ok {
			log.Printf("Error finding template params")
			return err
		}

		// Execute template to the file
		err = t.Execute(f, templateParams)
		if err != nil {
			log.Printf("Error executing template %s: %v", templateName, err)
			f.Close()
			return err
		}

		f.Close()
		log.Printf("Successfully created: %s", outputPath)
	}

	return nil
}
