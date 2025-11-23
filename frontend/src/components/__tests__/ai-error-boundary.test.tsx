import { fireEvent,render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { AIErrorBoundary } from '../ai-error-boundary'

const ThrowError = ({ shouldThrow }: { shouldThrow: boolean }) => {
  if (shouldThrow) {
    throw new Error('Test error message')
  }
  return <div>Normal content</div>
}

describe('AIErrorBoundary', () => {
  // Suppress console.error for these tests
  const consoleError = console.error
  beforeAll(() => {
    console.error = vi.fn()
  })
  afterAll(() => {
    console.error = consoleError
  })

  it('renders children when there is no error', () => {
    render(
      <AIErrorBoundary>
        <div>Test content</div>
      </AIErrorBoundary>
    )

    expect(screen.getByText('Test content')).toBeInTheDocument()
  })

  it('renders error UI when child component throws', () => {
    render(
      <AIErrorBoundary featureName="AI Assistant">
        <ThrowError shouldThrow={true} />
      </AIErrorBoundary>
    )

    expect(screen.getByText(/AI Assistant Unavailable/i)).toBeInTheDocument()
    expect(screen.getAllByText(/Test error message/i).length).toBeGreaterThan(0)
  })

  it('displays feature name in error message', () => {
    render(
      <AIErrorBoundary featureName="Natural Language Query">
        <ThrowError shouldThrow={true} />
      </AIErrorBoundary>
    )

    expect(screen.getByText(/Natural Language Query Unavailable/i)).toBeInTheDocument()
  })

  it('shows default feature name when none provided', () => {
    render(
      <AIErrorBoundary>
        <ThrowError shouldThrow={true} />
      </AIErrorBoundary>
    )

    expect(screen.getByText(/AI Assistant Unavailable/i)).toBeInTheDocument()
  })

  it('calls onError callback when error occurs', () => {
    const onError = vi.fn()

    render(
      <AIErrorBoundary onError={onError}>
        <ThrowError shouldThrow={true} />
      </AIErrorBoundary>
    )

    expect(onError).toHaveBeenCalled()
    expect(onError.mock.calls[0][0]).toBeInstanceOf(Error)
    expect(onError.mock.calls[0][0].message).toBe('Test error message')
  })

  it('allows retry after error', () => {
    let shouldThrow = true
    const TestComponent = () => <ThrowError shouldThrow={shouldThrow} />

    const { rerender } = render(
      <AIErrorBoundary>
        <TestComponent />
      </AIErrorBoundary>
    )

    // Initially shows error
    expect(screen.getByText(/AI Assistant Unavailable/i)).toBeInTheDocument()

    // Click retry button
    const retryButton = screen.getByRole('button', { name: /Retry/i })

    // Fix the component before retry
    shouldThrow = false
    fireEvent.click(retryButton)

    // After retry, should show normal content
    expect(screen.queryByText(/AI Assistant Unavailable/i)).not.toBeInTheDocument()
  })

  it('renders custom fallback when provided', () => {
    const customFallback = <div>Custom error message</div>

    render(
      <AIErrorBoundary fallback={customFallback}>
        <ThrowError shouldThrow={true} />
      </AIErrorBoundary>
    )

    expect(screen.getByText('Custom error message')).toBeInTheDocument()
    expect(screen.queryByText(/AI Assistant Unavailable/i)).not.toBeInTheDocument()
  })

  it('shows technical details in development mode', () => {
    const originalEnv = process.env.NODE_ENV
    process.env.NODE_ENV = 'development'

    render(
      <AIErrorBoundary>
        <ThrowError shouldThrow={true} />
      </AIErrorBoundary>
    )

    // In development mode, we should see technical details
    const details = screen.queryByText(/Technical Details/i)
    // The details element exists in dev mode
    if (process.env.NODE_ENV === 'development') {
      expect(details).toBeInTheDocument()
    }

    process.env.NODE_ENV = originalEnv
  })

  it('provides user-friendly messaging', () => {
    render(
      <AIErrorBoundary>
        <ThrowError shouldThrow={true} />
      </AIErrorBoundary>
    )

    expect(screen.getByText(/the rest of your workspace is still working normally/i)).toBeInTheDocument()
  })
})
