export interface ParsedCurl {
  method: string
  url: string
  headers: Record<string, string>
  body: string
}

/**
 * Parses a standard cURL command line string into method, URL, headers, and body.
 */
export function parseCurl(curlCommand: string): ParsedCurl {
  const result: ParsedCurl = {
    method: 'GET',
    url: '',
    headers: {},
    body: '',
  }

  if (!curlCommand || !curlCommand.trim()) {
    return result
  }

  // Normalize newlines and continuation slashes
  const normalized = curlCommand.replace(/\\\r?\n/g, ' ').trim()

  // Tokenize preserving single and double quoted strings
  const tokens: string[] = []
  const tokenRegex = /[^\s"']+|"([^"\\]*(?:\\.[^"\\]*)*)"|'([^'\\]*(?:\\.[^'\\]*)*)'/g
  let match: RegExpExecArray | null

  while ((match = tokenRegex.exec(normalized)) !== null) {
    if (match[1] !== undefined) {
      // Double quoted
      tokens.push(match[1].replace(/\\"/g, '"'))
    } else if (match[2] !== undefined) {
      // Single quoted
      tokens.push(match[2].replace(/\\'/g, "'"))
    } else {
      tokens.push(match[0])
    }
  }

  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i]

    // Method flag
    if ((token === '-X' || token === '--request') && i + 1 < tokens.length) {
      result.method = tokens[i + 1].toUpperCase()
      i++
      continue
    }

    // Header flag
    if ((token === '-H' || token === '--header') && i + 1 < tokens.length) {
      const headerStr = tokens[i + 1]
      const colonIdx = headerStr.indexOf(':')
      if (colonIdx > 0) {
        const key = headerStr.slice(0, colonIdx).trim()
        const val = headerStr.slice(colonIdx + 1).trim()
        result.headers[key] = val
      }
      i++
      continue
    }

    // Body flag
    if (
      (token === '-d' ||
        token === '--data' ||
        token === '--data-raw' ||
        token === '--data-binary' ||
        token === '--data-ascii') &&
      i + 1 < tokens.length
    ) {
      result.body = tokens[i + 1]
      if (result.method === 'GET') {
        result.method = 'POST'
      }
      i++
      continue
    }

    // URL detection (first non-flag token that isn't 'curl')
    if (
      !token.startsWith('-') &&
      token.toLowerCase() !== 'curl' &&
      !result.url
    ) {
      result.url = token
    }
  }

  return result
}
