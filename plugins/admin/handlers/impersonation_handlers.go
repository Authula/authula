package handlers

import (
	"net/http"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/types"
	"github.com/Authula/authula/plugins/admin/usecases"
	"github.com/Authula/authula/util"
)

type GetAllImpersonationsHandler struct {
	useCase usecases.ImpersonationUseCase
}

func NewGetAllImpersonationsHandler(useCase usecases.ImpersonationUseCase) *GetAllImpersonationsHandler {
	return &GetAllImpersonationsHandler{useCase: useCase}
}

func (h *GetAllImpersonationsHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())

		rows, err := h.useCase.GetAllImpersonations(r.Context(), reqCtx.Actor)
		if err != nil {
			respondImpersonationError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, rows)
	}
}

type GetImpersonationByIDHandler struct {
	useCase usecases.ImpersonationUseCase
}

func NewGetImpersonationByIDHandler(useCase usecases.ImpersonationUseCase) *GetImpersonationByIDHandler {
	return &GetImpersonationByIDHandler{useCase: useCase}
}

func (h *GetImpersonationByIDHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())

		impersonationID := r.PathValue("impersonation_id")
		impersonation, err := h.useCase.GetImpersonationByID(r.Context(), reqCtx.Actor, impersonationID)
		if err != nil {
			respondImpersonationError(reqCtx, err)
			return
		}

		reqCtx.SetJSONResponse(http.StatusOK, &types.GetImpersonationByIDResponse{Impersonation: impersonation})
	}
}

type StartImpersonationHandler struct {
	useCase      usecases.ImpersonationUseCase
	globalConfig *models.Config
}

func NewStartImpersonationHandler(
	useCase usecases.ImpersonationUseCase,
	globalConfig *models.Config,
) *StartImpersonationHandler {
	return &StartImpersonationHandler{
		useCase:      useCase,
		globalConfig: globalConfig,
	}
}

func (h *StartImpersonationHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		actor := reqCtx.Actor
		if actor == nil || actor.ID == "" {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
			reqCtx.Handled = true
			return
		}

		impersonatorScopes := make([]string, len(reqCtx.Actor.Scopes))
		copy(impersonatorScopes, reqCtx.Actor.Scopes)

		var req types.StartImpersonationRequest
		if err := util.ParseJSON(r, &req); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}
		if err := req.Validate(); err != nil {
			reqCtx.SetJSONResponse(http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
			reqCtx.Handled = true
			return
		}

		originalCookieValue, _ := r.Cookie(h.globalConfig.Session.CookieName)
		originalCookieMaxAge := 0
		if originalCookieValue != nil {
			originalCookieMaxAge = originalCookieValue.MaxAge
		}
		originalCookieVal := ""
		if originalCookieValue != nil {
			originalCookieVal = originalCookieValue.Value
		}

		userAgent := r.UserAgent()
		result, err := h.useCase.StartImpersonation(
			r.Context(), actor, getSessionID(reqCtx),
			&reqCtx.ClientIP, &userAgent, req,
			impersonatorScopes, originalCookieVal, originalCookieMaxAge,
		)
		if err != nil {
			respondImpersonationError(reqCtx, err)
			return
		}

		if result == nil || result.Impersonation == nil {
			reqCtx.SetJSONResponse(http.StatusInternalServerError, map[string]any{"message": "failed to start impersonation"})
			reqCtx.Handled = true
			return
		}

		sessionConfig := h.globalConfig.Session
		sameSite := sameSiteFromSessionConfig(&sessionConfig)

		if result.OriginalCookieToken != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     sessionConfig.CookieName + constants.OriginalSessionCookieSuffix,
				Value:    result.OriginalCookieToken,
				Path:     "/",
				HttpOnly: sessionConfig.HttpOnly,
				Secure:   sessionConfig.Secure,
				SameSite: sameSite,
				MaxAge:   result.OriginalCookieMaxAge,
			})
		}

		reqCtx.SetActorInContext(&models.Actor{
			ID:   result.TargetUserID,
			Type: models.ActorUser,
			Claims: map[string]any{
				constants.ImpersonatorID:     result.ImpersonatorUserID,
				constants.ImpersonatorScopes: result.ImpersonatorScopes,
			},
		})
		if result.SessionID != nil && *result.SessionID != "" {
			reqCtx.Values[models.ContextSessionID.String()] = *result.SessionID
		}
		if result.SessionToken != nil && *result.SessionToken != "" {
			reqCtx.Values[models.ContextSessionToken.String()] = *result.SessionToken
			reqCtx.Values[models.ContextAuthSuccess.String()] = true
		}

		reqCtx.SetJSONResponse(http.StatusCreated, &types.StartImpersonationResponse{
			Impersonation: result.Impersonation,
		})
	}
}

