import { describe, expect, it } from 'vitest'
import { authRoute, sanitizeAuthRedirect } from '../authRedirect'

describe('authentication return path validation', () => {
  it.each([
    '/keys', '/usage?model=GPT%2BClaude&range=week#records',
    '/oauth/mcp/authorize?redirect_uri=http%3A%2F%2F127.0.0.1%3A54321%2Fcallback&state=a%2Bb%26c',
    '/oauth/mcp/authorize?redirect_uri=http://127.0.0.1:54321/callback&state=a%2Bb%26c'
  ])('retains the complete internal path %s', target => {
    expect(sanitizeAuthRedirect(target)).toBe(target)
    expect(authRoute('/register', target)).toEqual({ path: '/register', query: { redirect: target } })
  })

  it.each([
    undefined, null, '', 'https://outside.example', '//outside.example', '/\\outside.example',
    '/%2foutside.example', '/%5coutside.example', '/usage\n', '/usage%0a', '/%zz',
    '/login?redirect=/keys', '/register', '/email-verify', '/forgot-password', '/reset-password',
    '/auth/oauth/callback', '/setup', '/a/../login', '/%6cogin', '/' + 'a'.repeat(2048),
    ['/keys', '/usage']
  ].map(target => ({ target })))('rejects external, malformed or looping path $target', ({ target }) => {
    expect(sanitizeAuthRedirect(target)).toBeUndefined()
    expect(authRoute('/login', target)).toBe('/login')
  })
})
