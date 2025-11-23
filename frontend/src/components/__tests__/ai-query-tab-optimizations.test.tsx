import { renderHook } from '@testing-library/react'
import { useCallback, useMemo } from 'react'
import { describe, expect,it } from 'vitest'

/**
 * Tests to verify React optimization patterns are correctly implemented
 * These tests check for proper memoization and callback stability
 */

describe('React Optimization Patterns', () => {
  describe('useCallback stability', () => {
    it('should create stable callback references', () => {
      const { result, rerender } = renderHook(
        ({ value }) => useCallback(() => value, [value]),
        { initialProps: { value: 'test' } }
      )

      const firstCallback = result.current

      // Rerender with same value
      rerender({ value: 'test' })
      expect(result.current).toBe(firstCallback)

      // Rerender with different value
      rerender({ value: 'different' })
      expect(result.current).not.toBe(firstCallback)
    })

    it('should handle callbacks with multiple dependencies', () => {
      const { result, rerender } = renderHook(
        ({ a, b }) => useCallback(() => a + b, [a, b]),
        { initialProps: { a: 1, b: 2 } }
      )

      const firstCallback = result.current

      // Rerender with same values
      rerender({ a: 1, b: 2 })
      expect(result.current).toBe(firstCallback)

      // Rerender with one changed value
      rerender({ a: 1, b: 3 })
      expect(result.current).not.toBe(firstCallback)
    })
  })

  describe('useMemo stability', () => {
    it('should create stable memoized values', () => {
      const { result, rerender } = renderHook(
        ({ items }) => useMemo(() => items.map(i => i * 2), [items]),
        { initialProps: { items: [1, 2, 3] } }
      )

      const firstValue = result.current

      // Rerender with same reference
      rerender({ items: [1, 2, 3] })
      expect(result.current).not.toBe(firstValue) // Different array reference

      // Rerender with same value but same reference
      const sameItems = [1, 2, 3]
      const { result: result2, rerender: rerender2 } = renderHook(
        ({ items }) => useMemo(() => items.map(i => i * 2), [items]),
        { initialProps: { items: sameItems } }
      )

      const value1 = result2.current
      rerender2({ items: sameItems })
      expect(result2.current).toBe(value1) // Same reference = stable memo
    })

    it('should handle expensive computations', () => {
      let computeCount = 0
      const expensiveCompute = (value: number) => {
        computeCount++
        return value * 2
      }

      const { result, rerender } = renderHook(
        ({ value }) => useMemo(() => expensiveCompute(value), [value]),
        { initialProps: { value: 10 } }
      )

      expect(computeCount).toBe(1)
      expect(result.current).toBe(20)

      // Rerender with same value
      rerender({ value: 10 })
      expect(computeCount).toBe(1) // Should not recompute

      // Rerender with different value
      rerender({ value: 20 })
      expect(computeCount).toBe(2) // Should recompute
      expect(result.current).toBe(40)
    })
  })

  describe('Optimization best practices', () => {
    it('demonstrates correct dependency array usage', () => {
      const { result, rerender } = renderHook(
        ({ config }) => {
          const processData = useCallback(
            (data: string) => {
              return `${config.prefix}-${data}`
            },
            [config.prefix] // Only depends on prefix, not entire config
          )
          return processData
        },
        { initialProps: { config: { prefix: 'test', suffix: 'end' } } }
      )

      const callback1 = result.current

      // Rerender with same prefix but different suffix
      rerender({ config: { prefix: 'test', suffix: 'different' } })
      expect(result.current).toBe(callback1) // Should be stable

      // Rerender with different prefix
      rerender({ config: { prefix: 'new', suffix: 'end' } })
      expect(result.current).not.toBe(callback1) // Should update
    })

    it('demonstrates proper memo usage for arrays', () => {
      const { result, rerender } = renderHook(
        () => {
          // Good: Static array memoized
          const staticOptions = useMemo(() => ['option1', 'option2', 'option3'], [])
          return staticOptions
        }
      )

      const options1 = result.current
      rerender()
      expect(result.current).toBe(options1) // Should be stable
    })
  })
})
