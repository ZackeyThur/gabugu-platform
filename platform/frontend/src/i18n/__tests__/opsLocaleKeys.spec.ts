import { describe, expect, it } from 'vitest'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

function flattenKeys(obj: Record<string, any>, prefix = ''): string[] {
  const keys: string[] = []
  for (const [k, v] of Object.entries(obj)) {
    const fullKey = prefix ? `${prefix}.${k}` : k
    if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
      keys.push(...flattenKeys(v, fullKey))
    } else {
      keys.push(fullKey)
    }
  }
  return keys
}

describe('ops locale key completeness', () => {
  const requiredKeys = [
    'admin.ops.result',
    'admin.ops.timeRange.custom',
    'admin.ops.customTimeRange.startTime',
    'admin.ops.customTimeRange.endTime',
    'admin.ops.errorDetail.upstreamStatus',
    'admin.ops.errorDetail.rootCause',
    'admin.ops.errorDetail.diagnosticPayloads',
    'admin.ops.errorDetail.payloads.client',
    'admin.ops.errorDetail.payloads.upstream_message',
    'admin.ops.errorDetail.payloads.upstream_detail',
    'admin.ops.errorDetail.payloads.upstream_events',
    'admin.ops.alertRules.form.minRequests',
    'admin.ops.alertRules.form.minErrors',
    'admin.ops.alertRules.form.globalDefaultPlaceholder',
    'admin.ops.alertRules.hints.minRequests',
    'admin.ops.alertRules.hints.minErrors',
    'admin.ops.runtime.noLimit',
    'admin.ops.runtime.noiseGuardTitle',
    'admin.ops.runtime.noiseGuardHint',
    'admin.ops.runtime.minSampleRequests',
    'admin.ops.runtime.minSampleRequestsHint',
    'admin.ops.runtime.minErrorCount',
    'admin.ops.runtime.minErrorCountHint',
    'admin.ops.runtime.validation.minSampleRequestsRange',
    'admin.ops.runtime.validation.minErrorCountRange',
    'admin.ops.email.sendTestAlert',
    'admin.ops.email.sendingTestAlert',
    'admin.ops.email.testAlertSuccess',
    'admin.ops.email.testAlertFailed',
    'admin.ops.email.aiDiagnosisTitle',
    'admin.ops.email.aiDiagnosisHint',
    'admin.ops.email.aiBaseUrl',
    'admin.ops.email.aiModel',
    'admin.ops.email.aiTimeoutSeconds',
    'admin.ops.email.aiApiKey',
    'admin.ops.email.aiFallbackTitle',
    'admin.ops.email.aiFallbackHint',
    'admin.ops.email.aiFallbackSummary',
    'admin.ops.email.aiFallbackBaseUrl',
    'admin.ops.email.aiFallbackModel',
    'admin.ops.email.aiFallbackApiKey',
    'admin.ops.email.secretConfigured',
    'admin.ops.email.secretNotConfigured',
    'admin.ops.email.clearSavedKey',
    'admin.ops.email.privateWebhookTitle',
    'admin.ops.email.privateWebhookHint',
    'admin.ops.email.privateWebhookUrl',
    'admin.ops.email.privateWebhookTarget',
    'admin.ops.email.privateWebhookToken',
    'admin.ops.email.privateWebhookIncludeReports',
    'admin.ops.email.notSet',
    'admin.ops.email.validation.aiBaseUrlInvalid',
    'admin.ops.email.validation.aiModelRequired',
    'admin.ops.email.validation.aiTimeoutRange',
    'admin.ops.email.validation.aiApiKeyRequired',
    'admin.ops.email.validation.aiFallbackBaseUrlInvalid',
    'admin.ops.email.validation.aiFallbackModelRequired',
    'admin.ops.email.validation.privateWebhookChatroomBlocked',
    'admin.ops.email.validation.privateWebhookUrlInvalid',
    'admin.ops.email.validation.privateWebhookTargetRequired',
  ]

  for (const key of requiredKeys) {
    it(`en locale has ${key}`, () => {
      const enKeys = flattenKeys(en)
      expect(enKeys).toContain(key)
    })
  }

  for (const key of requiredKeys) {
    it(`zh locale has ${key}`, () => {
      const zhKeys = flattenKeys(zh)
      expect(zhKeys).toContain(key)
    })
  }
})

describe('groups locale key completeness', () => {
  it('en locale has admin.groups.failedToSave', () => {
    const enKeys = flattenKeys(en)
    expect(enKeys).toContain('admin.groups.failedToSave')
  })

  const webSearchPricingKeys = [
    'admin.groups.webSearchPricing.title',
    'admin.groups.webSearchPricing.pricePerCall',
    'admin.groups.webSearchPricing.pricePerCallHint',
    'admin.groups.webSearchPricing.finalPricePreview',
  ]

  for (const key of webSearchPricingKeys) {
    it(`en and zh locales both have ${key}`, () => {
      expect(flattenKeys(en)).toContain(key)
      expect(flattenKeys(zh)).toContain(key)
    })
  }
})