type StopImpersonationHandler struct {
	useCase      usecases.ImpersonationUseCase
	globalConfig *models.Config
}

func NewStopImpersonationHandler(
	useCase usecases.ImpersonationUseCase,
	globalConfig *models.Config,
) *StopImpersonationHandler {
	return &StopImpersonationHandler{
		useCase:      useCase,
		globalConfig: globalConfig,
	}
}

func (h *StopImpersonationHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx, _ := models.GetRequestContext(r.Context())
		impersonatedUserID := getUserID(reqCtx)
		impersonatedSessionID := getSessionID(reqCtx)

		if impersonatedUserID == nil || impersonatedSessionID == nil {
			reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
			reqCtx.Handled = true
			return
		}

		originalCookieValue := ""
		originalCookie, err := r.Cookie(h.globalConfig.Session.CookieName + constants.OriginalSessionCookieSuffix)
		if err == nil {
			originalCookieValue = originalCookie.Value
		}

		if originalCookieValue == "" {
			if reqCtx.Actor == nil || reqCtx.Actor.Claims == nil {
				reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "no original session found"})
				reqCtx.Handled = true
				return
			}
			if _, ok := reqCtx.Actor.Claims[constants.ImpersonatorID]; !ok {
				reqCtx.SetJSONResponse(http.StatusUnauthorized, map[string]any{"message": "no original session found"})
				reqCtx.Handled = true
				return
			}
		}

		impersonationID := r.PathValue("impersonation_id")
		result, err := h.useCase.StopImpersonation(
			r.Context(), reqCtx.Actor, *impersonatedUserID, *impersonatedSessionID,
			originalCookieValue,
			types.StopImpersonationRequest{ImpersonationID: &impersonationID},
		)
		if err != nil {
			respondImpersonationError(reqCtx, err)
			return
		}

		sessionConfig := h.globalConfig.Session
		sameSite := sameSiteFromSessionConfig(&sessionConfig)

		http.SetCookie(w, &http.Cookie{
			Name:     sessionConfig.CookieName,
			Value:    result.OriginalSessionToken,
			Path:     "/",
			HttpOnly: sessionConfig.HttpOnly,
			Secure:   sessionConfig.Secure,
			SameSite: sameSite,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     sessionConfig.CookieName + constants.OriginalSessionCookieSuffix,
			Value:    "",
			Path:     "/",
			HttpOnly: sessionConfig.HttpOnly,
			Secure:   sessionConfig.Secure,
			SameSite: sameSite,
			MaxAge:   -1,
		})

		reqCtx.SetJSONResponse(http.StatusOK, &types.StopImpersonationResponse{Message: "Impersonation stopped"})
	}
}

func getUserID(reqCtx *models.RequestContext) *string {
	if reqCtx.Actor == nil || reqCtx.Actor.ID == "" {
		return nil
	}
	return &reqCtx.Actor.ID
}

func getSessionID(reqCtx *models.RequestContext) *string {
	value, ok := reqCtx.Values[models.ContextSessionID.String()]
	if !ok || value == nil {
		return nil
	}
	sessionID, ok := value.(string)
	if !ok || sessionID == "" {
		return nil
	}
	return &sessionID
}

func respondImpersonationError(reqCtx *models.RequestContext, err error) {
	reqCtx.SetJSONResponse(mapImpersonationErrorStatus(err), map[string]any{"message": mapAdminHttpErrorMessage(err)})
	reqCtx.Handled = true
}

func mapImpersonationErrorStatus(err error) int {
	return mapAdminHttpErrorStatus(err)
}

func sameSiteFromSessionConfig(sessionConfig *models.SessionConfig) http.SameSite {
	switch sessionConfig.SameSite {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteLaxMode
	}
}
