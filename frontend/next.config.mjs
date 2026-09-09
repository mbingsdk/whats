import { validateEnvironment } from "./env.mjs";
const env = validateEnvironment(process.env);
const nextConfig = {
  poweredByHeader: false,
  output: "standalone",
  async rewrites() {
    if (env.production) return [];
    return ["/healthz", "/readyz", "/api/:path*"].map((source) => ({
      source, destination: env.backendOrigin + source,
    }));
  },
  async headers() {
    return [{ source: "/(.*)", headers: [
      { key: "X-Content-Type-Options", value: "nosniff" },
      { key: "Referrer-Policy", value: "no-referrer" },
      { key: "X-Frame-Options", value: "DENY" },
    ] }];
  },
};
export default nextConfig;
