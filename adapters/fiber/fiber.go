// Package fiber provides a Fiber middleware adapter for go-better-auth.
//
// It bridges Fiber's fasthttp-based context to go-better-auth's standard
// net/http.Handler by manually constructing http.Request objects. This avoids
// fasthttpadaptor which can lose the request body — a critical issue for
// authentication payloads.
//
// Usage:
//
//	auth := gobetterauth.New(&gobetterauth.AuthConfig{...})
//
//	app := fiber.New()
//	app.Use("/api/auth", fiberadapter.New(fiberadapter.Config{
//	    Handler: auth.Handler(),
//	}))
package fiber

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v3"
)

// Config defines the config for the go-better-auth Fiber adapter middleware.
type Config struct {
	// Handler is the go-better-auth http.Handler. Required.
	// Obtain it via auth.Handler().
	Handler http.Handler `json:"-" toml:"-"`

	// Next defines a function to skip this middleware when returning true.
	// Optional. Default: nil (never skip).
	Next func(c fiber.Ctx) bool `json:"-" toml:"-"`

	// ErrorHandler is called when the adapter encounters an internal error
	// (e.g., a malformed request URL). Optional. Default: returns 500 JSON.
	ErrorHandler fiber.ErrorHandler `json:"-" toml:"-"`
}

func configDefault(config Config) Config {
	if config.ErrorHandler == nil {
		config.ErrorHandler = func(c fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal adapter error",
			})
		}
	}
	return config
}

// New creates a Fiber middleware that proxies requests to a go-better-auth
// http.Handler. It manually builds net/http requests from Fiber's fasthttp
// context to ensure request bodies are preserved correctly.
func New(config Config) fiber.Handler {
	if config.Handler == nil {
		panic("fiber adapter: Config.Handler is required")
	}
	cfg := configDefault(config)

	return func(c fiber.Ctx) error {
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		path := c.Path()
		parsedURL, err := url.ParseRequestURI(path)
		if err != nil {
			return cfg.ErrorHandler(c, err)
		}
		parsedURL.RawQuery = string(c.Request().URI().QueryString())

		body := c.Body()
		req := &http.Request{
			Method:        c.Method(),
			URL:           parsedURL,
			RequestURI:    parsedURL.RequestURI(),
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          io.NopCloser(bytes.NewReader(body)),
			ContentLength: int64(len(body)),
			Host:          string(c.Request().Header.Peek("Host")),
			RemoteAddr:    c.IP() + ":0",
		}

		for key, value := range c.Request().Header.All() {
			req.Header.Add(string(key), string(value))
		}

		if addr := c.RequestCtx().RemoteAddr(); addr != nil {
			req.RemoteAddr = addr.String()
		}

		rw := newFiberResponseWriter(c)
		cfg.Handler.ServeHTTP(rw, req)

		return nil
	}
}
