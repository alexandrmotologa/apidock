import React, { useState } from 'react'
import { X, GitCompare, AlertOctagon, CheckCircle2, Info, ArrowRight, Upload, Code } from 'lucide-react'

interface DiffModalProps {
  isOpen: boolean
  onClose: () => void
}

interface Change {
  type: 'BREAKING' | 'NON_BREAKING' | 'INFO'
  category: string
  path: string
  message: string
}

interface DiffReport {
  oldTitle: string
  newTitle: string
  oldVersion: string
  newVersion: string
  breakingCount: number
  nonBreakingCount: number
  changes: Change[]
}

export const DiffModal: React.FC<DiffModalProps> = ({ isOpen, onClose }) => {
  const [newSpecContent, setNewSpecContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [report, setReport] = useState<DiffReport | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<'ALL' | 'BREAKING' | 'NON_BREAKING'>('ALL')

  if (!isOpen) return null

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = (event) => {
      if (typeof event.target?.result === 'string') {
        setNewSpecContent(event.target.result)
      }
    }
    reader.readAsText(file)
  }

  const handleRunDiff = async () => {
    if (!newSpecContent.trim()) {
      setError('Please provide a target specification (paste YAML/JSON, enter URL, or upload file).')
      return
    }

    try {
      setLoading(true)
      setError(null)
      const resp = await fetch('/api/diff', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ newSpec: newSpecContent.trim() }),
      })

      if (!resp.ok) {
        const errorText = await resp.text()
        throw new Error(errorText || `Diff request failed with status ${resp.status}`)
      }

      const data: DiffReport = await resp.json()
      setReport(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to compare specifications')
    } finally {
      setLoading(false)
    }
  }

  const filteredChanges = report?.changes.filter((c) => {
    if (filter === 'BREAKING') return c.type === 'BREAKING'
    if (filter === 'NON_BREAKING') return c.type === 'NON_BREAKING'
    return true
  })

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-4">
      <div className="w-full max-w-2xl bg-[#121215] border border-zinc-800 rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800">
          <div className="flex items-center gap-2">
            <GitCompare className="w-4 h-4 text-indigo-400" />
            <span className="font-semibold text-sm text-zinc-100">Breaking Changes & Spec Diff</span>
          </div>
          <button onClick={onClose} className="text-zinc-500 hover:text-zinc-300">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          {/* Input Section */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="text-xs font-medium text-zinc-300 flex items-center gap-1.5">
                <Code className="w-3.5 h-3.5 text-zinc-400" />
                Target Specification (YAML, JSON, or URL)
              </label>
              <label className="text-xs text-indigo-400 hover:text-indigo-300 cursor-pointer flex items-center gap-1">
                <Upload className="w-3 h-3" />
                <span>Upload File</span>
                <input
                  type="file"
                  accept=".yaml,.yml,.json"
                  onChange={handleFileUpload}
                  className="hidden"
                />
              </label>
            </div>
            <textarea
              value={newSpecContent}
              onChange={(e) => setNewSpecContent(e.target.value)}
              placeholder="Paste updated OpenAPI YAML/JSON or enter local path / remote URL..."
              rows={4}
              className="w-full bg-[#18181b] border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-100 font-mono placeholder:text-zinc-600 focus:outline-hidden focus:border-indigo-500"
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={handleRunDiff}
                disabled={loading || !newSpecContent.trim()}
                className="px-3.5 py-1.5 text-xs font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? 'Analyzing Differences...' : 'Compare Against Current Spec'}
              </button>
            </div>
          </div>

          {error && (
            <div className="p-3 rounded-lg bg-rose-950/40 border border-rose-900/60 text-xs text-rose-300">
              {error}
            </div>
          )}

          {/* Results Section */}
          {report && (
            <div className="space-y-3 pt-2 border-t border-zinc-800">
              {/* Header Stats */}
              <div className="flex items-center justify-between bg-zinc-900/60 border border-zinc-800 p-3 rounded-lg">
                <div className="space-y-0.5">
                  <div className="flex items-center gap-2 text-xs font-medium text-zinc-200">
                    <span>{report.oldTitle} ({report.oldVersion})</span>
                    <ArrowRight className="w-3.5 h-3.5 text-zinc-500" />
                    <span className="text-indigo-300">{report.newTitle} ({report.newVersion})</span>
                  </div>
                  <div className="text-[11px] text-zinc-400">
                    Found {report.changes.length} total difference(s)
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <span
                    className={`px-2 py-0.5 rounded text-[11px] font-semibold ${
                      report.breakingCount > 0
                        ? 'bg-rose-950/80 text-rose-300 border border-rose-800/60'
                        : 'bg-zinc-800 text-zinc-400'
                    }`}
                  >
                    {report.breakingCount} Breaking
                  </span>
                  <span
                    className={`px-2 py-0.5 rounded text-[11px] font-semibold ${
                      report.nonBreakingCount > 0
                        ? 'bg-emerald-950/80 text-emerald-300 border border-emerald-800/60'
                        : 'bg-zinc-800 text-zinc-400'
                    }`}
                  >
                    {report.nonBreakingCount} Additions
                  </span>
                </div>
              </div>

              {/* Filters */}
              <div className="flex items-center gap-1.5">
                {(['ALL', 'BREAKING', 'NON_BREAKING'] as const).map((f) => (
                  <button
                    key={f}
                    onClick={() => setFilter(f)}
                    className={`px-2.5 py-1 rounded-md text-[11px] font-medium transition-colors ${
                      filter === f
                        ? 'bg-zinc-800 text-zinc-100'
                        : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50'
                    }`}
                  >
                    {f === 'ALL' ? 'All Changes' : f === 'BREAKING' ? 'Breaking Only' : 'Additions'}
                  </button>
                ))}
              </div>

              {/* Changes List */}
              <div className="space-y-2 max-h-64 overflow-y-auto pr-1">
                {filteredChanges && filteredChanges.length > 0 ? (
                  filteredChanges.map((change, idx) => (
                    <div
                      key={idx}
                      className={`p-2.5 rounded-lg border text-xs flex items-start gap-2.5 ${
                        change.type === 'BREAKING'
                          ? 'bg-rose-950/20 border-rose-900/40 text-rose-200'
                          : change.type === 'NON_BREAKING'
                          ? 'bg-emerald-950/20 border-emerald-900/40 text-emerald-200'
                          : 'bg-zinc-900 border-zinc-800 text-zinc-300'
                      }`}
                    >
                      <div className="mt-0.5">
                        {change.type === 'BREAKING' ? (
                          <AlertOctagon className="w-3.5 h-3.5 text-rose-400 shrink-0" />
                        ) : change.type === 'NON_BREAKING' ? (
                          <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                        ) : (
                          <Info className="w-3.5 h-3.5 text-sky-400 shrink-0" />
                        )}
                      </div>
                      <div className="flex-1 space-y-0.5 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-[10px] uppercase tracking-wider px-1.5 py-0.2 rounded bg-black/40">
                            {change.category}
                          </span>
                          <span className="font-mono text-[11px] truncate opacity-90">{change.path}</span>
                        </div>
                        <p className="text-[11px] opacity-80 leading-relaxed">{change.message}</p>
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="text-center py-6 text-xs text-zinc-500">
                    No changes match the selected filter.
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
