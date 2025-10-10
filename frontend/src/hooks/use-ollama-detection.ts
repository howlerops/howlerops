import { useState, useEffect, useCallback } from 'react'

export interface OllamaStatus {
  installed: boolean
  running: boolean
  version?: string
  endpoint: string
  available_models: string[]
  last_checked: string
  error?: string
  backend_available: boolean
}

export interface OllamaActions {
  detectOllama: () => Promise<OllamaStatus>
  getInstallInstructions: () => Promise<string>
  startOllamaService: () => Promise<void>
  pullModel: (modelName: string) => Promise<void>
  isDetecting: boolean
  isStarting: boolean
  isPulling: boolean
}

export const useOllamaDetection = (autoDetect: boolean = true): OllamaStatus & OllamaActions => {
  const [status, setStatus] = useState<OllamaStatus>({
    installed: false,
    running: false,
    endpoint: 'http://localhost:11434',
    available_models: [],
    last_checked: new Date().toISOString(),
    backend_available: true,
  })

  const [isDetecting, setIsDetecting] = useState(false)
  const [isStarting, setIsStarting] = useState(false)
  const [isPulling, setIsPulling] = useState(false)

  const detectOllama = useCallback(async (): Promise<OllamaStatus> => {
    setIsDetecting(true)
    try {
      const response = await fetch('/api/ai/ollama/detect')
      if (!response.ok) {
        throw new Error('Failed to detect Ollama')
      }
      const detectedStatus = await response.json()
      const normalized: OllamaStatus = {
        installed: !!detectedStatus.installed,
        running: !!detectedStatus.running,
        version: detectedStatus.version,
        endpoint: detectedStatus.endpoint || 'http://localhost:11434',
        available_models: Array.isArray(detectedStatus.available_models) ? detectedStatus.available_models : [],
        last_checked: detectedStatus.last_checked || new Date().toISOString(),
        error: detectedStatus.error,
        backend_available: true,
      }
      setStatus(normalized)
      return normalized
    } catch (error) {
      const message = error instanceof Error ? error.message || 'Unable to reach backend API (is the SQL Studio server running?)' : 'Unknown error'
      const fallback: OllamaStatus = {
        installed: status.installed,
        running: status.running,
        version: status.version,
        endpoint: status.endpoint,
        available_models: status.available_models,
        last_checked: new Date().toISOString(),
        error: message,
        backend_available: false,
      }
      setStatus(fallback)
      return fallback
    } finally {
      setIsDetecting(false)
    }
  }, [status])

  const getInstallInstructions = async (): Promise<string> => {
    try {
      const response = await fetch('/api/ai/ollama/install')
      if (!response.ok) {
        throw new Error('Failed to get installation instructions')
      }
      const data = await response.json()
      return data.instructions
    } catch (error) {
      throw new Error(error instanceof Error ? error.message : 'Failed to get installation instructions')
    }
  }

  const startOllamaService = async (): Promise<void> => {
    setIsStarting(true)
    try {
      const response = await fetch('/api/ai/ollama/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      })
      if (!response.ok) {
        const error = await response.text()
        throw new Error(error || 'Failed to start Ollama service')
      }
      await detectOllama()
    } catch (error) {
      throw new Error(error instanceof Error ? error.message : 'Failed to start Ollama service')
    } finally {
      setIsStarting(false)
    }
  }

  const pullModel = async (modelName: string): Promise<void> => {
    setIsPulling(true)
    try {
      const response = await fetch('/api/ai/ollama/pull', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: modelName }),
      })
      if (!response.ok) {
        const error = await response.text()
        throw new Error(error || 'Failed to pull model')
      }
      await detectOllama()
    } catch (error) {
      throw new Error(error instanceof Error ? error.message : 'Failed to pull model')
    } finally {
      setIsPulling(false)
    }
  }

  useEffect(() => {
    if (!autoDetect) {
      return
    }
    detectOllama()
  }, [autoDetect, detectOllama])

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }
    const handler = () => {
      detectOllama()
    }
    window.addEventListener('ollama-status:refresh', handler)
    return () => window.removeEventListener('ollama-status:refresh', handler)
  }, [detectOllama])

  return {
    ...status,
    detectOllama,
    getInstallInstructions,
    startOllamaService,
    pullModel,
    isDetecting,
    isStarting,
    isPulling,
  }
}
