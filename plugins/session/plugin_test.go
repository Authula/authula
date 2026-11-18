package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

const (
	testCookieName = "authula.session_token"
	testExpiresIn  = time.Hour * 24 * 7
	testUpdateAge  = time.Hour * 24
)

func newTestPlugin(sessionSvc *internaltests.MockSessionService, tokenSvc *internaltests.MockTokenService) *SessionPlugin {
	return &SessionPlugin{
		globalConfig: &models.Config{
			Session: models.SessionConfig{
				CookieName:   testCookieName,
				ExpiresIn:    testExpiresIn,
				UpdateAge:    testUpdateAge,
				CookieMaxAge: testExpiresIn,
				HttpOnly:     true,
				Secure:       true,
				SameSite:     "strict",
			},
		},
		sessionService: sessionSvc,
		tokenService:   tokenSvc,
		logger:         new(internaltests.MockLogger),
	}
}

func TestSessionPlugin_RenewSession(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	clientIP := "192.168.1.1"
	userAgent := "test-agent"

	tests := []struct {
		name       string
		setup      func(*internaltests.MockSessionService)
		session    *models.Session
		wantCookie bool
	}{
		{
			name: "extends expiry in place and re-issues the same cookie value",
			setup: func(mockSessionSvc *internaltests.MockSessionService) {
				mockSessionSvc.On("Update", mock.Anything, mock.MatchedBy(func(s *models.Session) bool {
					// The existing row must be extended, never replaced
					return s.ID == "session-id" &&
						s.Token == "hashed-old-token" &&
						s.ExpiresAt.After(now.Add(testExpiresIn-time.Minute))
				})).Return(&models.Session{
					ID:        "session-id",
					UserID:    "user-id",
					Token:     "hashed-old-token",
					ExpiresAt: now.Add(testExpiresIn),
				}, nil).Once()
			},
			session: &models.Session{
				ID:        "session-id",
				UserID:    "user-id",
				Token:     "hashed-old-token",
				ExpiresAt: now.Add(time.Hour),
				IPAddress: &clientIP,
				UserAgent: &userAgent,
			},
			wantCookie: true,
		},
		{
			name: "update failure aborts renewal and leaves the session untouched",
			setup: func(mockSessionSvc *internaltests.MockSessionService) {
				mockSessionSvc.On("Update", mock.Anything, mock.Anything).
					Return(nil, errors.New("update failed")).Once()
			},
			session: &models.Session{
				ID:        "session-id",
				UserID:    "user-id",
				Token:     "hashed-old-token",
				ExpiresAt: now.Add(time.Hour),
				IPAddress: &clientIP,
				UserAgent: &userAgent,
			},
			wantCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokenSvc := new(internaltests.MockTokenService)
			mockSessionSvc := new(internaltests.MockSessionService)
			tt.setup(mockSessionSvc)

			plugin := newTestPlugin(mockSessionSvc, mockTokenSvc)
			originalExpiry := tt.session.ExpiresAt

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			plugin.renewSession(w, r, tt.session, "old-plaintext-token")

			cookies := w.Result().Cookies()
			if tt.wantCookie {
				require.Len(t, cookies, 1)
				assert.Equal(t, testCookieName, cookies[0].Name)
				assert.Equal(t, "old-plaintext-token", cookies[0].Value, "cookie value must not rotate")
				assert.Equal(t, "session-id", tt.session.ID, "session ID must not change")
				assert.True(t, tt.session.ExpiresAt.After(originalExpiry), "expiry should be extended")
			} else {
				assert.Empty(t, cookies)
				assert.Equal(t, originalExpiry, tt.session.ExpiresAt, "expiry must not change on error")
			}

			// Renewal must never delete or create rows, nor generate new tokens
			mockSessionSvc.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			mockSessionSvc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			mockTokenSvc.AssertNotCalled(t, "Generate")
			mockTokenSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}

func TestSessionPlugin_ShouldRenewSession(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{name: "far from expiry does not renew", expiresAt: now.Add(testExpiresIn), want: false},
		{name: "just outside update window does not renew", expiresAt: now.Add(testUpdateAge + time.Minute), want: false},
		{name: "inside update window renews", expiresAt: now.Add(testUpdateAge - time.Minute), want: true},
		{name: "already expired still reports renewable", expiresAt: now.Add(-time.Minute), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := newTestPlugin(new(internaltests.MockSessionService), new(internaltests.MockTokenService))

			got := plugin.shouldRenewSession(&models.Session{ExpiresAt: tt.expiresAt})

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionPlugin_SetSessionCookie(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name         string
		cookieMaxAge time.Duration
		expiresAt    time.Time
		wantMaxAge   int
	}{
		{
			name:         "uses cookie max age when session outlives it",
			cookieMaxAge: time.Hour,
			expiresAt:    now.Add(testExpiresIn),
			wantMaxAge:   int(time.Hour.Seconds()),
		},
		{
			name:         "clamps to remaining session lifetime when shorter than cookie max age",
			cookieMaxAge: testExpiresIn,
			expiresAt:    now.Add(2 * time.Hour),
			wantMaxAge:   int((2 * time.Hour).Seconds()),
		},
		{
			name:         "falls back to remaining session lifetime when cookie max age is unset",
			cookieMaxAge: 0,
			expiresAt:    now.Add(3 * time.Hour),
			wantMaxAge:   int((3 * time.Hour).Seconds()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := newTestPlugin(new(internaltests.MockSessionService), new(internaltests.MockTokenService))
			plugin.globalConfig.Session.CookieMaxAge = tt.cookieMaxAge
			w := httptest.NewRecorder()

			plugin.SetSessionCookie(w, "plaintext-token", tt.expiresAt)

			cookies := w.Result().Cookies()
			require.Len(t, cookies, 1)
			cookie := cookies[0]
			assert.Equal(t, testCookieName, cookie.Name)
			assert.Equal(t, "plaintext-token", cookie.Value)
			assert.Equal(t, "/", cookie.Path)
			assert.True(t, cookie.HttpOnly)
			assert.True(t, cookie.Secure)
			assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
			// Allow a second of slack for the time elapsed between building expiresAt and the cookie
			assert.InDelta(t, tt.wantMaxAge, cookie.MaxAge, 1)
		})
	}
}

func TestSessionPlugin_ClearSessionCookie(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin(new(internaltests.MockSessionService), new(internaltests.MockTokenService))
	w := httptest.NewRecorder()

	plugin.ClearSessionCookie(w)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	assert.Equal(t, testCookieName, cookie.Name)
	assert.Empty(t, cookie.Value)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite, "clear must carry the same SameSite as the set")
	assert.Equal(t, -1, cookie.MaxAge)
}

func TestSessionPlugin_CookieDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		domain     string
		wantDomain string
	}{
		{name: "host-only when no domain is configured", domain: "", wantDomain: ""},
		{name: "shares the configured parent domain", domain: "example.com", wantDomain: "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := newTestPlugin(new(internaltests.MockSessionService), new(internaltests.MockTokenService))
			plugin.globalConfig.Session.Domain = tt.domain

			setRecorder := httptest.NewRecorder()
			plugin.SetSessionCookie(setRecorder, "plaintext-token", time.Now().UTC().Add(time.Hour))
			setCookies := setRecorder.Result().Cookies()
			require.Len(t, setCookies, 1)
			assert.Equal(t, tt.wantDomain, setCookies[0].Domain)

			clearRecorder := httptest.NewRecorder()
			plugin.ClearSessionCookie(clearRecorder)
			clearCookies := clearRecorder.Result().Cookies()
			require.Len(t, clearCookies, 1)
			assert.Equal(t, tt.wantDomain, clearCookies[0].Domain, "clear must carry the same Domain as the set or the browser keeps the cookie")
		})
	}
}

