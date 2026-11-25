import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
    setupFiles: ['./src/testing/setup.ts'],
    restoreMocks: true,
    // Return real CSS (instead of an empty module) so tokens.test.ts can read theme.css.
    css: { include: [/theme\.css/] },
  },
});
