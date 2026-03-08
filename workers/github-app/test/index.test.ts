import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type { Env } from "../src/types";

// We test the worker's fetch handler by importing it and calling it directly
// with constructed Request objects and a mock Env.

// For the integration test, we need to mock external dependencies. Since we
// can't easily mock module-level imports in a Workers environment, we test
// the request/response flow at the handler level by mocking global fetch.

/** Build a valid HMAC-SHA256 signature for a body + secret. */
async function signPayload(body: string, secret: string): Promise<string> {
  const encoder = new TextEncoder();
  const key = await crypto.subtle.importKey(
    "raw",
    encoder.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const mac = await crypto.subtle.sign("HMAC", key, encoder.encode(body));
  const hex = Array.from(new Uint8Array(mac))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
  return `sha256=${hex}`;
}

const TEST_SECRET = "test-secret-for-webhooks";

// Test RSA private key (2048-bit) — generated for testing only, not a real secret.
const TEST_PRIVATE_KEY = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAmZPIls6vQsmqrdrNfzWWEeBc4jv9Gmo5yGRRN3X0/UWWv+ES
ggnQLWSHiZd08mZ6YFhsGCdf+NFXmp8wvK3sJnEhUZXc7ZshHI6EI58sPr6aJbr7
mu1kx2Mxma76QmRPUuBlZ8hd5eVwZoB1x4Z1N2Y6uN6dqDPkwcUsuNfM/HygjwO+
OOuc1WRhdcVofdKi6YJsAdYDU2eEIrZu+OIh7eyj1pPp1aOkcjBg0zAMJs7iv2kz
xZZwxq7Q6sIQnlGu0W4RlWEpNBj5OAK1KOh/ycnOlqNpOr8Sc8+uAKEDVVbcY9P4
FusEkXcBOKuDHdWR306fXKwLqrAhkwmjkffOCwIDAQABAoIBAAr7JCaUVLfDz65q
rLLh0/8nObz7aReQbN1FPwFmL8RES4kgwMAHj5kPTRmreLM0XJ+y8tevSQ9zeH4X
z9ZN4UrGYAmDW66pnu55yjz5zqIV4tO70O28963CC/PfLQm+PmXAob+P9hbQFv9d
RA6mMI9rgdtiH4e9Xif0v0PgOkn7SurrF7YHIn0t8D7MW33IDn9086VhyAen6QXa
7ekWKYAWZn32IJx6pu4CyzkD5IETOn0428zaByMtOep3+x3uIoSf6VcYVssy7zZN
hfUnx+U72C+f8tvX6TMQchqOINKmZawK7IjQfBx2TBaZkq7OiOXDMc7RVw+B1VoB
OkT9YzECgYEAyrl08cQS5wDn2ddAlxfvhzTZ6/p34cCl+SThOEqoeLEax+rjOtYc
5y/faeUH1fsCagDnHOPW8vu6eMKKc5QqV79eiaxHaDV4zbF5Z0p3zDMIJ81fUqmU
rJ2ZHuq1QDRL1LrLs2uErGhla4f0BNZDK8IggmUjbSdmp2+sheJqaMcCgYEAwe/j
qRqM2r5IcYIT71P+MJnrTtfCyxGcr0d6Uhq304IY2PNp9cQ8zSzOjE0nXR/nX7La
USJUvkz0uMLT/U3T1vl0DbN4fWMNTMhhhMpZsvy2NfKrNGgmhER6PMHG3aMbZs2a
NdMBjfmEf3YP0NxR12btPDcPrFfAhn5Y1vtKFJ0CgYEArSo/v6iZ8OLwKT9aN/ZF
L7wwjgc0Ug1KeQhMrdXwFLBLzQtSMFbm94AIGh9+UwUHqd69jAr++C2YukCLHXEp
viyEp5sWn+hVGXcI2fddX3sT81PVofmjOtOgES2xx3ckc0FgcRFdkhvWzkSiZ2NS
m1VGibu0yC+I22tj9jVSac8CgYEAuI/0Z70lqRKHXMZ+9DdJ47THdAvvjFPhegmb
BkH5CWd5ABZ+k25CsrvegTT3ri8rgS5zh90VKtmP17lKB3kmjiJt6JAQrbszMAxO
ihIMVUMcoLClb8ViSmPktKdw+wI7lJU8GdcKVrPL/YU8vfa+SDDiunhoCQql5Rie
sVEKCh0CgYBQwlG7YrNKXZm6kGoBhSO8tbh438f6xhxM81ZE/+6Vdn2ccyATBVsy
uHzkSBzrmn5u1d+H2FtCLcUdZlD7cuJaW4np8g6ZoAmPq7eqYT4/+Nc89FMUo3dp
bVAAfmC90YVfg4UxqYOVij6rje8bySFPFs1IvwIEvCnpKaT2BWzSAg==
-----END RSA PRIVATE KEY-----`;

const TEST_ENV: Env = {
  GITHUB_APP_ID: "123456",
  GITHUB_APP_PRIVATE_KEY: TEST_PRIVATE_KEY,
  GITHUB_WEBHOOK_SECRET: TEST_SECRET,
  SOURCE_REPO_OWNER: "JoshLuedeman",
  SOURCE_REPO_NAME: "teamwork",
  SOURCE_REF: "main",
};

function makePayload(overrides: Record<string, unknown> = {}): string {
  return JSON.stringify({
    action: "created",
    repository: {
      name: "new-repo",
      owner: { login: "test-org" },
      full_name: "test-org/new-repo",
      fork: false,
      default_branch: "main",
    },
    installation: { id: 42 },
    ...overrides,
  });
}

async function makeRequest(options: {
  method?: string;
  body?: string;
  event?: string;
  signature?: string;
}): Promise<Request> {
  const method = options.method ?? "POST";
  const body = options.body ?? makePayload();
  const headers = new Headers();

  if (options.event !== undefined) {
    headers.set("X-GitHub-Event", options.event);
  } else {
    headers.set("X-GitHub-Event", "repository");
  }

  if (options.signature !== undefined) {
    if (options.signature !== "") {
      headers.set("X-Hub-Signature-256", options.signature);
    }
  } else {
    const sig = await signPayload(body, TEST_SECRET);
    headers.set("X-Hub-Signature-256", sig);
  }

  return new Request("https://worker.example.com/webhook", {
    method,
    headers,
    body: method !== "GET" ? body : undefined,
  });
}

describe("Worker fetch handler", () => {
  let originalFetch: typeof globalThis.fetch;
  let worker: { fetch: (request: Request, env: Env) => Promise<Response> };

  beforeEach(async () => {
    originalFetch = globalThis.fetch;
    // Dynamically import to get a fresh module
    const mod = await import("../src/index");
    worker = mod.default;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("returns 405 for non-POST requests", async () => {
    const request = new Request("https://worker.example.com/webhook", {
      method: "GET",
    });
    const response = await worker.fetch(request, TEST_ENV);
    expect(response.status).toBe(405);
    expect(await response.text()).toBe("Method not allowed");
  });

  it("returns 401 when signature header is missing", async () => {
    const body = makePayload();
    const request = await makeRequest({ body, signature: "" });
    const response = await worker.fetch(request, TEST_ENV);
    expect(response.status).toBe(401);
    expect(await response.text()).toBe("Invalid signature");
  });

  it("returns 401 for an invalid signature", async () => {
    const body = makePayload();
    const request = await makeRequest({
      body,
      signature: "sha256=0000000000000000000000000000000000000000000000000000000000000000",
    });
    const response = await worker.fetch(request, TEST_ENV);
    expect(response.status).toBe(401);
    expect(await response.text()).toBe("Invalid signature");
  });

  it("returns 200 with 'Event ignored' for non-repository events", async () => {
    const body = makePayload();
    const request = await makeRequest({ body, event: "push" });
    const response = await worker.fetch(request, TEST_ENV);
    expect(response.status).toBe(200);
    expect(await response.text()).toBe("Event ignored");
  });

  it("returns 200 with 'Action ignored' for non-created actions", async () => {
    const body = makePayload({ action: "deleted" });
    const request = await makeRequest({ body });
    const response = await worker.fetch(request, TEST_ENV);
    expect(response.status).toBe(200);
    expect(await response.text()).toBe("Action ignored");
  });

  it("returns 200 with 'Skipped (fork)' for forked repositories", async () => {
    const body = makePayload({
      repository: {
        name: "forked-repo",
        owner: { login: "test-org" },
        full_name: "test-org/forked-repo",
        fork: true,
        default_branch: "main",
      },
    });
    const request = await makeRequest({ body });
    const response = await worker.fetch(request, TEST_ENV);
    expect(response.status).toBe(200);
    expect(await response.text()).toBe("Skipped (fork)");
  });

  it("returns 500 when GitHub API calls fail (auth failure)", async () => {
    // Mock fetch to simulate auth failure
    globalThis.fetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ message: "Bad credentials" }), {
        status: 401,
        statusText: "Unauthorized",
      }),
    );

    const body = makePayload();
    const request = await makeRequest({ body });
    const response = await worker.fetch(request, TEST_ENV);
    expect(response.status).toBe(500);
    expect(await response.text()).toBe("Internal server error");
  });

  it("processes valid repository.created event with mocked API", async () => {
    const fetchMock = vi.fn();

    // Track call order for assertions
    const calls: string[] = [];

    fetchMock.mockImplementation(async (url: string | URL | Request, init?: RequestInit) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.toString() : url.url;
      calls.push(`${init?.method ?? "GET"} ${urlStr}`);

      // Installation token
      if (urlStr.includes("/access_tokens")) {
        return new Response(JSON.stringify({ token: "test-installation-token" }), {
          status: 201,
        });
      }

      // Check .teamwork-skip file — not found
      if (urlStr.includes("/contents/.teamwork-skip")) {
        return new Response("Not Found", { status: 404 });
      }

      // Source tree
      if (urlStr.includes("/git/trees/main")) {
        return new Response(
          JSON.stringify({
            tree: [
              { path: ".github/agents/coder.agent.md", mode: "100644", type: "blob", sha: "abc123" },
              { path: "go.mod", mode: "100644", type: "blob", sha: "def456" },
              { path: "docs/conventions.md", mode: "100644", type: "blob", sha: "ghi789" },
            ],
          }),
          { status: 200 },
        );
      }

      // Get blob
      if (urlStr.includes("/git/blobs/")) {
        return new Response(
          JSON.stringify({ content: "dGVzdCBjb250ZW50" }),
          { status: 200 },
        );
      }

      // Create blob
      if (urlStr.includes("/git/blobs") && init?.method === "POST") {
        return new Response(
          JSON.stringify({ sha: "new-blob-sha-" + Math.random().toString(36).slice(2, 8) }),
          { status: 201 },
        );
      }

      // Check refs (repo is empty)
      if (urlStr.includes("/git/refs/heads/main") && (!init?.method || init.method === "GET")) {
        return new Response("Not Found", { status: 404 });
      }

      // Create tree
      if (urlStr.includes("/git/trees") && init?.method === "POST") {
        return new Response(
          JSON.stringify({ sha: "new-tree-sha" }),
          { status: 201 },
        );
      }

      // Create commit
      if (urlStr.includes("/git/commits") && init?.method === "POST") {
        return new Response(
          JSON.stringify({ sha: "new-commit-sha" }),
          { status: 201 },
        );
      }

      // Create ref (empty repo)
      if (urlStr.includes("/git/refs") && init?.method === "POST") {
        return new Response(
          JSON.stringify({ ref: "refs/heads/main", object: { sha: "new-commit-sha" } }),
          { status: 201 },
        );
      }

      return new Response("Not Found", { status: 404 });
    });

    globalThis.fetch = fetchMock;

    const body = makePayload();
    const request = await makeRequest({ body });
    const response = await worker.fetch(request, TEST_ENV);

    expect(response.status).toBe(200);
    const text = await response.text();
    expect(text).toContain("Installed Teamwork framework");
    expect(text).toContain("test-org/new-repo");

    // Verify key API calls were made
    const callUrls = calls.map((c) => c.replace(/https:\/\/api\.github\.com/, ""));

    // Should have called installation token
    expect(callUrls.some((c) => c.includes("/access_tokens"))).toBe(true);

    // Should have checked .teamwork-skip
    expect(callUrls.some((c) => c.includes("/contents/.teamwork-skip"))).toBe(true);

    // Should have fetched source tree
    expect(callUrls.some((c) => c.includes("/git/trees/main"))).toBe(true);

    // Should have created blobs (framework files + starter templates)
    const blobCreates = callUrls.filter((c) => c.startsWith("POST") && c.includes("/git/blobs"));
    // 2 framework files (coder.agent.md + docs/conventions.md, go.mod filtered out) + 3 templates = 5 blob creates
    // Plus 2 blob GETs for the framework files
    expect(blobCreates.length).toBeGreaterThanOrEqual(5);

    // Should have created a ref (empty repo)
    expect(callUrls.some((c) => c.startsWith("POST") && c.includes("/git/refs") && !c.includes("refs/heads"))).toBe(true);
  });
});
