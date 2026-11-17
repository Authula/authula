package providers

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wneessen/go-mail"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/email/constants"
	emailtypes "github.com/Authula/authula/plugins/email/types"
)

func TestNewSMTPProvider(t *testing.T) {
	// t.Setenv forbids t.Parallel in this test.
	tests := []struct {
		name          string
		envHost       string
		envPort       string
		envUser       string
		envPass       string
		config        *emailtypes.EmailPluginConfig
		wantHost      string
		wantPort      int
		wantUser      string
		wantPass      string
		wantTLSPolicy mail.TLSPolicy
		wantErr       string
	}{
		{
			name:          "reads host and port from environment",
			envHost:       "smtp.env.test",
			envPort:       "2525",
			config:        new(emailtypes.EmailPluginConfig),
			wantHost:      "smtp.env.test",
			wantPort:      2525,
			wantTLSPolicy: mail.TLSOpportunistic,
		},
		{
			name:          "trims whitespace from environment values",
			envHost:       "  smtp.env.test  ",
			envPort:       " 2525 ",
			config:        new(emailtypes.EmailPluginConfig),
			wantHost:      "smtp.env.test",
			wantPort:      2525,
			wantTLSPolicy: mail.TLSOpportunistic,
		},
		{
			name:          "falls back to config host and port",
			config:        &emailtypes.EmailPluginConfig{SMTP: &emailtypes.SMTPConfig{Host: "smtp.config.test", Port: 587}},
			wantHost:      "smtp.config.test",
			wantPort:      587,
			wantTLSPolicy: mail.TLSOpportunistic,
		},
		{
			name:          "environment overrides config",
			envHost:       "smtp.env.test",
			envPort:       "2525",
			config:        &emailtypes.EmailPluginConfig{SMTP: &emailtypes.SMTPConfig{Host: "smtp.config.test", Port: 587}},
			wantHost:      "smtp.env.test",
			wantPort:      2525,
			wantTLSPolicy: mail.TLSOpportunistic,
		},
		{
			name:    "errors when host is missing from environment and config",
			envPort: "2525",
			config:  new(emailtypes.EmailPluginConfig),
			wantErr: constants.EnvSMTPHost,
		},
		{
			name:    "errors when config host is blank",
			envPort: "2525",
			config:  &emailtypes.EmailPluginConfig{SMTP: &emailtypes.SMTPConfig{Host: "   ", Port: 587}},
			wantErr: constants.EnvSMTPHost,
		},
		{
			name:    "errors when port is missing from environment and config",
			envHost: "smtp.env.test",
			config:  &emailtypes.EmailPluginConfig{SMTP: &emailtypes.SMTPConfig{Host: "smtp.config.test"}},
			wantErr: constants.EnvSMTPPort,
		},
		{
			name:    "errors when config port is zero",
			envHost: "smtp.env.test",
			config:  &emailtypes.EmailPluginConfig{SMTP: &emailtypes.SMTPConfig{Host: "smtp.config.test", Port: 0}},
			wantErr: constants.EnvSMTPPort,
		},
		{
			name:    "errors when port is not an integer",
			envHost: "smtp.env.test",
			envPort: "not-a-port",
			config:  new(emailtypes.EmailPluginConfig),
			wantErr: "must be a valid integer",
		},
		{
			name:          "maps off TLS mode to NoTLS",
			envHost:       "smtp.env.test",
			envPort:       "25",
			config:        &emailtypes.EmailPluginConfig{TLSMode: emailtypes.SMTPTLSModeOff},
			wantHost:      "smtp.env.test",
			wantPort:      25,
			wantTLSPolicy: mail.NoTLS,
		},
		{
			name:          "maps starttls TLS mode to opportunistic TLS",
			envHost:       "smtp.env.test",
			envPort:       "587",
			config:        &emailtypes.EmailPluginConfig{TLSMode: emailtypes.SMTPTLSModeStartTLS},
			wantHost:      "smtp.env.test",
			wantPort:      587,
			wantTLSPolicy: mail.TLSOpportunistic,
		},
		{
			name:          "maps tls TLS mode to mandatory TLS",
			envHost:       "smtp.env.test",
			envPort:       "465",
			config:        &emailtypes.EmailPluginConfig{TLSMode: emailtypes.SMTPTLSModeTLS},
			wantHost:      "smtp.env.test",
			wantPort:      465,
			wantTLSPolicy: mail.TLSMandatory,
		},
		{
			name:          "TLS mode is case insensitive",
			envHost:       "smtp.env.test",
			envPort:       "465",
			config:        &emailtypes.EmailPluginConfig{TLSMode: "TLS"},
			wantHost:      "smtp.env.test",
			wantPort:      465,
			wantTLSPolicy: mail.TLSMandatory,
		},
		{
			name:    "rejects unknown TLS mode",
			envHost: "smtp.env.test",
			envPort: "587",
			config:  &emailtypes.EmailPluginConfig{TLSMode: "ssl"},
			wantErr: "invalid SMTP_TLS_MODE",
		},
		{
			name:          "reads credentials from environment",
			envHost:       "smtp.env.test",
			envPort:       "587",
			envUser:       "env-user",
			envPass:       "env-pass",
			config:        &emailtypes.EmailPluginConfig{TLSMode: emailtypes.SMTPTLSModeStartTLS},
			wantHost:      "smtp.env.test",
			wantPort:      587,
			wantUser:      "env-user",
			wantPass:      "env-pass",
			wantTLSPolicy: mail.TLSOpportunistic,
		},
		{
			name:    "falls back to config credentials",
			envHost: "smtp.env.test",
			envPort: "465",
			config: &emailtypes.EmailPluginConfig{
				TLSMode: emailtypes.SMTPTLSModeTLS,
				SMTP:    &emailtypes.SMTPConfig{Username: "config-user", Password: "config-pass"},
			},
			wantHost:      "smtp.env.test",
			wantPort:      465,
			wantUser:      "config-user",
			wantPass:      "config-pass",
			wantTLSPolicy: mail.TLSMandatory,
		},
		{
			name:    "rejects credentials when TLS is disabled",
			envHost: "smtp.env.test",
			envPort: "25",
			envUser: "env-user",
			envPass: "env-pass",
			config:  &emailtypes.EmailPluginConfig{TLSMode: emailtypes.SMTPTLSModeOff},
			wantErr: "SMTP credentials supplied but TLS is disabled",
		},
		{
			name:    "rejects username without password when TLS is disabled",
			envHost: "smtp.env.test",
			envPort: "25",
			envUser: "env-user",
			config:  &emailtypes.EmailPluginConfig{TLSMode: emailtypes.SMTPTLSModeOff},
			wantErr: "SMTP credentials supplied but TLS is disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(constants.EnvSMTPHost, tt.envHost)
			t.Setenv(constants.EnvSMTPPort, tt.envPort)
			t.Setenv(constants.EnvSMTPUser, tt.envUser)
			t.Setenv(constants.EnvSMTPPass, tt.envPass)

			provider, err := NewSMTPProvider(tt.config, new(internaltests.RecordingLogger))

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Nil(t, provider)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)
			assert.Equal(t, tt.wantHost, provider.host)
			assert.Equal(t, tt.wantPort, provider.port)
			assert.Equal(t, tt.wantUser, provider.user)
			assert.Equal(t, tt.wantPass, provider.pass)
			assert.Equal(t, tt.wantTLSPolicy, provider.tlsPolicy)
			assert.Equal(t, tt.config, provider.config)
		})
	}
}

