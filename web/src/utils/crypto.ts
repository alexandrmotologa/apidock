/**
 * Client-Side AES-GCM 256-bit Vault Encryption
 * Uses Web Crypto API (SubtleCrypto) with PBKDF2 key derivation.
 */

export async function encryptVault(plainText: string, password: string): Promise<string> {
  const enc = new TextEncoder()
  const salt = window.crypto.getRandomValues(new Uint8Array(16))
  const iv = window.crypto.getRandomValues(new Uint8Array(12))

  const keyMaterial = await window.crypto.subtle.importKey(
    'raw',
    enc.encode(password),
    'PBKDF2',
    false,
    ['deriveKey']
  )

  const key = await window.crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt,
      iterations: 100000,
      hash: 'SHA-256',
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt']
  )

  const ciphertext = await window.crypto.subtle.encrypt(
    { name: 'AES-GCM', iv },
    key,
    enc.encode(plainText)
  )

  // Package as base64 envelope: [salt(16) + iv(12) + ciphertext]
  const combined = new Uint8Array(salt.length + iv.length + ciphertext.byteLength)
  combined.set(salt, 0)
  combined.set(iv, salt.length)
  combined.set(new Uint8Array(ciphertext), salt.length + iv.length)

  return btoa(String.fromCharCode(...combined))
}

export async function decryptVault(cipherEnvelope: string, password: string): Promise<string> {
  const binaryStr = atob(cipherEnvelope)
  const bytes = new Uint8Array(binaryStr.length)
  for (let i = 0; i < binaryStr.length; i++) {
    bytes[i] = binaryStr.charCodeAt(i)
  }

  if (bytes.length < 28) {
    throw new Error('Invalid encrypted payload')
  }

  const salt = bytes.slice(0, 16)
  const iv = bytes.slice(16, 28)
  const ciphertext = bytes.slice(28)

  const enc = new TextEncoder()
  const keyMaterial = await window.crypto.subtle.importKey(
    'raw',
    enc.encode(password),
    'PBKDF2',
    false,
    ['deriveKey']
  )

  const key = await window.crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt,
      iterations: 100000,
      hash: 'SHA-256',
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['decrypt']
  )

  const decrypted = await window.crypto.subtle.decrypt(
    { name: 'AES-GCM', iv },
    key,
    ciphertext
  )

  return new TextDecoder().decode(decrypted)
}
