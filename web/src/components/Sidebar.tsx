import React, { useState } from 'react'
import type { Spec, Endpoint, Environment } from '../types/spec'
import {
  Search,
  Server,
  History,
  CheckCircle2,
  AlertTriangle,
  Radio,
  FileCode2,
  ChevronDown,
  Layers,
} from 'lucide-react'

interface SidebarProps {
  spec: Spec
  selectedEndpoint: Endpoint | null
  onSelectEndpoint: (ep: Endpoint) => void
  onOpenSearch: () => void
  onOpenEnv: () => void
  onOpenHistory: () => void
  onOpenValidation: () => void
  activeEnv: Environment
  mockPort?: number
}

const METHOD_STYLES: Record<string, { bg: string; text: string; border: string }> = {
  GET: { bg: 'bg-emerald-950/40', text: 'text-emerald-400', border: 'border-emerald-800/40' },
  POST: { bg: 'bg-blue-950/40', text: 'text-blue-400', border: 'border-blue-800/40' },
  PUT: { bg: 'bg-amber-950/40', text: 'text-amber-400', border: 'border-amber-800/40' },
  DELETE: { bg: 'bg-rose-950/40', text: 'text-rose-400', border: 'border-rose-800/40' },
  PATCH: { bg: 'bg-purple-950/40', text: 'text-purple-400', border: 'border-purple-800/40' },
  OPTIONS: { bg: 'bg-zinc-800/40', text: 'text-zinc-400', border: 'border-zinc-700/40' },
  HEAD: { bg: 'bg-zinc-800/40', text: 'text-zinc-400', border: 'border-zinc-700/40' },
}