func TestSessionPlugin_ValidateSessionHook(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name          string
		cookie        *http.Cookie
		existingActor *models.Actor
		setup         func(*internaltests.MockTokenService, *internaltests.MockSessionService)
		wantHandled   bool
		wantStatus    int
		wantActorID   string
		wantCookie    bool
	}{
		{
			name:        "missing cookie returns unauthorized",
			setup:       func(*internaltests.MockTokenService, *internaltests.MockSessionService) {},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:   "unknown token returns unauthorized without touching the cookie",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").Return(nil, nil).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:   "ErrSessionNotFound returns unauthorized without touching the cookie",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").Return(nil, coreerrors.ErrSessionNotFound).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:   "expired session returns unauthorized",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(-time.Minute)}, nil).Once()
				sessionSvc.On("Delete", mock.Anything, "s1").Return(nil).Once()
			},
			wantHandled: true,
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:   "valid session outside renewal window sets actor and no cookie",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(testExpiresIn)}, nil).Once()
			},
			wantActorID: "u1",
		},
		{
			name:   "valid session inside renewal window extends and re-issues cookie",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", Token: "hashed", ExpiresAt: now.Add(time.Hour)}, nil).Once()
				sessionSvc.On("Update", mock.Anything, mock.Anything).
					Return(&models.Session{ID: "s1", UserID: "u1", Token: "hashed", ExpiresAt: now.Add(testExpiresIn)}, nil).Once()
			},
			wantActorID: "u1",
			wantCookie:  true,
		},
		{
			name:          "actor already set by another auth plugin is left untouched",
			existingActor: &models.Actor{ID: "external", Type: models.ActorUser},
			setup:         func(*internaltests.MockTokenService, *internaltests.MockSessionService) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokenSvc := new(internaltests.MockTokenService)
			mockSessionSvc := new(internaltests.MockSessionService)
			tt.setup(mockTokenSvc, mockSessionSvc)
			plugin := newTestPlugin(mockSessionSvc, mockTokenSvc)

			r := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.cookie != nil {
				r.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()
			reqCtx := &models.RequestContext{
				Request:         r,
				ResponseWriter:  w,
				Actor:           tt.existingActor,
				Values:          make(map[string]any),
				ResponseHeaders: make(http.Header),
			}

			err := plugin.validateSessionHook(reqCtx)

			require.NoError(t, err)
			assert.Equal(t, tt.wantHandled, reqCtx.Handled)
			switch {
			case tt.wantHandled:
				assert.Equal(t, tt.wantStatus, reqCtx.ResponseStatus)
				assert.Nil(t, reqCtx.Actor)
			case tt.existingActor != nil:
				assert.Same(t, tt.existingActor, reqCtx.Actor)
			default:
				require.NotNil(t, reqCtx.Actor)
				assert.Equal(t, tt.wantActorID, reqCtx.Actor.ID)
				assert.Equal(t, "s1", reqCtx.Values[models.ContextSessionID.String()])
			}

			cookies := w.Result().Cookies()
			if tt.wantCookie {
				require.Len(t, cookies, 1)
				assert.Equal(t, "plaintext", cookies[0].Value)
			} else {
				assert.Empty(t, cookies, "a rejected or non-renewed request must never emit Set-Cookie")
			}

			mockTokenSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}

func TestSessionPlugin_ValidateSessionCookie(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name      string
		cookie    *http.Cookie
		setup     func(*internaltests.MockTokenService, *internaltests.MockSessionService)
		wantErr   error
		wantMsg   string
		wantToken string
	}{
		{
			name:    "missing cookie returns the request error",
			setup:   func(*internaltests.MockTokenService, *internaltests.MockSessionService) {},
			wantErr: http.ErrNoCookie,
		},
		{
			name:   "nil session without an error maps to ErrSessionNotFound",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").Return(nil, nil).Once()
			},
			wantErr: coreerrors.ErrSessionNotFound,
		},
		{
			name:   "repository sentinel is passed through",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").Return(nil, coreerrors.ErrSessionNotFound).Once()
			},
			wantErr: coreerrors.ErrSessionNotFound,
		},
		{
			name:   "expired session is deleted and rejected",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(-time.Minute)}, nil).Once()
				sessionSvc.On("Delete", mock.Anything, "s1").Return(nil).Once()
			},
			wantMsg: "session expired",
		},
		{
			name:   "valid session returns the session and the raw cookie value",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(time.Hour)}, nil).Once()
			},
			wantToken: "plaintext",
		},
		{
			name:   "delete failure still rejects the expired session",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(-time.Minute)}, nil).Once()
				sessionSvc.On("Delete", mock.Anything, "s1").Return(errors.New("db down")).Once()
			},
			wantMsg: "session expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockTokenSvc := new(internaltests.MockTokenService)
			mockSessionSvc := new(internaltests.MockSessionService)
			tt.setup(mockTokenSvc, mockSessionSvc)
			plugin := newTestPlugin(mockSessionSvc, mockTokenSvc)

			r := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.cookie != nil {
				r.AddCookie(tt.cookie)
			}

			session, rawToken, err := plugin.validateSessionCookie(r)

			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantMsg != "":
				require.EqualError(t, err, tt.wantMsg)
			default:
				require.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, "u1", session.UserID)
			}

			if err != nil {
				assert.Nil(t, session)
				assert.Empty(t, rawToken)
			} else {
				assert.Equal(t, tt.wantToken, rawToken)
			}

			mockTokenSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}

