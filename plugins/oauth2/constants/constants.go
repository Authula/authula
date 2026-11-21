package constants

const (
	// CookieFlowBinding binds an OAuth2 state row to the browser that started
	// the flow. The state lookup hash includes this value.
	CookieFlowBinding = "oauth2_flow_binding"

	// TokenRequestMaxBodyBytes limits the token endpoint request body.
	TokenRequestMaxBodyBytes int64 = 64 << 10
)
