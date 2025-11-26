package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"text/template"
	"time"

	service "github.com/flaviogonzalez/instant-layer/internal/services"
)

// top config
type Layer struct {
	Name        string    `json:"name"` // project name
	Root        string    `json:"root"`
	GeneratedAt time.Time `json:"generated_at"`
	Services    []*service.Service
}

func (l *Layer) Save() error {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}

	err = os.Mkdir(l.Root, 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(l.Root, "layer.json"), data, 0644)
}

func (l *Layer) RegenerateDockerCompose() error {
	const tmpl = `name: {{.Name}}

services:
{{range .Services}}  {{.Name}}:
    build: ./{{.Dir}}
    ports:
      - "{{.Port}}:{{.Port}}"
    restart: unless-stopped
{{end}}
networks:
  default:
    name: {{.Name}}-network
`

	t := template.Must(template.New("dc").Parse(tmpl))
	path := filepath.Join(l.Root, "docker-compose.yml")

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, l)
}

func (l *Layer) Reload() error {
	path := filepath.Join(l.Root, "layer.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, l)
}
