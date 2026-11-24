<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { opsAPI } from '@/api/admin/ops'
import type {
  EmailNotificationConfig,
  AlertSeverity,
  OpsAIDiagnosisConfig,
  OpsPrivateWebhookConfig
} from '../types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'

const { t, locale } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const sendingTestAlert = ref(false)
const config = ref<EmailNotificationConfig | null>(null)

const showEditor = ref(false)
const saving = ref(false)
const draft = ref<EmailNotificationConfig | null>(null)
const alertRecipientInput = ref('')
const reportRecipientInput = ref('')
const alertRecipientError = ref('')
const reportRecipientError = ref('')

function defaultAIDiagnosisConfig(): OpsAIDiagnosisConfig {
  return {
    enabled: false,
    base_url: '',
    model: '',
    api_key: '',
    api_key_configured: false,
    api_key_masked: '',
    clear_api_key: false,
    fallback_base_url: '',
    fallback_model: '',
    fallback_api_key: '',
    fallback_api_key_configured: false,
    fallback_api_key_masked: '',
    clear_fallback_api_key: false,
    timeout_seconds: 8
  }
}

function defaultPrivateWebhookConfig(): OpsPrivateWebhookConfig {
  return {
    enabled: false,
    url: '',
    target: '',
    bearer_token: '',
    token_configured: false,
    token_masked: '',
    clear_token: false,
    include_reports: false
  }
}

function ensureExtendedConfigFields(cfg: EmailNotificationConfig): EmailNotificationConfig {
  if (!cfg.ai_diagnosis) {
    cfg.ai_diagnosis = defaultAIDiagnosisConfig()
  }
  if (!cfg.private_webhook) {
    cfg.private_webhook = defaultPrivateWebhookConfig()
  }
  return cfg
}

const severityOptions: Array<{ value: AlertSeverity | ''; label: string }> = [
  { value: '', label: t('admin.ops.email.minSeverityAll') },
  { value: 'critical', label: t('common.critical') },
  { value: 'warning', label: t('common.warning') },
  { value: 'info', label: t('common.info') }
]

