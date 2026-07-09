package openapi

import (
	"errors"
	"net/http"

	"github.com/Authula/authula/openapi"
	"github.com/Authula/authula/plugins/totp/types"
)

func RegisterOpenAPIDocs(svc openapi.OpenAPIService) error {
	return errors.Join(
		svc.AddOperation(
			http.MethodPost,
			"/totp/enable",
			openapi.WithOperationID("enableTotp"),
			openapi.WithSummary("Enable TOTP"),
			openapi.WithDescription("Enables TOTP two-factor authentication for the authenticated user. Returns the TOTP URI and backup codes."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithResponseStatus(http.StatusOK, &types.EnableResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			"/totp/disable",
			openapi.WithOperationID("disableTotp"),
			openapi.WithSummary("Disable TOTP"),
			openapi.WithDescription("Disables TOTP two-factor authentication for the authenticated user."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithResponseStatus(http.StatusOK, &types.DisableResponse{}),
		),
		svc.AddOperation(
			http.MethodGet,
			"/totp/get-uri",
			openapi.WithOperationID("getTotpURI"),
			openapi.WithSummary("Get TOTP URI"),
			openapi.WithDescription("Returns the current TOTP URI for the authenticated user's authenticator app configuration."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithResponseStatus(http.StatusOK, &types.GetTOTPURIResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			"/totp/verify",
			openapi.WithOperationID("verifyTotp"),
			openapi.WithSummary("Verify TOTP code"),
			openapi.WithDescription("Verifies a TOTP code from the authenticator app and completes authentication. Requires a pending TOTP cookie set during sign-in."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithRequest(&types.VerifyTOTPRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.VerifyTOTPResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			"/totp/verify-backup-code",
			openapi.WithOperationID("verifyTotpBackupCode"),
			openapi.WithSummary("Verify backup code"),
			openapi.WithDescription("Verifies a backup code as an alternative to TOTP verification. Completes authentication if the code is valid."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithRequest(&types.VerifyBackupCodeRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.VerifyBackupCodeResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			"/totp/generate-backup-codes",
			openapi.WithOperationID("generateTotpBackupCodes"),
			openapi.WithSummary("Generate backup codes"),
			openapi.WithDescription("Generates a new set of backup codes for the authenticated user. Previous backup codes are invalidated."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithResponseStatus(http.StatusOK, &types.GenerateBackupCodesResponse{}),
		),
	)
}
