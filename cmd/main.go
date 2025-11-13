package main

import (
	"html/template"
	"log"
	"os"
	"path/filepath"

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
	htmlTemplates, err := template.ParseGlob("src/templates/*")
	if err != nil {
		panic(err)
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
		log.Printf("Rendering template: %s", templateName)

		// Create output file
		outputPath := filepath.Join(outputDir, templateName)
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
