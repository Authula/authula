package email

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	emailtypes "github.com/Authula/authula/plugins/email/types"
	rootservices "github.com/Authula/authula/services"
)

// fakeMailer records delivery attempts so tests can assert fallback ordering.
type fakeMailer struct {
	mock.Mock
}

func (m *fakeMailer) SendEmail(ctx context.Context, to string, subject string, text string, html string) error {
	args := m.Called(ctx, to, subject, text, html)
	return args.Error(0)
}

func TestEmailService_SendEmail(t *testing.T) {
	const (
		to      = "user@example.com"
		subject = "Welcome"
		text    = "Hello"
		html    = "<p>Hello</p>"
	)

	primaryErr := errors.New("primary provider down")
	fallbackErr := errors.New("fallback provider down")
	ctx := context.Background()

	tests := []struct {
		name          string
		primaryErr    error
		withFallback  bool
		fallbackErr   error
		wantErr       error
		wantFallback  bool
		wantLogLevels []string
	}{
		{
			name:          "delivers through the primary provider",
			wantLogLevels: nil,
		},
		{
			name:          "does not call the fallback when the primary succeeds",
			withFallback:  true,
			wantLogLevels: nil,
		},
		{
			name:          "falls back when the primary fails",
			primaryErr:    primaryErr,
			withFallback:  true,
			wantFallback:  true,
			wantLogLevels: []string{"warn", "info"},
		},
		{
			name:          "returns the primary error when the fallback also fails",
			primaryErr:    primaryErr,
			withFallback:  true,
			fallbackErr:   fallbackErr,
			wantErr:       primaryErr,
			wantFallback:  true,
			wantLogLevels: []string{"warn", "error"},
		},
		{
			name:          "returns the primary error when no fallback is configured",
			primaryErr:    primaryErr,
			wantErr:       primaryErr,
			wantLogLevels: []string{"warn"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &internaltests.LevelRecordingLogger{}

			primary := new(fakeMailer)
			primary.On("SendEmail", ctx, to, subject, text, html).Return(tt.primaryErr)

			var fallback *fakeMailer
			var fallbackService rootservices.MailerService
			if tt.withFallback {
				fallback = new(fakeMailer)
				fallbackService = fallback
				if tt.wantFallback {
					fallback.On("SendEmail", ctx, to, subject, text, html).Return(tt.fallbackErr)
				}
			}

			service, err := NewEmailService(logger, &emailtypes.EmailPluginConfig{}, primary, fallbackService)
			require.NoError(t, err)

			gotErr := service.SendEmail(ctx, to, subject, text, html)

			if tt.wantErr != nil {
				require.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, tt.wantErr)
			} else {
				require.NoError(t, gotErr)
			}

			primary.AssertExpectations(t)

			if fallback != nil {
				fallback.AssertExpectations(t)
				if !tt.wantFallback {
					fallback.AssertNotCalled(t, "SendEmail", ctx, to, subject, text, html)
				}
			}

			records := logger.Records()
			require.Len(t, records, len(tt.wantLogLevels))
			for i, wantLevel := range tt.wantLogLevels {
				assert.Equal(t, wantLevel, records[i].Level, "log %d level", i)
			}

			if tt.primaryErr != nil {
				assert.Contains(t, records[0].Message, "primary email provider failed")
			}
			if tt.wantFallback && tt.fallbackErr == nil {
				assert.Contains(t, records[1].Message, "fallback email provider succeeded")
			}
			if tt.fallbackErr != nil {
				assert.Contains(t, records[1].Message, "fallback email provider also failed")
			}
		})
	}
}

func TestMailerServiceAdapter_SendEmail(t *testing.T) {
	const (
		to      = "user@example.com"
		subject = "Welcome"
		text    = "Hello"
		html    = "<p>Hello</p>"
	)

	sendErr := errors.New("delivery failed")
	ctx := context.Background()

	tests := []struct {
		name    string
		sendErr error
	}{
		{
			name: "delegates a successful delivery",
		},
		{
			name:    "propagates a delivery error",
			sendErr: sendErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mailer := new(fakeMailer)
			mailer.On("SendEmail", ctx, to, subject, text, html).Return(tt.sendErr)

			service, err := NewEmailService(
				&internaltests.MockLogger{},
				&emailtypes.EmailPluginConfig{FromAddress: "noreply@example.com"},
				mailer,
				nil,
			)
			require.NoError(t, err)

			adapter := NewMailerServiceAdapter(service)
			gotErr := adapter.SendEmail(ctx, to, subject, text, html)

			if tt.sendErr != nil {
				require.Error(t, gotErr)
				assert.ErrorIs(t, gotErr, tt.sendErr)
			} else {
				require.NoError(t, gotErr)
			}

			mailer.AssertExpectations(t)
		})
	}
}
