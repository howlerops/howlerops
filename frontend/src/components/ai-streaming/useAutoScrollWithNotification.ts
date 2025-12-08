import { useCallback, useEffect, useRef, useState } from "react"

interface UseAutoScrollWithNotificationOptions {
  /** Whether auto-scroll is enabled by default */
  enabled?: boolean
  /** Threshold in pixels from bottom to consider "at bottom" */
  threshold?: number
  /** Callback when new content arrives while scrolled up */
  onNewContentWhileScrolledUp?: (count: number) => void
}

interface UseAutoScrollWithNotificationResult {
  /** Ref to attach to the scrollable container */
  containerRef: React.RefObject<HTMLDivElement | null>
  /** Whether there's new content below the viewport */
  hasNewContent: boolean
  /** Number of new items since user scrolled up */
  newContentCount: number
  /** Whether the user is near the bottom */
  isNearBottom: boolean
  /** Scroll to the bottom of the container */
  scrollToBottom: () => void
  /** Mark all new content as seen */
  clearNewContent: () => void
  /** Called when items change - updates scroll and notification state */
  onItemsChange: (newLength: number) => void
}

/**
 * useAutoScrollWithNotification - Smart auto-scroll hook with new content notification
 *
 * Features:
 * - Auto-scrolls to bottom when user is already at bottom
 * - Shows notification when new content arrives while scrolled up
 * - Tracks new message count
 * - Provides smooth scroll to bottom function
 *
 * @example
 * ```tsx
 * const { containerRef, hasNewContent, scrollToBottom, onItemsChange } = useAutoScrollWithNotification()
 *
 * useEffect(() => {
 *   onItemsChange(messages.length)
 * }, [messages.length])
 *
 * return (
 *   <div ref={containerRef}>
 *     {messages.map(...)}
 *     {hasNewContent && <NewContentNotification onClick={scrollToBottom} />}
 *   </div>
 * )
 * ```
 */
export function useAutoScrollWithNotification(
  options: UseAutoScrollWithNotificationOptions = {}
): UseAutoScrollWithNotificationResult {
  const { enabled = true, threshold = 100, onNewContentWhileScrolledUp } = options

  const containerRef = useRef<HTMLDivElement>(null)
  const [hasNewContent, setHasNewContent] = useState(false)
  const [newContentCount, setNewContentCount] = useState(0)
  const [isNearBottom, setIsNearBottom] = useState(true)

  const lastItemCountRef = useRef(0)
  const isUserScrollingRef = useRef(false)
  const scrollTimeoutRef = useRef<NodeJS.Timeout>(undefined)

  // Check if user is near bottom
  const checkIsNearBottom = useCallback(() => {
    const container = containerRef.current
    if (!container) return true

    const { scrollTop, scrollHeight, clientHeight } = container
    return scrollHeight - scrollTop - clientHeight <= threshold
  }, [threshold])

  // Scroll to bottom with smooth behavior
  const scrollToBottom = useCallback(() => {
    const container = containerRef.current
    if (!container) return

    container.scrollTo({
      top: container.scrollHeight,
      behavior: "smooth",
    })

    // Clear new content notification
    setHasNewContent(false)
    setNewContentCount(0)
  }, [])

  // Clear new content notification
  const clearNewContent = useCallback(() => {
    setHasNewContent(false)
    setNewContentCount(0)
  }, [])

  // Handle scroll events
  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const handleScroll = () => {
      // Mark as user scrolling
      isUserScrollingRef.current = true

      // Clear existing timeout
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current)
      }

      // Update isNearBottom state
      const nearBottom = checkIsNearBottom()
      setIsNearBottom(nearBottom)

      // Clear notification if user scrolls to bottom
      if (nearBottom) {
        setHasNewContent(false)
        setNewContentCount(0)
      }

      // Reset user scrolling flag after scroll stops
      scrollTimeoutRef.current = setTimeout(() => {
        isUserScrollingRef.current = false
      }, 150)
    }

    container.addEventListener("scroll", handleScroll, { passive: true })

    return () => {
      container.removeEventListener("scroll", handleScroll)
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current)
      }
    }
  }, [checkIsNearBottom])

  // Handle items change
  const onItemsChange = useCallback(
    (newLength: number) => {
      if (!enabled) return

      const previousLength = lastItemCountRef.current
      const hasNewItems = newLength > previousLength

      if (hasNewItems) {
        const nearBottom = checkIsNearBottom()

        if (nearBottom && !isUserScrollingRef.current) {
          // User is at bottom, auto-scroll
          requestAnimationFrame(() => {
            scrollToBottom()
          })
        } else {
          // User has scrolled up, show notification
          const newCount = newLength - previousLength
          setHasNewContent(true)
          setNewContentCount((prev) => prev + newCount)
          onNewContentWhileScrolledUp?.(newCount)
        }
      }

      lastItemCountRef.current = newLength
    },
    [enabled, checkIsNearBottom, scrollToBottom, onNewContentWhileScrolledUp]
  )

  return {
    containerRef,
    hasNewContent,
    newContentCount,
    isNearBottom,
    scrollToBottom,
    clearNewContent,
    onItemsChange,
  }
}

/**
 * useStreamingStartTime - Track when streaming started for elapsed time display
 */
export function useStreamingStartTime(isStreaming: boolean): number | undefined {
  const [startTime, setStartTime] = useState<number | undefined>(undefined)

  useEffect(() => {
    if (isStreaming && !startTime) {
      setStartTime(Date.now())
    } else if (!isStreaming) {
      setStartTime(undefined)
    }
  }, [isStreaming, startTime])

  return startTime
}
