import { describe, expect, it } from 'vitest'
import { detectGuideOs, getGuideLinks, guideHashTarget, guideLaunchCommands } from '../beginnerGuide'

describe('beginner guide entry points', () => {
  it('follows disabled features and the recharge-only billing mode', () => {
    const links = getGuideLinks({ available_channels_enabled: false, channel_monitor_enabled: false, subscription_enabled: false }, false)
    expect(links.map(link => link.path)).toEqual(['/dashboard', '/keys', '/usage', '/purchase'])
    expect(links.find(link => link.id === 'purchase')?.label).toBe('purchaseRecharge')
  })
  it('shows subscription wording and removes payment when disabled', () => {
    expect(getGuideLinks({ payment_balance_disabled: true }, false).find(link => link.id === 'purchase')?.label).toBe('purchaseSubscription')
    expect(getGuideLinks({ payment_enabled: false }, false).some(link => link.id === 'purchase')).toBe(false)
  })
  it('respects simple mode and default feature visibility before settings arrive', () => {
    expect(getGuideLinks(null, true).map(link => link.path)).toEqual(['/dashboard', '/keys', '/monitor'])
    expect(getGuideLinks(undefined, false).some(link => link.id === 'channels')).toBe(false)
    expect(getGuideLinks(undefined, false).some(link => link.id === 'modelPlaza')).toBe(false)
    expect(getGuideLinks({ available_channels_enabled: true }, false).some(link => link.id === 'channels')).toBe(true)
    expect(getGuideLinks({ model_plaza_enabled: true }, false).find(link => link.id === 'modelPlaza')?.path).toBe('/model-plaza?embedded=1')
    expect(getGuideLinks({ model_plaza_enabled: true }, true).some(link => link.id === 'modelPlaza')).toBe(false)
  })
  it('accepts only known chapter hashes', () => {
    expect(guideHashTarget('#setup')).toBe('#setup')
    expect(guideHashTarget('#[bad-selector')).toBeUndefined()
    expect(guideHashTarget('#anything')).toBeUndefined()
  })
})

describe('beginner guide command selection', () => {
  it('keeps Claude Code commands in the selected shell', () => {
    const windows = guideLaunchCommands('windows', 'claude')!
    const macos = guideLaunchCommands('macos', 'claude')!
    expect(windows).toContain('Set-Location "$HOME/gabugu-start"')
    expect(windows).not.toContain('mkdir -p')
    expect(macos).toContain('mkdir -p ~/gabugu-start')
    expect(macos).not.toContain('New-Item')
    expect(windows.split('\n').at(-1)).toBe('claude')
    expect(macos.split('\n').at(-1)).toBe('claude')
  })
  it('never generates a terminal launch command for desktop Codex', () => {
    expect(guideLaunchCommands('windows', 'codex')).toBeNull()
    expect(guideLaunchCommands('macos', 'codex')).toBeNull()
  })
  it('detects Mac and uses Windows for unrecognized platforms', () => {
    expect(detectGuideOs('MacIntel')).toBe('macos')
    expect(detectGuideOs('Win32')).toBe('windows')
    expect(detectGuideOs('Linux x86_64')).toBe('windows')
  })
})
