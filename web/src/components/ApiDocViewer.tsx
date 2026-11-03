import React, { useState } from 'react'
import type { Endpoint, Spec, Server } from '../types/spec'
import { SchemaViewer } from './SchemaViewer'
import {
  Copy,
  Check,
  Shield,
  Server as ServerIcon,
  Code2,
  FileJson,
  Layers,
  Sparkles,
  Zap,
} from 'lucide-react'

interface ApiDocViewerProps {
  endpoint: Endpoint | null
  spec: Spec
  selectedServer: string
  onSelectServer: (url: string) => void
  onFillClientBody?: (exampleJSON: string) => void
  onOpenPlayground?: () => void
}

const METHOD_STYLES: Record<string, { bg: string; text: string; border: string }> = {
  GET: { bg: 'bg-emerald-950/50', text: 'text-emerald-400', border: 'border-emerald-800/60' },
  POST: { bg: 'bg-blue-950/50', text: 'text-blue-400', border: 'border-blue-800/60' },
  PUT: { bg: 'bg-amber-950/50', text: 'text-amber-400', border: 'border-amber-800/60' },
  DELETE: { bg: 'bg-rose-950/50', text: 'text-rose-400', border: 'border-rose-800/60' },
  PATCH: { bg: 'bg-purple-950/50', text: 'text-purple-400', border: 'border-purple-800/60' },
  OPTIONS: { bg: 'bg-zinc-800/50', text: 'text-zinc-400', border: 'border-zinc-700/60' },
  HEAD: { bg: 'bg-zinc-800/50', text: 'text-zinc-400', border: 'border-zinc-700/60' },
}

