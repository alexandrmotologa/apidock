/**
 * Interpolates variables formatted as {{varName}} in URLs, headers, and request bodies.
 */
export function interpolate(template: string, vars: Record<string, string>): string {
  if (!template) return ''
  return template.replace(/\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}/g, (_, key) => {
    return vars[key] !== undefined ? vars[key] : `{{${key}}}`
  })
}

export function extractTemplateVariables(template: string): string[] {
  const matches = template.matchAll(/\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}/g)
  const vars = new Set<string>()
  for (const m of matches) {
    if (m[1]) {
      vars.add(m[1])
    }
  }
  return Array.from(vars)
}
