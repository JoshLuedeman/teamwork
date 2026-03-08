import { describe, it, expect } from "vitest";
import { verifySignature } from "../src/verify";

/** Helper: compute HMAC-SHA256 hex digest using Web Crypto API. */
async function computeHMAC(body: string, secret: string): Promise<string> {
  const encoder = new TextEncoder();
  const key = await crypto.subtle.importKey(
    "raw",
    encoder.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const mac = await crypto.subtle.sign("HMAC", key, encoder.encode(body));
  const bytes = new Uint8Array(mac);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

describe("verifySignature", () => {
  const secret = "test-webhook-secret";
  const body = '{"action":"created","repository":{"name":"test-repo"}}';

  it("returns true for a valid signature", async () => {
    const hex = await computeHMAC(body, secret);
    const signature = `sha256=${hex}`;
    const result = await verifySignature(body, signature, secret);
    expect(result).toBe(true);
  });

  it("returns false for an invalid signature", async () => {
    const signature = "sha256=0000000000000000000000000000000000000000000000000000000000000000";
    const result = await verifySignature(body, signature, secret);
    expect(result).toBe(false);
  });

  it("returns false for a missing signature", async () => {
    const result = await verifySignature(body, "", secret);
    expect(result).toBe(false);
  });

  it("returns false for a malformed signature (no sha256= prefix)", async () => {
    const hex = await computeHMAC(body, secret);
    const result = await verifySignature(body, hex, secret);
    expect(result).toBe(false);
  });

  it("returns false for a signature with invalid hex characters", async () => {
    const result = await verifySignature(body, "sha256=zzzz", secret);
    expect(result).toBe(false);
  });

  it("returns false when signature has wrong length", async () => {
    const result = await verifySignature(body, "sha256=abcd", secret);
    expect(result).toBe(false);
  });
});
