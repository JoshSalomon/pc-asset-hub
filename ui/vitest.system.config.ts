import { defineConfig } from 'vitest/config'

// System tests use Playwright directly (not Vitest Browser Mode).
// They run in Node, launch a browser, and navigate to the live UI.
export default defineConfig({
  test: {
    include: ['src/**/*.system.test.ts'],
    testTimeout: 30000,
  },
})
