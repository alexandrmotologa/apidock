import React, { useState } from 'react'
import type { Environment } from '../types/spec'
import { X, Plus, Trash2, Lock, Unlock, Key, Layers, Globe } from 'lucide-react'

interface EnvSwitcherModalProps {
  isOpen: boolean
  onClose: () => void
  environments: Environment[]
  activeEnvId: string
  onSelectEnv: (id: string) => void
  onUpdateEnv: (env: Environment) => void
  onAddEnv: (env: Environment) => void
  onDeleteEnv: (id: string) => void
  onEncrypt: (password: string) => Promise<void>
  onDecrypt: (password: string) => Promise<void>
}

export const EnvSwitcherModal: React.FC<EnvSwitcherModalProps> = ({
  isOpen,
  onClose,
  environments,
  activeEnvId,
  onSelectEnv,
  onUpdateEnv,
  onAddEnv,
  onDeleteEnv,
  onEncrypt,
  onDecrypt,
}) => {
  const [selectedEnvId, setSelectedEnvId] = useState<string>(activeEnvId)
  const [passphrase, setPassphrase] = useState<string>('')
  const [vaultError, setVaultError] = useState<string | null>(null)
  const [newVarKey, setNewVarKey] = useState('')
  const [newVarVal, setNewVarVal] = useState('')

  const currentEnv = environments.find((e) => e.id === selectedEnvId) || environments[0]

  if (!isOpen) return null

  const handleAddVar = () => {
    if (!newVarKey.trim() || !currentEnv) return
    const updated = {
      ...currentEnv,
      variables: {
        ...currentEnv.variables,
        [newVarKey.trim()]: newVarVal,
      },
    }
    onUpdateEnv(updated)
    setNewVarKey('')
    setNewVarVal('')
  }

  const handleDeleteVar = (key: string) => {
    if (!currentEnv) return
    const copy = { ...currentEnv.variables }
    delete copy[key]
    onUpdateEnv({ ...currentEnv, variables: copy })
  }

  const handleCreateEnv = () => {
    const newId = `env-${Date.now()}`
    const newEnv: Environment = {
      id: newId,
      name: 'Custom Env',
      baseUrl: 'https://api.example.com',
      variables: { baseUrl: 'https://api.example.com' },
    }
    onAddEnv(newEnv)
    setSelectedEnvId(newId)
  }

  const handleVaultToggle = async () => {
    setVaultError(null)
    if (!passphrase) {
      setVaultError('Master passphrase is required')
      return
    }

    try {
      if (currentEnv.isEncrypted) {
        await onDecrypt(passphrase)
      } else {
        await onEncrypt(passphrase)
      }
      setPassphrase('')
    } catch (e) {
      setVaultError(e instanceof Error ? e.message : 'Vault operation failed')
    }
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4">
      <div className="w-full max-w-2xl bg-[#121215] border border-zinc-800 rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800">
          <div className="flex items-center gap-2">
            <Layers className="w-4 h-4 text-indigo-400" />
            <span className="font-semibold text-sm text-zinc-100">Environment & Secret Vault</span>
          </div>
          <button onClick={onClose} className="text-zinc-500 hover:text-zinc-300">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Content Body */}
        <div className="flex flex-1 overflow-hidden">
          {/* Environment List Sidebar */}
          <div className="w-48 border-r border-zinc-800 p-2 space-y-1 bg-zinc-950/40">
            <div className="flex items-center justify-between px-2 py-1 text-[11px] font-semibold text-zinc-500 uppercase tracking-wider">
              <span>Environments</span>
              <button
                onClick={handleCreateEnv}
                className="text-indigo-400 hover:text-indigo-300"
                title="Create New Environment"
              >
                <Plus className="w-3.5 h-3.5" />
              </button>
            </div>

            {environments.map((env) => (
              <button
                key={env.id}
                onClick={() => setSelectedEnvId(env.id)}
                className={`w-full text-left px-2.5 py-1.5 rounded text-xs flex items-center justify-between transition-colors cursor-pointer ${
                  selectedEnvId === env.id
                    ? 'bg-zinc-800 text-zinc-100 font-medium'
                    : 'text-zinc-400 hover:bg-zinc-800/50 hover:text-zinc-200'
                }`}
              >
                <span className="truncate">{env.name}</span>
                {env.id === activeEnvId && (
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0" title="Active" />
                )}
              </button>
            ))}
          </div>

          {/* Environment Details Editor */}
          <div className="flex-1 p-5 overflow-y-auto space-y-5">
            {/* Environment Name & Active Toggle */}
            <div className="flex items-center justify-between gap-3">
              <input
                type="text"
                value={currentEnv.name}
                onChange={(e) => onUpdateEnv({ ...currentEnv, name: e.target.value })}
                className="text-sm font-semibold bg-transparent border-b border-transparent hover:border-zinc-700 focus:border-indigo-500 text-zinc-100 focus:outline-none pb-0.5"
              />

              <div className="flex items-center gap-2">
                {currentEnv.id !== activeEnvId && (
                  <button
                    onClick={() => onSelectEnv(currentEnv.id)}
                    className="px-2.5 py-1 text-xs font-medium rounded bg-emerald-950/40 text-emerald-300 border border-emerald-800/50 hover:bg-emerald-900/40 transition-colors"
                  >
                    Set as Active
                  </button>
                )}

                {environments.length > 1 && (
                  <button
                    onClick={() => onDeleteEnv(currentEnv.id)}
                    className="text-zinc-500 hover:text-rose-400 p-1 rounded"
                    title="Delete Environment"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            </div>

            {/* Base URL */}
            <div className="space-y-1.5">
              <label className="text-xs text-zinc-400 flex items-center gap-1.5">
                <Globe className="w-3.5 h-3.5 text-zinc-500" />
                <span>Base URL:</span>
              </label>
              <input
                type="text"
                value={currentEnv.baseUrl}
                onChange={(e) => {
                  const val = e.target.value
                  onUpdateEnv({
                    ...currentEnv,
                    baseUrl: val,
                    variables: { ...currentEnv.variables, baseUrl: val },
                  })
                }}
                className="w-full bg-zinc-900 border border-zinc-800 rounded px-3 py-1.5 text-xs font-mono text-zinc-200 focus:outline-none focus:border-zinc-600"
                placeholder="http://localhost:4010"
              />
            </div>

            {/* Client-Side AES-GCM Vault Section */}
            <div className="p-3.5 rounded-lg border border-zinc-800 bg-zinc-950/60 space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Key className="w-3.5 h-3.5 text-amber-400" />
                  <span className="text-xs font-semibold text-zinc-200">Local Secret Vault (AES-GCM)</span>
                </div>
                <span
                  className={`text-[10px] font-mono px-2 py-0.5 rounded border ${
                    currentEnv.isEncrypted
                      ? 'bg-rose-950/50 text-rose-400 border-rose-800/40'
                      : 'bg-zinc-800 text-zinc-400 border-zinc-700'
                  }`}
                >
                  {currentEnv.isEncrypted ? 'LOCKED' : 'UNLOCKED'}
                </span>
              </div>

              <p className="text-[11px] text-zinc-400 leading-relaxed">
                Encrypt your API tokens with AES-GCM 256-bit encryption right in your browser. Secrets never leave your machine unencrypted.
              </p>

              <div className="flex items-center gap-2">
                <input
                  type="password"
                  value={passphrase}
                  onChange={(e) => setPassphrase(e.target.value)}
                  placeholder="Master Passphrase"
                  className="flex-1 bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1 text-xs text-zinc-200 focus:outline-none focus:border-zinc-600"
                />
                <button
                  onClick={handleVaultToggle}
                  className={`flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded transition-colors ${
                    currentEnv.isEncrypted
                      ? 'bg-amber-600 hover:bg-amber-500 text-white'
                      : 'bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700'
                  }`}
                >
                  {currentEnv.isEncrypted ? <Unlock className="w-3 h-3" /> : <Lock className="w-3 h-3" />}
                  <span>{currentEnv.isEncrypted ? 'Unlock Vault' : 'Encrypt Vault'}</span>
                </button>
              </div>

              {vaultError && (
                <div className="text-[11px] text-rose-400 bg-rose-950/40 p-2 rounded border border-rose-900/50 font-mono">
                  {vaultError}
                </div>
              )}
            </div>

            {/* Variable Key-Value Pairs */}
            <div className="space-y-3">
              <span className="text-xs font-semibold text-zinc-300 block">Variables</span>

              {currentEnv.isEncrypted ? (
                <div className="p-6 text-center text-xs text-zinc-500 border border-zinc-800/80 rounded-lg">
                  🔒 Variables are encrypted with AES-GCM. Enter your passphrase above to unlock.
                </div>
              ) : (
                <div className="space-y-2">
                  {Object.entries(currentEnv.variables).map(([k, v]) => (
                    <div key={k} className="flex items-center gap-2">
                      <span className="font-mono text-xs text-indigo-300 w-32 truncate shrink-0">
                        {`{{${k}}}`}
                      </span>
                      <input
                        type="text"
                        value={v}
                        onChange={(e) => {
                          onUpdateEnv({
                            ...currentEnv,
                            variables: { ...currentEnv.variables, [k]: e.target.value },
                          })
                        }}
                        className="flex-1 bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1 text-xs font-mono text-zinc-200 focus:outline-none focus:border-zinc-600"
                      />
                      <button
                        onClick={() => handleDeleteVar(k)}
                        className="text-zinc-500 hover:text-rose-400 text-xs px-1"
                      >
                        ×
                      </button>
                    </div>
                  ))}

                  {/* Add New Variable Row */}
                  <div className="flex items-center gap-2 pt-2 border-t border-zinc-800/60">
                    <input
                      type="text"
                      value={newVarKey}
                      placeholder="variableName"
                      onChange={(e) => setNewVarKey(e.target.value)}
                      className="w-32 bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1 text-xs font-mono text-zinc-200 focus:outline-none focus:border-zinc-600"
                    />
                    <input
                      type="text"
                      value={newVarVal}
                      placeholder="value"
                      onChange={(e) => setNewVarVal(e.target.value)}
                      className="flex-1 bg-zinc-900 border border-zinc-800 rounded px-2.5 py-1 text-xs font-mono text-zinc-200 focus:outline-none focus:border-zinc-600"
                    />
                    <button
                      onClick={handleAddVar}
                      className="px-2.5 py-1 text-xs bg-indigo-600 hover:bg-indigo-500 text-white rounded font-medium transition-colors"
                    >
                      Add
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-zinc-800 bg-zinc-950 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-1.5 text-xs font-medium bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded transition-colors"
          >
            Done
          </button>
        </div>
      </div>
    </div>
  )
}
