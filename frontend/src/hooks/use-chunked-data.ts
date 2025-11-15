/**
 * useChunkedData Hook
 *
 * Manages on-demand loading of data chunks from IndexedDB for large datasets.
 * Works with TanStack Virtual to load only visible rows + overscan.
 */

import { useState, useEffect, useCallback, useRef } from 'react'
import {
  loadChunks,
  calculateRequiredChunks,
  CHUNK_CONFIG,
} from '../lib/query-result-storage'

interface UseChunkedDataOptions {
  resultId: string
  totalRows: number
  isLarge: boolean
  initialData: unknown[]
}

interface ChunkedDataState {
  data: unknown[]
  loadedChunks: Set<number>
  isLoading: boolean
  error: string | null
}

interface UseChunkedDataResult extends ChunkedDataState {
  ensureRangeLoaded: (startIndex: number, endIndex: number) => void
}

/**
 * Hook to manage chunked data loading for large result sets
 */
export function useChunkedData({
  resultId,
  totalRows,
  isLarge,
  initialData,
}: UseChunkedDataOptions): UseChunkedDataResult {
  const chunksRef = useRef(new Map<number, unknown[]>())
  const loadingChunksRef = useRef(new Set<number>())

  const [state, setState] = useState<ChunkedDataState>(() => {
    if (!isLarge) {
      return {
        data: initialData,
        loadedChunks: new Set([0]),
        isLoading: false,
        error: null,
      }
    }

    const seededChunks = seedInitialChunks(initialData)
    chunksRef.current = seededChunks
    loadingChunksRef.current = new Set()

    return {
      data: buildDataFromChunks(seededChunks, totalRows),
      loadedChunks: seededChunks.size ? new Set([0]) : new Set(),
      isLoading: false,
      error: null,
    }
  })

  useEffect(() => {
    if (!isLarge) {
      setState({
        data: initialData,
        loadedChunks: new Set([0]),
        isLoading: false,
        error: null,
      })
      chunksRef.current = new Map()
      loadingChunksRef.current = new Set()
      return
    }

    const seededChunks = seedInitialChunks(initialData)
    chunksRef.current = seededChunks
    loadingChunksRef.current = new Set()

    setState({
      data: buildDataFromChunks(seededChunks, totalRows),
      loadedChunks: seededChunks.size ? new Set([0]) : new Set(),
      isLoading: false,
      error: null,
    })
  }, [resultId, initialData, totalRows, isLarge])

  /**
   * Load chunks from IndexedDB if not already loaded
   */
  const loadChunksIfNeeded = useCallback(
    async (chunkIndices: number[]) => {
      if (!isLarge) {
        return
      }
      // Filter out already loaded/loading chunks
      const chunksToLoad = chunkIndices.filter(
        (idx) =>
          !state.loadedChunks.has(idx) && !loadingChunksRef.current.has(idx)
      )

      if (chunksToLoad.length === 0) return

      // Mark as loading
      chunksToLoad.forEach((idx) => loadingChunksRef.current.add(idx))
      setState((prev) => ({ ...prev, isLoading: true }))

      try {
        const newChunks = await loadChunks(resultId, chunksToLoad)

        // Update chunks map
        newChunks.forEach((rows, idx) => {
          chunksRef.current.set(idx, rows)
        })

        // Apply LRU eviction if needed
        if (chunksRef.current.size > CHUNK_CONFIG.MAX_CHUNKS_IN_MEMORY) {
          evictOldChunks(chunksRef.current, chunkIndices)
        }

        // Rebuild data array from loaded chunks
        const newData = buildDataFromChunks(chunksRef.current, totalRows)

        setState({
          data: newData,
          loadedChunks: new Set(chunksRef.current.keys()),
          isLoading: false,
          error: null,
        })
      } catch (error) {
        setState((prev) => ({
          ...prev,
          isLoading: false,
          error:
            error instanceof Error
              ? error.message
              : 'Failed to load chunks',
        }))
      } finally {
        chunksToLoad.forEach((idx) => loadingChunksRef.current.delete(idx))
      }
    },
    [isLarge, resultId, totalRows, state.loadedChunks]
  )

  /**
   * Load the chunks required to cover the requested visible range
   */
  const ensureRangeLoaded = useCallback((startIndex: number, endIndex: number) => {
    if (!isLarge) return
    if (startIndex < 0 || endIndex < startIndex) return
    const requiredChunks = calculateRequiredChunks(startIndex, endIndex)
    loadChunksIfNeeded(requiredChunks)
  }, [isLarge, loadChunksIfNeeded])

  if (!isLarge) {
    return {
      ...state,
      ensureRangeLoaded: () => {},
    }
  }

  return {
    ...state,
    ensureRangeLoaded,
  }
}

/**
 * Build data array with placeholders for unloaded chunks
 */
function buildDataFromChunks(
  chunks: Map<number, unknown[]>,
  totalRows: number
): unknown[] {
  const data: unknown[] = []

  for (let i = 0; i < totalRows; i++) {
    const chunkIndex = Math.floor(i / CHUNK_CONFIG.CHUNK_SIZE)
    const chunk = chunks.get(chunkIndex)

    if (chunk) {
      const indexInChunk = i % CHUNK_CONFIG.CHUNK_SIZE
      data[i] = chunk[indexInChunk]
    } else {
      // Placeholder for unloaded row
      data[i] = {
        __rowId: `placeholder-${i}`,
        __isPlaceholder: true,
      }
    }
  }

  return data
}

function seedInitialChunks(initialData: unknown[]): Map<number, unknown[]> {
  const seeded = new Map<number, unknown[]>()
  if (initialData && initialData.length > 0) {
    seeded.set(0, initialData.slice(0, CHUNK_CONFIG.CHUNK_SIZE))
  }
  return seeded
}

/**
 * LRU eviction: Remove old chunks to stay under memory limit
 */
function evictOldChunks(
  chunks: Map<number, unknown[]>,
  keepChunks: number[]
) {
  const keepSet = new Set(keepChunks)
  const toEvict: number[] = []

  chunks.forEach((_, chunkIndex) => {
    if (!keepSet.has(chunkIndex)) {
      toEvict.push(chunkIndex)
    }
  })

  // Evict oldest chunks first (lower indices = older)
  toEvict.sort((a, b) => a - b)
  const evictCount = chunks.size - CHUNK_CONFIG.MAX_CHUNKS_IN_MEMORY

  for (let i = 0; i < Math.min(evictCount, toEvict.length); i++) {
    chunks.delete(toEvict[i])
  }
}
