/**
 * Helpers for mapping LMArena organization slugs to display labels, accent colors,
 * and icon keys shared with ModelIcon / ArenaOrgIcon.
 */

export interface ArenaOrgMeta {
  key: string
  displayName: string
  shortBadge: string
  color: string
  iconKey?:
    | 'claude'
    | 'openai'
    | 'gemini'
    | 'xai'
    | 'deepseek'
    | 'qwen'
    | 'zhipu'
    | 'moonshot'
    | 'meta'
    | 'mistral'
    | 'minimax'
    | 'doubao'
    | 'wenxin'
    | 'hunyuan'
    | 'cohere'
    | 'midjourney'
}

const ORG_MAP: Record<string, ArenaOrgMeta> = {
  google: { key: 'google', displayName: 'Google', shortBadge: 'G', color: '#4285F4', iconKey: 'gemini' },
  anthropic: { key: 'anthropic', displayName: 'Anthropic', shortBadge: 'A', color: '#D97706', iconKey: 'claude' },
  openai: { key: 'openai', displayName: 'OpenAI', shortBadge: 'O', color: '#10A37F', iconKey: 'openai' },
  xai: { key: 'xai', displayName: 'xAI', shortBadge: 'X', color: '#94A3B8', iconKey: 'xai' },
  deepseek: { key: 'deepseek', displayName: 'DeepSeek', shortBadge: 'DS', color: '#4D6BFE', iconKey: 'deepseek' },
  alibaba: { key: 'alibaba', displayName: 'Alibaba', shortBadge: 'Q', color: '#615EFF', iconKey: 'qwen' },
  qwen: { key: 'alibaba', displayName: 'Alibaba', shortBadge: 'Q', color: '#615EFF', iconKey: 'qwen' },
  zai: { key: 'zai', displayName: 'Z.ai', shortBadge: 'Z', color: '#3859FF', iconKey: 'zhipu' },
  zhipu: { key: 'zai', displayName: 'Zhipu AI', shortBadge: 'Z', color: '#3859FF', iconKey: 'zhipu' },
  'zhipu ai': { key: 'zai', displayName: 'Zhipu AI', shortBadge: 'Z', color: '#3859FF', iconKey: 'zhipu' },
  moonshot: { key: 'moonshot', displayName: 'Moonshot', shortBadge: 'K', color: '#6366F1', iconKey: 'moonshot' },
  meta: { key: 'meta', displayName: 'Meta', shortBadge: 'M', color: '#0668E1', iconKey: 'meta' },
  mistral: { key: 'mistral', displayName: 'Mistral', shortBadge: 'M', color: '#F59E0B', iconKey: 'mistral' },
  minimax: { key: 'minimax', displayName: 'MiniMax', shortBadge: 'MM', color: '#F23F5D', iconKey: 'minimax' },
  bytedance: { key: 'bytedance', displayName: 'ByteDance', shortBadge: 'BD', color: '#1C64F2', iconKey: 'doubao' },
  baidu: { key: 'baidu', displayName: 'Baidu', shortBadge: 'B', color: '#167ADF', iconKey: 'wenxin' },
  tencent: { key: 'tencent', displayName: 'Tencent', shortBadge: 'T', color: '#0053E0', iconKey: 'hunyuan' },
  cohere: { key: 'cohere', displayName: 'Cohere', shortBadge: 'C', color: '#39594D', iconKey: 'cohere' },
  midjourney: { key: 'midjourney', displayName: 'Midjourney', shortBadge: 'MJ', color: '#64748B', iconKey: 'midjourney' },
  'microsoft-ai': { key: 'microsoft-ai', displayName: 'Microsoft AI', shortBadge: 'MS', color: '#0EA5E9' },
  microsoft: { key: 'microsoft-ai', displayName: 'Microsoft', shortBadge: 'MS', color: '#0EA5E9' },
  xiaomi: { key: 'xiaomi', displayName: 'Xiaomi', shortBadge: 'MI', color: '#F97316' },
  bfl: { key: 'bfl', displayName: 'Black Forest Labs', shortBadge: 'BF', color: '#8B5CF6' },
  'black-forest-labs': { key: 'bfl', displayName: 'Black Forest Labs', shortBadge: 'BF', color: '#8B5CF6' },
  'black forest labs': { key: 'bfl', displayName: 'Black Forest Labs', shortBadge: 'BF', color: '#8B5CF6' },
  amazon: { key: 'amazon', displayName: 'Amazon', shortBadge: 'AM', color: '#F59E0B' },
  nvidia: { key: 'nvidia', displayName: 'NVIDIA', shortBadge: 'NV', color: '#22C55E' },
  ideogram: { key: 'ideogram', displayName: 'Ideogram', shortBadge: 'ID', color: '#EC4899' },
  recraft: { key: 'recraft', displayName: 'Recraft', shortBadge: 'RC', color: '#14B8A6' },
  reve: { key: 'reve', displayName: 'Reve', shortBadge: 'RV', color: '#A855F7' },
  stepfun: { key: 'stepfun', displayName: 'StepFun', shortBadge: 'SF', color: '#3B82F6' },
  hidream: { key: 'hidream', displayName: 'HiDream', shortBadge: 'HD', color: '#6366F1' },
  krea: { key: 'krea', displayName: 'Krea', shortBadge: 'KR', color: '#0EA5E9' },
  'leonardo-ai': { key: 'leonardo-ai', displayName: 'Leonardo AI', shortBadge: 'LA', color: '#A855F7' },
  'luma-ai': { key: 'luma-ai', displayName: 'Luma AI', shortBadge: 'LU', color: '#38BDF8' },
  runway: { key: 'runway', displayName: 'Runway', shortBadge: 'RW', color: '#14B8A6' },
  reflection: { key: 'reflection', displayName: 'Reflection', shortBadge: 'RF', color: '#6366F1' },
  unisound: { key: 'unisound', displayName: 'Unisound', shortBadge: 'US', color: '#2563EB' },
  thinky: { key: 'thinky', displayName: 'Thinky', shortBadge: 'TH', color: '#8B5CF6' },
  synlan: { key: 'synlan', displayName: 'Synlan', shortBadge: 'SL', color: '#64748B' },
  stability: { key: 'stability', displayName: 'Stability AI', shortBadge: 'ST', color: '#7C3AED' }
}

