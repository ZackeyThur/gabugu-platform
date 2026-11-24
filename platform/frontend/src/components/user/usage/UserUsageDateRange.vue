<template>
  <div class="usage-date-range">
    <div
      class="overview-periods g-segmented"
      :style="{ '--seg-count': periods.length + 1, '--seg-index': activePeriodIndex }"
      :aria-label="t('admin.dashboard.timeRange')"
    >
      <span class="g-segmented-thumb" aria-hidden="true" />
      <button v-for="period in periods" :key="period.days" type="button" :aria-pressed="isSelected(period.days)" @click="select(period.days)">{{ t(period.label) }}</button>
      <button type="button" :aria-pressed="customOpen || !periods.some(p => isSelected(p.days))" :aria-expanded="customOpen" aria-controls="usage-custom-dates" @click="openCustom">{{ t('usageOverview.custom') }}</button>
    </div>
    <form v-if="customOpen" id="usage-custom-dates" class="usage-custom-dates" @submit.prevent="apply">
      <label>{{ t('dates.startDate') }}<input ref="startInput" v-model="draftStart" type="date" required :max="draftEnd || today" /></label>
      <label>{{ t('dates.endDate') }}<input v-model="draftEnd" type="date" required :min="draftStart" :max="today" /></label>
      <button type="submit" class="overview-primary-button" :disabled="!valid">{{ t('dates.apply') }}</button>
      <button type="button" class="overview-link" @click="customOpen = false">{{ t('common.cancel') }}</button>
    </form>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { dashboardToday, shiftDate } from '@/utils/dashboardOverview'
const props = defineProps<{ startDate: string; endDate: string }>()
const emit = defineEmits<{ change: [range: { startDate: string; endDate: string; preset: string | null }] }>()
const { t } = useI18n()
const periods = [{ days: 1, label: 'dates.today' }, { days: 7, label: 'dates.last7Days' }, { days: 30, label: 'dates.last30Days' }]
const customOpen = ref(false)
const draftStart = ref(props.startDate)
const draftEnd = ref(props.endDate)
const today = ref(dashboardToday())
const startInput = ref<HTMLInputElement | null>(null)
const valid = computed(() => !!draftStart.value && !!draftEnd.value && draftStart.value <= draftEnd.value && draftEnd.value <= today.value)
function isSelected(days: number) { return props.endDate === today.value && props.startDate === shiftDate(today.value, 1 - days) }
const activePeriodIndex = computed(() => {
  if (customOpen.value) return periods.length
  const idx = periods.findIndex(p => isSelected(p.days))
  return idx >= 0 ? idx : periods.length
})
function select(days: number) {
  today.value = dashboardToday()
  customOpen.value = false
  emit('change', { startDate: shiftDate(today.value, 1 - days), endDate: today.value, preset: days === 1 ? 'today' : `${days}days` })
}
async function openCustom() {
  today.value = dashboardToday()
  draftStart.value = props.startDate
  draftEnd.value = props.endDate
  customOpen.value = !customOpen.value
  if (customOpen.value) { await nextTick(); startInput.value?.focus() }
}
function apply() {
  if (!valid.value) return
  emit('change', { startDate: draftStart.value, endDate: draftEnd.value, preset: null })
  customOpen.value = false
}
</script>