func TestSessionPlugin_ValidateSessionHookOptional(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name          string
		cookie        *http.Cookie
		existingActor *models.Actor
		setup         func(*internaltests.MockTokenService, *internaltests.MockSessionService)
		wantActorID   string
		wantCookie    bool
	}{
		{
			name:  "missing cookie is skipped silently",
			setup: func(*internaltests.MockTokenService, *internaltests.MockSessionService) {},
		},
		{
			name:   "unknown token is skipped silently",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").Return(nil, nil).Once()
			},
		},
		{
			name:   "expired session is deleted and skipped",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(-time.Minute)}, nil).Once()
				sessionSvc.On("Delete", mock.Anything, "s1").Return(nil).Once()
			},
		},
		{
			name:   "valid session sets the actor",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(testExpiresIn)}, nil).Once()
			},
			wantActorID: "u1",
		},
		{
			name:   "valid session inside renewal window refreshes the cookie",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(time.Hour)}, nil).Once()
				sessionSvc.On("Update", mock.Anything, mock.Anything).
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(testExpiresIn)}, nil).Once()
			},
			wantActorID: "u1",
			wantCookie:  true,
		},
		{
			name:          "actor already set by another auth plugin is left untouched",
			existingActor: &models.Actor{ID: "external", Type: models.ActorUser},
			setup:         func(*internaltests.MockTokenService, *internaltests.MockSessionService) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockTokenSvc := new(internaltests.MockTokenService)
			mockSessionSvc := new(internaltests.MockSessionService)
			tt.setup(mockTokenSvc, mockSessionSvc)
			plugin := newTestPlugin(mockSessionSvc, mockTokenSvc)

			r := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.cookie != nil {
				r.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()
			reqCtx := &models.RequestContext{
				Request:         r,
				ResponseWriter:  w,
				Actor:           tt.existingActor,
				Values:          make(map[string]any),
				ResponseHeaders: make(http.Header),
			}

			err := plugin.validateSessionHookOptional(reqCtx)

			require.NoError(t, err)
			assert.False(t, reqCtx.Handled)
			switch {
			case tt.existingActor != nil:
				assert.Same(t, tt.existingActor, reqCtx.Actor)
			case tt.wantActorID == "":
				assert.Nil(t, reqCtx.Actor)
			default:
				require.NotNil(t, reqCtx.Actor)
				assert.Equal(t, tt.wantActorID, reqCtx.Actor.ID)
				assert.Equal(t, "s1", reqCtx.Values[models.ContextSessionID.String()])
			}

			cookies := w.Result().Cookies()
			if tt.wantCookie {
				require.Len(t, cookies, 1)
				assert.Equal(t, "plaintext", cookies[0].Value)
			} else {
				assert.Empty(t, cookies, "a skipped request must never emit Set-Cookie")
			}

			mockTokenSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}

func TestSessionPlugin_AuthMiddleware(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name       string
		cookie     *http.Cookie
		setup      func(*internaltests.MockTokenService, *internaltests.MockSessionService)
		wantStatus int
		wantNext   bool
		wantCookie bool
	}{
		{
			name:       "missing cookie returns unauthorized and stops the chain",
			setup:      func(*internaltests.MockTokenService, *internaltests.MockSessionService) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "unknown token returns unauthorized",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").Return(nil, nil).Once()
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "expired session is deleted and rejected",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(-time.Minute)}, nil).Once()
				sessionSvc.On("Delete", mock.Anything, "s1").Return(nil).Once()
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "valid session sets the user in context",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(testExpiresIn)}, nil).Once()
			},
			wantStatus: http.StatusNoContent,
			wantNext:   true,
		},
		{
			name:   "valid session inside renewal window re-issues the cookie",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(time.Hour)}, nil).Once()
				sessionSvc.On("Update", mock.Anything, mock.Anything).
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(testExpiresIn)}, nil).Once()
			},
			wantStatus: http.StatusNoContent,
			wantNext:   true,
			wantCookie: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockTokenSvc := new(internaltests.MockTokenService)
			mockSessionSvc := new(internaltests.MockSessionService)
			tt.setup(mockTokenSvc, mockSessionSvc)
			plugin := newTestPlugin(mockSessionSvc, mockTokenSvc)

			nextCalled := false
			handler := plugin.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				assert.Equal(t, "u1", r.Context().Value(models.ContextUserID))
				assert.Equal(t, "s1", r.Context().Value(models.ContextSessionID))
				w.WriteHeader(http.StatusNoContent)
			}))

			r := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.cookie != nil {
				r.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.wantNext, nextCalled)
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantCookie {
				cookies := w.Result().Cookies()
				require.Len(t, cookies, 1)
				assert.Equal(t, "plaintext", cookies[0].Value)
			} else {
				assert.Empty(t, w.Result().Cookies(), "a rejected or non-renewed request must never emit Set-Cookie")
			}

			mockTokenSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}

