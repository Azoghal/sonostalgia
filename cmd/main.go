package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"

	sonostalgia "github.com/azoghal/sonostalgia/src"
)

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

	memories, err := loadMemories("src/memories/*.yaml")
	if err != nil {
		log.Fatal("Error parsing memories: ", err)
	}

	if len(memories) < 1 {
		log.Fatal("No memories parsed")
	}

	params, err := produceTemplateParams(memories)
	if err != nil {
		log.Fatal("failed to produce template params", err)
	}

	outputDir := "output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Error creating output directory:", err)
	}

	err = doTemplate(htmlTemplates, outputDir, params)
	if err != nil {
		log.Fatal("error doing templates")
	}
}

// load all the memories
func loadMemories(pattern string) ([]*sonostalgia.Memory, error) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var memories []*sonostalgia.Memory
	for _, file := range files {
		memory, err := sonostalgia.LoadMemory(file)
		if err != nil {
			return nil, fmt.Errorf("error loading %s: %w", file, err)
		}
		memories = append(memories, memory)
	}

	return memories, nil
}

// parse the loaded memories for the details to fill in each page's params
func produceTemplateParams(memories []*sonostalgia.Memory) (map[string]any, error) {

	// todo noddy for now, sort later
	memory := *memories[0]

	templateParamMap := map[string]any{
		"style.css": struct{}{},
		// "index.template.html":    sonostalgia.ExampleIndex,
		// "about.template.html": sonostalgia.ExampleAbout,
		// "memories.template.html": sonostalgia.ExampleMemories,
		// "years.template.html":    sonostalgia.ExampleYears,
		"memory.template.html": memory,
	}

	return templateParamMap, nil

}

func doTemplate(htmlTemplates *template.Template, outputDir string, templateParamMap map[string]any) error {

	// Execute each template and save to a file
	for _, t := range htmlTemplates.Templates() {
		templateName := t.Name()
		var (
			templateParams any
			ok             bool
		)

		// skip any templates that we've not got params for
		if templateParams, ok = templateParamMap[templateName]; !ok {
			log.Printf("Params not found, skipping template: %s", templateName)
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
