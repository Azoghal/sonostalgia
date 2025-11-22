package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

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
		"statcard": func(label string, value any) sonostalgia.StatCard {
			return sonostalgia.StatCard{
				Label: label,
				Value: value,
			}
		},
	}

	htmlTemplates, err := template.New("").Funcs(funcMap).ParseGlob("src/templates/*")
	if err != nil {
		log.Fatal("Error parsing templates: ", err)
	}

	templateParams, err := load("src/memories/*.yaml")
	if err != nil {
		log.Fatal("Error parsing memories: ", err)
	}

	outputDir := "output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Error creating output directory:", err)
	}

	err = doTemplate(htmlTemplates, outputDir, templateParams)
	if err != nil {
		log.Fatal("error doing templates")
	}
}

// load all the memories
func load(pattern string) (*sonostalgia.Sonostalgia, error) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	sonostalgia, err := sonostalgia.LoadSonostalgia(files)
	if err != nil {
		return nil, err
	}

	return sonostalgia, nil
}

type page struct {
	templateName   string
	outputName     string
	templateParams any
}

func doTemplate(htmlTemplates *template.Template, outputDir string, templateParams *sonostalgia.Sonostalgia) error {

	staticPages := []page{
		{
			templateName:   "style.css",
			outputName:     "style.css",
			templateParams: nil,
		},
		{
			templateName:   "about.template.html",
			outputName:     "about.html",
			templateParams: templateParams.AboutParams,
		},
		{
			templateName:   "index.template.html",
			outputName:     "index.html",
			templateParams: templateParams.IndexParams,
		},
		{
			templateName:   "memories.template.html",
			outputName:     "memories.html",
			templateParams: templateParams.MemoriesParams,
		},
		{
			templateName:   "years.template.html",
			outputName:     "years.html",
			templateParams: templateParams.YearsParams,
		},
	}

	allMemories := make([]page, len(templateParams.MemoryParams))
	for i, memory := range templateParams.MemoryParams {
		allMemories[i] = page{
			templateName:   "memory.template.html",
			outputName:     fmt.Sprintf("%s.html", memory.OutputTitle),
			templateParams: memory,
		}
	}
	allPages := append(staticPages, allMemories...)

	// Execute each template and save to a file
	for _, page := range allPages {
		t := htmlTemplates.Lookup(page.templateName)
		templateName := t.Name()

		log.Printf("Rendering template: %s", templateName)

		// Create output file
		outputPath := filepath.Join(outputDir, page.outputName)
		f, err := os.Create(outputPath)
		if err != nil {
			log.Printf("Error creating file %s: %v", outputPath, err)
			return err
		}

		// Execute template to the file
		err = t.Execute(f, page.templateParams)
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
