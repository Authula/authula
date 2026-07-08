package openapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Authula/authula/openapi"
	"github.com/Authula/authula/plugins/totp/types"
)

func RegisterOpenAPIDocs(svc openapi.OpenAPIService, basePath string) error {
	return errors.Join(
		svc.AddOperation(
			http.MethodPost,
			fmt.Sprintf("%s/totp/enable", basePath),
			openapi.WithOperationID("enableTotp"),
			openapi.WithSummary("Enable TOTP"),
			openapi.WithDescription("Enables TOTP two-factor authentication for the authenticated user. Returns the TOTP URI and backup codes."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithResponseStatus(http.StatusOK, &types.EnableResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			fmt.Sprintf("%s/totp/disable", basePath),
			openapi.WithOperationID("disableTotp"),
			openapi.WithSummary("Disable TOTP"),
			openapi.WithDescription("Disables TOTP two-factor authentication for the authenticated user."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithResponseStatus(http.StatusOK, &types.DisableResponse{}),
		),
		svc.AddOperation(
			http.MethodGet,
			fmt.Sprintf("%s/totp/get-uri", basePath),
			openapi.WithOperationID("getTotpURI"),
			openapi.WithSummary("Get TOTP URI"),
			openapi.WithDescription("Returns the current TOTP URI for the authenticated user's authenticator app configuration."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithResponseStatus(http.StatusOK, &types.GetTOTPURIResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			fmt.Sprintf("%s/totp/verify", basePath),
			openapi.WithOperationID("verifyTotp"),
			openapi.WithSummary("Verify TOTP code"),
			openapi.WithDescription("Verifies a TOTP code from the authenticator app and completes authentication. Requires a pending TOTP cookie set during sign-in."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithRequest(&types.VerifyTOTPRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.VerifyTOTPResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			fmt.Sprintf("%s/totp/verify-backup-code", basePath),
			openapi.WithOperationID("verifyTotpBackupCode"),
			openapi.WithSummary("Verify backup code"),
			openapi.WithDescription("Verifies a backup code as an alternative to TOTP verification. Completes authentication if the code is valid."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithRequest(&types.VerifyBackupCodeRequest{}),
			openapi.WithResponseStatus(http.StatusOK, &types.VerifyBackupCodeResponse{}),
		),
		svc.AddOperation(
			http.MethodPost,
			fmt.Sprintf("%s/totp/generate-backup-codes", basePath),
			openapi.WithOperationID("generateTotpBackupCodes"),
			openapi.WithSummary("Generate backup codes"),
			openapi.WithDescription("Generates a new set of backup codes for the authenticated user. Previous backup codes are invalidated."),
			openapi.WithTags("TOTP Plugin"),
			openapi.WithResponseStatus(http.StatusOK, &types.GenerateBackupCodesResponse{}),
		),
	)
}
