# Troubleshooting

This page collects common CCX Desktop issues and resolution steps.

For first-run setup or the full user flow, see [CCX Desktop](./index).

## First things to check

### Disable global proxy and TUN mode

**This is the most common first troubleshooting step.**

Global proxies and TUN mode (e.g. Clash TUN, Surge enhanced mode, v2rayN TUN) hijack all local network traffic, causing requests between CCX and upstream providers to be misrouted or intercepted.

**Symptoms**:

- Requests time out or connections are reset
- Unexpected error pages or HTML content is returned
- Upstream channels show unhealthy, but direct curl to upstream works fine

**Resolution**:

1. Disable the system proxy or switch to rule-based mode (PAC / Rule) to avoid global hijacking.
2. Disable TUN mode, or add CCX-related addresses to the bypass list.
3. Retry the request.

## Startup failures

### Binary not found

**Symptom**: Starting the service reports that the CCX binary is missing.

**Resolution**:

1. If you are running from source, build the backend first:

   ```bash
   cd backend-go && make build
   ```

2. Confirm CCX Desktop can locate `ccx-go`.
3. If the build output path changed, reopen CCX Desktop and try again.

### Port conflict

**Symptom**: Health check times out after startup, with errors such as `connection refused` or a port conflict.

**Resolution**:

1. Check the process using the port:

   ```bash
   # macOS / Linux
   lsof -i :3688

   # Windows
   netstat -ano | findstr :3688
   ```

2. Stop the conflicting process or change `PORT` in **Environment Params**, then restart the service.

### Health check timeout

**Symptom**: The backend process starts, but `http://localhost:3688/health` does not become healthy for a long time.

**Possible causes**:

- Incorrect `.env` values
- Bad channel configuration
- Slow first-run initialization

**Resolution**: Review errors in **Gateway Monitor** or **Log Viewer** and fix configuration step by step.

### Permission denied

**Symptom**: The error includes `permission denied`.

**Resolution**:

```bash
# macOS / Linux
chmod +x backend-go/ccx-go

# Windows
Run Desktop as administrator
```

## Keys and access

### Base URL root vs `/v1`

- **Claude Code**: use `http://localhost:3688`
- **Codex CLI / Codex App**: use `http://localhost:3688/v1`
- **OpenCode**: use `http://localhost:3688/v1`

This is a common source of mistakes. Always match the Base URL to the client type.

### 401 Unauthorized

Check in this order:

1. The client API key matches CCX `PROXY_ACCESS_KEY`
2. The running CCX Desktop instance was started with the expected `PROXY_ACCESS_KEY`
3. No older environment variable is overriding the current configuration
4. You did not enter the upstream provider API key by mistake

### Environment overrides prevent config from taking effect

If the client CLI already has these variables set, they may override agent configuration:

- `ANTHROPIC_API_KEY`
- `ANTHROPIC_BASE_URL`
- `OPENAI_API_KEY`
- `OPENAI_BASE_URL`

Check them in the terminal:

```bash
printenv ANTHROPIC_API_KEY
printenv ANTHROPIC_BASE_URL
printenv OPENAI_API_KEY
printenv OPENAI_BASE_URL
```

If stale values exist, clear them or test again in a clean shell session.

### `localhost` does not work on Windows

If the client runs inside cmd, PowerShell, WSL, or Docker, `localhost` may not reach CCX.

Use the Windows host LAN IPv4 address instead, for example:

```text
http://192.168.1.23:3688
```

Keep the client API key set to `PROXY_ACCESS_KEY`.

## Configuration problems

### Agent configuration applied but not effective

1. Confirm the gateway is running and the port is correct.
2. Confirm the target configuration file path matches the client tool.
3. Restart the client and send a test prompt.
4. If it still fails, remove old environment variables and try again.

### Channel added but requests fail

1. Confirm the upstream API key is correct.
2. Confirm the upstream Base URL is reachable.
3. Check whether the model allowlist or model mapping covers the requested model.
4. Review detailed errors in **Log Viewer** or the web UI.

### Request does not reach the expected endpoint

Different clients use different request paths:

- Claude Code: `/v1/messages`
- Codex CLI / Codex App: `/v1/responses`
- OpenCode: `/v1/chat/completions`

If the log path does not match expectations, the client is not connected to CCX in the intended way. Recheck agent configuration and client provider settings.

### Quick verify each endpoint with curl

When requests fail or you are unsure whether the CCX path is healthy, test each endpoint directly with curl.

Replace `localhost:3688` with your actual address and port, and `your-proxy-key` with your `PROXY_ACCESS_KEY`.

**Messages (Claude Code)**

```bash
curl -sS -i "http://localhost:3688/v1/messages" \
  -H "x-api-key: your-proxy-key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 64,
    "messages": [{"role": "user", "content": "Reply with exactly: pong"}]
  }'
```

**Responses (Codex)**

```bash
curl -sS -i "http://localhost:3688/v1/responses" \
  -H "Authorization: Bearer your-proxy-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "input": "Reply with exactly: pong"
  }'
```

**Chat Completions (OpenCode)**

```bash
curl -sS -i "http://localhost:3688/v1/chat/completions" \
  -H "Authorization: Bearer your-proxy-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "messages": [{"role": "user", "content": "Reply with exactly: pong"}]
  }'
```

**How to interpret the result**:

- `HTTP/1.1 200 OK` with valid response body → CCX path is healthy, the problem is in client configuration.
- `401` → Check that `PROXY_ACCESS_KEY` matches the client API key.
- `400` with model not found → Check channel model mapping.
- `502` or connection timeout → Upstream is unreachable, check channel configuration and network.

## Auto update issues

### macOS shows an unverified developer warning

Open **System Settings → Privacy & Security**, find the blocked app, and click **Open Anyway**.

### Linux AppImage cannot update

Only AppImage supports in-app auto updates. If you installed through deb or rpm packages:

```bash
# deb
sudo apt update && sudo apt upgrade ccx-desktop

# rpm
sudo dnf update ccx-desktop
```

### Update download fails

The GitHub installer build checks and downloads updates from GitHub Releases.

Troubleshooting steps:

1. Check network connectivity.
2. If you use a proxy, confirm GitHub Releases is reachable.
3. If needed, download the update manually from the [Releases page](https://github.com/BenedictKing/ccx/releases).

The Microsoft Store build is updated by the Store. If Store updates fail, retry from the library page or reinstall.

## Other issues

### Window position or size is not restored

CCX Desktop saves window state to the data directory.

If the saved state becomes invalid:

1. Close Desktop.
2. Delete `window-state.json` in the data directory.
3. Open the app again.

### Launch at login does not work

- macOS: check **System Settings → General → Login Items**
- Windows: check **Task Manager → Startup**
- Linux: check your desktop environment autostart settings

### Web UI is not accessible

1. Confirm the gateway is running.
2. Confirm `ENABLE_WEB_UI` is `true`.
3. Try opening `http://localhost:3688` directly in a browser.
