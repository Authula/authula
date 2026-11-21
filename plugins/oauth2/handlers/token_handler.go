package handlers

import (
	"encoding/base64"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
)

const tokenGrantType = "client_credentials"

// TokenHandler serves the OAuth 2.0 client credentials token endpoint.
type TokenHandler struct {
	UseCase usecases.ClientCredentialsUseCase
	Logger  models.Logger
}

func (h *TokenHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())

		form, ok := h.parseForm(reqCtx, r)
		if !ok {
			h.writeError(reqCtx, http.StatusBadRequest, types.TokenErrorResponse{
				Error:            "invalid_request",
				ErrorDescription: "invalid token request",
			}, nil)
			return
		}

		grantType, present, duplicate := formValue(form, "grant_type")
		clientID, clientIDPresent, clientIDDuplicate := formValue(form, "client_id")
		clientSecret, clientSecretPresent, clientSecretDuplicate := formValue(form, "client_secret")
		scope, scopePresent, scopeDuplicate := formValue(form, "scope")

		if duplicate || clientIDDuplicate || clientSecretDuplicate || scopeDuplicate {
			h.writeError(reqCtx, http.StatusBadRequest, types.TokenErrorResponse{
				Error:            "invalid_request",
				ErrorDescription: "duplicate token request parameter",
			}, nil)
			return
		}

		authHeaders := r.Header.Values("Authorization")
		if len(authHeaders) > 1 {
			h.writeClientError(reqCtx)
			return
		}

		basicUsed := false
		if len(authHeaders) == 1 {
			basicID, basicSecret, ok := parseBasicCredentials(authHeaders[0])
			if !ok {
				h.writeClientError(reqCtx)
				return
			}
			basicUsed = true
			clientID = basicID
			clientSecret = basicSecret
		}

		if basicUsed {
			if clientIDPresent || clientSecretPresent {
				h.writeError(reqCtx, http.StatusBadRequest, types.TokenErrorResponse{
					Error:            "invalid_request",
					ErrorDescription: "multiple client authentication methods",
				}, nil)
				return
			}
		} else if !clientIDPresent || !clientSecretPresent || clientID == "" || clientSecret == "" {
			h.writeClientError(reqCtx)
			return
		}

		if !present {
			h.writeError(reqCtx, http.StatusBadRequest, types.TokenErrorResponse{
				Error:            "invalid_request",
				ErrorDescription: "grant_type is required",
			}, nil)
			return
		}
		if grantType != tokenGrantType {
			h.writeError(reqCtx, http.StatusBadRequest, types.TokenErrorResponse{
				Error:            "unsupported_grant_type",
				ErrorDescription: "unsupported grant type",
			}, nil)
			return
		}

		req := &types.TokenRequest{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		}
		if scopePresent {
			req.Scope = &scope
		}

		result, err := h.UseCase.IssueToken(r.Context(), req)
		if err != nil {
			h.writeUseCaseError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.TokenResponse{
			AccessToken: result.AccessToken,
			TokenType:   result.TokenType,
			ExpiresIn:   int64(result.ExpiresIn / time.Second),
			Scope:       types.JoinScope(result.Scopes),
		})
		setTokenResponseHeaders(reqCtx, http.StatusOK)
	}
}

func (h *TokenHandler) parseForm(reqCtx *models.RequestContext, r *http.Request) (url.Values, bool) {
	contentTypes := r.Header.Values("Content-Type")
	if len(contentTypes) != 1 {
		return nil, false
	}

	mediaType, _, err := mime.ParseMediaType(contentTypes[0])
	if err != nil || !strings.EqualFold(mediaType, "application/x-www-form-urlencoded") {
		return nil, false
	}

	r.Body = http.MaxBytesReader(reqCtx.ResponseWriter, r.Body, constants.TokenRequestMaxBodyBytes)
	if err := r.ParseForm(); err != nil {
		return nil, false
	}

	return r.PostForm, true
}

func (h *TokenHandler) writeUseCaseError(reqCtx *models.RequestContext, err error) {
	switch {
	case errors.Is(err, constants.ErrInvalidClient):
		h.writeClientError(reqCtx)
	case errors.Is(err, constants.ErrUnauthorizedClient):
		h.writeError(reqCtx, http.StatusBadRequest, types.TokenErrorResponse{
			Error:            "unauthorized_client",
			ErrorDescription: "client is not allowed to use the client_credentials grant",
		}, nil)
	case errors.Is(err, constants.ErrInvalidScope):
		h.writeError(reqCtx, http.StatusBadRequest, types.TokenErrorResponse{
			Error:            "invalid_scope",
			ErrorDescription: "requested scope is not allowed",
		}, nil)
	default:
		h.writeError(reqCtx, http.StatusInternalServerError, types.TokenErrorResponse{}, err)
	}
}

func (h *TokenHandler) writeClientError(reqCtx *models.RequestContext) {
	h.writeError(reqCtx, http.StatusUnauthorized, types.TokenErrorResponse{
		Error:            "invalid_client",
		ErrorDescription: "client authentication failed",
	}, nil)
}

func (h *TokenHandler) writeError(reqCtx *models.RequestContext, status int, payload types.TokenErrorResponse, cause error) {
	if status >= http.StatusInternalServerError {
		if h.Logger != nil && cause != nil {
			h.Logger.Error("oauth2 token request failed", "error", cause)
		}
		reqCtx.SetJSONResponse(status, &types.ErrorResponse{Message: "internal server error"})
	} else {
		reqCtx.SetJSONResponse(status, payload)
	}

	setTokenResponseHeaders(reqCtx, status)
	reqCtx.Handled = true
}

func setTokenResponseHeaders(reqCtx *models.RequestContext, status int) {
	if reqCtx.ResponseHeaders == nil {
		reqCtx.ResponseHeaders = make(http.Header)
	}
	reqCtx.ResponseHeaders.Set("Cache-Control", "no-store")
	reqCtx.ResponseHeaders.Set("Pragma", "no-cache")
	if status == http.StatusUnauthorized {
		reqCtx.ResponseHeaders.Set("WWW-Authenticate", `Basic realm="oauth2/token", charset="UTF-8"`)
	}
}

func formValue(form url.Values, key string) (value string, present bool, duplicate bool) {
	values, ok := form[key]
	if !ok {
		return "", false, false
	}
	if len(values) > 1 {
		return "", true, true
	}
	return values[0], true, false
}

func parseBasicCredentials(header string) (string, string, bool) {
	scheme, encoded, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Basic") {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", false
	}

	clientID, clientSecret, found := strings.Cut(string(decoded), ":")
	if !found {
		return "", "", false
	}

	clientID, err = url.QueryUnescape(clientID)
	if err != nil {
		return "", "", false
	}
	clientSecret, err = url.QueryUnescape(clientSecret)
	if err != nil {
		return "", "", false
	}

	if clientID == "" || clientSecret == "" {
		return "", "", false
	}
	return clientID, clientSecret, true
}