// fakeSMTPServer is a minimal in-process SMTP sink used to exercise the send path.
type fakeSMTPServer struct {
	listener net.Listener

	mu       sync.Mutex
	messages []string
	conns    map[net.Conn]struct{}
	wg       sync.WaitGroup
}

func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &fakeSMTPServer{listener: listener, conns: make(map[net.Conn]struct{})}

	server.wg.Add(1)
	go func() {
		defer server.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			server.mu.Lock()
			server.conns[conn] = struct{}{}
			server.mu.Unlock()

			server.wg.Add(1)
			go func() {
				defer server.wg.Done()
				defer func() {
					server.mu.Lock()
					delete(server.conns, conn)
					server.mu.Unlock()
				}()
				server.handle(conn)
			}()
		}
	}()

	t.Cleanup(func() {
		_ = listener.Close()

		// Clients that abort before sending QUIT leave their connection open.
		server.mu.Lock()
		for conn := range server.conns {
			_ = conn.Close()
		}
		server.mu.Unlock()

		server.wg.Wait()
	})

	return server
}

func (s *fakeSMTPServer) port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *fakeSMTPServer) received() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages := make([]string, len(s.messages))
	copy(messages, s.messages)
	return messages
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	write := func(response string) bool {
		_, err := conn.Write([]byte(response))
		return err == nil
	}

	if !write("220 fake ESMTP ready\r\n") {
		return
	}

	reader := bufio.NewReader(conn)
	var data strings.Builder
	inData := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		if inData {
			if strings.TrimRight(line, "\r\n") == "." {
				inData = false
				s.record(data.String())
				data.Reset()
				if !write("250 2.0.0 queued\r\n") {
					return
				}
				continue
			}
			data.WriteString(line)
			continue
		}

		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			if !write("250 fake\r\n") {
				return
			}
		case strings.HasPrefix(command, "DATA"):
			inData = true
			if !write("354 End data with <CR><LF>.<CR><LF>\r\n") {
				return
			}
		case strings.HasPrefix(command, "QUIT"):
			_ = write("221 2.0.0 Bye\r\n")
			return
		default:
			if !write("250 2.1.0 OK\r\n") {
				return
			}
		}
	}
}

