import test from "node:test";
import assert from "node:assert/strict";
import { validateEnvironment } from "../env.mjs";
import { requestJSON, APIError } from "../lib/api.ts";
test("environment rejects credentials and remote plaintext", () => {
  assert.throws(() => validateEnvironment({ APP_ENV: "production" }));
  assert.throws(() => validateEnvironment({ APP_ENV: "production", BACKEND_ORIGIN: "http://remote.example.test" }));
  assert.throws(() => validateEnvironment({ NEXT_PUBLIC_DATABASE_URL: "sensitive" }));
  assert.equal(validateEnvironment({}).backendOrigin, "http://127.0.0.1:8080");
});
test("API uses same-origin reads and exposes a safe error envelope", async () => {
  await assert.rejects(requestJSON("/readyz", undefined, async (path, options) => {
    assert.equal(path, "/readyz"); assert.equal(options.credentials, "same-origin"); assert.equal(options.method, "GET");
    return new Response(JSON.stringify({ error: { code: "DEPENDENCY_UNAVAILABLE", message: "private", request_id: "req-1" } }), { status: 503 });
  }), (e) => e instanceof APIError && e.code === "DEPENDENCY_UNAVAILABLE" && e.requestId === "req-1" && !e.message.includes("private"));
  await assert.rejects(requestJSON("https://elsewhere.example.test"), /Unsupported/);
});
