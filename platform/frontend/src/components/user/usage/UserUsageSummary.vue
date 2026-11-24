<template>
  <section class="usage-summary" :aria-label="t('usageOverview.summary')" :aria-busy="loading">
    <div v-if="loading" class="usage-summary-loading" role="status"><span class="overview-skeleton" /><span class="overview-skeleton" /><span class="sr-only">{{ t('common.loading') }}</span></div>
    <div v-else-if="error || !stats" class="overview-surface overview-empty usage-summary-error" role="alert">{{ t('usageOverview.statsError') }}<button class="overview-link" @click="$emit('retry')">{{ t('common.retry') }}</button></div>
    <template v-else>
      <article class="overview-surface usage-spend-card">
        <div class="overview-card-label"><h2>{{ t('usageOverview.actualCost') }}</h2><span>{{ t('usageOverview.filtered') }}</span></div>
        <p class="overview-amount" data-testid="usage-total-actual"><CountUp :value="stats.total_actual_cost" :formatter="overviewMoney" /></p>
        <div class="usage-summary-footer"><p>{{ t('usageOverview.recordCount', { count: stats.total_requests.toLocaleString() }) }}</p><button class="overview-link" @click="open = true">{{ t('usageOverview.billingDetails') }}<Icon name="arrowRight" size="sm" /></button></div>
      </article>
      <article data-star-border class="usage-discount-card" :class="{ 'is-surcharge': difference < 0 }">
        <div class="overview-card-label"><h2>{{ t(difference < 0 ? 'usageOverview.aboveStandard' : 'usageOverview.discount') }}</h2><span v-if="stats.total_cost > 0 && difference > 0" class="usage-discount-rate">{{ t('dashboard.overview.lessBy', { percent: (difference / stats.total_cost * 100).toFixed(1) }) }}</span></div>
        <p class="overview-amount" data-testid="usage-total-discount"><CountUp :value="Math.abs(difference)" :formatter="overviewMoney" /></p>
        <div class="usage-summary-footer"><p>{{ t('usageOverview.standardCost') }} <span>{{ overviewMoney(stats.total_cost) }}</span></p><span>{{ t('usageOverview.siteComparison') }}</span></div>
      </article>
    </template>
  </section>
  <DashboardDrawer :open="open" :title="t('usageOverview.billingDetails')" @close="open = false">
    <p class="overview-detail-intro">{{ t('usageOverview.billingExplanation') }}</p>
    <p class="overview-detail-intro">{{ t('dashboard.overview.subscriptionExplanation') }}</p>
    <dl v-if="stats && !loading && !error" class="overview-detail-list">
      <div><dt>{{ t('usageOverview.standardCost') }}</dt><dd>{{ preciseMoney(stats.total_cost) }}</dd></div>
      <div><dt>{{ t('usageOverview.actualCost') }}</dt><dd>{{ preciseMoney(stats.total_actual_cost) }}</dd></div>
      <div><dt>{{ t('dashboard.overview.signedDifference') }}</dt><dd>{{ preciseMoney(difference) }}</dd></div>
      <div><dt>{{ t('usageOverview.records') }}</dt><dd>{{ stats.total_requests.toLocaleString() }}</dd></div>
    </dl>
    <p v-else>{{ t(loading ? 'common.loading' : 'usageOverview.statsError') }}</p>
  </DashboardDrawer>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageStatsResponse } from '@/types'
import { overviewMoney, preciseMoney } from '@/utils/dashboardOverview'
import DashboardDrawer from '@/components/user/dashboard/DashboardDrawer.vue'
import CountUp from '@/components/common/CountUp.vue'
import Icon from '@/components/icons/Icon.vue'
const props = defineProps<{ stats: UsageStatsResponse | null; loading: boolean; error: boolean }>()
defineEmits<{ retry: [] }>()
const { t } = useI18n()
const open = ref(false)
const difference = computed(() => props.stats ? props.stats.total_cost - props.stats.total_actual_cost : 0)
</script>