func (s *fakeSMTPServer) record(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, message)
}

func newSMTPTestProvider(port int, config *emailtypes.EmailPluginConfig) *SMTPProvider {
	if config == nil {
		config = new(emailtypes.EmailPluginConfig)
	}

	return &SMTPProvider{
		config:    config,
		logger:    new(internaltests.RecordingLogger),
		host:      "127.0.0.1",
		port:      port,
		tlsPolicy: mail.NoTLS,
	}
}

func TestSMTPProvider_SendEmail_SendsMessage(t *testing.T) {
	t.Parallel()

	server := newFakeSMTPServer(t)
	provider := newSMTPTestProvider(server.port(), &emailtypes.EmailPluginConfig{FromAddress: "noreply@example.com"})

	err := provider.SendEmail(context.Background(), "user@example.com", "Hello there", "plain body", "<p>hi</p>")

	require.NoError(t, err)

	messages := server.received()
	require.Len(t, messages, 1)
	assert.Contains(t, messages[0], "From: <noreply@example.com>")
	assert.Contains(t, messages[0], "To: <user@example.com>")
	assert.Contains(t, messages[0], "Subject: Hello there")
	assert.Contains(t, messages[0], "plain body")
	assert.Contains(t, messages[0], "<p>hi</p>")
}

func TestSMTPProvider_SendEmail_SendsEmptyBodyWithoutValidation(t *testing.T) {
	t.Parallel()

	server := newFakeSMTPServer(t)
	provider := newSMTPTestProvider(server.port(), &emailtypes.EmailPluginConfig{FromAddress: "noreply@example.com"})

	err := provider.SendEmail(context.Background(), "user@example.com", "No body", "", "")

	require.NoError(t, err)
	assert.Len(t, server.received(), 1)
}

func TestSMTPProvider_SendEmail_RejectsInvalidAddresses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		from       string
		to         string
		wantErrMsg string
	}{
		{
			name:       "invalid from address",
			from:       "",
			to:         "user@example.com",
			wantErrMsg: "invalid from address",
		},
		{
			name:       "invalid recipient address",
			from:       "noreply@example.com",
			to:         "not-an-address",
			wantErrMsg: "invalid recipient address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newFakeSMTPServer(t)
			provider := newSMTPTestProvider(server.port(), &emailtypes.EmailPluginConfig{FromAddress: tt.from})

			err := provider.SendEmail(context.Background(), tt.to, "Subject", "body", "")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
			assert.Empty(t, server.received(), "no message should be sent for invalid addresses")
		})
	}
}

func TestSMTPProvider_SendEmail_ReturnsDialError(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	provider := newSMTPTestProvider(port, &emailtypes.EmailPluginConfig{FromAddress: "noreply@example.com"})

	dialErr := provider.SendEmail(context.Background(), "user@example.com", "Subject", "body", "")

	require.Error(t, dialErr)
	assert.Contains(t, dialErr.Error(), "failed to send email")
	assert.Contains(t, dialErr.Error(), strconv.Itoa(port))
}

func TestSMTPProvider_SendEmail_RejectsInvalidPort(t *testing.T) {
	t.Parallel()

	provider := newSMTPTestProvider(0, &emailtypes.EmailPluginConfig{FromAddress: "noreply@example.com"})

	err := provider.SendEmail(context.Background(), "user@example.com", "Subject", "body", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create SMTP client")
}

func TestSMTPProvider_SendEmail_ConfiguresCredentials(t *testing.T) {
	t.Parallel()

	server := newFakeSMTPServer(t)
	provider := newSMTPTestProvider(server.port(), &emailtypes.EmailPluginConfig{FromAddress: "noreply@example.com"})
	provider.user = "smtp-user"
	provider.pass = "smtp-pass"
	provider.tlsPolicy = mail.NoTLS

	// The fake server does not offer STARTTLS, so authenticating on the
	// resulting plaintext connection must be refused.
	err := provider.SendEmail(context.Background(), "user@example.com", "Subject", "body", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}
