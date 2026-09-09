export function validateEnvironment(env) {
  for (const key of Object.keys(env)) {
    if (/^NEXT_PUBLIC_.*(SECRET|TOKEN|PASSWORD|DATABASE)/i.test(key) && env[key]) {
      throw new Error("Secret-like NEXT_PUBLIC configuration is forbidden.");
    }
  }
  if (env.APP_ENV && !["development", "test", "production"].includes(env.APP_ENV)) throw new Error("APP_ENV invalid.");
  const production = env.APP_ENV === "production";
  const origin = env.BACKEND_ORIGIN || (production ? "" : "http://127.0.0.1:8080");
  let parsed;
  try { parsed = new URL(origin); } catch { throw new Error("BACKEND_ORIGIN must be configured."); }
  if (!["http:", "https:"].includes(parsed.protocol) || parsed.username || parsed.password ||
      parsed.search || parsed.hash || parsed.pathname !== "/") {
    throw new Error("BACKEND_ORIGIN must be an HTTP(S) origin without credentials.");
  }
  if (production && parsed.protocol !== "https:" && !["127.0.0.1", "localhost", "[::1]"].includes(parsed.hostname)) {
    throw new Error("Remote production backend requires HTTPS.");
  }
  return { backendOrigin: parsed.origin, production };
}
