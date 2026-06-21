# Zee

Zee is the Avagenc Tuya smart-home agent, built on the [Google ADK](https://google.golang.org/adk)
and a Gemini LLM. It ships as a **pure Go module** — there is no standalone
service or HTTP API. You embed Zee into your own runner/session lifecycle, and
the module gives you a configured agent wired to the Tuya device tools.

```
go get go.avagenc.com/zee
```

---

## Usage

Zee exposes two constructors. Pick the one that matches how much of the
lifecycle you want to own.

### `zee.New` — agent + runner

Returns a ready-to-use `*runner.Runner`. Zee wires the session service for you
(`AutoCreateSession` is on).

```go
r, err := zee.New(ctx, zee.Config{
    Name:               "Zee",
    AppName:            "avagenc",
    Description:        "Avagenc Tuya smart-home agent",
    ChannelInstruction: channelInstruction, // per-channel tone/behaviour
    Model:              model,              // a google.golang.org/adk/model.LLM
    Session:            sessionService,     // a google.golang.org/adk/session.Service
    TuyaClient:         tuyaClient,         // a go.naturallyfunny.dev/tuya.Client
})
```

### `zee.NewAgent` — agent only

Returns the bare `agent.Agent`, without a runner or session service. Use this
when the caller manages the session lifecycle itself — for example the ADK web
launcher. `AppName` and `Session` are ignored here.

```go
a, err := zee.NewAgent(zee.Config{
    Name:               "Zee",
    Description:        "Avagenc Tuya smart-home agent",
    ChannelInstruction: channelInstruction,
    Model:              model,
    TuyaClient:         tuyaClient,
})
```

### Config

| Field | Type | Used by | Description |
|---|---|---|---|
| `Name` | `string` | both | Agent name. |
| `AppName` | `string` | `New` | App name passed to the runner. |
| `Description` | `string` | both | Short agent description. |
| `ChannelInstruction` | `string` | both | Per-channel instruction appended to Zee's base system instruction. |
| `Model` | `model.LLM` | both | The LLM backing the agent (e.g. Gemini). |
| `Session` | `session.Service` | `New` | Session service for the runner. |
| `TuyaClient` | `*tuya.Client` | both | Tuya client; its device tools are exposed to the agent. |

The base system instruction is embedded from `internal/system-instruction.txt`;
`ChannelInstruction` is appended to it at build time.

---

## Dev entry points

Two `cmd/` programs let you run Zee locally without any database. Both load a
`.env` (see below) and use an in-memory session plus a `staticAccountStore`
that links **any** session `user_id` to a single fixed Tuya account
(`DEV_TUYA_UID`) — so there's no account-linking or postgres to set up.

> [!WARNING]
> These are local dev tools with **no authentication**, and any `user_id`
> controls the real Tuya devices behind `DEV_TUYA_UID`. Do not expose them
> to the public internet.

### `cmd/web` — ADK web UI / API

Launches the ADK `web`, `webui`, and `api` servers (chat playground + REST API)
on `http://localhost:8080`.

```
go run ./cmd/web
```

### `cmd/cli` — ADK console (`adk run`)

Launches the ADK `console` launcher — the Go equivalent of Python's `adk run`:
an interactive terminal chat loop. Extra ADK console flags pass through.

```
go run ./cmd/cli
go run ./cmd/cli -streaming_mode=none
```

---

## Environment variables (dev entry points)

Copy `.env.example` to `.env` and fill these in. All are required by `cmd/web`
and `cmd/cli`; the module itself reads no environment variables.

| Variable | Description |
|---|---|
| `GEMINI_API_KEY` | Google Gemini API key (model is `gemini-2.5-flash`). |
| `TUYA_ACCESS_ID` | Tuya platform access ID. |
| `TUYA_ACCESS_SECRET` | Tuya platform access secret. |
| `TUYA_BASE_URL` | Tuya API base URL (e.g. `https://openapi.tuyaus.com`). |
| `DEV_TUYA_UID` | Fixed Tuya account UID every dev session links to. |
