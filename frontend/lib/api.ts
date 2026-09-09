export class APIError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  constructor(status: number, code: string, requestId?: string) {
    super("The API request could not be completed.");
    this.status = status; this.code = code; this.requestId = requestId;
  }
}
// Browser-only, same-origin boundary. Future callers validate domain response schemas.
export async function requestJSON(
  path: `/api/v1/${string}` | "/healthz" | "/readyz",
  signal?: AbortSignal,
  fetcher: typeof fetch = fetch,
): Promise<unknown> {
  if (!path.startsWith("/api/v1/") && path !== "/healthz" && path !== "/readyz") {
    throw new Error("Unsupported API path.");
  }
  const response = await fetcher(path, { method: "GET", credentials: "same-origin", cache: "no-store", signal });
  let body: unknown;
  try { body = await response.json(); } catch { throw new APIError(response.status, "INVALID_RESPONSE"); }
  if (!response.ok) {
    const error = (body as { error?: { code?: unknown; request_id?: unknown } })?.error;
    throw new APIError(response.status, typeof error?.code === "string" ? error.code : "REQUEST_FAILED",
      typeof error?.request_id === "string" ? error.request_id : undefined);
  }
  return body;
}