async function loadConfig() {
  loading.value = true
  try {
    const data = await opsAPI.getEmailNotificationConfig()
    config.value = ensureExtendedConfigFields(data)
  } catch (err: any) {
    console.error('[OpsEmailNotificationCard] Failed to load config', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.email.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function sendTestAlert() {
  sendingTestAlert.value = true
  try {
    const res = await opsAPI.sendTestAlertEmail({ locale: locale.value })
    appStore.showSuccess(
      t('admin.ops.email.testAlertSuccess', {
        recipient: res.recipient,
        mode: res.diagnosis_badge
      })
    )
  } catch (err: any) {
    console.error('[OpsEmailNotificationCard] Failed to send test alert email', err)
    appStore.showError(
      err?.response?.data?.message ||
        err?.response?.data?.detail ||
        t('admin.ops.email.testAlertFailed')
    )
  } finally {
    sendingTestAlert.value = false
  }
}

async function saveConfig() {
  if (!draft.value) return
  if (!editorValidation.value.valid) {
    appStore.showError(editorValidation.value.errors[0] || t('admin.ops.email.validation.invalid'))
    return
  }
  saving.value = true
  try {
    config.value = ensureExtendedConfigFields(await opsAPI.updateEmailNotificationConfig(draft.value))
    showEditor.value = false
    appStore.showSuccess(t('admin.ops.email.saveSuccess'))
  } catch (err: any) {
    console.error('[OpsEmailNotificationCard] Failed to save config', err)
    appStore.showError(
      err?.response?.data?.message ||
        err?.response?.data?.detail ||
        t('admin.ops.email.saveFailed')
    )
  } finally {
    saving.value = false
  }
}

function openEditor() {
  if (!config.value) return
  const cloned = ensureExtendedConfigFields(JSON.parse(JSON.stringify(config.value)))
  cloned.ai_diagnosis = {
    ...defaultAIDiagnosisConfig(),
    ...cloned.ai_diagnosis,
    api_key: '',
    clear_api_key: false,
    fallback_api_key: '',
    clear_fallback_api_key: false
  }
  cloned.private_webhook = {
    ...defaultPrivateWebhookConfig(),
    ...cloned.private_webhook,
    bearer_token: '',
    clear_token: false
  }
  draft.value = cloned
  alertRecipientInput.value = ''
  reportRecipientInput.value = ''
  alertRecipientError.value = ''
  reportRecipientError.value = ''
  showEditor.value = true
}

function isValidEmailAddress(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}

function isValidHttpUrl(raw: string): boolean {
  try {
    const parsed = new URL(raw.trim())
    return (parsed.protocol === 'http:' || parsed.protocol === 'https:') && !!parsed.host
  } catch {
    return false
  }
}

function hasDisallowedChatroomTarget(target: string): boolean {
  return target.trim().toLowerCase().includes('@chatroom')
}

function isNonNegativeNumber(value: unknown): boolean {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

function validateCronField(enabled: boolean, cron: string): string | null {
  if (!enabled) return null
  if (!cron || !cron.trim()) return t('admin.ops.email.validation.cronRequired')
  if (cron.trim().split(/\s+/).length < 5) return t('admin.ops.email.validation.cronFormat')
  return null
}

const editorValidation = computed(() => {
  const errors: string[] = []
  if (!draft.value) return { valid: true, errors }

  if (draft.value.alert.enabled && draft.value.alert.recipients.length === 0) {
    errors.push(t('admin.ops.email.validation.alertRecipientsRequired'))
  }
  if (draft.value.report.enabled && draft.value.report.recipients.length === 0) {
    errors.push(t('admin.ops.email.validation.reportRecipientsRequired'))
  }

  const invalidAlertRecipients = draft.value.alert.recipients.filter((e) => !isValidEmailAddress(e))
  if (invalidAlertRecipients.length > 0) errors.push(t('admin.ops.email.validation.invalidRecipients'))

  const invalidReportRecipients = draft.value.report.recipients.filter((e) => !isValidEmailAddress(e))
  if (invalidReportRecipients.length > 0) errors.push(t('admin.ops.email.validation.invalidRecipients'))

  if (!isNonNegativeNumber(draft.value.alert.rate_limit_per_hour)) {
    errors.push(t('admin.ops.email.validation.rateLimitRange'))
  }
  if (
    !isNonNegativeNumber(draft.value.alert.batching_window_seconds) ||
    draft.value.alert.batching_window_seconds > 86400
  ) {
    errors.push(t('admin.ops.email.validation.batchWindowRange'))
  }

  const dailyErr = validateCronField(
    draft.value.report.daily_summary_enabled,
    draft.value.report.daily_summary_schedule
  )
  if (dailyErr) errors.push(dailyErr)
  const weeklyErr = validateCronField(
    draft.value.report.weekly_summary_enabled,
    draft.value.report.weekly_summary_schedule
  )
  if (weeklyErr) errors.push(weeklyErr)
  const digestErr = validateCronField(
    draft.value.report.error_digest_enabled,
    draft.value.report.error_digest_schedule
  )
  if (digestErr) errors.push(digestErr)
  const accErr = validateCronField(
    draft.value.report.account_health_enabled,
    draft.value.report.account_health_schedule
  )
  if (accErr) errors.push(accErr)

  if (!isNonNegativeNumber(draft.value.report.error_digest_min_count)) {
    errors.push(t('admin.ops.email.validation.digestMinCountRange'))
  }

  const thr = draft.value.report.account_health_error_rate_threshold
  if (!(typeof thr === 'number' && Number.isFinite(thr) && thr >= 0 && thr <= 100)) {
    errors.push(t('admin.ops.email.validation.accountHealthThresholdRange'))
  }

  const ai = draft.value.ai_diagnosis
  if (ai) {
    const timeout = ai.timeout_seconds
    if (!(typeof timeout === 'number' && Number.isInteger(timeout) && timeout >= 3 && timeout <= 30)) {
      errors.push(t('admin.ops.email.validation.aiTimeoutRange'))
    }
    if (ai.enabled) {
      if (!isValidHttpUrl(ai.base_url || '')) {
        errors.push(t('admin.ops.email.validation.aiBaseUrlInvalid'))
      }
      if (!ai.model || !ai.model.trim()) {
        errors.push(t('admin.ops.email.validation.aiModelRequired'))
      }
      const hasEnteredKey = Boolean(!ai.clear_api_key && ai.api_key && ai.api_key.trim())
      const keepsExistingKey = Boolean(ai.api_key_configured && !ai.clear_api_key)
      if (!hasEnteredKey && !keepsExistingKey) {
        errors.push(t('admin.ops.email.validation.aiApiKeyRequired'))
      }
    }
    const fbUrl = (ai.fallback_base_url || '').trim()
    const fbModel = (ai.fallback_model || '').trim()
    if (fbUrl) {
      if (!isValidHttpUrl(fbUrl)) {
        errors.push(t('admin.ops.email.validation.aiFallbackBaseUrlInvalid'))
      }
      if (!fbModel) {
        errors.push(t('admin.ops.email.validation.aiFallbackModelRequired'))
      }
    } else if (fbModel) {
      errors.push(t('admin.ops.email.validation.aiFallbackBaseUrlInvalid'))
    }
  }

  const wh = draft.value.private_webhook
  if (wh) {
    if (hasDisallowedChatroomTarget(wh.target || '')) {
      errors.push(t('admin.ops.email.validation.privateWebhookChatroomBlocked'))
    }
    if (wh.enabled) {
      if (!isValidHttpUrl(wh.url || '')) {
        errors.push(t('admin.ops.email.validation.privateWebhookUrlInvalid'))
      }
      if (!wh.target || !wh.target.trim()) {
        errors.push(t('admin.ops.email.validation.privateWebhookTargetRequired'))
      }
    }
  }

  return { valid: errors.length === 0, errors }
})

const privateTargetIsChatroom = computed(() => {
  return hasDisallowedChatroomTarget(draft.value?.private_webhook?.target || '')
})

function addRecipient(target: 'alert' | 'report') {
  if (!draft.value) return
  const raw = (target === 'alert' ? alertRecipientInput.value : reportRecipientInput.value).trim()
  if (!raw) return

  if (!isValidEmailAddress(raw)) {
    const msg = t('common.invalidEmail')
    if (target === 'alert') alertRecipientError.value = msg
    else reportRecipientError.value = msg
    return
  }

  const normalized = raw.toLowerCase()
  const list = target === 'alert' ? draft.value.alert.recipients : draft.value.report.recipients
  if (!list.includes(normalized)) {
    list.push(normalized)
  }
  if (target === 'alert') alertRecipientInput.value = ''
  else reportRecipientInput.value = ''
  if (target === 'alert') alertRecipientError.value = ''
  else reportRecipientError.value = ''
}

function removeRecipient(target: 'alert' | 'report', email: string) {
  if (!draft.value) return
  const list = target === 'alert' ? draft.value.alert.recipients : draft.value.report.recipients
  const idx = list.indexOf(email)
  if (idx >= 0) list.splice(idx, 1)
}

onMounted(() => {
  loadConfig()
})
</script>

<template>
  <div class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
    <div class="mb-4 flex items-start justify-between gap-4">
      <div>
        <h3 class="text-sm font-bold text-gray-900 dark:text-white">{{ t('admin.ops.email.title') }}</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.ops.email.description') }}</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          class="btn btn-sm btn-secondary"
          :disabled="!config || sendingTestAlert || config.alert.recipients.length === 0"
          @click="sendTestAlert"
        >
          {{ sendingTestAlert ? t('admin.ops.email.sendingTestAlert') : t('admin.ops.email.sendTestAlert') }}
        </button>
        <button
          class="flex items-center gap-1.5 rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-bold text-gray-700 transition-colors hover:bg-gray-200 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600"
          :disabled="loading"
          @click="loadConfig"
        >
          <svg class="h-3.5 w-3.5" :class="{ 'animate-spin': loading }" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {{ t('common.refresh') }}
        </button>
        <button class="btn btn-sm btn-secondary" :disabled="!config" @click="openEditor">{{ t('common.edit') }}</button>
      </div>
    </div>

    <div v-if="!config" class="text-sm text-gray-500 dark:text-gray-400">
      <span v-if="loading">{{ t('admin.ops.email.loading') }}</span>
      <span v-else>{{ t('admin.ops.email.noData') }}</span>
    </div>

    <div v-else class="space-y-4">
      <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/50">
        <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.email.alertTitle') }}</h4>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('common.enabled') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">
              {{ config.alert.enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('admin.ops.email.recipients') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">{{ config.alert.recipients.length }}</span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('admin.ops.email.minSeverity') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">{{
              config.alert.min_severity || t('admin.ops.email.minSeverityAll')
            }}</span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('admin.ops.email.includeResolved') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">
              {{ config.alert.include_resolved_alerts ? t('common.enabled') : t('common.disabled') }}
            </span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('admin.ops.email.rateLimitPerHour') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">{{ config.alert.rate_limit_per_hour }}</span>
          </div>
        </div>
      </div>

      <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/50">
        <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.email.reportTitle') }}</h4>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('common.enabled') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">
              {{ config.report.enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('admin.ops.email.recipients') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">{{ config.report.recipients.length }}</span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300 md:col-span-2">
            {{ t('admin.ops.email.dailySummary') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">
              {{
                config.report.daily_summary_enabled
                  ? `${t('common.enabled')} (${config.report.daily_summary_schedule || '0 9 * * *'})`
                  : t('common.disabled')
              }}
            </span>
          </div>
        </div>
      </div>

      <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/50">
        <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.email.aiDiagnosisTitle') }}</h4>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('common.enabled') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">
              {{ config.ai_diagnosis?.enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('admin.ops.email.aiModel') }}:
            <span class="ml-1 font-mono font-medium text-gray-900 dark:text-white">
              {{ config.ai_diagnosis?.model || t('admin.ops.email.notSet') }}
            </span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('admin.ops.email.aiApiKey') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">
              {{
                config.ai_diagnosis?.api_key_configured
                  ? `${t('admin.ops.email.secretConfigured')} (${config.ai_diagnosis.api_key_masked || '****'})`
                  : t('admin.ops.email.secretNotConfigured')
              }}
            </span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('admin.ops.email.aiTimeoutSeconds') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">{{ config.ai_diagnosis?.timeout_seconds || 8 }}s</span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300 md:col-span-2">
            {{ t('admin.ops.email.aiFallbackSummary') }}:
            <span class="ml-1 font-mono font-medium text-gray-900 dark:text-white">
              {{
                config.ai_diagnosis?.fallback_model
                  ? `${config.ai_diagnosis.fallback_model} (${config.ai_diagnosis.fallback_base_url || '-'})`
                  : t('admin.ops.email.notSet')
              }}
            </span>
          </div>
        </div>
      </div>

      <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/50">
        <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.email.privateWebhookTitle') }}</h4>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('common.enabled') }}:
            <span class="ml-1 font-medium text-gray-900 dark:text-white">
              {{ config.private_webhook?.enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
          </div>
          <div class="text-xs text-gray-600 dark:text-gray-300">
            {{ t('admin.ops.email.privateWebhookTarget') }}:
            <span class="ml-1 font-mono font-medium text-gray-900 dark:text-white">
              {{ config.private_webhook?.target || t('admin.ops.email.notSet') }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>

  <BaseDialog :show="showEditor" :title="t('admin.ops.email.title')" width="extra-wide" @close="showEditor = false">
    <div v-if="draft" class="space-y-6">
      <div
        v-if="!editorValidation.valid"
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-200"
      >
        <div class="font-bold">{{ t('admin.ops.email.validation.title') }}</div>
        <ul class="mt-1 list-disc space-y-1 pl-4">
          <li v-for="msg in editorValidation.errors" :key="msg">{{ msg }}</li>
        </ul>
      </div>
      <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/50">
        <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.email.alertTitle') }}</h4>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('common.enabled') }}</div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="draft.alert.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
              <span>{{ draft.alert.enabled ? t('common.enabled') : t('common.disabled') }}</span>
            </label>
          </div>

          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.minSeverity') }}</div>
            <Select v-model="draft.alert.min_severity" :options="severityOptions" />
          </div>

          <div class="md:col-span-2">
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.recipients') }}</div>
            <div class="flex gap-2">
              <input
                v-model="alertRecipientInput"
                type="email"
                class="input"
                :placeholder="t('admin.ops.email.recipients')"
                @keydown.enter.prevent="addRecipient('alert')"
              />
              <button class="btn btn-secondary whitespace-nowrap" type="button" @click="addRecipient('alert')">
                {{ t('common.add') }}
              </button>
            </div>
            <p v-if="alertRecipientError" class="mt-1 text-xs text-red-600 dark:text-red-400">{{ alertRecipientError }}</p>
            <div class="mt-2 flex flex-wrap gap-2">
              <span
                v-for="email in draft.alert.recipients"
                :key="email"
                class="inline-flex items-center gap-2 rounded-full bg-blue-100 px-3 py-1 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
              >
                {{ email }}
                <button
                  type="button"
                  class="text-blue-700/80 hover:text-blue-900 dark:text-blue-300"
                  @click="removeRecipient('alert', email)"
                >
                  ×
                </button>
              </span>
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.ops.email.recipientsHint') }}</div>
          </div>

          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.rateLimitPerHour') }}</div>
            <input v-model.number="draft.alert.rate_limit_per_hour" type="number" min="0" max="100000" class="input" />
          </div>

          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.batchWindowSeconds') }}</div>
            <input v-model.number="draft.alert.batching_window_seconds" type="number" min="0" max="86400" class="input" />
          </div>

          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.includeResolved') }}</div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="draft.alert.include_resolved_alerts" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
              <span>{{ draft.alert.include_resolved_alerts ? t('common.enabled') : t('common.disabled') }}</span>
            </label>
          </div>
        </div>
      </div>

      <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/50">
        <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.email.reportTitle') }}</h4>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('common.enabled') }}</div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="draft.report.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
              <span>{{ draft.report.enabled ? t('common.enabled') : t('common.disabled') }}</span>
            </label>
          </div>

          <div class="md:col-span-2">
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.recipients') }}</div>
            <div class="flex gap-2">
              <input
                v-model="reportRecipientInput"
                type="email"
                class="input"
                :placeholder="t('admin.ops.email.recipients')"
                @keydown.enter.prevent="addRecipient('report')"
              />
              <button class="btn btn-secondary whitespace-nowrap" type="button" @click="addRecipient('report')">
                {{ t('common.add') }}
              </button>
            </div>
            <p v-if="reportRecipientError" class="mt-1 text-xs text-red-600 dark:text-red-400">{{ reportRecipientError }}</p>
            <div class="mt-2 flex flex-wrap gap-2">
              <span
                v-for="email in draft.report.recipients"
                :key="email"
                class="inline-flex items-center gap-2 rounded-full bg-blue-100 px-3 py-1 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
              >
                {{ email }}
                <button
                  type="button"
                  class="text-blue-700/80 hover:text-blue-900 dark:text-blue-300"
                  @click="removeRecipient('report', email)"
                >
                  ×
                </button>
              </span>
            </div>
          </div>

          <div class="md:col-span-2">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div>
                <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.dailySummary') }}</div>
                <div class="flex items-center gap-2">
                  <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                    <input v-model="draft.report.daily_summary_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
                  </label>
                  <input v-model="draft.report.daily_summary_schedule" type="text" class="input" :placeholder="t('admin.ops.email.cronPlaceholder')" />
                </div>
              </div>
              <div>
                <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.weeklySummary') }}</div>
                <div class="flex items-center gap-2">
                  <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                    <input v-model="draft.report.weekly_summary_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
                  </label>
                  <input v-model="draft.report.weekly_summary_schedule" type="text" class="input" :placeholder="t('admin.ops.email.cronPlaceholder')" />
                </div>
              </div>
              <div>
                <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.errorDigest') }}</div>
                <div class="flex items-center gap-2">
                  <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                    <input v-model="draft.report.error_digest_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
                  </label>
                  <input v-model="draft.report.error_digest_schedule" type="text" class="input" :placeholder="t('admin.ops.email.cronPlaceholder')" />
                </div>
              </div>
              <div>
                <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.errorDigestMinCount') }}</div>
                <input v-model.number="draft.report.error_digest_min_count" type="number" min="0" max="1000000" class="input" />
              </div>
              <div>
                <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.accountHealth') }}</div>
                <div class="flex items-center gap-2">
                  <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                    <input v-model="draft.report.account_health_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
                  </label>
                  <input v-model="draft.report.account_health_schedule" type="text" class="input" :placeholder="t('admin.ops.email.cronPlaceholder')" />
                </div>
              </div>
              <div>
                <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.accountHealthThreshold') }}</div>
                <input v-model.number="draft.report.account_health_error_rate_threshold" type="number" min="0" max="100" step="0.1" class="input" />
              </div>
            </div>
            <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.ops.email.reportHint') }}</div>
          </div>
        </div>
      </div>

      <div v-if="draft.ai_diagnosis" class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/50">
        <div class="mb-3">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.email.aiDiagnosisTitle') }}</h4>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.ops.email.aiDiagnosisHint') }}</p>
        </div>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('common.enabled') }}</div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="draft.ai_diagnosis.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
              <span>{{ draft.ai_diagnosis.enabled ? t('common.enabled') : t('common.disabled') }}</span>
            </label>
          </div>
          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.aiTimeoutSeconds') }}</div>
            <input v-model.number="draft.ai_diagnosis.timeout_seconds" type="number" min="3" max="30" class="input" />
          </div>
          <div class="md:col-span-2">
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.aiBaseUrl') }}</div>
            <input
              v-model="draft.ai_diagnosis.base_url"
              type="url"
              class="input font-mono text-sm"
              :placeholder="t('admin.ops.email.aiBaseUrlPlaceholder')"
            />
          </div>
          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.aiModel') }}</div>
            <input
              v-model="draft.ai_diagnosis.model"
              type="text"
              class="input font-mono text-sm"
              :placeholder="t('admin.ops.email.aiModelPlaceholder')"
            />
          </div>
          <div>
            <div class="mb-1 flex items-center justify-between gap-2 text-xs font-medium text-gray-600 dark:text-gray-300">
              <span>{{ t('admin.ops.email.aiApiKey') }}</span>
              <span
                v-if="draft.ai_diagnosis.api_key_configured && !draft.ai_diagnosis.clear_api_key"
                class="rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
              >
                {{ t('admin.ops.email.secretConfigured') }} ({{ draft.ai_diagnosis.api_key_masked || '****' }})
              </span>
            </div>
            <input
              v-model="draft.ai_diagnosis.api_key"
              type="password"
              autocomplete="new-password"
              class="input font-mono text-sm"
              :disabled="Boolean(draft.ai_diagnosis.clear_api_key)"
              :placeholder="
                draft.ai_diagnosis.api_key_configured
                  ? t('admin.ops.email.aiApiKeyPlaceholderReplace', { masked: draft.ai_diagnosis.api_key_masked || '****' })
                  : t('admin.ops.email.aiApiKeyPlaceholderNew')
              "
            />
            <label
              v-if="draft.ai_diagnosis.api_key_configured"
              class="mt-1.5 inline-flex items-center gap-2 text-xs text-gray-600 dark:text-gray-400"
            >
              <input v-model="draft.ai_diagnosis.clear_api_key" type="checkbox" class="h-3.5 w-3.5 rounded border-gray-300" />
              <span>{{ t('admin.ops.email.clearSavedKey') }}</span>
            </label>
          </div>
          <div class="md:col-span-2 border-t border-gray-200 pt-3 dark:border-dark-600">
            <div class="text-xs font-semibold text-gray-700 dark:text-gray-200">{{ t('admin.ops.email.aiFallbackTitle') }}</div>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.ops.email.aiFallbackHint') }}</p>
          </div>
          <div class="md:col-span-2">
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.aiFallbackBaseUrl') }}</div>
            <input
              v-model="draft.ai_diagnosis.fallback_base_url"
              type="url"
              class="input font-mono text-sm"
              :placeholder="t('admin.ops.email.aiBaseUrlPlaceholder')"
            />
          </div>
          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.aiFallbackModel') }}</div>
            <input
              v-model="draft.ai_diagnosis.fallback_model"
              type="text"
              class="input font-mono text-sm"
              :placeholder="t('admin.ops.email.aiModelPlaceholder')"
            />
          </div>
          <div>
            <div class="mb-1 flex items-center justify-between gap-2 text-xs font-medium text-gray-600 dark:text-gray-300">
              <span>{{ t('admin.ops.email.aiFallbackApiKey') }}</span>
              <span
                v-if="draft.ai_diagnosis.fallback_api_key_configured && !draft.ai_diagnosis.clear_fallback_api_key"
                class="rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
              >
                {{ t('admin.ops.email.secretConfigured') }} ({{ draft.ai_diagnosis.fallback_api_key_masked || '****' }})
              </span>
            </div>
            <input
              v-model="draft.ai_diagnosis.fallback_api_key"
              type="password"
              autocomplete="new-password"
              class="input font-mono text-sm"
              :disabled="Boolean(draft.ai_diagnosis.clear_fallback_api_key)"
              :placeholder="
                draft.ai_diagnosis.fallback_api_key_configured
                  ? t('admin.ops.email.aiApiKeyPlaceholderReplace', { masked: draft.ai_diagnosis.fallback_api_key_masked || '****' })
                  : t('admin.ops.email.aiFallbackApiKeyPlaceholderNew')
              "
            />
            <label
              v-if="draft.ai_diagnosis.fallback_api_key_configured"
              class="mt-1.5 inline-flex items-center gap-2 text-xs text-gray-600 dark:text-gray-400"
            >
              <input v-model="draft.ai_diagnosis.clear_fallback_api_key" type="checkbox" class="h-3.5 w-3.5 rounded border-gray-300" />
              <span>{{ t('admin.ops.email.clearSavedKey') }}</span>
            </label>
          </div>
        </div>
      </div>

      <div v-if="draft.private_webhook" class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-700/50">
        <div class="mb-3">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.ops.email.privateWebhookTitle') }}</h4>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.ops.email.privateWebhookHint') }}</p>
        </div>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('common.enabled') }}</div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="draft.private_webhook.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
              <span>{{ draft.private_webhook.enabled ? t('common.enabled') : t('common.disabled') }}</span>
            </label>
          </div>
          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.privateWebhookIncludeReports') }}</div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input v-model="draft.private_webhook.include_reports" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
              <span>{{ draft.private_webhook.include_reports ? t('common.enabled') : t('common.disabled') }}</span>
            </label>
          </div>
          <div class="md:col-span-2">
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.privateWebhookUrl') }}</div>
            <input
              v-model="draft.private_webhook.url"
              type="url"
              class="input font-mono text-sm"
              :placeholder="t('admin.ops.email.privateWebhookUrlPlaceholder')"
            />
          </div>
          <div>
            <div class="mb-1 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.ops.email.privateWebhookTarget') }}</div>
            <input
              v-model="draft.private_webhook.target"
              type="text"
              class="input font-mono text-sm"
              :class="{ 'border-red-500 focus:border-red-500 focus:ring-red-500': privateTargetIsChatroom }"
              :placeholder="t('admin.ops.email.privateWebhookTargetPlaceholder')"
            />
            <p v-if="privateTargetIsChatroom" class="mt-1 text-xs text-red-600 dark:text-red-400">
              {{ t('admin.ops.email.validation.privateWebhookChatroomBlocked') }}
            </p>
          </div>
          <div>
            <div class="mb-1 flex items-center justify-between gap-2 text-xs font-medium text-gray-600 dark:text-gray-300">
              <span>{{ t('admin.ops.email.privateWebhookToken') }}</span>
              <span
                v-if="draft.private_webhook.token_configured && !draft.private_webhook.clear_token"
                class="rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
              >
                {{ t('admin.ops.email.secretConfigured') }} ({{ draft.private_webhook.token_masked || '****' }})
              </span>
            </div>
            <input
              v-model="draft.private_webhook.bearer_token"
              type="password"
              autocomplete="new-password"
              class="input font-mono text-sm"
              :disabled="Boolean(draft.private_webhook.clear_token)"
              :placeholder="
                draft.private_webhook.token_configured
                  ? t('admin.ops.email.privateWebhookTokenPlaceholderReplace', { masked: draft.private_webhook.token_masked || '****' })
                  : ''
              "
            />
            <label
              v-if="draft.private_webhook.token_configured"
              class="mt-1.5 inline-flex items-center gap-2 text-xs text-gray-600 dark:text-gray-400"
            >
              <input v-model="draft.private_webhook.clear_token" type="checkbox" class="h-3.5 w-3.5 rounded border-gray-300" />
              <span>{{ t('admin.ops.email.clearSavedKey') }}</span>
            </label>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button class="btn btn-secondary" @click="showEditor = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="saving || !editorValidation.valid" @click="saveConfig">
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>
