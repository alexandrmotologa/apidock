import React, { useState, useEffect, useMemo, useRef } from 'react'
import Editor from '@monaco-editor/react'
import type { Endpoint, Environment, ProxyResponse } from '../types/spec'
import { interpolate } from '../utils/interpolation'
import { parseCurl } from '../utils/curlParser'
import { validateContract, type ContractValidationResult } from '../utils/contractValidator'
import {
  Play,
  Copy,
  Check,
  Zap,
  Globe,
  Clock,
  HardDrive,
  FileCode,
  Sliders,
  CheckCircle2,
  AlertTriangle,
  X,
  Radio,
  Square,
  Activity,
} from 'lucide-react'

interface InteractiveClientProps {
  endpoint: Endpoint | null
  activeEnv: Environment
  activeServer: string
  initialBody?: string
  onAddToHistory?: (resp: ProxyResponse) => void
}

type TabType = 'params' | 'headers' | 'auth' | 'body'
type ResponseTabType = 'body' | 'headers' | 'snippets' | 'contract'

export const InteractiveClient: React.FC<InteractiveClientProps> = ({
  endpoint,
  activeEnv,
  activeServer,
  initialBody = '',
}) => {
  const [activeTab, setActiveTab] = useState<TabType>('params')
  const [activeResponseTab, setActiveResponseTab] = useState<ResponseTabType>('body')
  const [isProxyMode, setIsProxyMode] = useState<boolean>(true)

  // Param states
  const [pathParamValues, setPathParamValues] = useState<Record<string, string>>({})
  const [queryParams, setQueryParams] = useState<Array<{ key: string; value: string; enabled: boolean }>>([])
  const [headers, setHeaders] = useState<Array<{ key: string; value: string; enabled: boolean }>>([])
  const [requestBody, setRequestBody] = useState<string>(initialBody)

  // Auth states
  const [authType, setAuthType] = useState<'none' | 'bearer' | 'apiKey' | 'basic'>('none')
  const [bearerToken, setBearerToken] = useState<string>('{{token}}')
  const [apiKeyHeader, setApiKeyHeader] = useState<string>('X-API-KEY')
  const [apiKeyValue, setApiKeyValue] = useState<string>('{{apiKey}}')
  const [basicUser, setBasicUser] = useState<string>('admin')
  const [basicPass, setBasicPass] = useState<string>('')

  // Mock control states
  const [mockStatusOverride, setMockStatusOverride] = useState<string>('')
  const [mockExampleOverride, setMockExampleOverride] = useState<string>('')
  const [mockLatency, setMockLatency] = useState<string>('')
  const [showMockToolbar, setShowMockToolbar] = useState<boolean>(false)

  // cURL Import modal state
  const [isCurlModalOpen, setIsCurlModalOpen] = useState<boolean>(false)
  const [curlCommandInput, setCurlCommandInput] = useState<string>('')

  // Contract validation state
  const [contractResult, setContractResult] = useState<ContractValidationResult | null>(null)

  // Snippets language selection
  const [snippetLang, setSnippetLang] = useState<'curl' | 'fetch' | 'axios' | 'pythonReq' | 'pythonHttpx' | 'go'>('curl')

  // Execution states
  const [isLoading, setIsLoading] = useState<boolean>(false)
  const [isStreamMode, setIsStreamMode] = useState<boolean>(false)
  const [isStreaming, setIsStreaming] = useState<boolean>(false)
  const [streamChunksCount, setStreamChunksCount] = useState<number>(0)
  const abortControllerRef = useRef<AbortController | null>(null)
  const [response, setResponse] = useState<ProxyResponse | null>(null)
  const [copiedResponse, setCopiedResponse] = useState<boolean>(false)
  const [copiedSnippet, setCopiedSnippet] = useState<boolean>(false)

  const handleStopStream = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
      abortControllerRef.current = null
    }
    setIsStreaming(false)
    setIsLoading(false)
  }

  // Available named examples in active endpoint
  const availableExamples = useMemo(() => {
    if (!endpoint?.responses) return []
    const exList: string[] = []
    for (const resp of Object.values(endpoint.responses)) {
      if (resp.content) {
        for (const mt of Object.values(resp.content)) {
          if (mt.examples) {
            for (const exKey of Object.keys(mt.examples)) {
              if (!exList.includes(exKey)) exList.push(exKey)
            }
          }
        }
      }
    }
    return exList
  }, [endpoint])

  // Update initial body when prop changes
  useEffect(() => {
    if (initialBody) {
      setRequestBody(initialBody)
      setActiveTab('body')
    }
  }, [initialBody])

  // Initialize parameters when endpoint changes
  useEffect(() => {
    if (!endpoint) return

    // Path params
    const initialPaths: Record<string, string> = {}
    endpoint.parameters?.filter((p) => p.in === 'path').forEach((p) => {
      initialPaths[p.name] = p.example !== undefined ? String(p.example) : (p.schema?.default ? String(p.schema.default) : '')
    })
    setPathParamValues(initialPaths)

    // Query params
    const initialQueries = endpoint.parameters?.filter((p) => p.in === 'query').map((p) => ({
      key: p.name,
      value: p.example !== undefined ? String(p.example) : (p.schema?.default !== undefined ? String(p.schema.default) : ''),
      enabled: true,
    })) || []
    setQueryParams(initialQueries)

    // Default headers
    const initialHeaders = [
      { key: 'Content-Type', value: 'application/json', enabled: true },
    ]
    endpoint.parameters?.filter((p) => p.in === 'header').forEach((p) => {
      initialHeaders.push({
        key: p.name,
        value: p.example !== undefined ? String(p.example) : '',
        enabled: true,
      })
    })
    setHeaders(initialHeaders)

    // Request body default
    if (endpoint.requestBody) {
      const media = Object.values(endpoint.requestBody.content)[0]
      const sample = media?.example || (media?.examples ? Object.values(media.examples)[0]?.value : null) || media?.schema?.example
      if (sample) {
        setRequestBody(JSON.stringify(sample, null, 2))
      } else {
        setRequestBody('{\n  \n}')
      }
    } else {
      setRequestBody('')
    }

    // Default Auth detection
    if (endpoint.security && endpoint.security.length > 0) {
      const firstSec = Object.keys(endpoint.security[0])[0]
      if (firstSec.toLowerCase().includes('bearer') || firstSec.toLowerCase().includes('jwt')) {
        setAuthType('bearer')
      } else if (firstSec.toLowerCase().includes('key') || firstSec.toLowerCase().includes('api')) {
        setAuthType('apiKey')
      }
    } else {
      setAuthType('none')
    }

    setResponse(null)
    setContractResult(null)
    setMockStatusOverride('')
    setMockExampleOverride('')
  }, [endpoint])

  // Construct target URL with interpolated path & query parameters
  const computedUrl = useMemo(() => {
    if (!endpoint) return ''
    let baseUrl = activeServer || activeEnv.baseUrl || 'http://localhost:4010'
    baseUrl = interpolate(baseUrl, activeEnv.variables)

    let path = endpoint.path
    for (const [k, v] of Object.entries(pathParamValues)) {
      const cleanVal = interpolate(v, activeEnv.variables)
      path = path.replace(new RegExp(`\\{${k}\\}`, 'g'), encodeURIComponent(cleanVal || `:${k}`))
    }
    path = interpolate(path, activeEnv.variables)

    const queryParts: string[] = []
    for (const q of queryParams) {
      if (q.enabled && q.key.trim() !== '') {
        const val = interpolate(q.value, activeEnv.variables)
        queryParts.push(`${encodeURIComponent(q.key)}=${encodeURIComponent(val)}`)
      }
    }

    if (authType === 'apiKey' && apiKeyHeader.toLowerCase() === 'query') {
      const keyVal = interpolate(apiKeyValue, activeEnv.variables)
      queryParts.push(`apiKey=${encodeURIComponent(keyVal)}`)
    }

    const queryString = queryParts.length > 0 ? `?${queryParts.join('&')}` : ''
    const cleanBase = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl
    const cleanPath = path.startsWith('/') ? path : `/${path}`

    return `${cleanBase}${cleanPath}${queryString}`
  }, [endpoint, activeServer, activeEnv, pathParamValues, queryParams, authType, apiKeyHeader, apiKeyValue])

  // Computed Headers for execution (including Mock Server Directives)
  const computedHeaders = useMemo(() => {
    const res: Record<string, string> = {}
    for (const h of headers) {
      if (h.enabled && h.key.trim() !== '') {
        res[h.key] = interpolate(h.value, activeEnv.variables)
      }
    }

    // Apply Auth
    if (authType === 'bearer') {
      const token = interpolate(bearerToken, activeEnv.variables)
      res['Authorization'] = `Bearer ${token}`
    } else if (authType === 'apiKey' && apiKeyHeader.toLowerCase() !== 'query') {
      const keyVal = interpolate(apiKeyValue, activeEnv.variables)
      res[apiKeyHeader] = keyVal
    } else if (authType === 'basic') {
      const creds = btoa(`${basicUser}:${basicPass}`)
      res['Authorization'] = `Basic ${creds}`
    }

    // Apply Mock Server Directives
    const preferDirectives: string[] = []
    if (mockStatusOverride) {
      preferDirectives.push(`code=${mockStatusOverride}`)
    }
    if (mockExampleOverride) {
      preferDirectives.push(`example=${mockExampleOverride}`)
    }
    if (preferDirectives.length > 0) {
      res['Prefer'] = preferDirectives.join(', ')
    }

    if (mockLatency) {
      res['X-Mock-Delay'] = mockLatency
    }

    return res
  }, [
    headers,
    authType,
    bearerToken,
    apiKeyHeader,
    apiKeyValue,
    basicUser,
    basicPass,
    activeEnv.variables,
    mockStatusOverride,
    mockExampleOverride,
    mockLatency,
  ])

  // Execute request
  const handleSend = async () => {
    if (!endpoint || isLoading) return

    setIsLoading(true)
    const startTime = performance.now()

    try {
      if (isStreamMode) {
        setIsStreaming(true)
        setStreamChunksCount(0)
        setResponse({
          status: 200,
          statusText: 'Connecting stream...',
          headers: {},
          body: '',
          timing: {
            dnsLookupMs: 0,
            tcpConnectMs: 0,
            tlsHandshakeMs: 0,
            ttfbMs: 0,
            totalMs: 0,
            sizeBytes: 0,
          },
        })

        const controller = new AbortController()
        abortControllerRef.current = controller

        const proxyPayload = {
          url: computedUrl,
          method: endpoint.method,
          headers: computedHeaders,
          body: ['GET', 'HEAD'].includes(endpoint.method) ? '' : requestBody,
          stream: true,
        }

        const resp = await fetch('/api/proxy?stream=true', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(proxyPayload),
          signal: controller.signal,
        })

        if (!resp.body) {
          throw new Error('Response body stream is not available')
        }

        const reader = resp.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        let accumulatedBody = ''
        let streamHeaders: Record<string, string> = {}
        let streamStatus = 200
        let streamStatusText = 'OK'

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split('\n')
          buffer = lines.pop() || ''

          let currentEvent = ''
          for (const line of lines) {
            if (line.startsWith('event: ')) {
              currentEvent = line.slice(7).trim()
            } else if (line.startsWith('data: ')) {
              const dataStr = line.slice(6).trim()
              try {
                const dataJson = JSON.parse(dataStr)
                if (currentEvent === 'meta') {
                  streamStatus = dataJson.status || 200
                  streamStatusText = dataJson.statusText || 'OK'
                  streamHeaders = dataJson.headers || {}
                } else if (currentEvent === 'chunk') {
                  accumulatedBody += dataJson.chunk
                  setStreamChunksCount((c) => c + 1)
                  const elapsed = performance.now() - startTime
                  setResponse({
                    status: streamStatus,
                    statusText: streamStatusText,
                    headers: streamHeaders,
                    body: accumulatedBody,
                    timing: {
                      dnsLookupMs: 0,
                      tcpConnectMs: 0,
                      tlsHandshakeMs: 0,
                      ttfbMs: 0,
                      totalMs: Math.round(elapsed * 10) / 10,
                      sizeBytes: accumulatedBody.length,
                    },
                  })
                } else if (currentEvent === 'error') {
                  throw new Error(dataJson.error)
                }
              } catch {
                // Ignore raw line parse error
              }
            }
          }
        }

        setIsStreaming(false)
        setIsLoading(false)
        return
      }

      let finalResp: ProxyResponse

      if (isProxyMode) {
        // Route through Go CORS Proxy
        const proxyPayload = {
          url: computedUrl,
          method: endpoint.method,
          headers: computedHeaders,
          body: ['GET', 'HEAD'].includes(endpoint.method) ? '' : requestBody,
          timeoutMs: 30000,
        }

        const resp = await fetch('/api/proxy', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(proxyPayload),
        })

        finalResp = await resp.json()
      } else {
        // Direct browser fetch
        const resp = await fetch(computedUrl, {
          method: endpoint.method,
          headers: computedHeaders,
          body: ['GET', 'HEAD'].includes(endpoint.method) ? undefined : requestBody,
        })

        const totalTime = performance.now() - startTime
        const text = await resp.text()

        const respHeaders: Record<string, string> = {}
        resp.headers.forEach((v, k) => {
          respHeaders[k] = v
        })

        finalResp = {
          status: resp.status,
          statusText: resp.statusText,
          headers: respHeaders,
          body: text,
          timing: {
            dnsLookupMs: 0,
            tcpConnectMs: 0,
            tlsHandshakeMs: 0,
            ttfbMs: Math.round(totalTime * 0.8 * 10) / 10,
            totalMs: Math.round(totalTime * 10) / 10,
            sizeBytes: text.length,
          },
        }
      }

      setResponse(finalResp)

      // Run Response Contract Validation
      if (finalResp.body) {
        try {
          const parsedJSON = JSON.parse(finalResp.body)
          const statusCodeStr = String(finalResp.status)
          const respDef =
            endpoint.responses?.[statusCodeStr] ||
            endpoint.responses?.['default']
          const mediaDef = respDef?.content
            ? Object.values(respDef.content)[0]
            : undefined

          if (mediaDef?.schema) {
            const vResult = validateContract(parsedJSON, mediaDef.schema)
            setContractResult(vResult)
          } else {
            setContractResult(null)
          }
        } catch {
          setContractResult(null)
        }
      }
    } catch (err) {
      const totalTime = performance.now() - startTime
      setResponse({
        status: 0,
        statusText: 'Client Error',
        headers: {},
        body: '',
        error: err instanceof Error ? err.message : 'Network execution failed',
        timing: {
          dnsLookupMs: 0,
          tcpConnectMs: 0,
          tlsHandshakeMs: 0,
          ttfbMs: 0,
          totalMs: Math.round(totalTime * 10) / 10,
          sizeBytes: 0,
        },
      })
      setContractResult(null)
    } finally {
      setIsLoading(false)
    }
  }

  // Handle Ctrl+Enter shortcut
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault()
        handleSend()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleSend])

  // Handle cURL Import
  const handleImportCurl = () => {
    if (!curlCommandInput.trim()) return
    const parsed = parseCurl(curlCommandInput)

    // Apply headers
    if (Object.keys(parsed.headers).length > 0) {
      const newHeaders = Object.entries(parsed.headers).map(([k, v]) => ({
        key: k,
        value: v,
        enabled: true,
      }))
      setHeaders(newHeaders)
    }

    // Apply body
    if (parsed.body) {
      try {
        const js = JSON.parse(parsed.body)
        setRequestBody(JSON.stringify(js, null, 2))
      } catch {
        setRequestBody(parsed.body)
      }
      setActiveTab('body')
    }

    setIsCurlModalOpen(false)
    setCurlCommandInput('')
  }

  const copyResponse = () => {
    if (!response) return
    navigator.clipboard.writeText(response.body)
    setCopiedResponse(true)
    setTimeout(() => setCopiedResponse(false), 1500)
  }

  const prettifyBody = () => {
    try {
      const parsed = JSON.parse(requestBody)
      setRequestBody(JSON.stringify(parsed, null, 2))
    } catch {
      // Ignored
    }
  }

  // Code snippets generation
  const activeSnippetCode = useMemo(() => {
    const method = endpoint?.method || 'GET'
    const headersList = computedHeaders
    const bodyContent = ['GET', 'HEAD'].includes(method) ? '' : requestBody

    switch (snippetLang) {
      case 'curl': {
        const headerFlags = Object.entries(headersList)
          .map(([k, v]) => `  -H "${k}: ${v}"`)
          .join(' \\\n')
        const bodyFlag = bodyContent ? ` \\\n  -d '${bodyContent.replace(/'/g, "'\\''")}'` : ''
        return `curl -X ${method} "${computedUrl}"${headerFlags ? ` \\\n${headerFlags}` : ''}${bodyFlag}`
      }
      case 'fetch': {
        return `const url = "${computedUrl}";\nconst response = await fetch(url, {\n  method: "${method}",\n  headers: ${JSON.stringify(headersList, null, 4)},\n${bodyContent ? `  body: JSON.stringify(${bodyContent}),\n` : ''}});\nconst data = await response.json();\nconsole.log(data);`
      }
      case 'axios': {
        return `import axios from 'axios';\n\nconst response = await axios({\n  method: "${method.toLowerCase()}",\n  url: "${computedUrl}",\n  headers: ${JSON.stringify(headersList, null, 4)},\n${bodyContent ? `  data: ${bodyContent},\n` : ''}});\nconsole.log(response.data);`
      }
      case 'pythonReq': {
        return `import requests\n\nurl = "${computedUrl}"\nheaders = ${JSON.stringify(headersList, null, 4)}\n${bodyContent ? `payload = ${bodyContent}\nresponse = requests.${method.toLowerCase()}(url, headers=headers, json=payload)\n` : `response = requests.${method.toLowerCase()}(url, headers=headers)\n`}print(response.status_code)\nprint(response.json())`
      }
      case 'pythonHttpx': {
        return `import httpx\n\nurl = "${computedUrl}"\nheaders = ${JSON.stringify(headersList, null, 4)}\n${bodyContent ? `payload = ${bodyContent}\nwith httpx.Client() as client:\n    response = client.${method.toLowerCase()}(url, headers=headers, json=payload)\n` : `with httpx.Client() as client:\n    response = client.${method.toLowerCase()}(url, headers=headers)\n`}print(response.status_code)\nprint(response.json())`
      }
      case 'go': {
        return `package main\n\nimport (\n\t"fmt"\n\t"io"\n\t"net/http"\n\t"strings"\n)\n\nfunc main() {\n\turl := "${computedUrl}"\n\treq, _ := http.NewRequest("${method}", url, ${bodyContent ? `strings.NewReader(\`${bodyContent}\`)` : 'nil'})\n${Object.entries(headersList).map(([k, v]) => `\treq.Header.Set("${k}", "${v}")`).join('\n')}\n\tresp, _ := http.DefaultClient.Do(req)\n\tdefer resp.Body.Close()\n\tbody, _ := io.ReadAll(resp.Body)\n\tfmt.Println(resp.Status)\n\tfmt.Println(string(body))\n}`
      }
    }
  }, [snippetLang, endpoint, computedUrl, computedHeaders, requestBody])

  const copySnippet = () => {
    navigator.clipboard.writeText(activeSnippetCode)
    setCopiedSnippet(true)
    setTimeout(() => setCopiedSnippet(false), 1500)
  }

  if (!endpoint) {
    return (
      <div className="w-full lg:w-[460px] xl:w-[500px] h-full lg:h-screen bg-[#09090b] border-t lg:border-t-0 lg:border-l border-zinc-800/70 p-6 flex flex-col items-center justify-center text-center text-zinc-500">
        <Zap className="w-10 h-10 mb-2 text-zinc-700" />
        <span className="text-xs">Interactive Playground</span>
      </div>
    )
  }

  return (
    <div className="w-full lg:w-[460px] xl:w-[500px] h-full lg:h-screen bg-[#09090b] border-t lg:border-t-0 lg:border-l border-zinc-800/70 flex flex-col select-none">
      {/* Client Header */}
      <div className="p-3.5 border-b border-zinc-800/70 space-y-2.5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1.5 text-xs font-semibold text-zinc-200">
              <Zap className="w-3.5 h-3.5 text-amber-400" />
              <span>Playground</span>
            </div>

            {/* Mock Control Toggle */}
            <button
              onClick={() => setShowMockToolbar(!showMockToolbar)}
              className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono transition-colors border ${
                showMockToolbar || mockStatusOverride || mockLatency
                  ? 'bg-amber-950/40 text-amber-300 border-amber-800/50'
                  : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:text-zinc-200'
              }`}
              title="Toggle Mock Server Overrides"
            >
              <Sliders className="w-2.5 h-2.5" />
              <span>Mock Controls</span>
              {(mockStatusOverride || mockLatency) && (
                <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse ml-0.5" />
              )}
            </button>
          </div>

          <div className="flex items-center gap-2">
            {/* Import cURL Button */}
            <button
              onClick={() => setIsCurlModalOpen(true)}
              className="flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 transition-colors"
              title="Import cURL command"
            >
              <FileCode className="w-3 h-3 text-indigo-400" />
              <span>Import cURL</span>
            </button>

            {/* Proxy vs Direct Toggle */}
            <button
              onClick={() => setIsProxyMode(!isProxyMode)}
              className={`flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium transition-colors border ${
                isProxyMode
                  ? 'bg-emerald-950/40 text-emerald-300 border-emerald-800/50'
                  : 'bg-zinc-800/60 text-zinc-400 border-zinc-700/60'
              }`}
              title={
                isProxyMode
                  ? 'CORS Proxy Active (Requests dispatched via local Go backend)'
                  : 'Direct Mode (In-browser fetch)'
              }
            >
              <Globe className="w-3 h-3" />
              <span>{isProxyMode ? 'Proxy: ON' : 'Direct'}</span>
            </button>

            {/* Stream Mode Toggle */}
            <button
              onClick={() => setIsStreamMode(!isStreamMode)}
              className={`flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium transition-colors border ${
                isStreamMode
                  ? 'bg-purple-950/50 text-purple-300 border-purple-800/60'
                  : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:text-zinc-200'
              }`}
              title="Stream mode: Receive Server-Sent Events (SSE) or NDJSON chunks in real time"
            >
              <Activity className="w-3 h-3 text-purple-400" />
              <span>{isStreamMode ? 'Stream: ON' : 'Stream'}</span>
            </button>
          </div>
        </div>

        {/* Mock Controls Toolbar (Expandable) */}
        {showMockToolbar && (
          <div className="p-2.5 bg-zinc-950 rounded border border-zinc-800/80 space-y-2 text-xs animate-in fade-in duration-100">
            <div className="flex items-center justify-between text-[11px] text-zinc-400">
              <span className="font-semibold text-zinc-300 flex items-center gap-1">
                <Radio className="w-3 h-3 text-emerald-400" />
                Mock Response Directives
              </span>
              {(mockStatusOverride || mockLatency || mockExampleOverride) && (
                <button
                  onClick={() => {
                    setMockStatusOverride('')
                    setMockLatency('')
                    setMockExampleOverride('')
                  }}
                  className="text-zinc-500 hover:text-zinc-300 text-[10px]"
                >
                  Reset Overrides
                </button>
              )}
            </div>

            {/* Status Code Chips */}
            <div className="flex items-center gap-1.5 flex-wrap">
              <span className="text-[10px] text-zinc-500 font-mono">Status:</span>
              {['200', '201', '400', '401', '404', '500'].map((code) => {
                const isActive = mockStatusOverride === code
                return (
                  <button
                    key={code}
                    onClick={() => setMockStatusOverride(isActive ? '' : code)}
                    className={`px-1.5 py-0.5 rounded text-[10px] font-mono border transition-colors ${
                      isActive
                        ? 'bg-amber-950/60 text-amber-300 border-amber-700 font-bold'
                        : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:text-zinc-200'
                    }`}
                  >
                    {code}
                  </button>
                )
              })}
            </div>

            {/* Latency Presets */}
            <div className="flex items-center gap-1.5 flex-wrap">
              <span className="text-[10px] text-zinc-500 font-mono">Delay:</span>
              {[
                { label: '0ms', val: '' },
                { label: '150ms', val: '150ms' },
                { label: '500ms', val: '500ms' },
                { label: '1s', val: '1000ms' },
              ].map((lat) => {
                const isActive = mockLatency === lat.val
                return (
                  <button
                    key={lat.label}
                    onClick={() => setMockLatency(lat.val)}
                    className={`px-1.5 py-0.5 rounded text-[10px] font-mono border transition-colors ${
                      isActive
                        ? 'bg-indigo-950/60 text-indigo-300 border-indigo-700 font-bold'
                        : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:text-zinc-200'
                    }`}
                  >
                    {lat.label}
                  </button>
                )
              })}

              {/* Named Example Dropdown */}
              {availableExamples.length > 0 && (
                <div className="flex items-center gap-1 ml-auto">
                  <span className="text-[10px] text-zinc-500 font-mono">Example:</span>
                  <select
                    value={mockExampleOverride}
                    onChange={(e) => setMockExampleOverride(e.target.value)}
                    className="bg-zinc-900 border border-zinc-800 rounded px-1.5 py-0.5 text-[10px] font-mono text-zinc-200 focus:outline-none"
                  >
                    <option value="">(default)</option>
                    {availableExamples.map((ex) => (
                      <option key={ex} value={ex}>
                        {ex}
                      </option>
                    ))}
                  </select>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Target URL Bar & Send Button */}
        <div className="flex items-center gap-1.5">
          <div className="flex-1 flex items-center bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1.5 text-xs font-mono text-zinc-300 truncate">
            <span className="font-bold text-zinc-500 mr-1.5">{endpoint.method}</span>
            <span className="truncate text-zinc-200" title={computedUrl}>
              {computedUrl}
            </span>
          </div>

          {isStreaming ? (
            <button
              onClick={handleStopStream}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-rose-600 hover:bg-rose-500 text-white rounded font-medium text-xs shadow-sm shadow-rose-900/30 transition-all cursor-pointer shrink-0"
              title="Cancel live stream"
            >
              <Square className="w-3.5 h-3.5 fill-current" />
              <span>Stop</span>
            </button>
          ) : (
            <button
              onClick={handleSend}
              disabled={isLoading}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white rounded font-medium text-xs shadow-sm shadow-emerald-900/30 transition-all cursor-pointer disabled:opacity-50 shrink-0"
            >
              {isLoading ? (
                <span className="w-3.5 h-3.5 border-2 border-white/20 border-t-white rounded-full animate-spin" />
              ) : (
                <Play className="w-3.5 h-3.5 fill-current" />
              )}
              <span>Send</span>
            </button>
          )}
        </div>
      </div>

      {/* Request Configuration Tabs */}
      <div className="flex items-center border-b border-zinc-800 px-3 bg-zinc-950/40 text-xs">
        <button
          onClick={() => setActiveTab('params')}
          className={`px-3 py-2 font-medium border-b-2 transition-colors cursor-pointer ${
            activeTab === 'params'
              ? 'border-indigo-500 text-indigo-400'
              : 'border-transparent text-zinc-400 hover:text-zinc-200'
          }`}
        >
          Params
        </button>
        <button
          onClick={() => setActiveTab('headers')}
          className={`px-3 py-2 font-medium border-b-2 transition-colors cursor-pointer ${
            activeTab === 'headers'
              ? 'border-indigo-500 text-indigo-400'
              : 'border-transparent text-zinc-400 hover:text-zinc-200'
          }`}
        >
          Headers
        </button>
        <button
          onClick={() => setActiveTab('auth')}
          className={`px-3 py-2 font-medium border-b-2 transition-colors cursor-pointer ${
            activeTab === 'auth'
              ? 'border-indigo-500 text-indigo-400'
              : 'border-transparent text-zinc-400 hover:text-zinc-200'
          }`}
        >
          Auth
        </button>
        {!['GET', 'HEAD'].includes(endpoint.method) && (
          <button
            onClick={() => setActiveTab('body')}
            className={`px-3 py-2 font-medium border-b-2 transition-colors cursor-pointer ${
              activeTab === 'body'
                ? 'border-indigo-500 text-indigo-400'
                : 'border-transparent text-zinc-400 hover:text-zinc-200'
            }`}
          >
            Body
          </button>
        )}
      </div>

      {/* Tab Panels Content */}
      <div className="flex-1 overflow-y-auto p-3 space-y-3 min-h-[200px]">
        {/* PARAMS TAB */}
        {activeTab === 'params' && (
          <div className="space-y-4 text-xs">
            {Object.keys(pathParamValues).length > 0 && (
              <div className="space-y-2">
                <div className="font-semibold text-zinc-400 uppercase tracking-wider text-[10px]">
                  Path Parameters
                </div>
                {Object.entries(pathParamValues).map(([name, val]) => (
                  <div key={name} className="flex items-center gap-2">
                    <span className="font-mono text-zinc-300 w-28 truncate shrink-0">{name}</span>
                    <input
                      type="text"
                      value={val}
                      onChange={(e) =>
                        setPathParamValues((prev) => ({ ...prev, [name]: e.target.value }))
                      }
                      placeholder={`value for {${name}}`}
                      className="flex-1 bg-zinc-900 border border-zinc-800 rounded px-2 py-1 text-xs text-zinc-200 focus:outline-none focus:border-zinc-600 font-mono"
                    />
                  </div>
                ))}
              </div>
            )}

            {/* Query Params */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-zinc-400 uppercase tracking-wider text-[10px]">
                  Query Parameters
                </span>
                <button
                  onClick={() => setQueryParams((prev) => [...prev, { key: '', value: '', enabled: true }])}
                  className="text-[11px] text-indigo-400 hover:text-indigo-300"
                >
                  + Add Query
                </button>
              </div>

              {queryParams.length === 0 ? (
                <div className="text-zinc-500 italic text-[11px]">No query parameters</div>
              ) : (
                queryParams.map((q, idx) => (
                  <div key={idx} className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={q.enabled}
                      onChange={(e) => {
                        const copy = [...queryParams]
                        copy[idx].enabled = e.target.checked
                        setQueryParams(copy)
                      }}
                      className="rounded border-zinc-700 bg-zinc-900"
                    />
                    <input
                      type="text"
                      value={q.key}
                      placeholder="key"
                      onChange={(e) => {
                        const copy = [...queryParams]
                        copy[idx].key = e.target.value
                        setQueryParams(copy)
                      }}
                      className="w-32 bg-zinc-900 border border-zinc-800 rounded px-2 py-1 text-xs text-zinc-200 font-mono"
                    />
                    <input
                      type="text"
                      value={q.value}
                      placeholder="value"
                      onChange={(e) => {
                        const copy = [...queryParams]
                        copy[idx].value = e.target.value
                        setQueryParams(copy)
                      }}
                      className="flex-1 bg-zinc-900 border border-zinc-800 rounded px-2 py-1 text-xs text-zinc-200 font-mono"
                    />
                    <button
                      onClick={() => setQueryParams((prev) => prev.filter((_, i) => i !== idx))}
                      className="text-zinc-500 hover:text-rose-400 text-xs px-1"
                    >
                      ×
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {/* HEADERS TAB */}
        {activeTab === 'headers' && (
          <div className="space-y-2 text-xs">
            <div className="flex items-center justify-between">
              <span className="font-semibold text-zinc-400 uppercase tracking-wider text-[10px]">
                Request Headers
              </span>
              <button
                onClick={() => setHeaders((prev) => [...prev, { key: '', value: '', enabled: true }])}
                className="text-[11px] text-indigo-400 hover:text-indigo-300"
              >
                + Add Header
              </button>
            </div>

            {headers.map((h, idx) => (
              <div key={idx} className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={h.enabled}
                  onChange={(e) => {
                    const copy = [...headers]
                    copy[idx].enabled = e.target.checked
                    setHeaders(copy)
                  }}
                  className="rounded border-zinc-700 bg-zinc-900"
                />
                <input
                  type="text"
                  value={h.key}
                  placeholder="Header Name"
                  onChange={(e) => {
                    const copy = [...headers]
                    copy[idx].key = e.target.value
                    setHeaders(copy)
                  }}
                  className="w-36 bg-zinc-900 border border-zinc-800 rounded px-2 py-1 text-xs text-zinc-200 font-mono"
                />
                <input
                  type="text"
                  value={h.value}
                  placeholder="Value"
                  onChange={(e) => {
                    const copy = [...headers]
                    copy[idx].value = e.target.value
                    setHeaders(copy)
                  }}
                  className="flex-1 bg-zinc-900 border border-zinc-800 rounded px-2 py-1 text-xs text-zinc-200 font-mono"
                />
                <button
                  onClick={() => setHeaders((prev) => prev.filter((_, i) => i !== idx))}
                  className="text-zinc-500 hover:text-rose-400 text-xs px-1"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
        )}

        {/* AUTH TAB */}
        {activeTab === 'auth' && (
          <div className="space-y-4 text-xs">
            <div className="flex items-center gap-2">
              <span className="text-zinc-400 w-16">Type:</span>
              <select
                value={authType}
                onChange={(e) => setAuthType(e.target.value as any)}
                className="bg-zinc-900 border border-zinc-800 rounded px-2 py-1 text-zinc-200 text-xs font-mono"
              >
                <option value="none">No Auth</option>
                <option value="bearer">Bearer Token</option>
                <option value="apiKey">API Key</option>
                <option value="basic">Basic Auth</option>
              </select>
            </div>

            {authType === 'bearer' && (
              <div className="space-y-1.5">
                <label className="text-zinc-400 block text-[11px]">Bearer Token / JWT:</label>
                <input
                  type="text"
                  value={bearerToken}
                  onChange={(e) => setBearerToken(e.target.value)}
                  placeholder="e.g. {{token}} or eyJhbGciOi..."
                  className="w-full bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1 text-xs font-mono text-zinc-200"
                />
              </div>
            )}

            {authType === 'apiKey' && (
              <div className="space-y-2">
                <div className="space-y-1">
                  <label className="text-zinc-400 block text-[11px]">Header Name (or &apos;query&apos;):</label>
                  <input
                    type="text"
                    value={apiKeyHeader}
                    onChange={(e) => setApiKeyHeader(e.target.value)}
                    className="w-full bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1 text-xs font-mono text-zinc-200"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-zinc-400 block text-[11px]">API Key Value:</label>
                  <input
                    type="text"
                    value={apiKeyValue}
                    onChange={(e) => setApiKeyValue(e.target.value)}
                    placeholder="e.g. {{apiKey}}"
                    className="w-full bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1 text-xs font-mono text-zinc-200"
                  />
                </div>
              </div>
            )}

            {authType === 'basic' && (
              <div className="space-y-2">
                <div className="space-y-1">
                  <label className="text-zinc-400 block text-[11px]">Username:</label>
                  <input
                    type="text"
                    value={basicUser}
                    onChange={(e) => setBasicUser(e.target.value)}
                    className="w-full bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1 text-xs font-mono text-zinc-200"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-zinc-400 block text-[11px]">Password:</label>
                  <input
                    type="password"
                    value={basicPass}
                    onChange={(e) => setBasicPass(e.target.value)}
                    className="w-full bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1 text-xs font-mono text-zinc-200"
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* BODY TAB (Monaco Editor) */}
        {activeTab === 'body' && (
          <div className="h-full flex flex-col space-y-1.5">
            <div className="flex items-center justify-between text-xs">
              <span className="text-[11px] font-mono text-zinc-400">JSON Request Body</span>
              <div className="flex items-center gap-2">
                <button
                  onClick={prettifyBody}
                  className="text-zinc-400 hover:text-zinc-200 text-[11px]"
                  title="Format JSON"
                >
                  Prettify
                </button>
                <button
                  onClick={() => setRequestBody('')}
                  className="text-zinc-500 hover:text-rose-400 text-[11px]"
                >
                  Clear
                </button>
              </div>
            </div>

            <div className="h-44 rounded border border-zinc-800 overflow-hidden bg-zinc-950">
              <Editor
                height="100%"
                defaultLanguage="json"
                theme="vs-dark"
                value={requestBody}
                onChange={(val) => setRequestBody(val || '')}
                options={{
                  minimap: { enabled: false },
                  fontSize: 12,
                  fontFamily: 'Consolas, monospace',
                  scrollBeyondLastLine: false,
                  wordWrap: 'on',
                  tabSize: 2,
                  automaticLayout: true,
                }}
              />
            </div>
          </div>
        )}
      </div>

      {/* Response Console */}
      <div className="h-72 border-t border-zinc-800/80 flex flex-col bg-[#0b0b0e]">
        {/* Response Status & Telemetry Bar */}
        <div className="px-3 py-2 border-b border-zinc-800/80 flex items-center justify-between text-xs">
          <div className="flex items-center gap-2">
            {response ? (
              <span
                className={`font-mono font-bold px-2 py-0.5 rounded text-xs ${
                  response.status >= 200 && response.status < 300
                    ? 'bg-emerald-950/60 text-emerald-400 border border-emerald-800/50'
                    : response.status >= 400 && response.status < 500
                    ? 'bg-amber-950/60 text-amber-400 border border-amber-800/50'
                    : 'bg-rose-950/60 text-rose-400 border border-rose-800/50'
                }`}
              >
                {response.status} {response.statusText}
              </span>
            ) : (
              <span className="text-zinc-500 text-[11px]">Ready to execute</span>
            )}

            {/* Contract Conformity Pill */}
            {contractResult && (
              <button
                onClick={() => setActiveResponseTab('contract')}
                className={`flex items-center gap-1 font-mono text-[10px] px-1.5 py-0.5 rounded border transition-colors ${
                  contractResult.isValid
                    ? 'bg-emerald-950/50 text-emerald-400 border-emerald-800/50'
                    : 'bg-amber-950/50 text-amber-400 border-amber-800/50'
                }`}
                title="Open Response Contract Audit"
              >
                {contractResult.isValid ? (
                  <CheckCircle2 className="w-3 h-3 text-emerald-400" />
                ) : (
                  <AlertTriangle className="w-3 h-3 text-amber-400" />
                )}
                <span>Contract {contractResult.conformityScore}%</span>
              </button>
            )}

            {/* Live Streaming Badge */}
            {(isStreaming || streamChunksCount > 0) && (
              <span
                className={`flex items-center gap-1 font-mono text-[10px] px-1.5 py-0.5 rounded border ${
                  isStreaming
                    ? 'bg-purple-950/60 text-purple-300 border-purple-800/60 animate-pulse'
                    : 'bg-zinc-800/60 text-zinc-400 border-zinc-700/60'
                }`}
              >
                <Activity className="w-2.5 h-2.5 text-purple-400" />
                <span>
                  {isStreaming ? 'Streaming' : 'Stream Done'} ({streamChunksCount} chunks)
                </span>
              </span>
            )}

            {response?.timing && (
              <div className="flex items-center gap-3 text-[11px] text-zinc-400 font-mono ml-1">
                <span className="flex items-center gap-1" title="Total Execution Duration">
                  <Clock className="w-3 h-3 text-zinc-500" />
                  {response.timing.totalMs} ms
                </span>
                <span className="flex items-center gap-1" title="Payload Size">
                  <HardDrive className="w-3 h-3 text-zinc-500" />
                  {(response.timing.sizeBytes / 1024).toFixed(1)} KB
                </span>
              </div>
            )}
          </div>

          {/* Response Sub-tabs */}
          <div className="flex items-center gap-1">
            <button
              onClick={() => setActiveResponseTab('body')}
              className={`px-2 py-0.5 rounded text-[11px] transition-colors ${
                activeResponseTab === 'body'
                  ? 'bg-zinc-800 text-zinc-100 font-medium'
                  : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              Body
            </button>
            <button
              onClick={() => setActiveResponseTab('headers')}
              className={`px-2 py-0.5 rounded text-[11px] transition-colors ${
                activeResponseTab === 'headers'
                  ? 'bg-zinc-800 text-zinc-100 font-medium'
                  : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              Headers
            </button>
            <button
              onClick={() => setActiveResponseTab('snippets')}
              className={`px-2 py-0.5 rounded text-[11px] transition-colors ${
                activeResponseTab === 'snippets'
                  ? 'bg-zinc-800 text-zinc-100 font-medium'
                  : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              Snippets
            </button>
            {contractResult && (
              <button
                onClick={() => setActiveResponseTab('contract')}
                className={`px-2 py-0.5 rounded text-[11px] transition-colors ${
                  activeResponseTab === 'contract'
                    ? 'bg-zinc-800 text-zinc-100 font-medium'
                    : 'text-zinc-400 hover:text-zinc-200'
                }`}
              >
                Contract ({contractResult.issues.length})
              </button>
            )}

            {response?.body && activeResponseTab === 'body' && (
              <button
                onClick={copyResponse}
                className="text-zinc-500 hover:text-zinc-300 p-1 ml-1"
                title="Copy response body"
              >
                {copiedResponse ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            )}
          </div>
        </div>

        {/* Response Body Area */}
        <div className="flex-1 overflow-hidden">
          {activeResponseTab === 'body' && (
            response ? (
              response.error ? (
                <div className="p-4 text-xs font-mono text-rose-400 bg-rose-950/20 m-2 rounded border border-rose-900/40">
                  {response.error}
                </div>
              ) : (
                <Editor
                  height="100%"
                  defaultLanguage="json"
                  theme="vs-dark"
                  value={response.body}
                  options={{
                    readOnly: true,
                    minimap: { enabled: false },
                    fontSize: 12,
                    fontFamily: 'Consolas, monospace',
                    scrollBeyondLastLine: false,
                    wordWrap: 'on',
                    automaticLayout: true,
                  }}
                />
              )
            ) : (
              <div className="h-full flex flex-col items-center justify-center text-zinc-600 text-xs">
                <span>Hit &quot;Send&quot; or press Ctrl+Enter</span>
              </div>
            )
          )}

          {/* Response Headers Table */}
          {activeResponseTab === 'headers' && (
            <div className="h-full overflow-y-auto p-3">
              {response?.headers && Object.keys(response.headers).length > 0 ? (
                <table className="w-full text-left text-xs font-mono">
                  <tbody>
                    {Object.entries(response.headers).map(([k, v]) => (
                      <tr key={k} className="border-b border-zinc-800/60">
                        <td className="py-1 px-2 text-zinc-400 font-semibold">{k}</td>
                        <td className="py-1 px-2 text-zinc-200 break-all">{v}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <div className="text-zinc-500 text-xs italic text-center py-8">
                  No response headers yet
                </div>
              )}
            </div>
          )}

          {/* Code Snippets Panel */}
          {activeResponseTab === 'snippets' && (
            <div className="h-full flex flex-col">
              <div className="flex items-center justify-between px-3 py-1.5 border-b border-zinc-800 bg-zinc-950 text-xs">
                <div className="flex items-center gap-1">
                  {(['curl', 'fetch', 'axios', 'pythonReq', 'pythonHttpx', 'go'] as const).map((lang) => (
                    <button
                      key={lang}
                      onClick={() => setSnippetLang(lang)}
                      className={`px-2 py-0.5 rounded text-[10px] font-mono transition-colors ${
                        snippetLang === lang
                          ? 'bg-indigo-950/60 text-indigo-300 border border-indigo-800/40'
                          : 'text-zinc-500 hover:text-zinc-300'
                      }`}
                    >
                      {lang === 'pythonReq' ? 'requests' : lang === 'pythonHttpx' ? 'httpx' : lang}
                    </button>
                  ))}
                </div>

                <button
                  onClick={copySnippet}
                  className="flex items-center gap-1 text-[11px] text-zinc-400 hover:text-zinc-200"
                >
                  {copiedSnippet ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                  <span>Copy</span>
                </button>
              </div>

              <div className="flex-1 overflow-hidden bg-zinc-950">
                <Editor
                  height="100%"
                  defaultLanguage={
                    snippetLang === 'go'
                      ? 'go'
                      : snippetLang.startsWith('python')
                      ? 'python'
                      : snippetLang === 'curl'
                      ? 'shell'
                      : 'typescript'
                  }
                  theme="vs-dark"
                  value={activeSnippetCode}
                  options={{
                    readOnly: true,
                    minimap: { enabled: false },
                    fontSize: 11,
                    fontFamily: 'Consolas, monospace',
                    scrollBeyondLastLine: false,
                    wordWrap: 'on',
                    automaticLayout: true,
                  }}
                />
              </div>
            </div>
          )}

          {/* Response Contract Audit Panel */}
          {activeResponseTab === 'contract' && contractResult && (
            <div className="h-full overflow-y-auto p-4 space-y-3">
              <div className="flex items-center justify-between border-b border-zinc-800 pb-2">
                <div className="flex items-center gap-2">
                  {contractResult.isValid ? (
                    <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                  ) : (
                    <AlertTriangle className="w-4 h-4 text-amber-400" />
                  )}
                  <span className="text-xs font-semibold text-zinc-200">
                    Schema Conformity: {contractResult.conformityScore}%
                  </span>
                </div>
                <span className="text-[10px] text-zinc-500 font-mono">
                  {contractResult.issues.length} issue(s) detected
                </span>
              </div>

              {contractResult.issues.length === 0 ? (
                <div className="text-center py-6 text-zinc-400 text-xs">
                  ✅ Response payload conforms 100% to the OpenAPI response schema.
                </div>
              ) : (
                <div className="space-y-1.5">
                  {contractResult.issues.map((issue, idx) => (
                    <div
                      key={idx}
                      className={`p-2 rounded text-xs border ${
                        issue.severity === 'ERROR'
                          ? 'bg-rose-950/30 text-rose-300 border-rose-900/40'
                          : 'bg-amber-950/30 text-amber-300 border-amber-900/40'
                      }`}
                    >
                      <div className="font-mono text-[10px] opacity-75">{issue.path}</div>
                      <div>{issue.message}</div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* cURL Import Modal */}
      {isCurlModalOpen && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="w-full max-w-lg bg-[#121215] border border-zinc-800 rounded-xl shadow-2xl overflow-hidden flex flex-col">
            <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-800">
              <div className="flex items-center gap-2">
                <FileCode className="w-4 h-4 text-indigo-400" />
                <span className="font-semibold text-xs text-zinc-200">Import cURL Command</span>
              </div>
              <button onClick={() => setIsCurlModalOpen(false)} className="text-zinc-500 hover:text-zinc-300">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-4 space-y-2">
              <p className="text-[11px] text-zinc-400 leading-relaxed">
                Paste any standard cURL command. APIDock will automatically parse the headers, URL parameters, and JSON payload into the playground.
              </p>
              <textarea
                value={curlCommandInput}
                onChange={(e) => setCurlCommandInput(e.target.value)}
                placeholder={'curl -X POST "https://api.example.com/v1/orders" \\\n  -H "Authorization: Bearer token" \\\n  -d \'{"quantity": 5}\''}
                rows={6}
                className="w-full bg-zinc-950 border border-zinc-800 rounded p-2.5 text-xs font-mono text-zinc-200 focus:outline-none focus:border-indigo-500 resize-none"
              />
            </div>

            <div className="px-4 py-3 border-t border-zinc-800 bg-zinc-950 flex justify-end gap-2">
              <button
                onClick={() => setIsCurlModalOpen(false)}
                className="px-3 py-1 text-xs text-zinc-400 hover:text-zinc-200"
              >
                Cancel
              </button>
              <button
                onClick={handleImportCurl}
                disabled={!curlCommandInput.trim()}
                className="px-3 py-1 text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded transition-colors disabled:opacity-50"
              >
                Import to Playground
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
