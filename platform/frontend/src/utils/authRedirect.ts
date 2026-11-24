import type { RouteLocationRaw } from 'vue-router'

const AUTH_ENTRY_PATH = /^\/(?:login|register|email-verify|forgot-password|reset-password|setup)(?:\/|$)|^\/auth(?:\/|$)/i
function hasUnsafePathCharacters(value: string): boolean {
  return [...value].some(character => {
    const code = character.charCodeAt(0)
    return character === '\\' || code <= 0x20 || (code >= 0x7f && code <= 0x9f)
  })
}

/** Keep the original query and fragment, including MCP consent parameters. */
export function sanitizeAuthRedirect(value: unknown): string | undefined {
  if (typeof value !== 'string' || value.length > 2048 || !value.startsWith('/') ||
      value.startsWith('//') || hasUnsafePathCharacters(value)) return undefined
  try {
    const url = new URL(value, 'https://auth-return.invalid')
    const path = decodeURIComponent(url.pathname)
    if (url.origin !== 'https://auth-return.invalid' || path.startsWith('//') ||
        hasUnsafePathCharacters(path) || AUTH_ENTRY_PATH.test(path)) return undefined
    return value
  } catch {
    return undefined
  }
}

export function authRoute(path: string, redirect: unknown): RouteLocationRaw {
  const target = sanitizeAuthRedirect(redirect)
  return target ? { path, query: { redirect: target } } : path
}
