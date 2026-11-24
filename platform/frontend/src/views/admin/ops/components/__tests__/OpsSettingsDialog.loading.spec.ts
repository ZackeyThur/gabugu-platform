import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, shallowMount } from '@vue/test-utils'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh' } }),
}))
enableAutoUnmount(afterEach)

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

import OpsSettingsDialog from '../OpsSettingsDialog.vue'
const { opsAPI, showError, showSuccess } = vi.hoisted(() => ({
  opsAPI: {
    getAlertRuntimeSettings: vi.fn(), getEmailNotificationConfig: vi.fn(),
    getAdvancedSettings: vi.fn(), getMetricThresholds: vi.fn(),
    updateAlertRuntimeSettings: vi.fn(), updateEmailNotificationConfig: vi.fn(),
    updateAdvancedSettings: vi.fn(), updateMetricThresholds: vi.fn(),
    sendTestAlertEmail: vi.fn(),
  }, showError: vi.fn(), showSuccess: vi.fn(),
}))
vi.mock('@/api/admin/ops', () => ({ opsAPI }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError, showSuccess }) }))
async function openDialog() {
  const wrapper = shallowMount(OpsSettingsDialog, {
    props: { show: false },
    global: { stubs: { BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' } } },
  })
  await wrapper.setProps({ show: true })
  return wrapper
}
type View = { saveAllSettings: () => Promise<void> }
describe('ops settings load guard', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    vi.spyOn(console, 'error').mockImplementation(() => {})
    opsAPI.getAlertRuntimeSettings.mockResolvedValue({ evaluation_interval_seconds: 30 })
    opsAPI.getEmailNotificationConfig.mockResolvedValue({ alert: { enabled: false }, report: { enabled: false } })
    opsAPI.getAdvancedSettings.mockResolvedValue({
      data_retention: {
        cleanup_enabled: false,
        cleanup_schedule: '0 2 * * *',
        error_log_retention_days: 7,
        minute_metrics_retention_days: 7,
        hourly_metrics_retention_days: 7,
      },
      aggregation: { aggregation_enabled: true },
      ignore_count_tokens_errors: false,
      ignore_context_canceled: false,
      ignore_no_available_accounts: false,
      ignore_insufficient_balance_errors: false,
      auto_refresh_enabled: false,
      auto_refresh_interval_seconds: 30,
      display_alert_events: true,
      display_openai_token_stats: false,
    })
    opsAPI.getMetricThresholds.mockResolvedValue({ sla_percent_min: 98 })
  })
  afterEach(() => { vi.restoreAllMocks() })
  it('blocks saving while any settings request is still loading', async () => {
    const pending = deferred<object>()
    opsAPI.getMetricThresholds.mockReturnValueOnce(pending.promise)
    const wrapper = await openDialog()
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeDefined()
    await (wrapper.vm as unknown as View).saveAllSettings()
    expect(opsAPI.updateMetricThresholds).not.toHaveBeenCalled()
    pending.resolve({ sla_percent_min: 98 })
    await flushPromises()
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeUndefined()
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()
    expect(opsAPI.updateMetricThresholds).toHaveBeenCalledWith(expect.objectContaining({ sla_percent_min: 98 }))
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })
  it('does not save default thresholds after a failed initial load', async () => {
    opsAPI.getMetricThresholds.mockRejectedValueOnce(new Error('load failed'))
    const wrapper = await openDialog()
    await flushPromises()
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeDefined()
    await (wrapper.vm as unknown as View).saveAllSettings()
    expect(opsAPI.updateMetricThresholds).not.toHaveBeenCalled()
    expect(wrapper.emitted('saved')).toBeUndefined()
  })
  it('blocks saving stale settings after a failed reopen and allows a later retry', async () => {
    const wrapper = await openDialog()
    await flushPromises()
    await wrapper.setProps({ show: false })
    opsAPI.getMetricThresholds.mockRejectedValueOnce(new Error('refresh failed'))
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeDefined()
    await (wrapper.vm as unknown as View).saveAllSettings()
    expect(opsAPI.updateAlertRuntimeSettings).not.toHaveBeenCalled()
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeUndefined()
  })

  it('sends test alert email from the settings dialog when recipients exist', async () => {
    opsAPI.getEmailNotificationConfig.mockResolvedValueOnce({
      alert: { enabled: true, recipients: ['owner@example.com'], include_resolved_alerts: true, rate_limit_per_hour: 6 },
      report: { enabled: false, recipients: [] },
      ai_diagnosis: {
        enabled: true,
        base_url: 'https://primary.example/v1',
        model: 'primary-model',
        api_key_configured: true,
        api_key_masked: 'sk-p***1234',
        fallback_base_url: 'https://backup.example/v1',
        fallback_model: 'backup-model',
        fallback_api_key_configured: true,
        fallback_api_key_masked: 'sk-b***5678',
        timeout_seconds: 8,
      },
    })
    opsAPI.sendTestAlertEmail.mockResolvedValueOnce({
      recipient: 'owner@example.com',
      diagnosis_badge: 'AI 智能诊断',
    })
    const wrapper = await openDialog()
    await flushPromises()

    const btn = wrapper.get('[data-testid="ops-send-test-alert-btn"]')
    expect(btn.attributes('disabled')).toBeUndefined()
    await btn.trigger('click')
    await flushPromises()

    expect(opsAPI.sendTestAlertEmail).toHaveBeenCalledWith({ locale: 'zh' })
    expect(showSuccess).toHaveBeenCalledTimes(1)
  })

  it('disables test alert button when alert recipients list is empty', async () => {
    const wrapper = await openDialog()
    await flushPromises()
    const btn = wrapper.get('[data-testid="ops-send-test-alert-btn"]')
    expect(btn.attributes('disabled')).toBeDefined()
  })

  it('blocks saving when private webhook target contains @chatroom or backup AI route is incomplete', async () => {
    opsAPI.getEmailNotificationConfig.mockResolvedValueOnce({
      alert: { enabled: true, recipients: ['owner@example.com'] },
      report: { enabled: false, recipients: [] },
      ai_diagnosis: {
        enabled: true,
        base_url: 'https://primary.example/v1',
        model: 'primary-model',
        api_key_configured: true,
        fallback_base_url: 'https://backup.example/v1',
        fallback_model: '',
        timeout_seconds: 8,
      },
      private_webhook: {
        enabled: false,
        url: '',
        target: '123456@chatroom',
        include_reports: false,
      },
    })
    const wrapper = await openDialog()
    await flushPromises()

    expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('admin.ops.email.validation.privateWebhookChatroomBlocked')
    expect(wrapper.text()).toContain('admin.ops.email.validation.aiFallbackModelRequired')
  })

  it('saves low-sample noise guard thresholds alongside runtime settings', async () => {
    opsAPI.getAlertRuntimeSettings.mockResolvedValueOnce({
      evaluation_interval_seconds: 30,
      min_sample_requests: 10,
      min_error_count: 3,
    })
    const wrapper = await openDialog()
    await flushPromises()

    await wrapper.get('[data-testid="ops-min-sample-requests"]').setValue('15')
    await wrapper.get('[data-testid="ops-min-error-count"]').setValue('4')
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(opsAPI.updateAlertRuntimeSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        evaluation_interval_seconds: 30,
        min_sample_requests: 15,
        min_error_count: 4,
      }),
    )
  })
})
