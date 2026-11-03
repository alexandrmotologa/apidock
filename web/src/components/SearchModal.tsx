import React, { useState, useEffect, useRef } from 'react'
import type { Endpoint } from '../types/spec'
import { Search, X } from 'lucide-react'

interface SearchModalProps {
  isOpen: boolean
  onClose: () => void
  endpoints: Endpoint[]
  onSelectEndpoint: (ep: Endpoint) => void
}

const METHOD_STYLES: Record<string, { bg: string; text: string }> = {
  GET: { bg: 'bg-emerald-950/60', text: 'text-emerald-400' },
  POST: { bg: 'bg-blue-950/60', text: 'text-blue-400' },
  PUT: { bg: 'bg-amber-950/60', text: 'text-amber-400' },
  DELETE: { bg: 'bg-rose-950/60', text: 'text-rose-400' },
  PATCH: { bg: 'bg-purple-950/60', text: 'text-purple-400' },
  OPTIONS: { bg: 'bg-zinc-800', text: 'text-zinc-400' },
  HEAD: { bg: 'bg-zinc-800', text: 'text-zinc-400' },
}

export const SearchModal: React.FC<SearchModalProps> = ({
  isOpen,
  onClose,
  endpoints,
  onSelectEndpoint,
}) => {
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (isOpen) {
      setQuery('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [isOpen])

  const filtered = React.useMemo(() => {
    if (!query.trim()) return endpoints.slice(0, 15)
    const q = query.toLowerCase()
    return endpoints
      .filter((ep) => {
        return (
          ep.path.toLowerCase().includes(q) ||
          ep.method.toLowerCase().includes(q) ||
          (ep.summary || '').toLowerCase().includes(q) ||
          (ep.description || '').toLowerCase().includes(q) ||
          ep.tags?.some((t) => t.toLowerCase().includes(q))
        )
      })
      .slice(0, 20)
  }, [endpoints, query])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return

      if (e.key === 'Escape') {
        onClose()
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIndex((prev) => (prev + 1) % (filtered.length || 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIndex((prev) => (prev - 1 + filtered.length) % (filtered.length || 1))
      } else if (e.key === 'Enter' && filtered[selectedIndex]) {
        e.preventDefault()
        onSelectEndpoint(filtered[selectedIndex])
        onClose()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, filtered, selectedIndex, onSelectEndpoint, onClose])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-start justify-center pt-24 p-4 animate-in fade-in duration-150">
      <div className="w-full max-w-xl bg-[#121215] border border-zinc-800 rounded-xl shadow-2xl overflow-hidden flex flex-col">
        {/* Search Input Bar */}
        <div className="flex items-center px-4 py-3 border-b border-zinc-800/80 gap-3">
          <Search className="w-4 h-4 text-zinc-400 shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setSelectedIndex(0)
            }}
            placeholder="Search API operations by path, tag, or description..."
            className="flex-1 bg-transparent text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none"
          />
          {query && (
            <button onClick={() => setQuery('')} className="text-zinc-500 hover:text-zinc-300">
              <X className="w-4 h-4" />
            </button>
          )}
          <kbd className="text-[10px] font-mono text-zinc-500 bg-zinc-800/80 px-1.5 py-0.5 rounded border border-zinc-700/60">
            ESC
          </kbd>
        </div>

        {/* Results List */}
        <div className="max-h-96 overflow-y-auto p-2 space-y-1">
          {filtered.length === 0 ? (
            <div className="text-center py-8 text-xs text-zinc-500">
              No operations found matching &quot;{query}&quot;
            </div>
          ) : (
            filtered.map((ep, idx) => {
              const isSelected = idx === selectedIndex
              const methodStyle = METHOD_STYLES[ep.method] || METHOD_STYLES.GET

              return (
                <button
                  key={`${ep.method}-${ep.path}-${ep.id}`}
                  onClick={() => {
                    onSelectEndpoint(ep)
                    onClose()
                  }}
                  onMouseEnter={() => setSelectedIndex(idx)}
                  className={`w-full text-left px-3 py-2 rounded-lg text-xs flex items-center justify-between gap-3 transition-colors cursor-pointer ${
                    isSelected ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-300 hover:bg-zinc-800/50'
                  }`}
                >
                  <div className="flex items-center gap-2.5 truncate">
                    <span
                      className={`text-[10px] font-mono font-bold px-1.5 py-0.5 rounded ${methodStyle.bg} ${methodStyle.text}`}
                    >
                      {ep.method}
                    </span>
                    <span className="font-mono text-zinc-200 truncate">{ep.path}</span>
                    {ep.summary && (
                      <span className="text-zinc-500 truncate max-w-xs text-[11px]">
                        — {ep.summary}
                      </span>
                    )}
                  </div>

                  {ep.tags && ep.tags.length > 0 && (
                    <span className="text-[10px] font-mono text-zinc-500 bg-zinc-900 px-1.5 py-0.5 rounded border border-zinc-800 shrink-0">
                      {ep.tags[0]}
                    </span>
                  )}
                </button>
              )
            })
          )}
        </div>

        {/* Footer */}
        <div className="px-4 py-2 border-t border-zinc-800/70 bg-zinc-950/60 text-[11px] text-zinc-500 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span>
              <kbd className="font-mono bg-zinc-800 px-1 rounded text-zinc-400">↑↓</kbd> to navigate
            </span>
            <span>
              <kbd className="font-mono bg-zinc-800 px-1 rounded text-zinc-400">↵</kbd> to select
            </span>
          </div>
          <span>{filtered.length} results</span>
        </div>
      </div>
    </div>
  )
}