func TestSessionPlugin_OptionalAuthMiddleware(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name        string
		cookie      *http.Cookie
		setup       func(*internaltests.MockTokenService, *internaltests.MockSessionService)
		wantActorID string
		wantCookie  bool
	}{
		{
			name:  "missing cookie passes through without an actor",
			setup: func(*internaltests.MockTokenService, *internaltests.MockSessionService) {},
		},
		{
			name:   "unknown token passes through without an actor",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").Return(nil, nil).Once()
			},
		},
		{
			name:   "expired session is deleted and passes through",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(-time.Minute)}, nil).Once()
				sessionSvc.On("Delete", mock.Anything, "s1").Return(nil).Once()
			},
		},
		{
			name:   "valid session sets the user in context",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(testExpiresIn)}, nil).Once()
			},
			wantActorID: "u1",
		},
		{
			name:   "valid session inside renewal window re-issues the cookie",
			cookie: &http.Cookie{Name: testCookieName, Value: "plaintext"},
			setup: func(tokenSvc *internaltests.MockTokenService, sessionSvc *internaltests.MockSessionService) {
				tokenSvc.On("Hash", "plaintext").Return("hashed").Once()
				sessionSvc.On("GetByToken", mock.Anything, "hashed").
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(time.Hour)}, nil).Once()
				sessionSvc.On("Update", mock.Anything, mock.Anything).
					Return(&models.Session{ID: "s1", UserID: "u1", ExpiresAt: now.Add(testExpiresIn)}, nil).Once()
			},
			wantActorID: "u1",
			wantCookie:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockTokenSvc := new(internaltests.MockTokenService)
			mockSessionSvc := new(internaltests.MockSessionService)
			tt.setup(mockTokenSvc, mockSessionSvc)
			plugin := newTestPlugin(mockSessionSvc, mockTokenSvc)

			nextCalled := false
			handler := plugin.OptionalAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				if tt.wantActorID == "" {
					assert.Nil(t, r.Context().Value(models.ContextUserID))
					assert.Nil(t, r.Context().Value(models.ContextSessionID))
				} else {
					assert.Equal(t, tt.wantActorID, r.Context().Value(models.ContextUserID))
					assert.Equal(t, "s1", r.Context().Value(models.ContextSessionID))
				}
				w.WriteHeader(http.StatusNoContent)
			}))

			r := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.cookie != nil {
				r.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			assert.True(t, nextCalled, "optional auth must always continue the chain")
			assert.Equal(t, http.StatusNoContent, w.Code)
			if tt.wantCookie {
				cookies := w.Result().Cookies()
				require.Len(t, cookies, 1)
				assert.Equal(t, "plaintext", cookies[0].Value)
			} else {
				assert.Empty(t, w.Result().Cookies(), "a non-renewed request must never emit Set-Cookie")
			}

			mockTokenSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}

