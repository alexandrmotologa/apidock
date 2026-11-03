import { useState, useEffect } from 'react'
import type { Environment } from '../types/spec'
import { encryptVault, decryptVault } from '../utils/crypto'

const STORAGE_KEY = 'apidock_environments_v1'
const ACTIVE_KEY = 'apidock_active_env_id_v1'

const DEFAULT_ENVIRONMENTS: Environment[] = [
  {
    id: 'local-mock',
    name: 'Local Mock',
    baseUrl: 'http://localhost:4010',
    variables: {
      baseUrl: 'http://localhost:4010',
      apiKey: 'dev_mock_key_9981',
      token: 'jwt_mock_token_abc',
    },
  },
  {
    id: 'staging',
    name: 'Staging',
    baseUrl: 'https://staging.api.example.com/v1',
    variables: {
      baseUrl: 'https://staging.api.example.com/v1',
      apiKey: 'stg_live_key_3810',
      token: '',
    },
  },
  {
    id: 'production',
    name: 'Production',
    baseUrl: 'https://api.example.com/v1',
    variables: {
      baseUrl: 'https://api.example.com/v1',
      apiKey: '',
      token: '',
    },
  },
]

export function useEnvironment() {
  const [environments, setEnvironments] = useState<Environment[]>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY)
      if (saved) {
        return JSON.parse(saved)
      }
    } catch {
      // Fallback
    }
    return DEFAULT_ENVIRONMENTS
  })

  const [activeEnvId, setActiveEnvId] = useState<string>(() => {
    return localStorage.getItem(ACTIVE_KEY) || 'local-mock'
  })

  const [vaultPassword, setVaultPassword] = useState<string | null>(null)

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(environments))
    } catch (e) {
      console.error('Failed to save environments to storage', e)
    }
  }, [environments])

  useEffect(() => {
    try {
      localStorage.setItem(ACTIVE_KEY, activeEnvId)
    } catch (e) {
      console.error('Failed to save active env id', e)
    }
  }, [activeEnvId])

  const activeEnv = environments.find((e) => e.id === activeEnvId) || environments[0]

  const updateEnvironment = (updated: Environment) => {
    setEnvironments((prev) => prev.map((e) => (e.id === updated.id ? updated : e)))
  }

  const addEnvironment = (newEnv: Environment) => {
    setEnvironments((prev) => [...prev, newEnv])
    setActiveEnvId(newEnv.id)
  }

  const deleteEnvironment = (id: string) => {
    if (environments.length <= 1) return
    setEnvironments((prev) => prev.filter((e) => e.id !== id))
    if (activeEnvId === id) {
      const remaining = environments.filter((e) => e.id !== id)
      setActiveEnvId(remaining[0].id)
    }
  }

  const encryptActiveEnv = async (password: string) => {
    if (!activeEnv) return
    const plain = JSON.stringify(activeEnv.variables)
    const cipher = await encryptVault(plain, password)
    updateEnvironment({
      ...activeEnv,
      isEncrypted: true,
      cipherData: cipher,
      variables: {}, // Redact plaintext in storage
    })
    setVaultPassword(password)
  }

  const decryptActiveEnv = async (password: string) => {
    if (!activeEnv || !activeEnv.cipherData) return
    const plain = await decryptVault(activeEnv.cipherData, password)
    const vars = JSON.parse(plain)
    updateEnvironment({
      ...activeEnv,
      isEncrypted: false,
      cipherData: undefined,
      variables: vars,
    })
    setVaultPassword(null)
  }

  return {
    environments,
    activeEnv,
    activeEnvId,
    setActiveEnvId,
    updateEnvironment,
    addEnvironment,
    deleteEnvironment,
    vaultPassword,
    encryptActiveEnv,
    decryptActiveEnv,
  }
}
