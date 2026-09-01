// Deterministic-enough mock for tests: real SHA-256 via node:crypto,
// pseudo-random bytes so PKCE strings are well-formed.
import { createHash, randomBytes } from "node:crypto";

export const CryptoDigestAlgorithm = { SHA256: "SHA-256" } as const;
export const CryptoEncoding = { BASE64: "base64", HEX: "hex" } as const;

export function getRandomBytes(count: number): Uint8Array {
  return new Uint8Array(randomBytes(count));
}

export async function digestStringAsync(
  _alg: string,
  data: string,
  opts?: { encoding?: string }
): Promise<string> {
  const h = createHash("sha256").update(data, "utf8");
  return opts?.encoding === "hex" ? h.digest("hex") : h.digest("base64");
}
