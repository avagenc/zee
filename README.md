# Zee

Zee is the Avagenc Tuya smart-home agent, built on the [Google ADK](https://pkg.go.dev/google.golang.org/adk).
It ships as a **pure Go module** — there is no standalone service or HTTP API.
You embed Zee into your own runner/session lifecycle and supply an LLM plus a
Tuya client; the module gives you a configured agent wired to the Tuya device
tools.

```
go get go.avagenc.com/zee
```

---

## Usage

`zee.New` returns a configured `agent.Agent`. Running it — the runner and
session lifecycle — is the consumer's responsibility. Zee's identity and
system instruction are owned by the module; you supply only its dependencies.

```go
a, err := zee.New(zee.Config{
    Model:      model,      // a google.golang.org/adk/model.LLM
    TuyaClient: tuyaClient, // a go.naturallyfunny.dev/tuya.Client
})
```

### Config

| Field | Type | Description |
|---|---|---|
| `Model` | `model.LLM` | The LLM backing the agent (e.g. Gemini). |
| `TuyaClient` | `*tuya.Client` | Tuya client; its device tools are exposed to the agent. |

The system instruction is embedded from `internal/system-instruction.txt`.

### Per-channel / per-run instruction

Zee ships one fixed identity and system instruction. Anything channel- or
run-specific (tone, extra context) is the **consumer's** concern, added at the
runner you own — the module exposes no hook for it. The idiomatic ADK way is a
`BeforeModelCallback` that appends to the request's system instruction per run,
registered as a plugin on your runner:

```go
p, _ := plugin.New(plugin.Config{
    Name: "channel-instruction",
    BeforeModelCallback: func(ctx agent.CallbackContext, req *model.LLMRequest) (*model.LLMResponse, error) {
        // append channel-specific instruction to req for this run
        return nil, nil
    },
})

r, _ := runner.New(runner.Config{
    Agent:          a, // the zee.New agent
    SessionService: sessionService,
    PluginConfig:   runner.PluginConfig{Plugins: []*plugin.Plugin{p}},
})
```

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
