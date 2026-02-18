import '@testing-library/jest-dom'
import { vi } from 'vitest'

// PatternFly components use browser APIs that jsdom doesn't implement.
// Without these mocks, tests using Select, Tabs, and other components
// will hang or timeout. See:
// - https://github.com/trurl-master/jsdom-testing-mocks
// - https://mantine.dev/guides/vitest/

// Mock ResizeObserver
class MockResizeObserver {
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
}
window.ResizeObserver = MockResizeObserver as unknown as typeof ResizeObserver

// Mock IntersectionObserver
class MockIntersectionObserver {
  observe = vi.fn()
  disconnect = vi.fn()
  unobserve = vi.fn()
}
window.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// Mock scrollIntoView (used by Tabs overflow)
window.HTMLElement.prototype.scrollIntoView = vi.fn()
