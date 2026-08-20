# Authentication Sessions

## Token boundary

- Access token: signed JWT, lifetime 15 minutes, held by the web client and sent
  as `Authorization: Bearer`.
- Refresh token: 256-bit random opaque value, lifetime 30 days, sent only in the
  `usloop_refresh` `HttpOnly` cookie.
- The database stores SHA-256 hashes of refresh, email-verification, and
  password-reset tokens. Raw token values must never be logged or persisted.
- Refresh is rotating and single-use. Replaying a rotated or revoked token is
  rejected.
- Password reset revokes every active refresh session belonging to the user.

Set `COOKIE_SECURE=true` and use HTTPS in staging/production. Configure CORS with
an explicit frontend origin; wildcard origins are incompatible with credentialed
cookies.

## Email delivery

`AUTH_TOKEN_DELIVERY=development_response` may expose a verification/reset token
inside the JSON response for local development. Never enable it outside local
development.

Production must connect an email delivery adapter before
`REQUIRE_VERIFIED_EMAIL=true` is enabled. Request endpoints deliberately return
the same response for known and unknown email addresses to reduce account
enumeration.

## Endpoints

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/sessions`
- `DELETE /api/v1/auth/sessions/{session_id}`
- `POST /api/v1/auth/email-verification/request`
- `POST /api/v1/auth/email-verification/verify`
- `POST /api/v1/auth/password-reset/request`
- `POST /api/v1/auth/password-reset/confirm`
