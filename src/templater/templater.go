package templater

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	sonostalgia "github.com/azoghal/sonostalgia/src"
)

type page struct {
	templateName   string
	outputName     string
	templateParams any
}

func Run(srcDir, outputDir string) error {
	funcMap := template.FuncMap{
		"markdown": func(md string) template.HTML {
			var buf bytes.Buffer
			goldmark.New(goldmark.WithExtensions(extension.Strikethrough)).Convert([]byte(md), &buf)
			return template.HTML(buf.String())
		},
		"statcard": func(label string, value any) sonostalgia.StatCard {
			return sonostalgia.StatCard{Label: label, Value: value}
		},
		"seeallcard": func() sonostalgia.Memory {
			return sonostalgia.Memory{
				OutputTitle: "memories",
				Title:       "More Memories",
				Subtitle:    "Click here to see all memories...",
			}
		},
	}

	htmlTemplates, err := template.New("").Funcs(funcMap).ParseGlob(filepath.Join(srcDir, "templates/*"))
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}

	templateParams, err := loadMemories(filepath.Join(srcDir, "memories/*.yaml"))
	if err != nil {
		return fmt.Errorf("parsing memories: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if err := renderPages(htmlTemplates, outputDir, templateParams); err != nil {
		return err
	}

	assetsIn := filepath.Join(srcDir, "assets")
	if _, err := os.Stat(assetsIn); err == nil {
		assetsOut := filepath.Join(outputDir, "assets")
		if err := os.MkdirAll(assetsOut, 0755); err != nil {
			return fmt.Errorf("creating assets output directory: %w", err)
		}
		if err := os.CopyFS(assetsOut, os.DirFS(assetsIn)); err != nil {
			return fmt.Errorf("copying assets: %w", err)
		}
	}

	return nil
}

func loadMemories(pattern string) (*sonostalgia.Sonostalgia, error) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return sonostalgia.LoadSonostalgia(files)
}

func renderPages(htmlTemplates *template.Template, outputDir string, templateParams *sonostalgia.Sonostalgia) error {
	staticPages := []page{
		{templateName: "style.css", outputName: "style.css"},
		{templateName: "about.template.html", outputName: "about.html", templateParams: templateParams.AboutParams},
		{templateName: "index.template.html", outputName: "index.html", templateParams: templateParams.IndexParams},
		{templateName: "memories.template.html", outputName: "memories.html", templateParams: templateParams.MemoriesParams},
		{templateName: "years.template.html", outputName: "years.html", templateParams: templateParams.YearsParams},
	}

	allMemories := make([]page, len(templateParams.MemoryParams))
	for i, memory := range templateParams.MemoryParams {
		allMemories[i] = page{
			templateName:   "memory.template.html",
			outputName:     fmt.Sprintf("%s.html", memory.OutputTitle),
			templateParams: memory,
		}
	}

	for _, p := range append(staticPages, allMemories...) {
		t := htmlTemplates.Lookup(p.templateName)
		log.Printf("Rendering template: %s", t.Name())

		outputPath := filepath.Join(outputDir, p.outputName)
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", outputPath, err)
		}

		err = t.Execute(f, p.templateParams)
		f.Close()
		if err != nil {
			return fmt.Errorf("executing template %s: %w", t.Name(), err)
		}

		log.Printf("Successfully created: %s", outputPath)
	}

	return nil
}