func TestSessionPlugin_IssueSessionCookieHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		actor      *models.Actor
		values     map[string]any
		wantCookie bool
	}{
		{
			name:       "issues cookie bounded by session lifetime for user actor",
			actor:      &models.Actor{ID: "u1", Type: models.ActorUser},
			values:     map[string]any{models.ContextSessionToken.String(): "plaintext"},
			wantCookie: true,
		},
		{
			name:   "skips when no session token in context",
			actor:  &models.Actor{ID: "u1", Type: models.ActorUser},
			values: map[string]any{},
		},
		{
			name:   "skips when actor is nil",
			values: map[string]any{models.ContextSessionToken.String(): "plaintext"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := newTestPlugin(new(internaltests.MockSessionService), new(internaltests.MockTokenService))
			plugin.globalConfig.Session.CookieMaxAge = 30 * 24 * time.Hour // longer than ExpiresIn
			w := httptest.NewRecorder()
			reqCtx := &models.RequestContext{
				Request:        httptest.NewRequest(http.MethodPost, "/sign-in", nil),
				ResponseWriter: w,
				Actor:          tt.actor,
				Values:         tt.values,
			}

			err := plugin.issueSessionCookieHook(reqCtx)

			require.NoError(t, err)
			cookies := w.Result().Cookies()
			if !tt.wantCookie {
				assert.Empty(t, cookies)
				return
			}
			require.Len(t, cookies, 1)
			assert.Equal(t, "plaintext", cookies[0].Value)
			assert.InDelta(t, int(testExpiresIn.Seconds()), cookies[0].MaxAge, 1, "cookie must not outlive the session")
		})
	}
}

func sessionInitConfig(session models.SessionConfig) *models.Config {
	return &models.Config{Session: session}
}

func defaultInitSessionConfig() models.SessionConfig {
	return models.SessionConfig{
		CookieName:   testCookieName,
		ExpiresIn:    testExpiresIn,
		UpdateAge:    testUpdateAge,
		CookieMaxAge: testExpiresIn,
	}
}

func TestSessionPlugin_New(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  SessionPluginConfig
	}{
		{name: "keeps the enabled flag", cfg: SessionPluginConfig{Enabled: true}},
		{name: "keeps the disabled flag", cfg: SessionPluginConfig{Enabled: false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(tt.cfg)

			require.NotNil(t, plugin)
			assert.Equal(t, tt.cfg, plugin.pluginConfig)
			assert.Equal(t, tt.cfg, plugin.Config())
		})
	}
}

