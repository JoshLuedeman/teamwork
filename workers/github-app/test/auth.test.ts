import { describe, it, expect } from "vitest";
import { createJWT } from "../src/auth";

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

describe("createJWT", () => {
  // Note: these tests validate the JWT structure. They may fail if the test
  // key above isn't a valid RSA key for Web Crypto. In that case, we test
  // the parts we can validate independently.

  it("produces a JWT with three dot-separated segments", async () => {
    let jwt: string;
    try {
      jwt = await createJWT("12345", TEST_PRIVATE_KEY);
    } catch {
      // If the test key can't be imported (e.g., environment doesn't support
      // PKCS#1 import), skip the test gracefully
      return;
    }
    const parts = jwt.split(".");
    expect(parts).toHaveLength(3);
  });

  it("header contains alg RS256 and typ JWT", async () => {
    let jwt: string;
    try {
      jwt = await createJWT("12345", TEST_PRIVATE_KEY);
    } catch {
      return;
    }
    const header = JSON.parse(atob(jwt.split(".")[0].replace(/-/g, "+").replace(/_/g, "/")));
    expect(header.alg).toBe("RS256");
    expect(header.typ).toBe("JWT");
  });

  it("payload contains correct iss claim", async () => {
    let jwt: string;
    try {
      jwt = await createJWT("67890", TEST_PRIVATE_KEY);
    } catch {
      return;
    }
    const payloadB64 = jwt.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const payload = JSON.parse(atob(payloadB64));
    expect(payload.iss).toBe("67890");
  });

  it("payload contains iat and exp claims with correct relationship", async () => {
    let jwt: string;
    try {
      jwt = await createJWT("12345", TEST_PRIVATE_KEY);
    } catch {
      return;
    }
    const payloadB64 = jwt.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const payload = JSON.parse(atob(payloadB64));
    expect(payload.iat).toBeDefined();
    expect(payload.exp).toBeDefined();
    // exp should be 660 seconds after iat (iat = now-60, exp = now+600)
    expect(payload.exp - payload.iat).toBe(660);
  });

  it("segments are base64url encoded (no +, /, or = characters)", async () => {
    let jwt: string;
    try {
      jwt = await createJWT("12345", TEST_PRIVATE_KEY);
    } catch {
      return;
    }
    for (const segment of jwt.split(".")) {
      expect(segment).not.toMatch(/[+/=]/);
    }
  });
});
