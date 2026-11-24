<template>
  <section class="overview-surface overview-onboarding" aria-labelledby="onboarding-heading">
    <div class="overview-onboarding-head">
      <div>
        <span class="overview-onboarding-kicker">{{ t('dashboard.overview.onboarding.kicker') }}</span>
        <h2 id="onboarding-heading">{{ t('dashboard.overview.onboarding.title') }}</h2>
        <p>{{ t('dashboard.overview.onboarding.subtitle') }}</p>
      </div>
      <div class="overview-onboarding-progress-wrap">
        <span class="overview-onboarding-progress">
          {{ t('dashboard.overview.onboarding.progress', { done: completedCount, total: isSimple ? 2 : 3 }) }}
        </span>
        <div class="overview-onboarding-meter" aria-hidden="true">
          <span
            v-for="idx in (isSimple ? 2 : 3)"
            :key="idx"
            class="overview-onboarding-meter-seg"
            :class="{ 'is-filled': idx <= completedCount }"
          />
        </div>
      </div>
    </div>

    <div class="overview-onboarding-steps" :class="{ 'is-simple': isSimple }">
      <!-- Step 01: Prepare Balance / Quota -->
      <article v-if="!isSimple" class="overview-onboarding-step" :class="{ 'is-done': step1Done }">
        <div class="overview-onboarding-step-top">
          <span class="overview-onboarding-num">01</span>
          <span v-if="step1Done" class="overview-onboarding-check" :aria-label="t('dashboard.overview.onboarding.done')">
            <Icon name="check" size="xs" />
          </span>
        </div>
        <h3>{{ t('dashboard.overview.onboarding.step1Title') }}</h3>
        <p class="overview-onboarding-desc">
          {{ t('dashboard.overview.onboarding.step1Desc') }}
          <span v-if="firstBonusSummary" class="overview-onboarding-bonus">{{ firstBonusSummary }}</span>
        </p>
        <div class="overview-onboarding-actions">
          <router-link v-if="paymentEnabled" to="/purchase" class="overview-primary-button overview-step-btn">
            {{ t('dashboard.overview.recharge') }}
            <Icon name="arrowRight" size="xs" />
          </router-link>
          <router-link v-else to="/redeem" class="overview-secondary-button overview-step-btn">
            {{ t('dashboard.redeemCode') }}
            <Icon name="chevronRight" size="xs" />
          </router-link>
        </div>
      </article>

      <!-- Step 02: Create API Key -->
      <article class="overview-onboarding-step" :class="{ 'is-done': step2Done }">
        <div class="overview-onboarding-step-top">
          <span class="overview-onboarding-num">{{ isSimple ? '01' : '02' }}</span>
          <span v-if="step2Done" class="overview-onboarding-check" :aria-label="t('dashboard.overview.onboarding.done')">
            <Icon name="check" size="xs" />
          </span>
        </div>
        <h3>{{ t('dashboard.overview.onboarding.step2Title') }}</h3>
        <p class="overview-onboarding-desc">
          {{ step2Done ? t('dashboard.overview.onboarding.step2DoneDesc', { count: totalApiKeys }) : t('dashboard.overview.onboarding.step2Desc') }}
        </p>
        <div class="overview-onboarding-actions">
          <router-link to="/keys" class="overview-secondary-button overview-step-btn">
            <Icon name="key" size="xs" />
            {{ step2Done ? t('dashboard.overview.manageKeys') : t('dashboard.overview.onboarding.createKeyCta') }}
          </router-link>
        </div>
      </article>

      <!-- Step 03: Connect or Try Studio -->
      <article class="overview-onboarding-step">
        <div class="overview-onboarding-step-top">
          <span class="overview-onboarding-num">{{ isSimple ? '02' : '03' }}</span>
        </div>
        <h3>{{ t('dashboard.overview.onboarding.step3Title') }}</h3>
        <p class="overview-onboarding-desc">{{ t('dashboard.overview.onboarding.step3Desc') }}</p>
        <div class="overview-onboarding-actions">
          <router-link to="/guide" class="overview-secondary-button overview-step-btn">
            <Icon name="book" size="xs" />
            {{ t('dashboard.overview.onboarding.guideCta') }}
          </router-link>
          <router-link v-if="!isSimple" to="/studio" class="overview-link">
            {{ t('dashboard.overview.onboarding.studioCta') }}
            <Icon name="chevronRight" size="xs" />
          </router-link>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { CheckoutInfoResponse } from '@/types/payment'
import { buildQuickRechargeOptions } from '@/utils/dashboardOverview'
import { formatRechargeBonusNumber } from '@/utils/rechargeBonus'

const props = withDefaults(defineProps<{
  balance: number | null
  hasActiveSubscription?: boolean
  totalApiKeys: number
  paymentEnabled?: boolean
  isSimple?: boolean
  checkoutInfo?: CheckoutInfoResponse | null
}>(), {
  hasActiveSubscription: false,
  paymentEnabled: true,
  isSimple: false,
  checkoutInfo: null,
})

const { t } = useI18n()

const step1Done = computed(() => (props.balance ?? 0) > 0 || props.hasActiveSubscription)
const step2Done = computed(() => props.totalApiKeys > 0)
const completedCount = computed(() => (props.isSimple ? (step2Done.value ? 1 : 0) : (step1Done.value ? 1 : 0) + (step2Done.value ? 1 : 0)))

const firstBonusSummary = computed(() => {
  const options = buildQuickRechargeOptions(props.checkoutInfo, 0)
  const firstBonus = options.find((opt) => opt.percent > 0)
  if (!firstBonus) return ''
  if (firstBonus.mode === 'discount') {
    return t('dashboard.overview.onboarding.firstDiscountHint', {
      symbol: firstBonus.currencySymbol,
      amount: formatRechargeBonusNumber(firstBonus.amount),
      percent: formatRechargeBonusNumber(firstBonus.percent),
    })
  }
  return t('dashboard.overview.onboarding.firstBonusHint', {
    symbol: firstBonus.currencySymbol,
    amount: formatRechargeBonusNumber(firstBonus.amount),
    bonus: formatRechargeBonusNumber(firstBonus.bonus),
  })
})
</script>
