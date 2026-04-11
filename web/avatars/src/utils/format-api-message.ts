import type { ApiErrorBody } from '../types/avatar'

export function formatApiMessage(body: unknown, fallback: string): string {
  if (!body || typeof body !== 'object') {
    return fallback
  }
  const o = body as ApiErrorBody
  const parts: string[] = []
  if (typeof o.error === 'string') {
    parts.push(o.error)
  }
  if (typeof o.details === 'string') {
    parts.push(o.details)
  }
  if (typeof o.max_size === 'number') {
    parts.push(`max ${o.max_size} bytes`)
  }
  if (parts.length > 0) {
    return parts.join(' - ')
  }
  return fallback
}
