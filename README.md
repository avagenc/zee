# zee-agent

Tuya Smart Home agent service for the Avagenc platform. Exposes `/chat` and `/chat/ava` endpoints backed by a Gemini LLM runner with Zep memory.

---

## Required request headers

All `/chat` and `/chat/ava` requests must include the following headers. Requests missing a required header or supplying an invalid value are rejected before reaching the handler.

| Header | Description | Validation | Error on failure |
|---|---|---|---|
| `user-id` | Caller's user identifier | Must be non-empty | `401 Unauthorized` — `"user ID cannot be empty"` |
| `session-id` | Conversation session identifier | Must be non-empty | `401 Unauthorized` — `"session ID cannot be empty"` |
| `time-zone` | IANA timezone string | Validated via `time.LoadLocation` | `400 Bad Request` — `"invalid IANA timezone …"` |

### IANA timezone requirement

The `time-zone` header must be a valid IANA timezone name (e.g. `Asia/Jakarta`, `America/New_York`, `UTC`). Informal or abbreviated strings (`Jakarta`, `WIB`, `EST`) are rejected with a 400. Coordinate with client teams before deploying if they currently send non-IANA values.

### Example

```
POST /chat
user-id: usr_abc123
session-id: ses_xyz789
time-zone: Asia/Jakarta
Content-Type: application/json

{"message": "Turn on the living room lights"}
```

---

## Outbound HTTP transport

When adding new outbound calls to other Avagenc microservices, wire the propagating client so identity headers are forwarded automatically:

```go
import (
    apihttp  "go.naturallyfunny.dev/api/http"
    apises   "go.naturallyfunny.dev/api/session"
    apitime  "go.naturallyfunny.dev/api/time"
    apiuser  "go.naturallyfunny.dev/api/user"
    nethttp  "net/http"
)

propagatingClient := &nethttp.Client{
    Transport: &apihttp.Transport{
        Propagators: []apihttp.Propagator{
            apihttp.WithHeader(apiuser.ContextKey, "user-id"),
            apihttp.WithHeader(apises.ContextKey,  "session-id"),
            apihttp.WithHeader(apitime.ContextKey, "time-zone"),
        },
    },
}
```

Inject `propagatingClient` into any service client that calls other Avagenc services. The Tuya client (`internal/tuya`) manages its own HMAC-signed auth and must **not** use this transport.

---

## Environment variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `TUYA_ACCESS_ID` | Tuya platform access ID |
| `TUYA_ACCESS_SECRET` | Tuya platform access secret |
| `TUYA_BASE_URL` | Tuya API base URL |
| `ZEP_API_KEY` | Zep memory service API key |
| `GEMINI_API_KEY` | Google Gemini API key |
| `APP_NAME` | Service name reported on `/` |
| `APP_VERSION` | Service version reported on `/` |
| `APP_ENV` | Deployment environment (`production`, `staging`, etc.) |
| `SERVER_PORT` | Port to listen on (default `8080`) |
