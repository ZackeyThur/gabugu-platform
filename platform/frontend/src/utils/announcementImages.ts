export interface AnnouncementImage {
  src: string
  alt: string
}

export const MAX_ANNOUNCEMENT_IMAGES = 4
export const MAX_ANNOUNCEMENT_IMAGE_BYTES = 5 * 1024 * 1024
export const MAX_ANNOUNCEMENT_IMAGE_TOTAL_BYTES = 8 * 1024 * 1024
export const ANNOUNCEMENT_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/webp', 'image/gif']

const START = '<!-- gabugu-announcement-images:v1 -->'
const END = '<!-- /gabugu-announcement-images -->'

export function isAnnouncementImageSource(src: string): boolean {
  if (/^data:image\/(?:png|jpeg|webp|gif);base64,[A-Za-z0-9+/]+={0,2}$/.test(src)) return true
  if (/^\/(?!\/)[^\s<>"'\\]+$/.test(src)) return true
  try {
    const url = new URL(src)
    return url.protocol === 'https:' && !url.username && !url.password
  } catch {
    return false
  }
}

function escapeAttribute(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// Keep old Markdown/HTML byte-for-byte; only our complete trailing block is managed here.
export function splitAnnouncementImages(content: string): { body: string; images: AnnouncementImage[] } {
  const marker = content.lastIndexOf(START)
  const prefix = content.slice(0, marker)
  const spacing = prefix.endsWith('\r\n\r\n') ? 4 : prefix.endsWith('\n\n') ? 2 : 0
  if (marker < 0 || !spacing || !content.endsWith(END)) return { body: content, images: [] }
  const start = marker - spacing
  const html = content.slice(marker + START.length, -END.length)
  const document = new DOMParser().parseFromString(html, 'text/html')
  const nodes = Array.from(document.body.childNodes).filter(node => node.nodeType !== 3 || node.textContent?.trim())
  if (!nodes.length || nodes.some(node => !(node instanceof HTMLImageElement))) return { body: content, images: [] }
  if ((nodes as HTMLImageElement[]).some(image => Array.from(image.attributes).some(attribute => !['src', 'alt'].includes(attribute.name)))) return { body: content, images: [] }
  const images = (nodes as HTMLImageElement[]).map(image => ({ src: image.getAttribute('src') || '', alt: image.alt }))
  if (images.some(image => !isAnnouncementImageSource(image.src))) return { body: content, images: [] }
  return { body: content.slice(0, start), images }
}

export function joinAnnouncementImages(body: string, images: AnnouncementImage[]): string {
  if (!images.length) return body
  if (images.some(image => !isAnnouncementImageSource(image.src))) throw new Error('Invalid announcement image source')
  const html = images.map(image => `<img src="${escapeAttribute(image.src)}" alt="${escapeAttribute(image.alt)}">`).join('\n')
  return `${body}\n\n${START}\n${html}\n${END}`
}

export function announcementImageBytes(images: AnnouncementImage[]): number {
  return images.reduce((sum, image) => {
    if (!image.src.startsWith('data:')) return sum
    const base64 = image.src.slice(image.src.indexOf(',') + 1)
    return sum + base64.length * 3 / 4 - (base64.endsWith('==') ? 2 : base64.endsWith('=') ? 1 : 0)
  }, 0)
}

/**
 * Extracts the primary poster image from announcement content.
 * Prefers the structured trailing image block; falls back to safe inline Markdown/HTML images.
 */
export function extractAnnouncementPosterImage(content: string | null | undefined): AnnouncementImage | null {
  if (!content) return null
  const { body, images } = splitAnnouncementImages(content)
  for (const image of images) {
    if (isAnnouncementImageSource(image.src)) {
      return image
    }
  }
  for (const mdMatch of body.matchAll(/!\[([^\]]*)\]\(([^)\s]+)\)/g)) {
    const src = mdMatch[2].trim()
    if (isAnnouncementImageSource(src)) {
      return { src, alt: mdMatch[1].trim() }
    }
  }
  for (const imgTagMatch of body.matchAll(/<img\b[^>]*\bsrc=["']([^"']+)["'][^>]*>/gi)) {
    const src = imgTagMatch[1].trim()
    if (isAnnouncementImageSource(src)) {
      const altMatch = /\balt=["']([^"']*)["']/i.exec(imgTagMatch[0])
      return { src, alt: altMatch?.[1]?.trim() ?? '' }
    }
  }
  return null
}

export interface SafeDashboardLink {
  kind: 'internal' | 'external'
  href: string
}

/**
 * Defense-in-depth validation for dashboard poster click links.
 * Allows internal paths starting with single `/` or standard `https://` URLs without credentials.
 */
export function parseSafeDashboardLink(raw: string | null | undefined): SafeDashboardLink | null {
  if (!raw) return null
  const trimmed = raw.trim()
  if (!trimmed || trimmed.length > 500) return null
  if (/[\s\\]/.test(trimmed)) return null
  for (let i = 0; i < trimmed.length; i++) {
    const code = trimmed.charCodeAt(i)
    if (code <= 0x1f || code === 0x7f) return null
  }

  if (trimmed.startsWith('/')) {
    if (trimmed.startsWith('//')) return null
    return { kind: 'internal', href: trimmed }
  }

  try {
    const parsed = new URL(trimmed)
    if (parsed.protocol !== 'https:' || !parsed.hostname || parsed.username || parsed.password) {
      return null
    }
    return { kind: 'external', href: trimmed }
  } catch {
    return null
  }
}
