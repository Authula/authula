package organizations

import (
	"embed"
	"io/fs"

	emailconstants "github.com/Authula/authula/core/email/constants"
	emailtmpl "github.com/Authula/authula/core/email/template"
)

//go:embed templates/*
var organizationTemplateFiles embed.FS

func newOrganizationEmailTemplateManager() (*emailtmpl.Manager, error) {
	mgr := emailtmpl.NewManager()

	definitions := []emailtmpl.Definition{
		{
			Name:    emailconstants.OrganizationInvitationEmailTemplateName,
			Subject: mustLoadOrgTemplate("templates/organization_invitation/subject.tmpl"),
			Text:    mustLoadOrgTemplate("templates/organization_invitation/text.tmpl"),
			HTML:    mustLoadOrgTemplate("templates/organization_invitation/html.tmpl"),
		},
	}

	for _, def := range definitions {
		if err := mgr.Register(def); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func mustLoadOrgTemplate(path string) string {
	data, err := fs.ReadFile(organizationTemplateFiles, path)
	if err != nil {
		panic(err)
	}
	return string(data)
}