export const Sidebar: React.FC<SidebarProps> = ({
  spec,
  selectedEndpoint,
  onSelectEndpoint,
  onOpenSearch,
  onOpenEnv,
  onOpenHistory,
  onOpenValidation,
  activeEnv,
  mockPort = 4010,
}) => {
  const [filterText, setFilterText] = useState('')
  const [collapsedTags, setCollapsedTags] = useState<Record<string, boolean>>({})

  // Group endpoints by tag
  const groupedEndpoints = React.useMemo(() => {
    const groups: Record<string, Endpoint[]> = {}
    const defaultTag = 'General'

    for (const ep of spec.endpoints) {
      // Filter check
      if (filterText) {
        const query = filterText.toLowerCase()
        const matchPath = ep.path.toLowerCase().includes(query)
        const matchSummary = (ep.summary || '').toLowerCase().includes(query)
        const matchMethod = ep.method.toLowerCase().includes(query)
        if (!matchPath && !matchSummary && !matchMethod) {
          continue
        }
      }

      const tags = ep.tags && ep.tags.length > 0 ? ep.tags : [defaultTag]
      for (const tag of tags) {
        if (!groups[tag]) {
          groups[tag] = []
        }
        groups[tag].push(ep)
      }
    }

    return groups
  }, [spec.endpoints, filterText])

  const toggleTag = (tag: string) => {
    setCollapsedTags((prev) => ({ ...prev, [tag]: !prev[tag] }))
  }

  const hasDiagnostics = spec.errors && spec.errors.length > 0

  return (
    <aside className="w-80 h-screen flex flex-col bg-[#09090b] border-r border-zinc-800/70 select-none">
      {/* Brand & API Info Header */}
      <div className="p-4 border-b border-zinc-800/70">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 rounded bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center font-mono font-bold text-xs text-white shadow-sm shadow-indigo-500/20">
              AD
            </div>
            <span className="font-semibold text-sm tracking-tight text-zinc-100">APIDock</span>
            <span className="text-[10px] text-zinc-500 font-mono bg-zinc-900 px-1.5 py-0.5 rounded border border-zinc-800">
              v{spec.info.version || '1.0'}
            </span>
          </div>

          <div className="flex items-center gap-1.5">
            <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-emerald-950/60 text-emerald-400 border border-emerald-800/40">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
              Offline
            </span>
          </div>
        </div>

        <h1 className="text-xs font-medium text-zinc-300 truncate" title={spec.info.title}>
          {spec.info.title}
        </h1>

        {/* Quick Search Button */}
        <button
          onClick={onOpenSearch}
          className="w-full mt-3 flex items-center justify-between px-2.5 py-1.5 text-xs text-zinc-400 bg-zinc-900/90 hover:bg-zinc-800/90 rounded border border-zinc-800 transition-colors cursor-pointer group"
        >
          <div className="flex items-center gap-2">
            <Search className="w-3.5 h-3.5 text-zinc-500 group-hover:text-zinc-300" />
            <span>Search operations...</span>
          </div>
          <kbd className="text-[10px] font-mono bg-zinc-800 px-1.5 py-0.5 rounded text-zinc-400 border border-zinc-700/60">
            Ctrl+K
          </kbd>
        </button>

        {/* Environment Picker Button */}
        <div className="mt-2.5">
          <button
            onClick={onOpenEnv}
            className="w-full flex items-center justify-between px-2.5 py-1.5 text-xs text-zinc-300 bg-zinc-900/60 hover:bg-zinc-800/60 rounded border border-zinc-800/60 transition-colors"
          >
            <div className="flex items-center gap-2 truncate">
              <Layers className="w-3.5 h-3.5 text-indigo-400 shrink-0" />
              <span className="text-zinc-400 text-[11px]">Env:</span>
              <span className="font-medium text-zinc-200 truncate">{activeEnv.name}</span>
            </div>
            <span className="text-[10px] text-zinc-500 font-mono shrink-0 ml-1">
              {activeEnv.isEncrypted ? '🔒 Vault' : 'Edit'}
            </span>
          </button>
        </div>
      </div>

      {/* Quick Nav Tools */}
      <div className="px-3 py-2 border-b border-zinc-800/50 flex items-center gap-1.5 text-[11px]">
        <button
          onClick={onOpenHistory}
          className="flex-1 flex items-center justify-center gap-1.5 px-2 py-1 bg-zinc-900/50 hover:bg-zinc-800/60 rounded text-zinc-400 hover:text-zinc-200 transition-colors border border-zinc-800/50"
          title="Recent Request History"
        >
          <History className="w-3 h-3 text-zinc-500" />
          <span>History</span>
        </button>

        <button
          onClick={onOpenValidation}
          className="flex-1 flex items-center justify-center gap-1.5 px-2 py-1 bg-zinc-900/50 hover:bg-zinc-800/60 rounded text-zinc-400 hover:text-zinc-200 transition-colors border border-zinc-800/50"
          title="OpenAPI Spec Diagnostics"
        >
          {hasDiagnostics ? (
            <AlertTriangle className="w-3 h-3 text-amber-400" />
          ) : (
            <CheckCircle2 className="w-3 h-3 text-emerald-400" />
          )}
          <span>Spec Check</span>
        </button>

        <a
          href={`http://localhost:${mockPort}`}
          target="_blank"
          rel="noreferrer"
          className="flex items-center justify-center gap-1 px-2 py-1 bg-zinc-900/50 hover:bg-zinc-800/60 rounded text-zinc-400 hover:text-zinc-200 transition-colors border border-zinc-800/50"
          title="Open Mock Server Port"
        >
          <Radio className="w-3 h-3 text-emerald-400" />
          <span className="font-mono text-[10px]">:{mockPort}</span>
        </a>
      </div>

      {/* Filter bar */}
      <div className="px-3 py-2">
        <input
          type="text"
          placeholder="Filter paths..."
          value={filterText}
          onChange={(e) => setFilterText(e.target.value)}
          className="w-full px-2.5 py-1 text-xs bg-zinc-900/70 border border-zinc-800/80 rounded text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-zinc-600 transition-colors"
        />
      </div>

      {/* Endpoints Tree Navigation */}
      <div className="flex-1 overflow-y-auto px-2 py-1 space-y-3">
        {Object.keys(groupedEndpoints).length === 0 ? (
          <div className="text-center py-8 text-zinc-500 text-xs">
            No endpoints match &quot;{filterText}&quot;
          </div>
        ) : (
          Object.entries(groupedEndpoints).map(([tag, endpoints]) => {
            const isCollapsed = !!collapsedTags[tag]
            return (
              <div key={tag} className="space-y-0.5">
                <button
                  onClick={() => toggleTag(tag)}
                  className="w-full flex items-center justify-between px-2 py-1 text-[11px] font-semibold text-zinc-400 hover:text-zinc-200 uppercase tracking-wider transition-colors"
                >
                  <span className="truncate">{tag}</span>
                  <div className="flex items-center gap-1 text-zinc-500 font-normal">
                    <span className="text-[10px] font-mono">{endpoints.length}</span>
                    <ChevronDown
                      className={`w-3 h-3 transition-transform ${isCollapsed ? '-rotate-90' : ''}`}
                    />
                  </div>
                </button>

                {!isCollapsed && (
                  <div className="space-y-0.5 mt-0.5">
                    {endpoints.map((ep) => {
                      const isSelected = selectedEndpoint?.id === ep.id
                      const methodStyle = METHOD_STYLES[ep.method] || METHOD_STYLES.GET

                      return (
                        <button
                          key={`${ep.method}-${ep.path}-${ep.id}`}
                          onClick={() => onSelectEndpoint(ep)}
                          className={`w-full text-left px-2 py-1.5 rounded text-xs flex items-center gap-2 transition-all cursor-pointer ${
                            isSelected
                              ? 'bg-zinc-800/90 text-zinc-100 shadow-sm border border-zinc-700/60'
                              : 'text-zinc-400 hover:bg-zinc-900/70 hover:text-zinc-200 border border-transparent'
                          }`}
                        >
                          <span
                            className={`text-[9px] font-mono font-bold px-1.5 py-0.5 rounded border shrink-0 ${methodStyle.bg} ${methodStyle.text} ${methodStyle.border}`}
                          >
                            {ep.method}
                          </span>
                          <span className="font-mono text-[11px] truncate flex-1" title={ep.path}>
                            {ep.path}
                          </span>
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
            )
          })
        )}
      </div>

      {/* Bottom Status Footer */}
      <div className="p-3 border-t border-zinc-800/70 text-[11px] text-zinc-500 flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Server className="w-3.5 h-3.5 text-zinc-500" />
          <span>{spec.endpoints.length} operations</span>
        </div>
        <a
          href="/api/raw"
          target="_blank"
          rel="noreferrer"
          className="flex items-center gap-1 hover:text-zinc-300 transition-colors"
        >
          <FileCode2 className="w-3 h-3" />
          <span>Raw Spec</span>
        </a>
      </div>
    </aside>
  )
}
