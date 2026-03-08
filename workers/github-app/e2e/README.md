# E2E Tests

## Prerequisites

Before running the E2E test:

1. **GitHub App** is registered, private key generated, and installed on your account
2. **Cloudflare Worker** is deployed (`cd workers/github-app && wrangler deploy`)
3. **Worker secrets** are configured (`GITHUB_APP_ID`, `GITHUB_APP_PRIVATE_KEY`, `GITHUB_WEBHOOK_SECRET`)
4. **GitHub CLI** (`gh`) is authenticated with your account

## Running the Test

```bash
cd workers/github-app/e2e
./test-auto-install.sh
```

Optionally provide a custom repo name:

```bash
./test-auto-install.sh my-test-repo
```

## What the Test Does

1. Creates a new private repository on your GitHub account
2. Waits up to 60 seconds for the GitHub App to push framework files
3. Verifies expected framework files exist (`.github/agents/`, `docs/`, `Makefile`, etc.)
4. Checks that the commit author is the GitHub App
5. Deletes the test repository (cleanup)

## Troubleshooting

If the test fails:

- **Check webhook deliveries:** Go to your GitHub App settings → Advanced → Recent Deliveries
- **Check Worker logs:** Run `wrangler tail` in another terminal
- **Common issues:**
  - Webhook URL mismatch (update in App settings)
  - Expired or incorrect secrets
  - Worker not deployed
