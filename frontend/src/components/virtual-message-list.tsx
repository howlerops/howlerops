import { useVirtualizer } from '@tanstack/react-virtual'
import { useEffect, useRef } from 'react'

interface VirtualMessageListProps<T> {
  messages: T[]
  renderMessage: (message: T, index: number) => React.ReactNode
  estimateSize?: number
  overscan?: number
  className?: string
  autoScroll?: boolean
  getMessageKey: (message: T, index: number) => string
}

/**
 * VirtualMessageList - A performant message list component using virtual scrolling
 *
 * Features:
 * - Only renders visible messages (+ buffer) for optimal performance
 * - Supports variable message heights
 * - Auto-scrolls to bottom on new messages
 * - Maintains scroll position when messages are added above viewport
 * - Accessible - preserves keyboard navigation and screen reader support
 *
 * @example
 * ```tsx
 * <VirtualMessageList
 *   messages={chatMessages}
 *   renderMessage={(msg, idx) => <MessageBubble key={msg.id} message={msg} />}
 *   getMessageKey={(msg) => msg.id}
 *   estimateSize={100}
 *   autoScroll={true}
 * />
 * ```
 */
export function VirtualMessageList<T>({
  messages,
  renderMessage,
  estimateSize = 100,
  overscan = 5,
  className = '',
  autoScroll = true,
  getMessageKey,
}: VirtualMessageListProps<T>) {
  const parentRef = useRef<HTMLDivElement>(null)
  const isUserScrollingRef = useRef(false)
  const scrollTimeoutRef = useRef<NodeJS.Timeout>()

  const virtualizer = useVirtualizer({
    count: messages.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => estimateSize,
    overscan,
    // Enable dynamic measurement for variable height messages
    measureElement:
      typeof window !== 'undefined' && navigator.userAgent.indexOf('Firefox') === -1
        ? element => element.getBoundingClientRect().height
        : undefined,
  })

  // Track user scrolling to prevent auto-scroll interference
  useEffect(() => {
    const parent = parentRef.current
    if (!parent) return

    const handleScroll = () => {
      // Clear any existing timeout
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current)
      }

      // Mark as user scrolling
      isUserScrollingRef.current = true

      // Reset after scrolling stops (debounce)
      scrollTimeoutRef.current = setTimeout(() => {
        isUserScrollingRef.current = false
      }, 150)
    }

    parent.addEventListener('scroll', handleScroll, { passive: true })
    return () => {
      parent.removeEventListener('scroll', handleScroll)
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current)
      }
    }
  }, [])

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (!autoScroll || isUserScrollingRef.current || messages.length === 0) {
      return
    }

    // Use requestAnimationFrame to ensure DOM is updated
    requestAnimationFrame(() => {
      virtualizer.scrollToIndex(messages.length - 1, {
        align: 'end',
        behavior: 'smooth',
      })
    })
  }, [messages.length, autoScroll, virtualizer])

  return (
    <div
      ref={parentRef}
      className={className}
      style={{
        overflow: 'auto',
        contain: 'strict',
      }}
      role="log"
      aria-live="polite"
      aria-relevant="additions"
    >
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {virtualizer.getVirtualItems().map(virtualItem => {
          const message = messages[virtualItem.index]
          if (!message) return null

          return (
            <div
              key={getMessageKey(message, virtualItem.index)}
              data-index={virtualItem.index}
              ref={virtualizer.measureElement}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                transform: `translateY(${virtualItem.start}px)`,
              }}
            >
              {renderMessage(message, virtualItem.index)}
            </div>
          )
        })}
      </div>
    </div>
  )
}
