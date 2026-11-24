import { describe, expect, it } from 'vitest'
import {
  announcementImageBytes,
  extractAnnouncementPosterImage,
  isAnnouncementImageSource,
  joinAnnouncementImages,
  parseSafeDashboardLink,
  splitAnnouncementImages,
} from '../announcementImages'

describe('announcement image persistence', () => {
  it('keeps legacy Markdown, HTML and whitespace unchanged', () => {
    const body = '\n## 原公告\n\n<div>内容</div>\n![old](/brand/logo.png)\n'
    expect(splitAnnouncementImages(body)).toEqual({ body, images: [] })
    expect(joinAnnouncementImages(body, [])).toBe(body)
  })

  it('round trips original bytes, descriptions and image order without putting data URLs into the body editor', () => {
    const body = '正文\n\n继续'
    const images = [{ src: 'data:image/png;base64,YWJjZA==', alt: '图片 "A" <说明> & 更多' }, { src: 'https://gabugu.com/a.png?x=1&y=2', alt: '第二张' }]
    const saved = joinAnnouncementImages(body, images)
    expect(splitAnnouncementImages(saved)).toEqual({ body, images })
    expect(announcementImageBytes(images)).toBe(4)
    expect(saved).not.toContain('alt="图片 "A"')
  })

  it('retains unrecognized or malformed attachment content rather than deleting it', () => {
    const content = '正文\n\n<!-- gabugu-announcement-images:v1 -->\n<p>其他内容</p>\n<!-- /gabugu-announcement-images -->'
    expect(splitAnnouncementImages(content)).toEqual({ body: content, images: [] })
  })

  it('recognizes persisted Windows newlines without rewriting the original body', () => {
    const content = joinAnnouncementImages('first\n\nsecond', [{src:'/poster.png',alt:'Poster'}]).replace(/\n/g,'\r\n')
    expect(splitAnnouncementImages(content)).toEqual({body:'first\r\n\r\nsecond',images:[{src:'/poster.png',alt:'Poster'}]})
  })

  it.each(['javascript:alert(1)', 'data:image/svg+xml;base64,YWJjZA==', 'http://example.com/image.png', '//example.com/image.png', 'https://user:pass@example.com/i.png', '/\\evil.com/i.png'])('rejects unsafe attachment URL %s', src => {
    expect(isAnnouncementImageSource(src)).toBe(false)
    expect(() => joinAnnouncementImages('body', [{ src, alt: 'x' }])).toThrow()
  })

  it('extracts dashboard poster image from attachment block, Markdown, or safe HTML', () => {
    const withBlock = joinAnnouncementImages('![fallback](https://gabugu.com/fb.png)', [{ src: 'https://gabugu.com/poster.png', alt: 'Banner' }])
    expect(extractAnnouncementPosterImage(withBlock)).toEqual({ src: 'https://gabugu.com/poster.png', alt: 'Banner' })

    expect(extractAnnouncementPosterImage('Hello\n![Promo](https://gabugu.com/md.png)')).toEqual({
      src: 'https://gabugu.com/md.png',
      alt: 'Promo',
    })

    expect(extractAnnouncementPosterImage('<p><img src="javascript:alert(1)" /><img src="/brand/promo.png" alt="Promo" /></p>')).toEqual({
      src: '/brand/promo.png',
      alt: 'Promo',
    })

    expect(extractAnnouncementPosterImage('Plain text only')).toBeNull()
  })

  it('validates safe dashboard poster links and rejects XSS / open-redirect / http payloads', () => {
    expect(parseSafeDashboardLink('/purchase?amount=100')).toEqual({ kind: 'internal', href: '/purchase?amount=100' })
    expect(parseSafeDashboardLink('https://gabugu.com/promo')).toEqual({ kind: 'external', href: 'https://gabugu.com/promo' })
    expect(parseSafeDashboardLink('')).toBeNull()
    expect(parseSafeDashboardLink('   ')).toBeNull()
    expect(parseSafeDashboardLink('javascript:alert(1)')).toBeNull()
    expect(parseSafeDashboardLink('data:text/html,<script>alert(1)</script>')).toBeNull()
    expect(parseSafeDashboardLink('http://gabugu.com/insecure')).toBeNull()
    expect(parseSafeDashboardLink('//evil.com/phish')).toBeNull()
    expect(parseSafeDashboardLink('/\\evil.com')).toBeNull()
    expect(parseSafeDashboardLink('https://user:pass@example.com')).toBeNull()
    expect(parseSafeDashboardLink('/' + 'a'.repeat(501))).toBeNull()
  })
})