export const ApiDocViewer: React.FC<ApiDocViewerProps> = ({
  endpoint,
  spec,
  selectedServer,
  onSelectServer,
  onFillClientBody,
  onOpenPlayground,
}) => {
  const [copied, setCopied] = useState(false)
  const [activeResponseCode, setActiveResponseCode] = useState<string>('200')
  const [responseViewMode, setResponseViewMode] = useState<'schema' | 'example'>('example')

  // Auto-switch to first response code when endpoint changes
  React.useEffect(() => {
    if (endpoint?.responses) {
      const codes = Object.keys(endpoint.responses)
      if (codes.length > 0 && !codes.includes(activeResponseCode)) {
        setActiveResponseCode(codes[0])
      }
    }
  }, [endpoint, activeResponseCode])

  if (!endpoint) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-8 text-center text-zinc-500">
        <Layers className="w-12 h-12 mb-3 text-zinc-700 animate-pulse" />
        <h2 className="text-sm font-medium text-zinc-300">Select an Operation</h2>
        <p className="text-xs text-zinc-500 mt-1 max-w-sm">
          Browse endpoints in the sidebar or press <kbd className="font-mono text-zinc-400 bg-zinc-800 px-1 py-0.5 rounded">Ctrl+K</kbd> to jump directly to any API route.
        </p>
      </div>
    )
  }

  const methodStyle = METHOD_STYLES[endpoint.method] || METHOD_STYLES.GET

  const copyPath = () => {
    navigator.clipboard.writeText(endpoint.path)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  // Parameters separated by location
  const pathParams = endpoint.parameters?.filter((p) => p.in === 'path') || []
  const queryParams = endpoint.parameters?.filter((p) => p.in === 'query') || []
  const headerParams = endpoint.parameters?.filter((p) => p.in === 'header') || []

  // Active response object
  const activeResponse = endpoint.responses ? endpoint.responses[activeResponseCode] : null
  const activeMediaType = activeResponse?.content ? Object.values(activeResponse.content)[0] : null

  // Extract example for active response
  const activeResponseExample = activeMediaType?.example ||
    (activeMediaType?.examples ? Object.values(activeMediaType.examples)[0]?.value : null) ||
    activeMediaType?.schema?.example

  // Request body schema & example
  const requestBodyMedia = endpoint.requestBody?.content ? Object.values(endpoint.requestBody.content)[0] : null
  const requestBodyExample = requestBodyMedia?.example ||
    (requestBodyMedia?.examples ? Object.values(requestBodyMedia.examples)[0]?.value : null) ||
    requestBodyMedia?.schema?.example

  return (
    <div className="flex-1 overflow-y-auto px-4 sm:px-6 lg:px-8 py-5 sm:py-6 space-y-6 sm:space-y-8 max-w-4xl w-full">
      {/* Endpoint Header */}
      <div>
        <div className="flex items-center justify-between gap-2.5 flex-wrap">
          <div className="flex items-center gap-2.5 flex-wrap">
            <span
              className={`text-xs font-mono font-bold px-2 py-0.5 rounded border ${methodStyle.bg} ${methodStyle.text} ${methodStyle.border}`}
            >
              {endpoint.method}
            </span>
            <span className="font-mono text-sm sm:text-base font-semibold text-zinc-100 break-all">{endpoint.path}</span>
            <button
              onClick={copyPath}
              className="text-zinc-500 hover:text-zinc-300 p-1 rounded hover:bg-zinc-800/60 transition-colors"
              title="Copy path"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
            </button>
            {endpoint.deprecated && (
              <span className="text-[10px] font-semibold text-amber-400 bg-amber-950/40 px-2 py-0.5 rounded border border-amber-800/40">
                DEPRECATED
              </span>
            )}
          </div>

          {onOpenPlayground && (
            <button
              onClick={onOpenPlayground}
              className="lg:hidden flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded-lg bg-indigo-600/20 text-indigo-300 border border-indigo-500/30 hover:bg-indigo-600/30 transition-colors cursor-pointer"
            >
              <Zap className="w-3.5 h-3.5 text-indigo-400" />
              <span>Try in Playground</span>
            </button>
          )}
        </div>

        {endpoint.summary && (
          <h1 className="text-xl font-semibold text-zinc-100 mt-2">{endpoint.summary}</h1>
        )}

        {endpoint.description && (
          <p className="text-sm text-zinc-400 mt-2 leading-relaxed whitespace-pre-line">
            {endpoint.description}
          </p>
        )}
      </div>

      {/* Server & Security Bar */}
      <div className="flex items-center gap-4 flex-wrap p-3 rounded-lg bg-zinc-900/60 border border-zinc-800/70 text-xs">
        {/* Server Selector */}
        <div className="flex items-center gap-2">
          <ServerIcon className="w-3.5 h-3.5 text-zinc-500 shrink-0" />
          <span className="text-zinc-400">Server:</span>
          <select
            value={selectedServer}
            onChange={(e) => onSelectServer(e.target.value)}
            className="bg-zinc-800/80 border border-zinc-700/60 text-zinc-200 rounded px-2 py-1 text-xs focus:outline-none focus:border-zinc-500 font-mono"
          >
            {spec.servers.length === 0 ? (
              <option value="http://localhost:4010">http://localhost:4010 (Mock Server)</option>
            ) : (
              spec.servers.map((srv: Server, idx: number) => (
                <option key={idx} value={srv.url}>
                  {srv.url} {srv.description ? `(${srv.description})` : ''}
                </option>
              ))
            )}
          </select>
        </div>

        {/* Security badges */}
        {endpoint.security && endpoint.security.length > 0 && (
          <div className="flex items-center gap-1.5 ml-auto">
            <Shield className="w-3.5 h-3.5 text-indigo-400 shrink-0" />
            <span className="text-zinc-400">Auth:</span>
            {endpoint.security.map((sec, idx) => {
              const schemeName = Object.keys(sec)[0]
              return (
                <span
                  key={idx}
                  className="px-1.5 py-0.5 rounded bg-indigo-950/40 text-indigo-300 border border-indigo-800/40 font-mono text-[11px]"
                >
                  {schemeName}
                </span>
              )
            })}
          </div>
        )}
      </div>

      {/* Path Parameters */}
      {pathParams.length > 0 && (
        <section className="space-y-3">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-zinc-400 flex items-center gap-1.5">
            <span>Path Parameters</span>
            <span className="text-[10px] font-mono text-zinc-500">({pathParams.length})</span>
          </h2>
          <div className="rounded-lg border border-zinc-800 overflow-hidden">
            <table className="w-full text-left text-xs">
              <thead className="bg-zinc-900/80 text-zinc-400 font-medium border-b border-zinc-800">
                <tr>
                  <th className="py-2 px-3 font-semibold">Parameter</th>
                  <th className="py-2 px-3 font-semibold">Type</th>
                  <th className="py-2 px-3 font-semibold">Description</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/60 bg-zinc-950/50">
                {pathParams.map((p) => (
                  <tr key={p.name} className="hover:bg-zinc-900/40">
                    <td className="py-2.5 px-3 font-mono font-semibold text-zinc-200">
                      {p.name}
                      <span className="text-rose-400 ml-1 font-bold">*</span>
                    </td>
                    <td className="py-2.5 px-3 font-mono text-[11px] text-emerald-400">
                      {p.schema?.type || 'string'}
                      {p.schema?.format && (
                        <span className="text-zinc-500 font-normal"> ({p.schema.format})</span>
                      )}
                    </td>
                    <td className="py-2.5 px-3 text-zinc-300">
                      <div>{p.description || <span className="text-zinc-600 italic">No description</span>}</div>
                      {p.example !== undefined && (
                        <div className="text-[11px] text-zinc-500 font-mono mt-0.5">
                          Example: <span className="text-zinc-300">{JSON.stringify(p.example)}</span>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {/* Query Parameters */}
      {queryParams.length > 0 && (
        <section className="space-y-3">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-zinc-400 flex items-center gap-1.5">
            <span>Query Parameters</span>
            <span className="text-[10px] font-mono text-zinc-500">({queryParams.length})</span>
          </h2>
          <div className="rounded-lg border border-zinc-800 overflow-hidden">
            <table className="w-full text-left text-xs">
              <thead className="bg-zinc-900/80 text-zinc-400 font-medium border-b border-zinc-800">
                <tr>
                  <th className="py-2 px-3 font-semibold">Parameter</th>
                  <th className="py-2 px-3 font-semibold">Type</th>
                  <th className="py-2 px-3 font-semibold">Description</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/60 bg-zinc-950/50">
                {queryParams.map((p) => (
                  <tr key={p.name} className="hover:bg-zinc-900/40">
                    <td className="py-2.5 px-3 font-mono font-semibold text-zinc-200">
                      {p.name}
                      {p.required && <span className="text-rose-400 ml-1 font-bold">*</span>}
                    </td>
                    <td className="py-2.5 px-3 font-mono text-[11px] text-emerald-400">
                      {p.schema?.type || 'string'}
                      {p.schema?.enum && (
                        <div className="text-[10px] text-amber-400/90 font-mono mt-0.5">
                          enum: [{p.schema.enum.join(', ')}]
                        </div>
                      )}
                    </td>
                    <td className="py-2.5 px-3 text-zinc-300">
                      <div>{p.description || <span className="text-zinc-600 italic">No description</span>}</div>
                      {p.schema?.default !== undefined && (
                        <div className="text-[11px] text-zinc-500 font-mono mt-0.5">
                          Default: <span className="text-zinc-300">{JSON.stringify(p.schema.default)}</span>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {/* Header Parameters */}
      {headerParams.length > 0 && (
        <section className="space-y-3">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-zinc-400 flex items-center gap-1.5">
            <span>Header Parameters</span>
            <span className="text-[10px] font-mono text-zinc-500">({headerParams.length})</span>
          </h2>
          <div className="rounded-lg border border-zinc-800 overflow-hidden">
            <table className="w-full text-left text-xs">
              <thead className="bg-zinc-900/80 text-zinc-400 font-medium border-b border-zinc-800">
                <tr>
                  <th className="py-2 px-3 font-semibold">Header</th>
                  <th className="py-2 px-3 font-semibold">Type</th>
                  <th className="py-2 px-3 font-semibold">Description</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/60 bg-zinc-950/50">
                {headerParams.map((p) => (
                  <tr key={p.name} className="hover:bg-zinc-900/40">
                    <td className="py-2.5 px-3 font-mono font-semibold text-zinc-200">
                      {p.name}
                      {p.required && <span className="text-rose-400 ml-1 font-bold">*</span>}
                    </td>
                    <td className="py-2.5 px-3 font-mono text-[11px] text-emerald-400">
                      {p.schema?.type || 'string'}
                    </td>
                    <td className="py-2.5 px-3 text-zinc-300">{p.description}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {/* Request Body Section */}
      {endpoint.requestBody && (
        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-zinc-400 flex items-center gap-1.5">
              <span>Request Body</span>
              {endpoint.requestBody.required && (
                <span className="text-rose-400 text-[10px] font-mono lowercase">(required)</span>
              )}
            </h2>

            {requestBodyExample && onFillClientBody && (
              <button
                onClick={() => onFillClientBody(JSON.stringify(requestBodyExample, null, 2))}
                className="flex items-center gap-1.5 px-2 py-1 text-[11px] text-indigo-300 hover:text-indigo-200 bg-indigo-950/40 hover:bg-indigo-900/50 rounded border border-indigo-800/50 transition-colors"
                title="Fill Interactive Client Body with this sample"
              >
                <Sparkles className="w-3 h-3 text-indigo-400" />
                <span>Fill in Playground</span>
              </button>
            )}
          </div>

          <div className="p-3.5 rounded-lg border border-zinc-800 bg-zinc-950/40">
            {endpoint.requestBody.description && (
              <p className="text-xs text-zinc-400 mb-3">{endpoint.requestBody.description}</p>
            )}

            {requestBodyMedia?.schema ? (
              <SchemaViewer schema={requestBodyMedia.schema} />
            ) : (
              <span className="text-xs text-zinc-500 italic">No schema declared</span>
            )}
          </div>
        </section>
      )}

      {/* Responses Section */}
      {endpoint.responses && (
        <section className="space-y-3">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-zinc-400">Responses</h2>

          {/* Response Code Tabs */}
          <div className="flex items-center gap-1.5 border-b border-zinc-800 pb-1.5">
            {Object.entries(endpoint.responses).map(([code, resp]) => {
              const isActive = activeResponseCode === code
              const is2xx = code.startsWith('2')
              const is4xx = code.startsWith('4')
              const is5xx = code.startsWith('5')

              let badgeColor = 'text-zinc-400 border-zinc-800 bg-zinc-900/50'
              if (is2xx) badgeColor = 'text-emerald-400 border-emerald-800/60 bg-emerald-950/30'
              if (is4xx) badgeColor = 'text-amber-400 border-amber-800/60 bg-amber-950/30'
              if (is5xx) badgeColor = 'text-rose-400 border-rose-800/60 bg-rose-950/30'

              return (
                <button
                  key={code}
                  onClick={() => setActiveResponseCode(code)}
                  className={`px-2.5 py-1 rounded text-xs font-mono transition-all flex items-center gap-1.5 cursor-pointer ${
                    isActive
                      ? `${badgeColor} font-semibold ring-1 ring-zinc-600`
                      : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-900/60'
                  }`}
                >
                  <span>{code}</span>
                  <span className="text-[10px] font-sans font-normal text-zinc-400 truncate max-w-[120px]">
                    {resp.description || 'Response'}
                  </span>
                </button>
              )
            })}
          </div>

          {/* Active Response Body */}
          {activeResponse && (
            <div className="p-3.5 rounded-lg border border-zinc-800 bg-zinc-950/40 space-y-3">
              <div className="flex items-center justify-between text-xs">
                <span className="text-zinc-300 font-medium">{activeResponse.description}</span>

                <div className="flex items-center gap-1 bg-zinc-900 p-0.5 rounded border border-zinc-800">
                  <button
                    onClick={() => setResponseViewMode('example')}
                    className={`flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium transition-colors ${
                      responseViewMode === 'example'
                        ? 'bg-zinc-800 text-zinc-100'
                        : 'text-zinc-400 hover:text-zinc-200'
                    }`}
                  >
                    <FileJson className="w-3 h-3" />
                    <span>Example</span>
                  </button>
                  <button
                    onClick={() => setResponseViewMode('schema')}
                    className={`flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium transition-colors ${
                      responseViewMode === 'schema'
                        ? 'bg-zinc-800 text-zinc-100'
                        : 'text-zinc-400 hover:text-zinc-200'
                    }`}
                  >
                    <Code2 className="w-3 h-3" />
                    <span>Schema</span>
                  </button>
                </div>
              </div>

              {responseViewMode === 'schema' ? (
                activeMediaType?.schema ? (
                  <SchemaViewer schema={activeMediaType.schema} />
                ) : (
                  <div className="text-xs text-zinc-500 italic">No schema defined for this response code</div>
                )
              ) : activeResponseExample !== null && activeResponseExample !== undefined ? (
                <pre className="text-xs font-mono text-zinc-300 bg-zinc-900/80 p-3 rounded border border-zinc-800/80 overflow-x-auto max-h-60 leading-relaxed">
                  {JSON.stringify(activeResponseExample, null, 2)}
                </pre>
              ) : activeMediaType?.schema ? (
                <div>
                  <div className="text-[11px] text-zinc-500 mb-1.5 italic">
                    Schema structure (No explicit example defined in specification):
                  </div>
                  <SchemaViewer schema={activeMediaType.schema} />
                </div>
              ) : (
                <div className="text-xs text-zinc-500 italic">No example payload provided</div>
              )}
            </div>
          )}
        </section>
      )}
    </div>
  )
}
