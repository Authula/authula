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

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func TestSessionPlugin_RenewSession(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	clientIP := "192.168.1.1"
	userAgent := "test-agent"

	tests := []struct {
		name          string
		setup         func(*internaltests.MockTokenService, *internaltests.MockSessionService)
		session       *models.Session
		wantCookie    bool
		wantToken     string
		wantSessionID string
	}{
		{
			name: "rotates token and creates new session",
			setup: func(mockTokenSvc *internaltests.MockTokenService, mockSessionSvc *internaltests.MockSessionService) {
				mockTokenSvc.On("Generate").Return("new-token", nil).Once()
				mockTokenSvc.On("Hash", "new-token").Return("hashed-new-token").Once()
				mockSessionSvc.On("Delete", mock.Anything, "session-id").Return(nil).Once()
				mockSessionSvc.On("Create", mock.Anything, "user-id", "hashed-new-token", &clientIP, &userAgent, time.Hour*24*7).
					Return(&models.Session{ID: "new-session-id", Token: "hashed-new-token", ExpiresAt: now.Add(time.Hour * 24 * 7)}, nil).Once()
			},
			session: &models.Session{
				ID:        "session-id",
				UserID:    "user-id",
				Token:     "hashed-old-token",
				ExpiresAt: now.Add(24 * time.Hour),
				IPAddress: &clientIP,
				UserAgent: &userAgent,
			},
			wantCookie:    true,
			wantToken:     "new-token",
			wantSessionID: "new-session-id",
		},
		{
			name: "token generation failure aborts renewal",
			setup: func(mockTokenSvc *internaltests.MockTokenService, mockSessionSvc *internaltests.MockSessionService) {
				mockTokenSvc.On("Generate").Return("", errors.New("generation failed")).Once()
			},
			session: &models.Session{
				ID:        "session-id",
				Token:     "hashed-old-token",
				IPAddress: &clientIP,
				UserAgent: &userAgent,
			},
			wantCookie: false,
		},
		{
			name: "delete failure aborts renewal",
			setup: func(mockTokenSvc *internaltests.MockTokenService, mockSessionSvc *internaltests.MockSessionService) {
				mockTokenSvc.On("Generate").Return("new-token", nil).Once()
				mockTokenSvc.On("Hash", "new-token").Return("hashed-new-token").Once()
				mockSessionSvc.On("Delete", mock.Anything, "session-id").Return(errors.New("delete failed")).Once()
			},
			session: &models.Session{
				ID:        "session-id",
				Token:     "hashed-old-token",
				IPAddress: &clientIP,
				UserAgent: &userAgent,
			},
			wantCookie: false,
		},
		{
			name: "create failure after delete aborts renewal",
			setup: func(mockTokenSvc *internaltests.MockTokenService, mockSessionSvc *internaltests.MockSessionService) {
				mockTokenSvc.On("Generate").Return("new-token", nil).Once()
				mockTokenSvc.On("Hash", "new-token").Return("hashed-new-token").Once()
				mockSessionSvc.On("Delete", mock.Anything, "session-id").Return(nil).Once()
				mockSessionSvc.On("Create", mock.Anything, "user-id", "hashed-new-token", &clientIP, &userAgent, time.Hour*24*7).
					Return(nil, errors.New("create failed")).Once()
			},
			session: &models.Session{
				ID:        "session-id",
				UserID:    "user-id",
				Token:     "hashed-old-token",
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

			tt.setup(mockTokenSvc, mockSessionSvc)

			plugin := &SessionPlugin{
				globalConfig: &models.Config{
					Session: models.SessionConfig{
						CookieName:   "authula.session_token",
						ExpiresIn:    time.Hour * 24 * 7,
						CookieMaxAge: time.Hour * 24 * 7,
						HttpOnly:     true,
						SameSite:     "lax",
					},
				},
				sessionService: mockSessionSvc,
				tokenService:   mockTokenSvc,
				logger:         new(internaltests.MockLogger),
			}

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.AddCookie(&http.Cookie{
				Name:  "authula.session_token",
				Value: "old-plaintext-token",
			})
			w := httptest.NewRecorder()

			plugin.renewSession(w, r, tt.session)

			cookies := w.Result().Cookies()
			if tt.wantCookie {
				require.Len(t, cookies, 1)
				assert.Equal(t, "authula.session_token", cookies[0].Name)
				assert.Equal(t, tt.wantToken, cookies[0].Value)
				assert.Equal(t, tt.wantSessionID, tt.session.ID, "session ID should be updated")
				assert.Equal(t, "hashed-new-token", tt.session.Token, "session token should be updated")
			} else {
				assert.Empty(t, cookies)
				assert.Equal(t, "hashed-old-token", tt.session.Token, "session token should not change on error")
			}

			mockTokenSvc.AssertExpectations(t)
			mockSessionSvc.AssertExpectations(t)
		})
	}
}
