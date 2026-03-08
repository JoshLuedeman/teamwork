# Teamwork Auto-Installer — Cloudflare Worker

A Cloudflare Worker that acts as a GitHub App webhook listener. When a new
repository is created in the installed organization, this worker automatically
initializes it with the [Teamwork](../../README.md) framework files.

## How It Works

1. Receives a `repository.created` webhook event from GitHub.
2. Verifies the webhook signature (HMAC-SHA256).
3. Authenticates as the GitHub App using a JWT → installation access token.
4. Checks for a `.teamwork-skip` file (opt-out mechanism).
5. Fetches framework files from the source repository.
6. Copies framework files and creates starter templates (`MEMORY.md`,
   `CHANGELOG.md`, `README.md`) in the new repository.
7. Creates a commit on the default branch.

Forks are automatically skipped.

## Prerequisites

- **Cloudflare account** with Workers enabled
- **Wrangler CLI** (`npm install -g wrangler`) — v4+
- **GitHub App** configured with:
  - Webhook URL pointing to your worker (e.g.,
    `https://teamwork-installer.<subdomain>.workers.dev/webhook`)
  - Webhook secret
  - `repository.created` event subscription
  - Permissions: `contents: write`

## Configuration

### Environment Variables

Public variables are set in [`wrangler.toml`](./wrangler.toml):

| Variable            | Description                                      |
| ------------------- | ------------------------------------------------ |
| `SOURCE_REPO_OWNER` | Owner of the repository containing framework files |
| `SOURCE_REPO_NAME`  | Name of the source repository                    |
| `SOURCE_REF`        | Git ref (branch/tag) to fetch framework files from |

### Secrets

Secrets **must not** be committed to source control. Set them using the
Wrangler CLI:

```bash
# GitHub App ID (numeric)
wrangler secret put GITHUB_APP_ID

# GitHub App private key (PEM-encoded RSA key)
wrangler secret put GITHUB_APP_PRIVATE_KEY

# Webhook secret (shared secret configured in the GitHub App)
wrangler secret put GITHUB_WEBHOOK_SECRET
```

Each command prompts for the secret value interactively.

## Development

### Install Dependencies

```bash
cd workers/github-app
npm install
```

### Run Locally

```bash
npm run dev
```

This starts a local development server with `wrangler dev`. You can send test
webhook payloads to `http://localhost:8787/webhook`.

### Type Check

```bash
npm run typecheck
```

### Run Tests

```bash
npm test
```

Tests use [Vitest](https://vitest.dev/) with
[`@cloudflare/vitest-pool-workers`](https://developers.cloudflare.com/workers/testing/vitest-integration/)
for Workers-compatible test execution.

To run tests in watch mode:

```bash
npm run test:watch
```

## Deployment

### Deploy to Cloudflare

```bash
npm run deploy
```

This runs `wrangler deploy`, which publishes the worker to your Cloudflare
account. Make sure you have authenticated with `wrangler login` first.

### Verify Deployment

After deploying, confirm the worker is running:

```bash
curl -s -o /dev/null -w "%{http_code}" \
  https://teamwork-installer.<subdomain>.workers.dev/webhook
```

A `405` response (Method Not Allowed) is expected — the worker only accepts
`POST` requests.

## Monitoring

### Tail Logs

Stream real-time logs from the deployed worker:

```bash
wrangler tail teamwork-installer
```

Add `--format json` for structured output or filter by status:

```bash
wrangler tail teamwork-installer --status error
```

### Cloudflare Dashboard

View metrics, logs, and error rates in the
[Cloudflare Workers dashboard](https://dash.cloudflare.com/).

## Troubleshooting

| Symptom | Likely Cause | Fix |
| --- | --- | --- |
| `401 Unauthorized` on webhook delivery | Invalid or missing webhook secret | Re-set the secret: `wrangler secret put GITHUB_WEBHOOK_SECRET` |
| `500 Internal Server Error` | GitHub API authentication failure | Verify `GITHUB_APP_ID` and `GITHUB_APP_PRIVATE_KEY` are correct |
| Worker not receiving events | Webhook URL misconfigured in GitHub App | Check the app settings — URL must end in `/webhook` |
| New repos not getting framework files | App not installed on the organization | Install the GitHub App on the target org |
| `405 Method Not Allowed` | Non-POST request sent to the worker | This is expected for GET requests; webhooks use POST |
| Repo skipped unexpectedly | Repository is a fork or has `.teamwork-skip` | Forks are always skipped; remove `.teamwork-skip` to opt in |
