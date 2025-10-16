import { useEffect, useRef, useCallback } from 'react'
import { useConnectionStore } from '@/store/connection-store'
import { RefreshSchema } from '../../wailsjs/go/main/App'

export function useSchemaRefresh(intervalMs = 60000) {
  const { connections } = useConnectionStore()
  const lastRefreshRef = useRef<Map<string, number>>(new Map())
  
  useEffect(() => {
    const interval = setInterval(async () => {
      const connected = connections.filter(c => c.isConnected && c.sessionId)
      
      if (connected.length === 0) return
      
      console.log('🔄 Background schema refresh for', connected.length, 'connections')
      
      // Refresh schemas (backend cache makes this fast!)
      const results = await Promise.allSettled(
        connected.map(async (conn) => {
          try {
            await RefreshSchema(conn.sessionId!)
            lastRefreshRef.current.set(conn.id, Date.now())
            console.log(`  ✓ ${conn.name}: Schema refreshed`)
          } catch {
            console.debug(`  ⊘ ${conn.name}: Schema unchanged (cache valid)`)
          }
        })
      )
      
      const successful = results.filter(r => r.status === 'fulfilled').length
      console.log(`🔄 Background refresh complete: ${successful}/${connected.length} successful`)
    }, intervalMs)
    
    return () => clearInterval(interval)
  }, [connections, intervalMs])

  const getLastRefresh = useCallback(() => lastRefreshRef.current, [])
  
  const forceRefresh = useCallback(async () => {
    const connected = connections.filter(c => c.isConnected && c.sessionId)
    await Promise.all(connected.map(c => RefreshSchema(c.sessionId!)))
  }, [connections])
  
  return {
    getLastRefresh,
    forceRefresh
  }
}

