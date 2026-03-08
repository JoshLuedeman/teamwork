const GITHUB_API = "https://api.github.com";

/**
 * Create a JWT for GitHub App authentication.
 *
 * Uses RS256 (RSASSA-PKCS1-v1_5 with SHA-256) via the Web Crypto API.
 */
export async function createJWT(
  appId: string,
  privateKey: string,
): Promise<string> {
  const header = { alg: "RS256", typ: "JWT" };
  const now = Math.floor(Date.now() / 1000);
  const payload = {
    iss: appId,
    iat: now - 60,
    exp: now + 600,
  };

  const encodedHeader = base64url(JSON.stringify(header));
  const encodedPayload = base64url(JSON.stringify(payload));
  const signingInput = `${encodedHeader}.${encodedPayload}`;

  const key = await importPrivateKey(privateKey);
  const signatureBuffer = await crypto.subtle.sign(
    "RSASSA-PKCS1-v1_5",
    key,
    new TextEncoder().encode(signingInput),
  );

  const encodedSignature = base64urlFromBuffer(new Uint8Array(signatureBuffer));
  return `${signingInput}.${encodedSignature}`;
}

/**
 * Exchange a JWT for an installation access token.
 */
export async function getInstallationToken(
  jwt: string,
  installationId: number,
): Promise<string> {
  const response = await fetch(
    `${GITHUB_API}/app/installations/${installationId}/access_tokens`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${jwt}`,
        Accept: "application/vnd.github.v3+json",
        "User-Agent": "teamwork-installer",
      },
    },
  );

  if (!response.ok) {
    throw new Error(
      `Failed to get installation token: ${response.status} ${response.statusText}`,
    );
  }

  const data = (await response.json()) as { token: string };
  return data.token;
}

/**
 * Parse a PEM-encoded RSA private key and import it as a CryptoKey.
 */
async function importPrivateKey(pem: string): Promise<CryptoKey> {
  const pemBody = pem
    .replace(/-----BEGIN RSA PRIVATE KEY-----/g, "")
    .replace(/-----END RSA PRIVATE KEY-----/g, "")
    .replace(/-----BEGIN PRIVATE KEY-----/g, "")
    .replace(/-----END PRIVATE KEY-----/g, "")
    .replace(/\s/g, "");

  const binaryDer = base64ToArrayBuffer(pemBody);

  // Web Crypto only supports PKCS#8 import. If the key is PKCS#1
  // (BEGIN RSA PRIVATE KEY), wrap it in a PKCS#8 envelope first.
  const keyData = pem.includes("BEGIN RSA PRIVATE KEY")
    ? wrapPKCS1inPKCS8(binaryDer)
    : binaryDer;

  return crypto.subtle.importKey(
    "pkcs8",
    keyData,
    { name: "RSASSA-PKCS1-v1_5", hash: "SHA-256" },
    false,
    ["sign"],
  );
}

/**
 * Wrap a PKCS#1 RSA private key in a PKCS#8 envelope.
 *
 * Web Crypto API only accepts PKCS#8 format, so PKCS#1 keys need wrapping.
 */
function wrapPKCS1inPKCS8(pkcs1: ArrayBuffer): ArrayBuffer {
  // PKCS#8 header for RSA keys (OID 1.2.840.113549.1.1.1)
  const pkcs1Bytes = new Uint8Array(pkcs1);

  // Build the OCTET STRING wrapping the PKCS#1 key
  const octetString = wrapASN1(0x04, pkcs1Bytes);

  // AlgorithmIdentifier: SEQUENCE { OID rsaEncryption, NULL }
  const rsaOID = new Uint8Array([
    0x06, 0x09, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x01,
  ]);
  const nullParam = new Uint8Array([0x05, 0x00]);
  const algorithmId = wrapASN1(
    0x30,
    concatBytes(rsaOID, nullParam),
  );

  // Version INTEGER 0
  const version = new Uint8Array([0x02, 0x01, 0x00]);

  // Outer SEQUENCE
  const wrapped = wrapASN1(
    0x30,
    concatBytes(version, algorithmId, octetString),
  );
  return wrapped.buffer.slice(wrapped.byteOffset, wrapped.byteOffset + wrapped.byteLength);
}

/** Wrap data in an ASN.1 TLV (tag-length-value). */
function wrapASN1(tag: number, data: Uint8Array): Uint8Array {
  const length = encodeASN1Length(data.length);
  const result = new Uint8Array(1 + length.length + data.length);
  result[0] = tag;
  result.set(length, 1);
  result.set(data, 1 + length.length);
  return result;
}

/** Encode an ASN.1 length in DER format. */
function encodeASN1Length(length: number): Uint8Array {
  if (length < 0x80) {
    return new Uint8Array([length]);
  }
  const bytes: number[] = [];
  let temp = length;
  while (temp > 0) {
    bytes.unshift(temp & 0xff);
    temp >>= 8;
  }
  return new Uint8Array([0x80 | bytes.length, ...bytes]);
}

/** Concatenate multiple Uint8Arrays. */
function concatBytes(...arrays: Uint8Array[]): Uint8Array {
  const totalLength = arrays.reduce((sum, arr) => sum + arr.length, 0);
  const result = new Uint8Array(totalLength);
  let offset = 0;
  for (const arr of arrays) {
    result.set(arr, offset);
    offset += arr.length;
  }
  return result;
}

/** Base64url-encode a string (no padding). */
function base64url(str: string): string {
  return btoa(str).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/** Base64url-encode a Uint8Array (no padding). */
function base64urlFromBuffer(buffer: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < buffer.length; i++) {
    binary += String.fromCharCode(buffer[i]);
  }
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

/** Decode a base64 string to an ArrayBuffer. */
function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}
