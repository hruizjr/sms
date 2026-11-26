import type { ProxyOptions } from 'vite';

/**
 * Development-server proxy to the local API (Phase 3 Group 6, owner decision L5).
 *
 * The Vite dev server forwards exactly these two paths to the API started by
 * infra/local/compose.yaml (published on 127.0.0.1:8080), so the portal and the API share one origin
 * in development and the API needs no CORS. Nothing else is proxied: no /v1 or other API path, and no
 * frontend code calls these endpoints. The production Content Security Policy (connect-src 'none'),
 * the ESLint network rules and the test network guard are unchanged.
 *
 * The target is fixed (API_PORT's default); reading an environment variable here would need Node.js
 * type definitions, which are not added.
 */
export const DEV_API_TARGET = 'http://127.0.0.1:8080';

export const DEV_API_PROXY: Record<string, ProxyOptions> = {
  '/healthz': { target: DEV_API_TARGET, changeOrigin: false },
  '/readyz': { target: DEV_API_TARGET, changeOrigin: false },
};
