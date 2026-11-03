import type { Schema } from '../types/spec'

export interface ContractIssue {
  path: string
  severity: 'ERROR' | 'WARNING'
  message: string
}

export interface ContractValidationResult {
  isValid: boolean
  issues: ContractIssue[]
  conformityScore: number // 0 to 100
}

const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/**
 * Validates a response JSON payload against its OpenAPI Schema definition.
 */
export function validateContract(
  data: unknown,
  schema?: Schema,
  path = '$'
): ContractValidationResult {
  const issues: ContractIssue[] = []

  if (!schema) {
    return { isValid: true, issues: [], conformityScore: 100 }
  }

  // Nullable check
  if (data === null) {
    if (!schema.nullable) {
      issues.push({
        path,
        severity: 'ERROR',
        message: 'Value is null but schema does not declare nullable: true',
      })
    }
    return {
      isValid: issues.length === 0,
      issues,
      conformityScore: issues.length === 0 ? 100 : 0,
    }
  }

  if (data === undefined) {
    issues.push({
      path,
      severity: 'ERROR',
      message: 'Value is undefined',
    })
    return { isValid: false, issues, conformityScore: 0 }
  }

  const expectedType = schema.type || (schema.types ? schema.types[0] : undefined)

  // Type verification
  if (expectedType) {
    if (expectedType === 'string' && typeof data !== 'string') {
      issues.push({
        path,
        severity: 'ERROR',
        message: `Expected type string, but received ${typeof data}`,
      })
    } else if (expectedType === 'integer') {
      if (typeof data !== 'number' || !Number.isInteger(data)) {
        issues.push({
          path,
          severity: 'ERROR',
          message: `Expected integer number, but received ${typeof data}`,
        })
      }
    } else if (expectedType === 'number' && typeof data !== 'number') {
      issues.push({
        path,
        severity: 'ERROR',
        message: `Expected number, but received ${typeof data}`,
      })
    } else if (expectedType === 'boolean' && typeof data !== 'boolean') {
      issues.push({
        path,
        severity: 'ERROR',
        message: `Expected boolean, but received ${typeof data}`,
      })
    } else if (expectedType === 'array' && !Array.isArray(data)) {
      issues.push({
        path,
        severity: 'ERROR',
        message: `Expected array, but received ${typeof data}`,
      })
    } else if (
      expectedType === 'object' &&
      (typeof data !== 'object' || Array.isArray(data) || data === null)
    ) {
      issues.push({
        path,
        severity: 'ERROR',
        message: `Expected object, but received ${Array.isArray(data) ? 'array' : typeof data}`,
      })
    }
  }

  // Format checks for strings
  if (typeof data === 'string' && schema.format) {
    if (schema.format === 'uuid' && !UUID_REGEX.test(data)) {
      issues.push({
        path,
        severity: 'WARNING',
        message: `Value "${data}" does not match format uuid`,
      })
    } else if (schema.format === 'email' && !EMAIL_REGEX.test(data)) {
      issues.push({
        path,
        severity: 'WARNING',
        message: `Value "${data}" does not match format email`,
      })
    } else if (schema.format === 'date-time' && isNaN(Date.parse(data))) {
      issues.push({
        path,
        severity: 'WARNING',
        message: `Value "${data}" is not a valid ISO date-time string`,
      })
    }
  }

  // Enum checks
  if (schema.enum && schema.enum.length > 0) {
    if (!schema.enum.includes(data)) {
      issues.push({
        path,
        severity: 'ERROR',
        message: `Value "${String(data)}" is not in declared enum [${schema.enum.join(', ')}]`,
      })
    }
  }

  // Object properties & required fields
  if (
    typeof data === 'object' &&
    data !== null &&
    !Array.isArray(data)
  ) {
    const obj = data as Record<string, unknown>

    // Check required fields
    if (schema.required) {
      for (const reqProp of schema.required) {
        if (obj[reqProp] === undefined) {
          issues.push({
            path: `${path}.${reqProp}`,
            severity: 'ERROR',
            message: `Required property "${reqProp}" is missing in response`,
          })
        }
      }
    }

    // Recursively check known properties
    if (schema.properties) {
      for (const [propName, propSchema] of Object.entries(schema.properties)) {
        if (obj[propName] !== undefined) {
          const subResult = validateContract(
            obj[propName],
            propSchema,
            `${path}.${propName}`
          )
          issues.push(...subResult.issues)
        }
      }
    }
  }

  // Array items check
  if (Array.isArray(data) && schema.items) {
    data.slice(0, 10).forEach((item, idx) => {
      const subResult = validateContract(item, schema.items, `${path}[${idx}]`)
      issues.push(...subResult.issues)
    })
  }

  // Calculate score
  const errorCount = issues.filter((i) => i.severity === 'ERROR').length
  const warningCount = issues.filter((i) => i.severity === 'WARNING').length
  const penalty = errorCount * 25 + warningCount * 10
  const conformityScore = Math.max(0, 100 - penalty)

  return {
    isValid: errorCount === 0,
    issues,
    conformityScore,
  }
}
