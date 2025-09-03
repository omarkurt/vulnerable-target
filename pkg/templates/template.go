package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
)

type Template struct {
	ID             string                    `yaml:"id"`
	Info           Info                      `yaml:"info"`
	ProofOfConcept map[string][]string       `yaml:"poc"`
	Remediation    []string                  `yaml:"remediation"`
	Providers      map[string]ProviderConfig `yaml:"providers"`
}

type Info struct {
	Name             string   `yaml:"name"`
	Description      string   `yaml:"description"`
	Author           string   `yaml:"author"`
	Target           []string `yaml:"target"`
	Type             string   `yaml:"type"`
	AffectedVersions []string `yaml:"affected_versions"`
	FixedVersion     string   `yaml:"fixed_version"`
	Cwe              string   `yaml:"cwe"`
	Cvss             Cvss     `yaml:"cvss"`
	Tags             []string `yaml:"tags"`
	References       []string `yaml:"references"`
}

type ProviderConfig struct {
	Path string `yaml:"path"`
}

type Cvss struct {
	Score   string `yaml:"score"`
	Metrics string `yaml:"metrics"`
}

var Templates = make(map[string]Template)

func Init() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal().Msgf("%v", err)
	}
	home := filepath.Join(wd, "templates")
	dirEntry, err := os.ReadDir(home)
	if err != nil {
		log.Fatal().Msgf("%v", err)
	}
	for _, entry := range dirEntry {
		template, err := LoadTemplate(filepath.Join(home, entry.Name()))
		if err != nil {
			log.Fatal().Msgf("%v", err)
		}
		if template.ID != entry.Name() {
			log.Fatal().Msgf("id and directory name should match")
		}
		Templates[template.ID] = template
	}
}

func List() {
	ListWithFilter("")
}

func ListWithFilter(filterTag string) {
	t := table.NewWriter()
	t.SetStyle(table.StyleDefault)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"ID", "Name", "Author", "Target", "Type", "Tags"})

	count := 0
	for _, template := range Templates {
		if filterTag != "" {
			hasTag := false
			for _, tag := range template.Info.Tags {
				if strings.EqualFold(tag, filterTag) || strings.Contains(strings.ToLower(tag), strings.ToLower(filterTag)) {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		tags := strings.Join(template.Info.Tags, ", ")
		targets := strings.Join(template.Info.Target, ", ")
		t.AppendRow(table.Row{
			template.ID,
			template.Info.Name,
			template.Info.Author,
			targets,
			template.Info.Type,
			tags,
		})
		count++
	}

	if count == 0 {
		if filterTag != "" {
			fmt.Printf("No templates found with tag matching '%s'\n", filterTag)
		} else {
			fmt.Println("No templates found")
		}
		return
	}

	if filterTag != "" {
		t.SetCaption("Found %d templates with tag matching '%s'", count, filterTag)
	} else {
		t.SetCaption("there are %d templates", count)
	}
	t.SetIndexColumn(0)
	t.Render()
}

func GetByID(templateID string) (*Template, error) {
	template := Templates[templateID]
	if template.ID == "" {
		return nil, fmt.Errorf("template %s not found", templateID)
	}
	return &template, nil
}
