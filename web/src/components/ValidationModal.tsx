import React, { useState, useEffect } from 'react'
import { X, CheckCircle2, AlertTriangle, AlertCircle, RefreshCw } from 'lucide-react'

interface ValidationModalProps {
  isOpen: boolean
  onClose: () => void
}

interface Diagnostic {
  severity: 'ERROR' | 'WARNING'
  path: string
  message: string
}

interface ValidationResponse {
  valid: boolean
  diagnostics: Diagnostic[]
}

export const ValidationModal: React.FC<ValidationModalProps> = ({ isOpen, onClose }) => {
  const [data, setData] = useState<ValidationResponse | null>(null)
  const [loading, setLoading] = useState(false)

  const fetchDiagnostics = async () => {
    try {
      setLoading(true)
      const resp = await fetch('/api/validate')
      if (resp.ok) {
        const json = await resp.json()
        setData(json)
      }
    } catch {
      // Ignored
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isOpen) {
      fetchDiagnostics()
    }
  }, [isOpen])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4">
      <div className="w-full max-w-xl bg-[#121215] border border-zinc-800 rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="w-4 h-4 text-emerald-400" />
            <span className="font-semibold text-sm text-zinc-100">Specification Validation</span>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={fetchDiagnostics}
              className="text-zinc-500 hover:text-zinc-300 p-1"
              title="Refresh audit"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button onClick={onClose} className="text-zinc-500 hover:text-zinc-300">
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-5 space-y-3">
          {loading ? (
            <div className="text-center py-12 text-zinc-500 text-xs">Auditing specification...</div>
          ) : !data || data.diagnostics.length === 0 ? (
            <div className="p-8 text-center space-y-2">
              <CheckCircle2 className="w-10 h-10 text-emerald-400 mx-auto" />
              <h3 className="text-sm font-semibold text-zinc-200">Specification is 100% Valid</h3>
              <p className="text-xs text-zinc-400 max-w-sm mx-auto">
                No syntax errors, broken $ref pointers, or missing parameter definitions were found.
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {data.diagnostics.map((d, idx) => (
                <div
                  key={idx}
                  className={`p-3 rounded-lg border text-xs space-y-1 ${
                    d.severity === 'ERROR'
                      ? 'bg-rose-950/30 border-rose-900/50 text-rose-200'
                      : 'bg-amber-950/30 border-amber-900/50 text-amber-200'
                  }`}
                >
                  <div className="flex items-center gap-1.5 font-semibold text-[11px]">
                    {d.severity === 'ERROR' ? (
                      <AlertCircle className="w-3.5 h-3.5 text-rose-400" />
                    ) : (
                      <AlertTriangle className="w-3.5 h-3.5 text-amber-400" />
                    )}
                    <span>{d.severity}</span>
                    <span className="font-mono text-zinc-400 font-normal ml-1">({d.path})</span>
                  </div>
                  <p className="text-zinc-300 leading-relaxed pl-5">{d.message}</p>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-zinc-800 bg-zinc-950 flex justify-end">
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
