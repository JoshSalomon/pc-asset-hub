import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    include: ['src/**/*.unit.test.{ts,tsx}'],
    environment: 'node',
    testTimeout: 10000,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'text-summary', 'json-summary', 'json'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/test/**',
        'src/**/*.test.{ts,tsx}',
        'src/**/*.unit.test.{ts,tsx}',
        'src/**/*.browser.test.{ts,tsx}',
        'src/**/*.system.test.{ts,tsx}',
        'src/vite-env.d.ts',
        'src/main.tsx',
        'src/main-operational.tsx',
        'src/types/**',
        'src/test-helpers/**',
      ],
    },
  },
})
