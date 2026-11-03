import React, { useState } from 'react'
import type { Schema } from '../types/spec'
import { ChevronRight, ChevronDown } from 'lucide-react'

interface SchemaViewerProps {
  schema?: Schema
  name?: string
  required?: boolean
  depth?: number
}

export const SchemaViewer: React.FC<SchemaViewerProps> = ({
  schema,
  name,
  required = false,
  depth = 0,
}) => {
  const [expanded, setExpanded] = useState<boolean>(depth < 2)

  if (!schema) {
    return <span className="text-zinc-500 italic text-xs">any</span>
  }

  const schemaType = schema.type || (schema.types ? schema.types.join(' | ') : 'object')
  const isObject = schemaType === 'object' || !!schema.properties
  const isArray = schemaType === 'array' || !!schema.items
  const hasChildren = (isObject && schema.properties && Object.keys(schema.properties).length > 0) || (isArray && schema.items)

  return (
    <div className={`text-xs font-mono ${depth > 0 ? 'ml-3 border-l border-zinc-800/80 pl-2.5 my-1' : ''}`}>
      <div className="flex items-baseline gap-1.5 flex-wrap py-0.5">
        {hasChildren && (
          <button
            onClick={() => setExpanded(!expanded)}
            className="text-zinc-500 hover:text-zinc-300 p-0.5 -ml-1 transition-colors"
          >
            {expanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
          </button>
        )}

        {name && (
          <span className="font-semibold text-zinc-200">
            {name}
            {required && <span className="text-rose-400 ml-0.5">*</span>}
          </span>
        )}

        <span className="text-emerald-400/90 text-[11px] bg-emerald-950/40 px-1 py-0.5 rounded border border-emerald-800/40">
          {schemaType}
          {schema.format && <span className="text-zinc-400 font-normal"> ({schema.format})</span>}
        </span>

        {schema.nullable && (
          <span className="text-zinc-500 text-[10px] bg-zinc-800/50 px-1 rounded">nullable</span>
        )}

        {schema.readOnly && (
          <span className="text-zinc-400 text-[10px] bg-zinc-800/60 px-1 rounded">read-only</span>
        )}

        {schema.enum && (
          <span className="text-amber-400/80 text-[10px] bg-amber-950/30 px-1 py-0.5 rounded border border-amber-900/30">
            enum: [{schema.enum.map(v => JSON.stringify(v)).join(', ')}]
          </span>
        )}

        {schema.description && (
          <span className="text-zinc-400 font-sans text-[11px] ml-1">
            — {schema.description}
          </span>
        )}

        {schema.example !== undefined && (
          <span className="text-zinc-500 text-[11px] font-sans">
            (e.g. <span className="text-zinc-300 font-mono">{JSON.stringify(schema.example)}</span>)
          </span>
        )}
      </div>

      {expanded && hasChildren && (
        <div className="mt-1">
          {isObject && schema.properties && (
            <div>
              {Object.entries(schema.properties).map(([propName, propSchema]) => (
                <SchemaViewer
                  key={propName}
                  name={propName}
                  schema={propSchema}
                  required={schema.required?.includes(propName)}
                  depth={depth + 1}
                />
              ))}
            </div>
          )}

          {isArray && schema.items && (
            <div className="mt-0.5">
              <span className="text-[10px] uppercase tracking-wider text-zinc-500 block mb-0.5">
                items:
              </span>
              <SchemaViewer schema={schema.items} depth={depth + 1} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}
