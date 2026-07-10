package magiclink

import (
	"embed"
	"io/fs"

	emailconstants "github.com/Authula/authula/core/email/constants"
	emailtmpl "github.com/Authula/authula/core/email/template"
)

//go:embed templates/*
var magicLinkTemplateFiles embed.FS

func newMagicLinkEmailTemplateManager() (*emailtmpl.Manager, error) {
	mgr := emailtmpl.NewManager()

	definitions := []emailtmpl.Definition{
		{
			Name:    emailconstants.MagicLinkSignInEmailTemplateName,
			Subject: mustLoadMagicLinkTemplate("templates/magic_link_sign_in/subject.tmpl"),
			Text:    mustLoadMagicLinkTemplate("templates/magic_link_sign_in/text.tmpl"),
			HTML:    mustLoadMagicLinkTemplate("templates/magic_link_sign_in/html.tmpl"),
		},
	}

	for _, def := range definitions {
		if err := mgr.Register(def); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func mustLoadMagicLinkTemplate(path string) string {
	data, err := fs.ReadFile(magicLinkTemplateFiles, path)
	if err != nil {
		panic(err)
	}
	return string(data)
}
