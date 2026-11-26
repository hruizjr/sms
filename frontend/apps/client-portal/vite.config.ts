import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';
import { contentSecurityPolicy } from './config/content-security-policy.ts';
import { DEV_API_PROXY } from './config/dev-proxy.ts';

export default defineConfig({
  plugins: [react(), tailwindcss(), contentSecurityPolicy()],
  // Development only: /healthz and /readyz go to the local API (config/dev-proxy.ts).
  server: { port: 5173, strictPort: true, proxy: DEV_API_PROXY },
  // `vite preview` serves the production build and proxies nothing.
  preview: { port: 4173, strictPort: true, proxy: {} },
  build: {
    // Keep inline assets out of the build so the Content Security Policy needs no data: URLs.
    assetsInlineLimit: 0,
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}', 'config/**/*.test.ts'],
    setupFiles: ['./src/test/setup.ts'],
    restoreMocks: true,
  },
});
