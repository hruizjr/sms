import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';
import { contentSecurityPolicy } from './config/content-security-policy.ts';

export default defineConfig({
  plugins: [react(), tailwindcss(), contentSecurityPolicy()],
  server: { port: 5173, strictPort: true },
  preview: { port: 4173, strictPort: true },
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
