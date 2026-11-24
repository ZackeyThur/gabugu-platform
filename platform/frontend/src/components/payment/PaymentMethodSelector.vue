<template>
  <div>
    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ t('payment.paymentMethod') }}
    </label>
    <div
      data-testid="payment-method-grid"
      :class="[
        'grid gap-3',
        sortedMethods.length <= 2
          ? 'grid-cols-2'
          : sortedMethods.length === 3
            ? 'grid-cols-2 sm:grid-cols-3'
            : 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-4',
      ]"
    >
      <button
        v-for="method in sortedMethods"
        :key="method.type"
        type="button"
        :title="methodLabel(method)"
        :disabled="!method.available"
        :class="[
          'relative flex h-[60px] min-w-0 flex-col items-center justify-center rounded-xl border px-3 transition-all duration-200 ease-out',
          !method.available
            ? 'cursor-not-allowed border-transparent bg-pearl-raised/35 opacity-45 dark:bg-white/[0.025]'
            : selected === method.type
              ? methodSelectedClass(method.type)
              : 'border-transparent bg-pearl-raised/45 text-gray-800 hover:-translate-y-0.5 hover:bg-pearl-raised/75 active:scale-[0.985] dark:border-white/[0.06] dark:bg-white/[0.024] dark:text-gray-200 dark:hover:bg-white/[0.055]',
        ]"
        @click="method.available && emit('select', method.type)"
      >
        <span
          v-if="method.available && selected === method.type"
          aria-hidden="true"
          class="pointer-events-none absolute right-2 top-2 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-black/[0.08] dark:bg-white/[0.14]"
        >
          <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
        </span>
        <span class="flex w-full min-w-0 items-center justify-center gap-2.5">
          <img :src="methodIcon(method.type)" :alt="methodLabel(method)" class="h-6 w-6 shrink-0 object-contain" />
          <span class="flex min-w-0 flex-col items-start leading-tight">
            <span data-testid="payment-method-label" class="block w-full truncate text-sm font-semibold">
              {{ methodLabel(method) }}
            </span>
            <span
              v-if="method.fee_rate > 0"
              class="text-xs tracking-wide text-gray-500 dark:text-dark-400"
            >
              {{ t('payment.fee') }} {{ method.fee_rate }}%
            </span>
          </span>
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { METHOD_ORDER, isBuiltInAlipayMethod, isBuiltInWxpayMethod } from './providerConfig'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import stripeIcon from '@/assets/icons/stripe.svg'
import airwallexIcon from '@/assets/icons/airwallex.svg'
import paymentIcon from '@/assets/icons/payment.svg'

export interface PaymentMethodOption {
  type: string
  display_name?: string
  fee_rate: number
  available: boolean
}

const props = defineProps<{
  methods: PaymentMethodOption[]
  selected: string
}>()

const emit = defineEmits<{
  select: [type: string]
}>()

const { t } = useI18n()

const METHOD_ICONS: Record<string, string> = {
  alipay: alipayIcon,
  wxpay: wxpayIcon,
  stripe: stripeIcon,
  airwallex: airwallexIcon,
  credit_card: paymentIcon,
}

const sortedMethods = computed(() => {
  const order: readonly string[] = METHOD_ORDER
  return [...props.methods].sort((a, b) => {
    const ai = order.indexOf(a.type)
    const bi = order.indexOf(b.type)
    return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
  })
})

function methodIcon(type: string): string {
  if (isBuiltInAlipayMethod(type)) return METHOD_ICONS.alipay
  if (isBuiltInWxpayMethod(type)) return METHOD_ICONS.wxpay
  if (type === 'airwallex') return METHOD_ICONS.airwallex
  return METHOD_ICONS[type] || paymentIcon
}

function methodLabel(method: PaymentMethodOption): string {
  return method.display_name || t(`payment.methods.${method.type}`, method.type)
}

function methodSelectedClass(type: string): string {
  if (isBuiltInAlipayMethod(type)) return 'border-[#1273BA]/65 bg-[#1273BA]/[0.08] text-[#10629E] ring-1 ring-[#1273BA]/20 dark:border-[#5682F5]/65 dark:bg-[#5682F5]/[0.16] dark:text-[#C5D8FF] dark:ring-[#5682F5]/25'
  if (isBuiltInWxpayMethod(type)) return 'border-[#1D874B]/65 bg-[#1D874B]/[0.08] text-[#176D3C] ring-1 ring-[#1D874B]/20 dark:border-[#38D3B4]/60 dark:bg-[#38D3B4]/[0.14] dark:text-[#9CF2DF] dark:ring-[#38D3B4]/25'
  if (type === 'stripe') return 'border-[#5256C9]/65 bg-[#5256C9]/[0.08] text-[#4346AD] ring-1 ring-[#5256C9]/20 dark:border-[#A78BFA]/60 dark:bg-[#A78BFA]/[0.15] dark:text-[#DDD6FE]'
  if (type === 'airwallex') return 'border-[#D9592E]/65 bg-[#D9592E]/[0.08] text-[#B84722] ring-1 ring-[#D9592E]/20 dark:border-[#F6D494]/60 dark:bg-[#F6D494]/[0.13] dark:text-[#FDE6B6]'
  return 'border-primary-500 bg-primary-600/[0.08] text-primary-700 ring-1 ring-primary-500/20 dark:border-primary-400 dark:bg-primary-500/20 dark:text-primary-200'
}
</script>