export function resolveArenaOrgMeta(organization?: string, modelName?: string): ArenaOrgMeta {
  const rawOrg = (organization || '').trim()
  const key = rawOrg.toLowerCase()
  if (key && ORG_MAP[key]) {
    return ORG_MAP[key]
  }

  // Fallback heuristic based on model name if org slug is missing or novel.
  const lowerModel = (modelName || '').toLowerCase()
  if (lowerModel.includes('gemini') || lowerModel.includes('imagen') || lowerModel.includes('gemma')) {
    return ORG_MAP.google
  }
  if (lowerModel.includes('claude')) {
    return ORG_MAP.anthropic
  }
  if (lowerModel.includes('gpt') || lowerModel.startsWith('o1') || lowerModel.startsWith('o3') || lowerModel.startsWith('o4')) {
    return ORG_MAP.openai
  }
  if (lowerModel.includes('grok')) {
    return ORG_MAP.xai
  }
  if (lowerModel.includes('deepseek')) {
    return ORG_MAP.deepseek
  }
  if (lowerModel.includes('qwen')) {
    return ORG_MAP.alibaba
  }
  if (lowerModel.includes('glm')) {
    return ORG_MAP.zai
  }
  if (lowerModel.includes('kimi')) {
    return ORG_MAP.moonshot
  }
  if (lowerModel.includes('flux')) {
    return ORG_MAP.bfl
  }
  if (lowerModel.includes('stable diffusion') || lowerModel.includes('sdxl')) {
    return ORG_MAP.stability
  }

  const label = rawOrg
    ? rawOrg
        .split(/[-_\s]+/)
        .filter(Boolean)
        .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
        .join(' ')
    : 'Independent'

  const shortBadge = label.slice(0, 2).toUpperCase()
  return {
    key: key || 'independent',
    displayName: label,
    shortBadge,
    color: '#64748B'
  }
}

export function formatArenaNumber(val?: number | null): string {
  if (val == null || Number.isNaN(val)) return '—'
  return Math.round(val).toLocaleString('en-US')
}

export function formatArenaScore(val?: number | null): string {
  if (val == null || Number.isNaN(val)) return '—'
  return Math.round(val).toString()
}

export function formatArenaExactScore(val?: number | null): string {
  if (val == null || Number.isNaN(val)) return '—'
  return val.toFixed(1)
}

export function formatScoreDelta(val?: number | null): string {
  if (val == null || Number.isNaN(val) || Math.abs(val) < 0.05) return ''
  const sign = val > 0 ? '+' : ''
  return `${sign}${val.toFixed(1)}`
}
