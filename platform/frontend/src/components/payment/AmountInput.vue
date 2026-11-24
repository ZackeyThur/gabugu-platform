<template>
  <div class="space-y-4">
    <!-- Quick Amount Buttons -->
    <div>
      <label class="mb-2.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.quickAmounts') }}
      </label>
      <div class="grid grid-cols-3 gap-3 pt-1.5">
        <button
          v-for="amt in filteredAmounts"
          :key="amt"
          type="button"
          data-tilt
          data-tilt-max="3.6"
          data-spark
          :class="[
            'group relative rounded-xl px-3 py-3 text-center font-medium transition-all duration-200 ease-out active:scale-[0.98]',
            modelValue === amt
              ? 'bg-primary-600/[0.09] text-primary-700 ring-1 ring-inset ring-primary-600/50 dark:bg-primary-500/[0.14] dark:text-white dark:ring-primary-400/55'
              : 'bg-pearl-raised/45 text-gray-800 hover:-translate-y-0.5 hover:bg-pearl-raised/75 dark:bg-white/[0.024] dark:text-gray-200 dark:ring-1 dark:ring-inset dark:ring-white/[0.06] dark:hover:bg-white/[0.055]',
          ]"
          :data-testid="`quick-amount-${amt}`"
          @click="selectAmount(amt)"
        >
          <!-- 选中态精致角落指示点 -->
          <span
            v-if="modelValue === amt"
            aria-hidden="true"
            class="pointer-events-none absolute left-2 top-2 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-primary-600 text-white dark:bg-primary-400 dark:text-dark-950"
          >
            <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          </span>
          <!-- 优惠档位角标：克制扁平胶囊，不倾斜、不遮挡金额 -->
          <span
            v-if="quoteFor(amt).percent > 0"
            class="pointer-events-none absolute -top-2.5 right-2 z-10 inline-flex items-center whitespace-nowrap rounded-full bg-pearl-surface px-2 py-0.5 text-xs font-semibold leading-none tracking-tight text-clay-600 shadow-glass-sm ring-1 ring-inset ring-clay-500/25 dark:bg-[#1E2740] dark:text-[#F6D494] dark:ring-[#F6D494]/30"
            data-testid="quick-amount-bonus-badge"
          >
            {{ badgeText(amt) }}
          </span>
          <span class="block text-lg font-bold tracking-tight tabular-nums">{{ amt }}</span>
          <!-- 配置了优惠阶梯时，所有按钮都显示第二行，保持高度一致：赠金显示到账 USD，折扣显示折后实付 -->
          <span
            v-if="showSecondLine"
            :class="[
              'mt-1 block text-xs font-normal leading-tight tabular-nums',
              quoteFor(amt).percent > 0 ? 'text-clay-600 dark:text-[#F6D494]/90' : 'text-gray-400 dark:text-dark-400',
            ]"
            data-testid="quick-amount-credited"
          >{{ secondLine(amt) }}</span>
        </button>
      </div>
    </div>

    <!-- Custom Amount Input -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.customAmount') }}
      </label>
      <div class="relative">
        <span
          class="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-sm font-medium text-gray-400 dark:text-dark-400"
          data-testid="amount-input-currency-symbol"
        >
          {{ inputCurrencyPrefix }}
        </span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          :class="['input w-full py-2.5 pr-4 tabular-nums', inputPrefixPaddingClass]"
          @input="handleInput"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RechargeBonusTier } from '@/types/payment'
import { formatRechargeBonusNumber, quoteRechargeBonus, type RechargeBonusMode } from '@/utils/rechargeBonus'
import { currencySymbol, formatPaymentAmount } from './currency'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  /** 充值优惠阶梯（按 min_amount 升序）；为空时不显示价签与第二行 */
  bonusTiers?: RechargeBonusTier[]
  /** 阶梯模式：bonus 赠金 / discount 折扣 */
  bonusMode?: RechargeBonusMode
  /** 充值倍率（1 支付币种 = multiplier USD），用于计算到账金额 */
  multiplier?: number
  /** 支付币种（折扣模式第二行实付金额的币种与精度） */
  currency?: string
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
  bonusTiers: () => [],
  bonusMode: 'bonus',
  multiplier: 1,
  currency: undefined,
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

const inputCurrencyPrefix = computed(() => (props.currency ? currencySymbol(props.currency) : '$'))

const inputPrefixPaddingClass = computed(() => {
  const len = inputCurrencyPrefix.value.length
  if (len >= 4) return 'pl-16'
  if (len === 3) return 'pl-14'
  if (len === 2) return 'pl-11'
  return 'pl-8'
})

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)

const showSecondLine = computed(() => props.bonusTiers.length > 0)

function currencyDigits(): number {
  if (!props.currency) return 2
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: props.currency }).resolvedOptions().maximumFractionDigits ?? 2
  } catch {
    return 2
  }
}

function quoteFor(amt: number) {
  return quoteRechargeBonus(props.bonusTiers, amt, {
    multiplier: props.multiplier,
    mode: props.bonusMode,
    currencyDigits: currencyDigits(),
  })
}

// 价签文案：赠金「+20%」，折扣「20% OFF」
function badgeText(amt: number): string {
  const percent = formatRechargeBonusNumber(quoteFor(amt).percent)
  return props.bonusMode === 'discount' ? `${percent}% OFF` : `+${percent}%`
}

function secondLine(amt: number): string {
  const quote = quoteFor(amt)
  if (props.bonusMode === 'discount') {
    return t('payment.rechargeBonus.payShort', { amount: formatPaymentAmount(quote.payBase, props.currency) })
  }
  return t('payment.rechargeBonus.creditedShort', { amount: '$' + quote.credited.toFixed(2) })
}

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function handleInput(e: Event) {
  const input = e.target as HTMLInputElement
  const val = input.value
  if (!AMOUNT_PATTERN.test(val)) {
    input.value = customText.value
    return
  }
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
}, { immediate: true })
</script>
