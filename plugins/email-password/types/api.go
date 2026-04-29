package types

import (
	"encoding/json"
	"strings"

	"github.com/Authula/authula/models"
)

type SignUpRequest struct {
	Name        string          `json:"name"`
	Email       string          `json:"email"`
	Password    string          `json:"password"`
	Image       *string         `json:"image,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CallbackURL *string         `json:"callback_url,omitempty"`
}

func (p *SignUpRequest) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	p.Email = strings.TrimSpace(p.Email)
	p.Password = strings.TrimSpace(p.Password)
	if p.Image != nil {
		*p.Image = strings.TrimSpace(*p.Image)
	}
	if p.CallbackURL != nil {
		*p.CallbackURL = strings.TrimSpace(*p.CallbackURL)
	}
	return nil
}

type SignUpResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string // Empty string if auto sign-in is disabled
}

type SignUpResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

type SignInRequest struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

func (p *SignInRequest) Validate() error {
	p.Email = strings.TrimSpace(p.Email)
	p.Password = strings.TrimSpace(p.Password)
	if p.CallbackURL != nil {
		*p.CallbackURL = strings.TrimSpace(*p.CallbackURL)
	}
	return nil
}

type SignInResult struct {
	User         *models.User
	Session      *models.Session
	SessionToken string
}

type SignInResponse struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

type VerifyEmailRequest struct {
	Token       string  `json:"token"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

func (p *VerifyEmailRequest) Validate() error {
	p.Token = strings.TrimSpace(p.Token)
	if p.CallbackURL != nil {
		*p.CallbackURL = strings.TrimSpace(*p.CallbackURL)
	}
	return nil
}

type SendEmailVerificationRequest struct {
	CallbackURL *string `json:"callback_url,omitempty"`
}

func (p *SendEmailVerificationRequest) Validate() error {
	if p.CallbackURL != nil {
		*p.CallbackURL = strings.TrimSpace(*p.CallbackURL)
	}
	return nil
}

type RequestPasswordResetRequest struct {
	Email       string  `json:"email"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

func (p *RequestPasswordResetRequest) Validate() error {
	p.Email = strings.TrimSpace(p.Email)
	if p.CallbackURL != nil {
		*p.CallbackURL = strings.TrimSpace(*p.CallbackURL)
	}
	return nil
}

type RequestEmailChangeRequest struct {
	NewEmail    string  `json:"new_email"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

func (p *RequestEmailChangeRequest) Validate() error {
	p.NewEmail = strings.TrimSpace(p.NewEmail)
	if p.CallbackURL != nil {
		*p.CallbackURL = strings.TrimSpace(*p.CallbackURL)
	}
	return nil
}

type ChangePasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (p *ChangePasswordRequest) Validate() error {
	p.Token = strings.TrimSpace(p.Token)
	p.Password = strings.TrimSpace(p.Password)
	return nil
}

type ChangePasswordResponse struct {
	Message string `json:"message"`
}

type ChangeEmailResponse struct {
	Message string `json:"message"`
}
