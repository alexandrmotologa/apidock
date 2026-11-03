import { useState, useEffect } from 'react'
import { useSpec } from './hooks/useSpec'
import { useEnvironment } from './hooks/useEnvironment'
import { Sidebar } from './components/Sidebar'
import { ApiDocViewer } from './components/ApiDocViewer'
import { InteractiveClient } from './components/InteractiveClient'
import { SearchModal } from './components/SearchModal'
import { EnvSwitcherModal } from './components/EnvSwitcherModal'
import { HistoryModal } from './components/HistoryModal'
import { ValidationModal } from './components/ValidationModal'
import { DiffModal } from './components/DiffModal'
import type { Endpoint } from './types/spec'
import { AlertCircle, RefreshCw } from 'lucide-react'

export function App() {
  const { spec, loading, error, reloadSpec } = useSpec()
  const {
    environments,
    activeEnv,
    activeEnvId,
    setActiveEnvId,
    updateEnvironment,
    addEnvironment,
    deleteEnvironment,
    encryptActiveEnv,
    decryptActiveEnv,
  } = useEnvironment()

  const [selectedEndpoint, setSelectedEndpoint] = useState<Endpoint | null>(null)
  const [selectedServer, setSelectedServer] = useState<string>('')
  const [injectedBody, setInjectedBody] = useState<string>('')

  // Modals state
  const [isSearchOpen, setIsSearchOpen] = useState(false)
  const [isEnvOpen, setIsEnvOpen] = useState(false)
  const [isHistoryOpen, setIsHistoryOpen] = useState(false)
  const [isValidationOpen, setIsValidationOpen] = useState(false)
  const [isDiffOpen, setIsDiffOpen] = useState(false)

  // Initialize selected endpoint and server when spec loads
  useEffect(() => {
    if (spec && spec.endpoints.length > 0 && !selectedEndpoint) {
      setSelectedEndpoint(spec.endpoints[0])
    }
    if (spec && spec.servers.length > 0 && !selectedServer) {
      setSelectedServer(spec.servers[0].url)
    } else if (spec && spec.servers.length === 0 && !selectedServer) {
      setSelectedServer('http://localhost:4010')
    }
  }, [spec, selectedEndpoint, selectedServer])

  // Global Keyboard Shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl + K => Search
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault()
        setIsSearchOpen((prev) => !prev)
      }
      // Cmd/Ctrl + E => Environment
      if ((e.ctrlKey || e.metaKey) && e.key === 'e') {
        e.preventDefault()
        setIsEnvOpen((prev) => !prev)
      }
      // Cmd/Ctrl + H => History
      if ((e.ctrlKey || e.metaKey) && e.key === 'h') {
        e.preventDefault()
        setIsHistoryOpen((prev) => !prev)
      }
      // Cmd/Ctrl + D => Spec Diff
      if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
        e.preventDefault()
        setIsDiffOpen((prev) => !prev)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  if (loading && !spec) {
    return (
      <div className="h-screen w-screen flex flex-col items-center justify-center bg-[#09090b] text-zinc-400 space-y-3">
        <div className="w-8 h-8 rounded-full border-2 border-indigo-500/20 border-t-indigo-500 animate-spin" />
        <span className="text-xs font-mono">Loading APIDock Studio...</span>
      </div>
    )
  }

  if (error && !spec) {
    return (
      <div className="h-screen w-screen flex flex-col items-center justify-center bg-[#09090b] text-zinc-300 p-6 text-center space-y-4">
        <AlertCircle className="w-10 h-10 text-rose-500" />
        <h2 className="text-base font-semibold text-zinc-100">Unable to Load Specification</h2>
        <p className="text-xs text-zinc-400 max-w-md font-mono bg-zinc-900 p-3 rounded border border-zinc-800">
          {error}
        </p>
        <button
          onClick={reloadSpec}
          className="flex items-center gap-2 px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-medium transition-colors"
        >
          <RefreshCw className="w-3.5 h-3.5" />
          <span>Retry</span>
        </button>
      </div>
    )
  }

  if (!spec) return null

  return (
    <div className="h-screen w-screen flex overflow-hidden bg-[#09090b] text-zinc-100 font-sans">
      {/* 1. Left Column: Sidebar Navigation */}
      <Sidebar
        spec={spec}
        selectedEndpoint={selectedEndpoint}
        onSelectEndpoint={(ep) => setSelectedEndpoint(ep)}
        onOpenSearch={() => setIsSearchOpen(true)}
        onOpenEnv={() => setIsEnvOpen(true)}
        onOpenHistory={() => setIsHistoryOpen(true)}
        onOpenValidation={() => setIsValidationOpen(true)}
        onOpenDiff={() => setIsDiffOpen(true)}
        activeEnv={activeEnv}
      />

      {/* 2. Center Column: API Documentation Viewer */}
      <ApiDocViewer
        endpoint={selectedEndpoint}
        spec={spec}
        selectedServer={selectedServer}
        onSelectServer={setSelectedServer}
        onFillClientBody={(json) => setInjectedBody(json)}
      />

      {/* 3. Right Column: Interactive API Client */}
      <InteractiveClient
        endpoint={selectedEndpoint}
        activeEnv={activeEnv}
        activeServer={selectedServer}
        initialBody={injectedBody}
      />

      {/* Modals */}
      <SearchModal
        isOpen={isSearchOpen}
        onClose={() => setIsSearchOpen(false)}
        endpoints={spec.endpoints}
        onSelectEndpoint={(ep) => setSelectedEndpoint(ep)}
      />

      <EnvSwitcherModal
        isOpen={isEnvOpen}
        onClose={() => setIsEnvOpen(false)}
        environments={environments}
        activeEnvId={activeEnvId}
        onSelectEnv={(id) => setActiveEnvId(id)}
        onUpdateEnv={updateEnvironment}
        onAddEnv={addEnvironment}
        onDeleteEnv={deleteEnvironment}
        onEncrypt={encryptActiveEnv}
        onDecrypt={decryptActiveEnv}
      />

      <HistoryModal
        isOpen={isHistoryOpen}
        onClose={() => setIsHistoryOpen(false)}
        endpoints={spec.endpoints}
        onSelectEndpoint={(ep) => setSelectedEndpoint(ep)}
      />

      <ValidationModal
        isOpen={isValidationOpen}
        onClose={() => setIsValidationOpen(false)}
      />

      <DiffModal
        isOpen={isDiffOpen}
        onClose={() => setIsDiffOpen(false)}
      />
    </div>
  )
}

export default App
