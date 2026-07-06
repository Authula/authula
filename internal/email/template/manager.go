package emailtemplate

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"os"
	"strings"
	texttemplate "text/template"
)

type Definition struct {
	Name    string
	Subject string
	Text    string
	HTML    string
}

type EmailTemplate struct {
	subjectTmpl *texttemplate.Template
	textTmpl    *texttemplate.Template
	htmlTmpl    *htmltemplate.Template
}

func (et *EmailTemplate) Execute(data any) (subject, text, html string, err error) {
	var buf bytes.Buffer
	if err := et.subjectTmpl.Execute(&buf, data); err != nil {
		return "", "", "", err
	}
	subject = buf.String()

	buf.Reset()
	if err := et.textTmpl.Execute(&buf, data); err != nil {
		return "", "", "", err
	}
	text = buf.String()

	buf.Reset()
	if err := et.htmlTmpl.Execute(&buf, data); err != nil {
		return "", "", "", err
	}
	html = buf.String()

	return
}

type Manager struct {
	templates map[string]*EmailTemplate
}

func NewManager() *Manager {
	return &Manager{
		templates: make(map[string]*EmailTemplate),
	}
}

func envVarName(name, suffix string) string {
	return "AUTHULA_TEMPLATE_" + strings.ToUpper(name) + "_" + suffix
}

func (m *Manager) Register(def Definition) error {
	subjectText := def.Subject
	if v := os.Getenv(envVarName(def.Name, "SUBJECT")); v != "" {
		subjectText = v
	}
	textText := def.Text
	if v := os.Getenv(envVarName(def.Name, "TEXT")); v != "" {
		textText = v
	}
	htmlText := def.HTML
	if v := os.Getenv(envVarName(def.Name, "HTML")); v != "" {
		htmlText = v
	}

	textFuncs := texttemplate.FuncMap{
		"humanDuration": FormatDuration,
		"plural":        Plural,
		"upper":         Upper,
		"lower":         Lower,
		"title":         Title,
		"currentYear":   CurrentYear,
	}

	htmlFuncs := htmltemplate.FuncMap{
		"humanDuration": FormatDuration,
		"plural":        Plural,
		"upper":         Upper,
		"lower":         Lower,
		"title":         Title,
		"currentYear":   CurrentYear,
	}

	subjectTmpl, err := texttemplate.New(def.Name + "_subject").Funcs(textFuncs).Parse(subjectText)
	if err != nil {
		return fmt.Errorf("failed to parse subject template %q: %w", def.Name, err)
	}

	textTmpl, err := texttemplate.New(def.Name + "_text").Funcs(textFuncs).Parse(textText)
	if err != nil {
		return fmt.Errorf("failed to parse text template %q: %w", def.Name, err)
	}

	htmlTmpl, err := htmltemplate.New(def.Name + "_html").Funcs(htmlFuncs).Option("missingkey=error").Parse(htmlText)
	if err != nil {
		return fmt.Errorf("failed to parse html template %q: %w", def.Name, err)
	}

	m.templates[def.Name] = &EmailTemplate{
		subjectTmpl: subjectTmpl,
		textTmpl:    textTmpl,
		htmlTmpl:    htmlTmpl,
	}

	return nil
}

func (m *Manager) Render(name string, data any) (subject, text, html string, err error) {
	tmpl, ok := m.templates[name]
	if !ok {
		return "", "", "", fmt.Errorf("email template %q not found", name)
	}
	return tmpl.Execute(data)
}
