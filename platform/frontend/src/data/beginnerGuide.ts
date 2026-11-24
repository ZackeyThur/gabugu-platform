import type { PublicSettings } from '@/types'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'

export type GuideClient = 'claude' | 'codex'
export type GuideOs = 'windows' | 'macos'
export const guideChapters = ['basics', 'setup', 'resources', 'faq'] as const
export const guideSteps = ['prepare', 'choose', 'installSwitch', 'installClient', 'connect', 'launch', 'verify'] as const
export const guideDesktopSteps = ['prepare', 'installClient', 'installSwitch', 'connect', 'launch', 'verify'] as const
export const codexDesktopResources = {
  download: 'https://chatgpt.com/download/',
  setup: 'https://learn.chatgpt.com/docs/quickstart?setup=app',
  gateway: 'https://learn.chatgpt.com/docs/enterprise/connect-to-a-gateway',
} as const
export const guideTerms = ['account', 'key', 'address', 'model', 'token', 'credit'] as const
export const guideFaqs = ['key', 'tools', 'command', 'import', 'login', 'model', 'credit', 'errors'] as const

export const guideResources = [
  { id: 'ccswitch', mark: 'CC', href: 'https://github.com/farion1231/cc-switch/releases/latest', action: 'download' },
  { id: 'ccswitchManual', mark: '↗', href: 'https://github.com/farion1231/cc-switch/blob/main/docs/user-manual/zh/README.md', action: 'manual' },
  { id: 'claude', mark: '✳', href: 'https://code.claude.com/docs/zh-CN/setup', action: 'officialDocs' },
  { id: 'codex', mark: '>_', href: codexDesktopResources.download, action: 'download' },
] as const

export function detectGuideOs(platform: string): GuideOs {
  return /mac/i.test(platform) ? 'macos' : 'windows'
}

export function guideLaunchCommands(os: GuideOs, client: GuideClient): string | null {
  // Desktop Codex is opened from the application launcher, never a terminal command.
  if (client === 'codex') return null
  // A new named practice folder keeps the first session separate from an existing project.
  const folder = os === 'windows'
    ? 'New-Item -ItemType Directory -Force "$HOME/gabugu-start" | Out-Null\nSet-Location "$HOME/gabugu-start"'
    : 'mkdir -p ~/gabugu-start\ncd ~/gabugu-start'
  return `${folder}\n${client}`
}

export function guideHashTarget(hash: string): string | undefined {
  return guideChapters.some(chapter => hash === `#${chapter}`) ? hash : undefined
}

export interface GuideInternalLink {
  id: string
  path: string
  label: string
  icon: 'key' | 'grid' | 'chart' | 'creditCard' | 'home'
}

/** Match the existing personal sidebar's visibility, including simple mode. */
export function getGuideLinks(settings: Partial<PublicSettings> | null | undefined, simple: boolean): GuideInternalLink[] {
  const links: GuideInternalLink[] = [
    { id: 'dashboard', path: '/dashboard', label: 'dashboard', icon: 'home' },
    { id: 'keys', path: '/keys', label: 'keys', icon: 'key' },
  ]
  if (!simple) {
    if (resolveFeatureFlag(settings, FeatureFlags.modelPlaza)) {
      links.push({ id: 'modelPlaza', path: '/model-plaza?embedded=1', label: 'modelPlaza', icon: 'grid' })
    }
    if (resolveFeatureFlag(settings, FeatureFlags.availableChannels)) {
      links.push({ id: 'channels', path: '/available-channels', label: 'channels', icon: 'grid' })
    }
    links.push({ id: 'usage', path: '/usage', label: 'usage', icon: 'chart' })
    if (resolveFeatureFlag(settings, FeatureFlags.payment)) {
      const mode = resolveSiteBillingMode(settings)
      links.push({ id: 'purchase', path: '/purchase', label: mode === 'recharge_only' ? 'purchaseRecharge' : mode === 'subscription_only' ? 'purchaseSubscription' : 'purchaseBoth', icon: 'creditCard' })
    }
    if (resolveFeatureFlag(settings, FeatureFlags.subscription)) {
      links.push({ id: 'subscriptions', path: '/subscriptions', label: 'subscriptions', icon: 'creditCard' })
    }
  }
  if (resolveFeatureFlag(settings, FeatureFlags.channelMonitor)) {
    links.push({ id: 'monitor', path: '/monitor', label: 'monitor', icon: 'chart' })
  }
  return links
}
