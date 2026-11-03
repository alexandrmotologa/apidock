export interface Spec {
  openapi: string
  info: Info
  servers: Server[]
  tags: Tag[]
  endpoints: Endpoint[]
  components: Components
  errors?: string[]
  warnings?: string[]
}

export interface Info {
  title: string
  description?: string
  version: string
  termsOfService?: string
  contact?: {
    name?: string
    url?: string
    email?: string
  }
  license?: {
    name: string
    url?: string
  }
}

export interface Server {
  url: string
  description?: string
  variables?: Record<string, {
    default: string
    description?: string
    enum?: string[]
  }>
}

export interface Tag {
  name: string
  description?: string
}

export interface Endpoint {
  id: string
  path: string
  method: string
  summary?: string
  description?: string
  operationId?: string
  tags?: string[]
  deprecated?: boolean
  security?: Array<Record<string, string[]>>
  parameters?: Parameter[]
  requestBody?: RequestBody
  responses?: Record<string, Response>
}

export interface Parameter {
  name: string
  in: 'path' | 'query' | 'header' | 'cookie'
  description?: string
  required: boolean
  deprecated?: boolean
  schema?: Schema
  example?: unknown
}

export interface RequestBody {
  description?: string
  required: boolean
  content: Record<string, MediaType>
}

export interface Response {
  statusCode: string
  description: string
  headers?: Record<string, Parameter>
  content?: Record<string, MediaType>
}

export interface MediaType {
  schema?: Schema
  example?: unknown
  examples?: Record<string, Example>
}

export interface Example {
  summary?: string
  description?: string
  value?: unknown
}

export interface Schema {
  type?: string
  types?: string[]
  format?: string
  title?: string
  description?: string
  default?: unknown
  example?: unknown
  examples?: unknown[]
  enum?: unknown[]
  nullable?: boolean
  readOnly?: boolean
  writeOnly?: boolean
  minimum?: number
  maximum?: number
  exclusiveMinimum?: unknown
  exclusiveMaximum?: unknown
  minLength?: number
  maxLength?: number
  pattern?: string
  minItems?: number
  maxItems?: number
  uniqueItems?: boolean
  required?: string[]
  properties?: Record<string, Schema>
  items?: Schema
  additionalProperties?: unknown
  oneOf?: Schema[]
  anyOf?: Schema[]
  allOf?: Schema[]
  $ref?: string
}

export interface Components {
  schemas?: Record<string, Schema>
  securitySchemes?: Record<string, SecurityScheme>
}

export interface SecurityScheme {
  type: string
  description?: string
  name?: string
  in?: string
  scheme?: string
  bearerFormat?: string
  openIdConnectUrl?: string
}

export interface TimingMetrics {
  dnsLookupMs: number
  tcpConnectMs: number
  tlsHandshakeMs: number
  ttfbMs: number
  totalMs: number
  sizeBytes: number
}

export interface ProxyResponse {
  status: number
  statusText: string
  headers: Record<string, string>
  body: string
  timing: TimingMetrics
  error?: string
}

export interface HistoryEntry {
  id: string
  timestamp: string
  request: {
    url: string
    method: string
    headers?: Record<string, string>
    body?: string
  }
  response: ProxyResponse
}

export interface Environment {
  id: string
  name: string
  baseUrl: string
  variables: Record<string, string>
  isEncrypted?: boolean
  cipherData?: string
}
