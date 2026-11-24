/**
 * GABUGU 画室 — 分组选择、实时按次计价估算与 USD 金额格式化。
 *
 * 与后端 `service/billing_service.go` (`CalculateImageCost`) 和
 * `service/image_billing_multiplier.go` (`resolveImageRateMultiplier`) 保持一致。
 */

import type { Group } from '@/types'
import { classifyImageBillingTier, type StudioTierId } from './models'

export interface StudioCostEstimate {
  known: boolean
  tier: StudioTierId
  unitPrice: number
  totalPrice: number
  multiplier: number
}

export interface StudioCostRange {
  known: boolean
  minUnitPrice: number
  maxUnitPrice: number
  minTotalPrice: number
  maxTotalPrice: number
}

function isDedicatedImageGroupName(name: string): boolean {
  const lower = name.trim().toLowerCase()
  return lower.includes('图像') || lower.includes('图片') || lower.includes('image') || lower.includes('画')
}

/**
 * 在用户可用分组中挑选最适合画室生图的 OpenAI 兼容图像分组。
 * 优先级：
 * 1. status === 'active' 且 platform === 'openai' 且 allow_image_generation === true
 * 2. 优先启用按次计费 (`image_count_billing === true`)
 * 3. 优先名称匹配图像专用分组（例如线上 Group 7「图像生成」）
 * 4. 其次按 ID 升序保持确定性
 */
export function selectStudioImageGroup(groups: Group[]): Group | null {
  const eligible = groups.filter(
    (g) => g.status === 'active' && g.platform === 'openai' && g.allow_image_generation === true
  )
  if (!eligible.length) return null

  const sorted = [...eligible].sort((a, b) => {
    const aCount = a.image_count_billing ? 1 : 0
    const bCount = b.image_count_billing ? 1 : 0
    if (aCount !== bCount) return bCount - aCount

    const aStandard = a.subscription_type !== 'subscription' ? 1 : 0
    const bStandard = b.subscription_type !== 'subscription' ? 1 : 0
    if (aStandard !== bStandard) return bStandard - aStandard

    const aNamed = isDedicatedImageGroupName(a.name) ? 1 : 0
    const bNamed = isDedicatedImageGroupName(b.name) ? 1 : 0
    if (aNamed !== bNamed) return bNamed - aNamed

    const aHasPrices = a.image_price_2k != null ? 1 : 0
    const bHasPrices = b.image_price_2k != null ? 1 : 0
    if (aHasPrices !== bHasPrices) return bHasPrices - aHasPrices

    return a.id - b.id
  })

  return sorted[0] ?? null
}

/**
 * 计算用户在该分组下的有效图片计费倍率（与后端 image_billing_multiplier.go 一致）。
 */
export function resolveEffectiveImageMultiplier(
  group: Group,
  userGroupRates: Record<number, number> = {}
): number {
  if (group.image_rate_independent) {
    const raw = Number(group.image_rate_multiplier)
    if (!Number.isFinite(raw)) return 1
    return raw < 0 ? 0 : raw
  }
  const userRate = userGroupRates[group.id]
  if (typeof userRate === 'number' && Number.isFinite(userRate)) {
    return userRate < 0 ? 0 : userRate
  }
  const groupRate = Number(group.rate_multiplier)
  if (!Number.isFinite(groupRate)) return 1
  return groupRate < 0 ? 0 : groupRate
}

function configuredGroupTierPrice(group: Group, tier: StudioTierId): number | null {
  const raw =
    tier === '1K'
      ? group.image_price_1k
      : tier === '4K'
        ? group.image_price_4k
        : group.image_price_2k
  if (typeof raw === 'number' && Number.isFinite(raw) && raw >= 0) {
    return raw
  }
  return null
}

/**
 * 估算单次调用预计费用（USD）。
 * 若分组未开启按次计费或未配置对应档位单价，则诚实返回 `known: false`。
 */
export function estimateStudioCost(
  group: Group | null,
  userGroupRates: Record<number, number>,
  apiSize: string,
  count: number
): StudioCostEstimate {
  const tier = classifyImageBillingTier(apiSize)
  const safeCount = Math.max(1, Math.floor(count) || 1)
  if (!group || !group.image_count_billing) {
    return { known: false, tier, unitPrice: 0, totalPrice: 0, multiplier: 1 }
  }
  const basePrice = configuredGroupTierPrice(group, tier)
  if (basePrice == null) {
    return { known: false, tier, unitPrice: 0, totalPrice: 0, multiplier: 1 }
  }
  const multiplier = resolveEffectiveImageMultiplier(group, userGroupRates)
  const unitPrice = roundMicro(basePrice * multiplier)
  const totalPrice = roundMicro(unitPrice * safeCount)
  return { known: true, tier, unitPrice, totalPrice, multiplier }
}

/**
 * 返回模型在当前分组下的最低起价（按模型实际支持的清晰度档位取最小值）。
 */
export function estimateModelMinPrice(
  group: Group | null,
  userGroupRates: Record<number, number>,
  tiers: StudioTierId[]
): number | null {
  const range = estimateStudioCostRange(group, userGroupRates, tiers, 1)
  return range.known ? range.minUnitPrice : null
}

/**
 * 返回模型在当前分组下跨支持档位的单次费用区间（用于 auto 自适应比例的透明预估）。
 */
export function estimateStudioCostRange(
  group: Group | null,
  userGroupRates: Record<number, number>,
  tiers: StudioTierId[],
  count: number
): StudioCostRange {
  const safeCount = Math.max(1, Math.floor(count) || 1)
  if (!group || !group.image_count_billing || !tiers.length) {
    return { known: false, minUnitPrice: 0, maxUnitPrice: 0, minTotalPrice: 0, maxTotalPrice: 0 }
  }
  let minVal: number | null = null
  let maxVal: number | null = null
  for (const tier of tiers) {
    const est = estimateStudioCost(group, userGroupRates, tier, 1)
    if (est.known) {
      if (minVal == null || est.unitPrice < minVal) minVal = est.unitPrice
      if (maxVal == null || est.unitPrice > maxVal) maxVal = est.unitPrice
    }
  }
  if (minVal == null || maxVal == null) {
    return { known: false, minUnitPrice: 0, maxUnitPrice: 0, minTotalPrice: 0, maxTotalPrice: 0 }
  }
  return {
    known: true,
    minUnitPrice: minVal,
    maxUnitPrice: maxVal,
    minTotalPrice: roundMicro(minVal * safeCount),
    maxTotalPrice: roundMicro(maxVal * safeCount),
  }
}

function roundMicro(n: number): number {
  return Math.round(n * 1_000_000) / 1_000_000
}

/**
 * 格式化图片单次/预计费用（保留 3 位小数若含千分位，如 $0.189 / $0.268 / $0.371；否则 2 位）。
 */
export function formatStudioUsd(amount: number): string {
  if (!Number.isFinite(amount) || amount < 0) return '$0.00'
  const rounded3 = Math.round(amount * 1000) / 1000
  const hasMilli = Math.abs(rounded3 * 100 - Math.round(rounded3 * 100)) > 1e-6
  return `$${rounded3.toFixed(hasMilli ? 3 : 2)}`
}

/**
 * 格式化账户余额（与全站主页一致，2 位小数；低于 $1 且有千分位时保留 3 位避免误显 $0.00）。
 */
export function formatWalletUsd(balance: number): string {
  if (!Number.isFinite(balance)) return '$ 0.00'
  if (balance > 0 && balance < 0.01) {
    return `$ ${balance.toFixed(4)}`
  }
  return `$ ${balance.toFixed(2)}`
}
