<template>
  <BaseDialog :show="show" :title="t('admin.accounts.customOrder.title')" :close-on-escape="!saving" :show-close-button="!saving" width="normal" @close="close">
    <div ref="orderRegion" class="account-order-region space-y-4">
      <p class="text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('admin.accounts.customOrder.hint') }}</p>
      <div v-if="loading" class="flex items-center justify-center gap-2 py-10 text-sm text-gray-500" data-testid="account-order-loading">
        <Icon name="refresh" size="sm" class="animate-spin" />{{ t('common.loading') }}
      </div>
      <div v-else-if="loadError" class="space-y-3 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300" role="alert">
        <p>{{ loadError }}</p>
        <button type="button" class="btn btn-secondary" @click="loadAccounts">{{ t('admin.accounts.customOrder.retry') }}</button>
      </div>
      <p v-else-if="accounts.length === 0" class="py-8 text-center text-sm text-gray-500">{{ t('empty.noData') }}</p>
      <VueDraggable
        v-else
        v-model="accounts"
        target=".account-order-list"
        :animation="preferredMotion === 'reduce' ? 0 : 260"
        easing="cubic-bezier(0.22, 1, 0.36, 1)"
        :disabled="saving"
        handle=".account-order-handle"
        ghost-class="account-order-ghost"
        chosen-class="account-order-chosen"
        @start="startDrag"
        @end="finishDrag"
      >
        <TransitionGroup tag="div" class="account-order-list space-y-2" :css="false" :move-class="animateReorder && !dragging && preferredMotion !== 'reduce' ? 'account-order-move' : 'account-order-no-move'">
        <div v-for="(account, index) in accounts" :key="account.id" :data-account-id="account.id" data-testid="account-order-row" class="account-order-row flex flex-wrap items-center gap-2 rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800 sm:flex-nowrap" :class="{ 'account-order-highlight': highlightedId === account.id }">
          <span class="account-order-handle flex h-9 w-7 shrink-0 cursor-grab touch-none items-center justify-center text-gray-400 active:cursor-grabbing" :title="t('admin.accounts.customOrder.drag')" aria-hidden="true"><Icon name="menu" size="sm" /></span>
          <span class="account-order-position w-7 shrink-0 text-right font-mono text-xs text-gray-400">{{ index + 1 }}</span>
          <div class="min-w-0 flex-1">
            <p class="break-words text-sm font-medium text-gray-900 dark:text-gray-100">{{ account.name }}</p>
            <p class="text-xs text-gray-500">#{{ account.id }} · {{ account.platform }}</p>
          </div>
          <div class="account-order-controls flex w-full shrink-0 justify-end gap-1 sm:w-auto">
            <button type="button" class="btn btn-secondary h-9 w-9 justify-center p-0 disabled:opacity-30" :disabled="saving || index === 0" :title="t('admin.accounts.customOrder.moveToTop', { name: account.name })" :aria-label="t('admin.accounts.customOrder.moveToTop', { name: account.name })" data-testid="account-order-top" @click="moveAccount(index, 0)"><Icon name="arrowUpToLine" size="sm" /></button>
            <button type="button" class="btn btn-secondary h-9 w-9 justify-center p-0 disabled:opacity-30" :disabled="saving || index === 0" :title="t('admin.accounts.customOrder.moveUp', { name: account.name })" :aria-label="t('admin.accounts.customOrder.moveUp', { name: account.name })" data-testid="account-order-up" @click="moveAccount(index, index - 1)"><Icon name="chevronUp" size="sm" /></button>
            <button type="button" class="btn btn-secondary h-9 w-9 justify-center p-0 disabled:opacity-30" :disabled="saving || index === accounts.length - 1" :title="t('admin.accounts.customOrder.moveDown', { name: account.name })" :aria-label="t('admin.accounts.customOrder.moveDown', { name: account.name })" data-testid="account-order-down" @click="moveAccount(index, index + 1)"><Icon name="chevronDown" size="sm" /></button>
            <button type="button" class="btn btn-secondary h-9 w-9 justify-center p-0 disabled:opacity-30" :disabled="saving || index === accounts.length - 1" :title="t('admin.accounts.customOrder.moveToBottom', { name: account.name })" :aria-label="t('admin.accounts.customOrder.moveToBottom', { name: account.name })" data-testid="account-order-bottom" @click="moveAccount(index, accounts.length - 1)"><Icon name="arrowDownToLine" size="sm" /></button>
          </div>
        </div>
        </TransitionGroup>
      </VueDraggable>
      <p v-if="saveError" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ saveError }}</p>
      <p class="sr-only" aria-live="polite">{{ announcement }}</p>
    </div>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="saving" data-testid="account-order-cancel" @click="close">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="loading || saving || !!loadError || accounts.length === 0" data-testid="account-order-save" @click="save">
          <Icon v-if="saving" name="refresh" size="sm" class="animate-spin" />{{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { nextTick, onUnmounted, ref, watch } from 'vue'
import { usePreferredReducedMotion } from '@vueuse/core'
import { VueDraggable, type DraggableEvent } from 'vue-draggable-plus'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { AccountListItem } from '@/types'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t } = useI18n()
const accounts = ref<AccountListItem[]>([])
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const saveError = ref('')
const announcement = ref('')
const orderRegion = ref<HTMLElement | null>(null)
const preferredMotion = usePreferredReducedMotion()
const dragging = ref(false)
const animateReorder = ref(true)
const highlightedId = ref<number | null>(null)
let highlightTimer: ReturnType<typeof setTimeout> | undefined
let moveVersion = 0
let controller: AbortController | null = null
let requestVersion = 0
const pageSize = 1000
const maxAccounts = 10000

const loadAccounts = async () => {
  controller?.abort()
  const current = new AbortController()
  controller = current
  const version = ++requestVersion
  loading.value = true
  loadError.value = ''
  saveError.value = ''
  accounts.value = []
  try {
    const rows: AccountListItem[] = []
    const seen = new Set<number>()
    let total: number | undefined
    for (let page = 1; rows.length < (total ?? 1); page++) {
      const result = await adminAPI.accounts.list(page, pageSize, {
        lite: '1', include_scheduler_score: '0', sort_by: 'display_order', sort_order: 'asc'
      }, { signal: current.signal })
      if (version !== requestVersion || !props.show) return
      if (result.total > maxAccounts) throw new Error(t('admin.accounts.customOrder.tooMany', { count: maxAccounts }))
      if (total !== undefined && total !== result.total) throw new Error(t('admin.accounts.customOrder.listChanged'))
      total = result.total
      for (const account of result.items) {
        if (seen.has(account.id)) throw new Error(t('admin.accounts.customOrder.listChanged'))
        seen.add(account.id)
        rows.push(account)
      }
      if (rows.length > total || (result.items.length === 0 && rows.length < total)) throw new Error(t('admin.accounts.customOrder.listChanged'))
    }
    accounts.value = rows
  } catch (error) {
    if (version === requestVersion && !current.signal.aborted) loadError.value = extractApiErrorMessage(error, t('admin.accounts.customOrder.loadFailed'))
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

const announceOrder = () => {
  announcement.value = accounts.value.map((account, index) => `${index + 1}. ${account.name}`).join(', ')
}
const clearMotionFeedback = () => {
  moveVersion++
  clearTimeout(highlightTimer)
  highlightedId.value = null
}
const highlightAccount = (id: number) => {
  clearTimeout(highlightTimer)
  highlightedId.value = id
  highlightTimer = setTimeout(() => { highlightedId.value = null }, 1500)
}
const startDrag = () => {
  clearMotionFeedback()
  dragging.value = true
}
const finishDrag = async (event?: DraggableEvent<AccountListItem>) => {
  announceOrder()
  if (event?.item.dataset.accountId) highlightAccount(Number(event.item.dataset.accountId))
  // Let Sortable finish updating the DOM before enabling button-driven transitions.
  await nextTick()
  dragging.value = false
}
const revealAccount = (id: number) => {
  const row = orderRegion.value?.querySelector<HTMLElement>(`[data-account-id="${id}"]`)
  const viewport = orderRegion.value?.closest<HTMLElement>('.modal-body')
  if (!row?.parentElement || !viewport) return
  // offsetTop excludes the FLIP transform, so scrolling follows the final position.
  const top = row.parentElement.getBoundingClientRect().top - viewport.getBoundingClientRect().top + viewport.scrollTop + row.offsetTop
  const bottom = top + row.offsetHeight
  const inset = 12
  let scrollTop = viewport.scrollTop
  if (top < scrollTop + inset) scrollTop = top - inset
  else if (bottom > scrollTop + viewport.clientHeight - inset) scrollTop = bottom - viewport.clientHeight + inset
  scrollTop = Math.max(0, Math.min(scrollTop, viewport.scrollHeight - viewport.clientHeight))
  if (Math.abs(scrollTop - viewport.scrollTop) > 1) {
    viewport.scrollTo({ top: scrollTop, behavior: preferredMotion.value === 'reduce' ? 'auto' : 'smooth' })
  }
}
const moveAccount = async (index: number, target: number) => {
  if (saving.value || dragging.value || index === target || index < 0 || index >= accounts.value.length || target < 0 || target >= accounts.value.length) return
  const version = ++moveVersion
  // Long jumps use viewport scrolling rather than flying a card through the entire list.
  animateReorder.value = Math.abs(target - index) === 1
  const next = [...accounts.value]
  const [account] = next.splice(index, 1)
  next.splice(target, 0, account)
  accounts.value = next
  highlightAccount(account.id)
  announcement.value = t('admin.accounts.customOrder.moved', { name: account.name, position: target + 1 })
  await nextTick()
  if (props.show && version === moveVersion) revealAccount(account.id)
}
const close = () => { if (!saving.value) emit('close') }
const save = async () => {
  if (loading.value || saving.value || loadError.value || accounts.value.length === 0) return
  saving.value = true
  saveError.value = ''
  try {
    await adminAPI.accounts.reorder(accounts.value.map(account => account.id))
    emit('saved')
  } catch (error) {
    saveError.value = extractApiErrorMessage(error, t('admin.accounts.customOrder.saveFailed'))
  } finally {
    saving.value = false
  }
}
watch(() => props.show, (show) => {
  if (show) void loadAccounts()
  else { requestVersion++; controller?.abort(); clearMotionFeedback(); dragging.value = false }
}, { immediate: true })
onUnmounted(() => { requestVersion++; controller?.abort(); clearMotionFeedback() })
</script>

<style scoped>
.account-order-region {
  overflow-anchor: none;
}

.account-order-list,
.account-order-row {
  position: relative;
}

.account-order-row {
  transition: border-color 180ms ease, background-color 240ms ease, box-shadow 320ms ease;
}

.account-order-move {
  transition: transform 320ms cubic-bezier(0.22, 1, 0.36, 1), border-color 180ms ease, background-color 240ms ease, box-shadow 320ms ease;
}

.account-order-highlight {
  @apply border-primary-400 bg-primary-50 dark:border-primary-500/70 dark:bg-primary-500/10;
  box-shadow: 0 4px 16px -8px rgb(59 130 246 / 28%);
}

.account-order-position {
  transition: color 180ms ease;
}

.account-order-highlight .account-order-position {
  @apply text-primary-600 dark:text-primary-300;
}

.account-order-chosen {
  @apply border-primary-400 dark:border-primary-400;
  z-index: 1;
  box-shadow: 0 12px 28px -10px rgb(15 23 42 / 35%);
}

.account-order-ghost {
  @apply border-dashed border-primary-400 bg-primary-50 dark:bg-primary-500/10;
  opacity: 0.35;
}

.account-order-controls > button {
  transition: transform 160ms cubic-bezier(0.22, 1, 0.36, 1), color 160ms ease, background-color 160ms ease, border-color 160ms ease, opacity 160ms ease;
}

@media (hover: hover) {
  .account-order-controls > button:not(:disabled):hover {
    @apply border-primary-400 text-primary-600 dark:border-primary-500/70 dark:text-primary-300;
    transform: translateY(-1px);
  }
}

.account-order-controls > button:not(:disabled):active {
  transform: scale(0.94);
  transition-duration: 80ms;
}

@media (prefers-reduced-motion: reduce) {
  .account-order-row,
  .account-order-position,
  .account-order-controls > button {
    transition: none;
  }

  .account-order-controls > button:not(:disabled):hover,
  .account-order-controls > button:not(:disabled):active {
    transform: none;
  }
}
</style>