func TestSessionPlugin_Metadata(t *testing.T) {
	t.Parallel()

	metadata := New(SessionPluginConfig{Enabled: true}).Metadata()

	assert.Equal(t, models.PluginSession.String(), metadata.ID)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, "Provides cookie-based session authentication", metadata.Description)
}

func TestSessionPlugin_Config(t *testing.T) {
	t.Parallel()

	cfg := SessionPluginConfig{Enabled: true}
	plugin := New(cfg)

	assert.Equal(t, cfg, plugin.Config())
}

func TestSessionPlugin_Close(t *testing.T) {
	t.Parallel()

	require.NoError(t, New(SessionPluginConfig{Enabled: true}).Close())
}

func TestSessionPlugin_ApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   SessionPluginConfig
		want SessionPluginConfig
	}{
		{
			name: "zero value stays zero",
			in:   SessionPluginConfig{},
			want: SessionPluginConfig{},
		},
		{
			name: "custom enabled value is preserved",
			in:   SessionPluginConfig{Enabled: true},
			want: SessionPluginConfig{Enabled: true},
		},
		{
			name: "explicitly disabled value is preserved",
			in:   SessionPluginConfig{Enabled: false},
			want: SessionPluginConfig{Enabled: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.in
			cfg.ApplyDefaults()

			assert.Equal(t, tt.want, cfg)
		})
	}
}

func TestSessionPlugin_SessionHookIDString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   SessionHookID
		want string
	}{
		{name: "auth hook id", id: HookIDSessionAuth, want: "session.auth"},
		{name: "optional auth hook id", id: HookIDSessionAuthOptional, want: "session.auth.optional"},
		{name: "custom hook id", id: SessionHookID("session.custom"), want: "session.custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.id.String())
		})
	}
}

func TestSessionPlugin_Init(t *testing.T) {
	t.Parallel()

	mockUserSvc := new(internaltests.MockUserService)
	mockSessionSvc := new(internaltests.MockSessionService)
	mockTokenSvc := new(internaltests.MockTokenService)

	tests := []struct {
		name        string
		config      *models.Config
		services    map[string]any
		wantErr     string
		wantEnabled bool
	}{
		{
			name:     "config decode failure is returned",
			config:   &models.Config{Session: defaultInitSessionConfig(), Plugins: models.PluginsConfig{"session": 12345}},
			services: map[string]any{},
			wantErr:  "expected a map",
		},
		{
			name:     "missing user service is rejected",
			config:   sessionInitConfig(defaultInitSessionConfig()),
			services: map[string]any{},
			wantErr:  "user service not available",
		},
		{
			name:   "user service of the wrong type is rejected",
			config: sessionInitConfig(defaultInitSessionConfig()),
			services: map[string]any{
				models.ServiceUser.String(): "not-a-user-service",
			},
			wantErr: "user service not available",
		},
		{
			name:   "missing session service is rejected",
			config: sessionInitConfig(defaultInitSessionConfig()),
			services: map[string]any{
				models.ServiceUser.String(): mockUserSvc,
			},
			wantErr: "session service not available",
		},
		{
			name:   "session service of the wrong type is rejected",
			config: sessionInitConfig(defaultInitSessionConfig()),
			services: map[string]any{
				models.ServiceUser.String():    mockUserSvc,
				models.ServiceSession.String(): mockUserSvc,
			},
			wantErr: "session service not available",
		},
		{
			name:   "missing token service is rejected",
			config: sessionInitConfig(defaultInitSessionConfig()),
			services: map[string]any{
				models.ServiceUser.String():    mockUserSvc,
				models.ServiceSession.String(): mockSessionSvc,
			},
			wantErr: "token service not available",
		},
		{
			name:   "token service of the wrong type is rejected",
			config: sessionInitConfig(defaultInitSessionConfig()),
			services: map[string]any{
				models.ServiceUser.String():    mockUserSvc,
				models.ServiceSession.String(): mockSessionSvc,
				models.ServiceToken.String():   mockSessionSvc,
			},
			wantErr: "token service not available",
		},
		{
			name: "pre-parsed config is loaded and services are wired",
			config: &models.Config{
				Session:          defaultInitSessionConfig(),
				Plugins:          models.PluginsConfig{},
				PreParsedConfigs: map[string]any{models.PluginSession.String(): SessionPluginConfig{Enabled: true}},
			},
			services: map[string]any{
				models.ServiceUser.String():    mockUserSvc,
				models.ServiceSession.String(): mockSessionSvc,
				models.ServiceToken.String():   mockTokenSvc,
			},
			wantEnabled: true,
		},
		{
			name:   "raw plugin config map is decoded",
			config: &models.Config{Session: defaultInitSessionConfig(), Plugins: models.PluginsConfig{"session": map[string]any{"enabled": true}}},
			services: map[string]any{
				models.ServiceUser.String():    mockUserSvc,
				models.ServiceSession.String(): mockSessionSvc,
				models.ServiceToken.String():   mockTokenSvc,
			},
			wantEnabled: true,
		},
		{
			name:   "config without plugin entries keeps constructor values",
			config: sessionInitConfig(defaultInitSessionConfig()),
			services: map[string]any{
				models.ServiceUser.String():    mockUserSvc,
				models.ServiceSession.String(): mockSessionSvc,
				models.ServiceToken.String():   mockTokenSvc,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := new(internaltests.MockLogger)
			registry := &internaltests.TestServiceRegistry{Services: tt.services}
			ctx := &models.PluginContext{
				Logger:          logger,
				ServiceRegistry: registry,
				GetConfig:       func() *models.Config { return tt.config },
			}
			plugin := New(SessionPluginConfig{Enabled: false})

			err := plugin.Init(ctx)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.False(t, plugin.pluginConfig.Enabled)
				return
			}

			require.NoError(t, err)
			assert.Same(t, ctx, plugin.ctx)
			assert.Same(t, logger, plugin.logger)
			assert.Same(t, tt.config, plugin.globalConfig)
			assert.Same(t, mockUserSvc, plugin.userService)
			assert.Same(t, mockSessionSvc, plugin.sessionService)
			assert.Same(t, mockTokenSvc, plugin.tokenService)
			assert.Equal(t, tt.wantEnabled, plugin.pluginConfig.Enabled)
		})
	}
}

