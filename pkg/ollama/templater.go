package ollama

import (
	"bytes"
	"text/template"
)

// RenderTemplate renders a prompt template with the provided data.
func RenderTemplate(tmpl string, data any) (string, error) {
	tpl, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}
