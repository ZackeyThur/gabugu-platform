<template>
  <aside v-if="campaign && phase !== 'unavailable' && phase !== 'ended' && !dismissed" class="campaign-banner" :class="{ 'campaign-banner--home': surface === 'home' && !inline, 'campaign-banner--inline': inline }" aria-label="BLG夺冠活动">
    <router-link to="/events/blg" class="campaign-banner__link">
      <span class="campaign-banner__tag">BLG夺冠</span>
      <span v-if="phase === 'celebration-week'" class="campaign-banner__message">GPT / Claude / DeepSeek / Gemini 指定分组9折，持续一周</span>
      <span v-else class="campaign-banner__message">首日GPT 2折，Claude / DeepSeek / Gemini 5折；随后9折一周</span>
      <span class="campaign-banner__time">{{ phase === 'scheduled' ? '距开始' : phase === 'championship-day' ? '首日剩余' : '距结束' }} <span class="campaign-banner__clock">{{ countdown }}</span></span>
      <span class="campaign-banner__more">查看规则 <span aria-hidden="true">↗</span></span>
    </router-link>
    <button v-if="surface !== 'home'" type="button" class="campaign-banner__close" aria-label="关闭本次活动提示" @click="dismissed = true">×</button>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useBillingCampaign } from '@/composables/useBillingCampaign'
withDefaults(defineProps<{ surface?: 'home' | 'workspace'; inline?: boolean }>(), { surface: 'workspace', inline: false })
const { campaign, phase, countdown } = useBillingCampaign()
const dismissed = ref(false)
</script>

<style scoped>
.campaign-banner { position: relative; margin-bottom: 22px; border: 1px solid #c7a66d50; border-radius: 12px; background: #111a29; color: #f9fafb; }
.campaign-banner__link { display: flex; align-items: center; flex-wrap: wrap; gap: 10px 18px; padding: 15px 42px 15px 18px; text-decoration: none; }
.campaign-banner__tag { color: #eed3a2; font-size: 13px; font-weight: 600; letter-spacing: .06em; }
.campaign-banner__message { font-size: 13px; line-height: 1.6; }
.campaign-banner__time { margin-left: auto; font-size: 13px; color: #aeb9c9; white-space: nowrap; }
.campaign-banner__clock { color: #f4e1bd; font-variant-numeric: tabular-nums; }
.campaign-banner__more { font-size: 13px; white-space: nowrap; color: #e6d1a9; }
.campaign-banner__close { position: absolute; right: 5px; top: 50%; transform: translateY(-50%); width: 32px; height: 36px; color: #b9c3d3; font-size: 24px; border-radius: 6px; }
.campaign-banner__link:focus-visible, .campaign-banner__close:focus-visible { outline: 2px solid #e9c990; outline-offset: 3px; }
.campaign-banner--home { position: fixed; z-index: 45; bottom: 22px; left: 50%; transform: translateX(-50%); width: var(--campaign-home-width, min(1100px, calc(100% - 40px))); margin: 0; background: #0a1220f5; box-shadow: 0 12px 35px #0005; }
.campaign-banner--home .campaign-banner__link { padding-right: 18px; }
.campaign-banner--inline { flex-shrink: 0; width: calc(100% - 24px); margin: 8px auto calc(12px + env(safe-area-inset-bottom, 0px)); }
.campaign-banner--inline .campaign-banner__link { padding-right: 14px; }
@media(max-width: 640px) { .campaign-banner__link { gap: 6px 12px; padding: 13px 35px 13px 14px; } .campaign-banner__message { flex-basis: 100%; order: 2; font-size: 13px; } .campaign-banner__time { margin-left: auto; } .campaign-banner__more { order: 3; margin-top: 2px; } .campaign-banner--home { bottom: 12px; width: calc(100% - 24px); } }
</style>