func TestSessionPlugin_InitWarning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		updateAge time.Duration
		expiresIn time.Duration
		wantWarn  bool
	}{
		{
			name:      "warns when update age equals expires in",
			updateAge: time.Hour,
			expiresIn: time.Hour,
			wantWarn:  true,
		},
		{
			name:      "warns when update age is longer than expires in",
			updateAge: 2 * time.Hour,
			expiresIn: time.Hour,
			wantWarn:  true,
		},
		{
			name:      "stays quiet when update age is shorter than expires in",
			updateAge: time.Minute,
			expiresIn: time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := new(internaltests.LevelRecordingLogger)
			registry := &internaltests.TestServiceRegistry{Services: map[string]any{
				models.ServiceUser.String():    new(internaltests.MockUserService),
				models.ServiceSession.String(): new(internaltests.MockSessionService),
				models.ServiceToken.String():   new(internaltests.MockTokenService),
			}}
			cfg := sessionInitConfig(models.SessionConfig{
				CookieName: testCookieName,
				ExpiresIn:  tt.expiresIn,
				UpdateAge:  tt.updateAge,
			})
			ctx := &models.PluginContext{
				Logger:          logger,
				ServiceRegistry: registry,
				GetConfig:       func() *models.Config { return cfg },
			}

			require.NoError(t, New(SessionPluginConfig{Enabled: true}).Init(ctx))

			warnings := 0
			for _, record := range logger.Records() {
				if record.Level == "warn" {
					warnings++
				}
			}
			if tt.wantWarn {
				assert.Equal(t, 1, warnings, "expected the short update window warning")
			} else {
				assert.Zero(t, warnings, "unexpected warning")
			}
		})
	}
}

