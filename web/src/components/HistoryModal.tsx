import React, { useState, useEffect } from 'react'
import type { HistoryEntry, Endpoint } from '../types/spec'
import { X, History, Trash2, Clock, Play } from 'lucide-react'

interface HistoryModalProps {
  isOpen: boolean
  onClose: () => void
  endpoints: Endpoint[]
  onSelectEndpoint: (ep: Endpoint) => void
}

export const HistoryModal: React.FC<HistoryModalProps> = ({
  isOpen,
  onClose,
  endpoints,
  onSelectEndpoint,
}) => {
  const [entries, setEntries] = useState<HistoryEntry[]>([])
  const [loading, setLoading] = useState(false)

  const fetchHistory = async () => {
    try {
      setLoading(true)
      const resp = await fetch('/api/history')
      if (resp.ok) {
        const data = await resp.json()
        setEntries(data || [])
      }
    } catch {
      // Ignored
    } finally {
      setLoading(false)
    }
  }

  const handleClear = async () => {
    try {
      await fetch('/api/history', { method: 'DELETE' })
      setEntries([])
    } catch {
      // Ignored
    }
  }

  useEffect(() => {
    if (isOpen) {
      fetchHistory()
    }
  }, [isOpen])

  if (!isOpen) return null

  const handleReplay = (entry: HistoryEntry) => {
    // Find closest endpoint matching method and path
    const reqUrl = new URL(entry.request.url)
    const ep = endpoints.find(
      (e) =>
        e.method.toUpperCase() === entry.request.method.toUpperCase() &&
        reqUrl.pathname.includes(e.path.replace(/\{[^}]+\}/g, ''))
    )
    if (ep) {
      onSelectEndpoint(ep)
      onClose()
    }
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4">
      <div className="w-full max-w-2xl bg-[#121215] border border-zinc-800 rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800">
          <div className="flex items-center gap-2">
            <History className="w-4 h-4 text-indigo-400" />
            <span className="font-semibold text-sm text-zinc-100">Execution History</span>
          </div>

          <div className="flex items-center gap-3">
            {entries.length > 0 && (
              <button
                onClick={handleClear}
                className="flex items-center gap-1 text-xs text-zinc-500 hover:text-rose-400 transition-colors"
                title="Clear all history"
              >
                <Trash2 className="w-3.5 h-3.5" />
                <span>Clear History</span>
              </button>
            )}
            <button onClick={onClose} className="text-zinc-500 hover:text-zinc-300">
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* List of Entries */}
        <div className="flex-1 overflow-y-auto p-4 space-y-2">
          {loading ? (
            <div className="text-center py-12 text-zinc-500 text-xs">Loading execution history...</div>
          ) : entries.length === 0 ? (
            <div className="text-center py-12 text-zinc-500 text-xs">
              No executions recorded yet. Execute API calls from the playground to see them here.
            </div>
          ) : (
            entries.map((entry) => (
              <div
                key={entry.id}
                className="p-3 bg-zinc-900/60 border border-zinc-800/80 rounded-lg flex items-center justify-between gap-3 text-xs"
              >
                <div className="flex items-center gap-2.5 truncate">
                  <span className="font-mono font-bold text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-300">
                    {entry.request.method}
                  </span>

                  <span
                    className={`font-mono font-bold text-[10px] px-1.5 py-0.5 rounded ${
                      entry.response.status >= 200 && entry.response.status < 300
                        ? 'bg-emerald-950/50 text-emerald-400 border border-emerald-800/40'
                        : 'bg-rose-950/50 text-rose-400 border border-rose-800/40'
                    }`}
                  >
                    {entry.response.status}
                  </span>

                  <span className="font-mono text-zinc-200 truncate max-w-sm" title={entry.request.url}>
                    {entry.request.url}
                  </span>
                </div>

                <div className="flex items-center gap-3 shrink-0">
                  <span className="flex items-center gap-1 font-mono text-[11px] text-zinc-400">
                    <Clock className="w-3 h-3 text-zinc-500" />
                    {entry.response.timing?.totalMs || 0} ms
                  </span>

                  <span className="text-[10px] text-zinc-500 font-mono">
                    {new Date(entry.timestamp).toLocaleTimeString()}
                  </span>

                  <button
                    onClick={() => handleReplay(entry)}
                    className="p-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 hover:text-white transition-colors"
                    title="Load in playground"
                  >
                    <Play className="w-3 h-3 fill-current" />
                  </button>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-zinc-800 bg-zinc-950 flex justify-between items-center text-xs text-zinc-500">
          <span>{entries.length} requests recorded</span>
          <button
            onClick={onClose}
            className="px-4 py-1.5 text-xs font-medium bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
