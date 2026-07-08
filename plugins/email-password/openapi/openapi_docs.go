package openapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Authula/authula/openapi"
	"github.com/Authula/authula/plugins/email-password/types"
)

func RegisterOpenAPIDocs(svc openapi.OpenAPIService, basePath string) error {
	var errs []error

	errs = append(errs, svc.AddOperation(
		http.MethodPost,
		fmt.Sprintf("%s/email-password/sign-up", basePath),
		openapi.WithOperationID("signUp"),
		openapi.WithSummary("Register new user"),
		openapi.WithDescription("Registers a new user with email and password"),
		openapi.WithTags("Email Password Plugin"),
		openapi.WithRequest(&types.SignUpRequest{}),
		openapi.WithResponseStatus(http.StatusCreated, &types.SignUpResponse{}),
	))

	errs = append(errs, svc.AddOperation(
		http.MethodPost,
		fmt.Sprintf("%s/email-password/sign-in", basePath),
		openapi.WithOperationID("signIn"),
		openapi.WithSummary("Sign in"),
		openapi.WithDescription("Authenticates a user with email and password"),
		openapi.WithTags("Email Password Plugin"),
		openapi.WithRequest(&types.SignInRequest{}),
		openapi.WithResponseStatus(http.StatusOK, &types.SignInResponse{}),
	))

	errs = append(errs, svc.AddOperation(
		http.MethodGet,
		fmt.Sprintf("%s/email-password/verify-email", basePath),
		openapi.WithOperationID("verifyEmail"),
		openapi.WithSummary("Verify email"),
		openapi.WithDescription("Verifies an email address or processes a password reset token using a verification token"),
		openapi.WithTags("Email Password Plugin"),
		openapi.WithRequest(&types.VerifyEmailRequest{}),
		openapi.WithResponseStatus(http.StatusOK, &types.ChangePasswordResponse{}),
	))

	errs = append(errs, svc.AddOperation(
		http.MethodPost,
		fmt.Sprintf("%s/email-password/send-email-verification", basePath),
		openapi.WithOperationID("sendEmailVerification"),
		openapi.WithSummary("Send email verification"),
		openapi.WithDescription("Sends a verification email to the authenticated user"),
		openapi.WithTags("Email Password Plugin"),
		openapi.WithRequest(&types.SendEmailVerificationRequest{}),
		openapi.WithResponseStatus(http.StatusOK, &types.ChangePasswordResponse{}),
	))

	errs = append(errs, svc.AddOperation(
		http.MethodPost,
		fmt.Sprintf("%s/email-password/request-password-reset", basePath),
		openapi.WithOperationID("requestPasswordReset"),
		openapi.WithSummary("Request password reset"),
		openapi.WithDescription("Requests a password reset link to be sent to the user's email"),
		openapi.WithTags("Email Password Plugin"),
		openapi.WithRequest(&types.RequestPasswordResetRequest{}),
		openapi.WithResponseStatus(http.StatusOK, &types.ChangePasswordResponse{}),
	))

	errs = append(errs, svc.AddOperation(
		http.MethodPost,
		fmt.Sprintf("%s/email-password/change-password", basePath),
		openapi.WithOperationID("changePassword"),
		openapi.WithSummary("Change password"),
		openapi.WithDescription("Changes the user's password using a reset token"),
		openapi.WithTags("Email Password Plugin"),
		openapi.WithRequest(&types.ChangePasswordRequest{}),
		openapi.WithResponseStatus(http.StatusOK, &types.ChangePasswordResponse{}),
	))

	errs = append(errs, svc.AddOperation(
		http.MethodPost,
		fmt.Sprintf("%s/email-password/request-email-change", basePath),
		openapi.WithOperationID("requestEmailChange"),
		openapi.WithSummary("Request email change"),
		openapi.WithDescription("Requests to change the authenticated user's email address"),
		openapi.WithTags("Email Password Plugin"),
		openapi.WithRequest(&types.RequestEmailChangeRequest{}),
		openapi.WithResponseStatus(http.StatusOK, &types.ChangeEmailResponse{}),
	))

	return errors.Join(errs...)
}