func TestSessionPlugin_BuildHooks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		index          int
		wantStage      models.HookStage
		wantPluginID   string
		wantOrder      int
		wantMatcher    bool
		wantHandlerSet bool
	}{
		{
			name:           "required auth validation runs before the route",
			index:          0,
			wantStage:      models.HookBefore,
			wantPluginID:   HookIDSessionAuth.String(),
			wantOrder:      10,
			wantHandlerSet: true,
		},
		{
			name:           "optional auth validation runs before the route",
			index:          1,
			wantStage:      models.HookBefore,
			wantPluginID:   HookIDSessionAuthOptional.String(),
			wantOrder:      10,
			wantHandlerSet: true,
		},
		{
			name:           "cookie issuance runs after a successful auth",
			index:          2,
			wantStage:      models.HookAfter,
			wantOrder:      10,
			wantMatcher:    true,
			wantHandlerSet: true,
		},
		{
			name:           "cookie clearing runs after sign out",
			index:          3,
			wantStage:      models.HookAfter,
			wantOrder:      10,
			wantMatcher:    true,
			wantHandlerSet: true,
		},
	}

	hooks := New(SessionPluginConfig{Enabled: true}).Hooks()
	require.Len(t, hooks, len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hook := hooks[tt.index]
			assert.Equal(t, tt.wantStage, hook.Stage)
			assert.Equal(t, tt.wantPluginID, hook.PluginID)
			assert.Equal(t, tt.wantOrder, hook.Order)
			assert.False(t, hook.Async, "auth hooks must never run asynchronously")
			if tt.wantMatcher {
				assert.NotNil(t, hook.Matcher)
			} else {
				assert.Nil(t, hook.Matcher)
			}
			if tt.wantHandlerSet {
				assert.NotNil(t, hook.Handler)
			}
		})
	}

	t.Run("buildHooks and Hooks return the same wiring", func(t *testing.T) {
		t.Parallel()

		plugin := New(SessionPluginConfig{Enabled: true})
		built := plugin.buildHooks()
		exposed := plugin.Hooks()

		require.Len(t, exposed, len(built))
		for i := range built {
			assert.Equal(t, built[i].Stage, exposed[i].Stage)
			assert.Equal(t, built[i].PluginID, exposed[i].PluginID)
			assert.Equal(t, built[i].Order, exposed[i].Order)
			assert.Equal(t, built[i].Matcher == nil, exposed[i].Matcher == nil)
		}
	})

	t.Run("sign out hook clears the session cookie", func(t *testing.T) {
		t.Parallel()

		plugin := New(SessionPluginConfig{Enabled: true})
		plugin.globalConfig = sessionInitConfig(defaultInitSessionConfig())
		w := httptest.NewRecorder()
		reqCtx := &models.RequestContext{
			Request:         httptest.NewRequest(http.MethodPost, "/sign-out", nil),
			ResponseWriter:  w,
			Values:          make(map[string]any),
			ResponseHeaders: make(http.Header),
		}

		require.NoError(t, plugin.buildHooks()[3].Handler(reqCtx))

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, testCookieName, cookies[0].Name)
		assert.Empty(t, cookies[0].Value)
		assert.Less(t, cookies[0].MaxAge, 0, "cookie must be expired")
		assert.False(t, reqCtx.Handled)
	})
}

func TestSessionPlugin_AuthSuccessMatcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]any
		want   bool
	}{
		{name: "true auth success value matches", values: map[string]any{models.ContextAuthSuccess.String(): true}, want: true},
		{name: "false auth success value does not match", values: map[string]any{models.ContextAuthSuccess.String(): false}},
		{name: "missing auth success value does not match", values: map[string]any{}},
		{name: "nil values do not match"},
		{name: "non-boolean auth success value does not match", values: map[string]any{models.ContextAuthSuccess.String(): "true"}},
		{name: "sign out value is ignored", values: map[string]any{models.ContextAuthSignOut.String(): true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(SessionPluginConfig{Enabled: true})
			reqCtx := &models.RequestContext{Values: tt.values}

			assert.Equal(t, tt.want, plugin.authSuccessMatcher(reqCtx))
		})
	}
}

func TestSessionPlugin_SignedOutMatcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]any
		want   bool
	}{
		{name: "true sign out value matches", values: map[string]any{models.ContextAuthSignOut.String(): true}, want: true},
		{name: "false sign out value does not match", values: map[string]any{models.ContextAuthSignOut.String(): false}},
		{name: "missing sign out value does not match", values: map[string]any{}},
		{name: "nil values do not match"},
		{name: "non-boolean sign out value does not match", values: map[string]any{models.ContextAuthSignOut.String(): 1}},
		{name: "auth success value is ignored", values: map[string]any{models.ContextAuthSuccess.String(): true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(SessionPluginConfig{Enabled: true})
			reqCtx := &models.RequestContext{Values: tt.values}

			assert.Equal(t, tt.want, plugin.signedOutMatcher(reqCtx))
		})
	}
}

func TestSessionPlugin_ClearSessionCookieHook(t *testing.T) {
	t.Parallel()

	plugin := New(SessionPluginConfig{Enabled: true})
	plugin.globalConfig = sessionInitConfig(defaultInitSessionConfig())
	w := httptest.NewRecorder()
	reqCtx := &models.RequestContext{
		Request:         httptest.NewRequest(http.MethodPost, "/sign-out", nil),
		ResponseWriter:  w,
		Values:          make(map[string]any),
		ResponseHeaders: make(http.Header),
	}

	err := plugin.clearSessionCookieHook(reqCtx)

	require.NoError(t, err)
	assert.False(t, reqCtx.Handled)
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, testCookieName, cookies[0].Name)
	assert.Empty(t, cookies[0].Value, "cleared cookie must carry no value")
	assert.Less(t, cookies[0].MaxAge, 0, "cleared cookie must expire immediately")
}
