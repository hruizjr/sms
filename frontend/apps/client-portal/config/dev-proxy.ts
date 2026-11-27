import type { ProxyOptions } from 'vite';

/**
 * Development-server proxy to the local API (Phase 3 Group 6, owner decision L5; /portal added in
 * Phase 4 Group 3, owner decision G3-6).
 *
 * The Vite dev server forwards exactly these paths to the API started by infra/local/compose.yaml
 * (published on 127.0.0.1:8080), so the portal and the API share one origin in development: the
 * session cookie is first-party and the API needs no CORS. The public API (/v1) is deliberately not
 * proxied: the browser never uses API keys.
 *
 * The target is fixed (API_PORT's default); reading an environment variable here would need Node.js
 * type definitions, which are not added.
 */
export const DEV_API_TARGET = 'http://127.0.0.1:8080';

export const DEV_API_PROXY: Record<string, ProxyOptions> = {
  '/healthz': { target: DEV_API_TARGET, changeOrigin: false },
  '/readyz': { target: DEV_API_TARGET, changeOrigin: false },
  '/portal': { target: DEV_API_TARGET, changeOrigin: false },
};
