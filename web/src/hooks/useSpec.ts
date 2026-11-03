import { useEffect, useState, useCallback } from 'react'
import type { Spec } from '../types/spec'

export function useSpec() {
  const [spec, setSpec] = useState<Spec | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchSpec = useCallback(async () => {
    try {
      setLoading(true)
      const resp = await fetch('/api/spec')
      if (!resp.ok) {
        throw new Error(`Failed to load specification: HTTP ${resp.status}`)
      }
      const data: Spec = await resp.json()
      setSpec(data)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error loading specification')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchSpec()

    // Setup Server-Sent Events (SSE) for hot-reload on disk change
    const events = new EventSource('/api/events')

    events.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data)
        if (payload.type === 'reload') {
          console.log('[APIDock] Received hot-reload event, refreshing specification...')
          fetchSpec()
        }
      } catch (e) {
        console.warn('[APIDock] Failed to parse SSE event data', e)
      }
    }

    events.onerror = () => {
      // EventSource automatically retries
    }

    return () => {
      events.close()
    }
  }, [fetchSpec])

  return { spec, loading, error, reloadSpec: fetchSpec }
}
