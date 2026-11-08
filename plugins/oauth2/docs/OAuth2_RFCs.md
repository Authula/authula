# Core OAuth 2.0 (IETF RFCs)

* **RFC 6749** – *The OAuth 2.0 Authorization Framework* (Core protocol, grant types, roles)
* **RFC 6750** – *The OAuth 2.0 Authorization Framework: Bearer Token Usage* (How bearer tokens are transmitted and accepted)
* **RFC 7636** – *Proof Key for Code Exchange (PKCE)* (Essential for securing public clients and modern authorization code flows)
* **RFC 6819** – *OAuth 2.0 Threat Model and Security Considerations*
* **RFC 9700** – *OAuth 2.0 Security Best Current Practice* (The modern baseline security guide replacing older threat assumptions)

### Token Management & Introspection

* **RFC 7009** – *OAuth 2.0 Token Revocation* (Enables invalidation of access and refresh tokens)
* **RFC 7662** – *OAuth 2.0 Token Introspection* (Allows resource servers to query token state)
* **RFC 9068** – *JSON Web Token (JWT) Profile for OAuth 2.0 Access Tokens* (Standardizes structured, self-contained JWT access tokens)

### Discovery & Client Configuration

* **RFC 8414** – *OAuth 2.0 Authorization Server Metadata* (Standard discovery endpoint config `/.well-known/oauth-authorization-server`)
* **RFC 7591** – *OAuth 2.0 Dynamic Client Registration Protocol*
* **RFC 9207** – *OAuth 2.0 Authorization Server Issuer Identification* (Mitigates mix-up attacks)

### Advanced Request Flow Securing

* **RFC 9126** – *OAuth 2.0 Pushed Authorization Requests (PAR)* (Moves authorization parameters to a direct back-channel call)
* **RFC 9101** – *JWT-Secured Authorization Request (JAR)* (Passing signed/encrypted request objects)
* **RFC 9449** – *OAuth 2.0 Demonstrating Proof of Possession (DPoP)* (Sender-constraining access tokens via cryptographic proofs)
* **RFC 8705** – *OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens*

---

### Where to fetch them for your codebase:

* For all **IETF RFCs**, you can pull the raw text or markdown format directly from the [RFC Editor](https://www.rfc-editor.org/) (e.g., `https://www.rfc-editor.org/rfc/rfc7009.txt`) or index maps via [oauth.net/specs](https://oauth.net/specs/).
* For **OpenID Connect specs**, the canonical markdown and PDF documents are hosted on the [OpenID Foundation Specifications Site](https://openid.net/developers/specs/).
