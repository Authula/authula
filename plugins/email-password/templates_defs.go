package email_password

import (
	"embed"
	"io/fs"

	emailconstants "github.com/Authula/authula/internal/email/constants"
	emailtmpl "github.com/Authula/authula/internal/email/template"
)

//go:embed templates/*
var emailPasswordTemplateFiles embed.FS

func newEmailPasswordEmailTemplateManager() (*emailtmpl.Manager, error) {
	mgr := emailtmpl.NewManager()

	definitions := []emailtmpl.Definition{
		{
			Name:    emailconstants.VerifyEmailEmailTemplateName,
			Subject: mustLoadTemplate("templates/verify_email/subject.tmpl"),
			Text:    mustLoadTemplate("templates/verify_email/text.tmpl"),
			HTML:    mustLoadTemplate("templates/verify_email/html.tmpl"),
		},
		{
			Name:    emailconstants.PasswordResetRequestEmailTemplateName,
			Subject: mustLoadTemplate("templates/password_reset_request/subject.tmpl"),
			Text:    mustLoadTemplate("templates/password_reset_request/text.tmpl"),
			HTML:    mustLoadTemplate("templates/password_reset_request/html.tmpl"),
		},
		{
			Name:    emailconstants.PasswordChangedEmailTemplateName,
			Subject: mustLoadTemplate("templates/password_changed/subject.tmpl"),
			Text:    mustLoadTemplate("templates/password_changed/text.tmpl"),
			HTML:    mustLoadTemplate("templates/password_changed/html.tmpl"),
		},
		{
			Name:    emailconstants.EmailChangeRequestEmailTemplateName,
			Subject: mustLoadTemplate("templates/email_change_request/subject.tmpl"),
			Text:    mustLoadTemplate("templates/email_change_request/text.tmpl"),
			HTML:    mustLoadTemplate("templates/email_change_request/html.tmpl"),
		},
		{
			Name:    emailconstants.EmailChangedEmailTemplateName,
			Subject: mustLoadTemplate("templates/email_changed/subject.tmpl"),
			Text:    mustLoadTemplate("templates/email_changed/text.tmpl"),
			HTML:    mustLoadTemplate("templates/email_changed/html.tmpl"),
		},
	}

	for _, def := range definitions {
		if err := mgr.Register(def); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func mustLoadTemplate(path string) string {
	data, err := fs.ReadFile(emailPasswordTemplateFiles, path)
	if err != nil {
		panic(err)
	}
	return string(data)
}
