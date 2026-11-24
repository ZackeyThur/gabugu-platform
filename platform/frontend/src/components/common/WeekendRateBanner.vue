<template>
  <aside v-if="bannerVisible && !dismissed" class="weekend-banner" :class="{ 'weekend-banner--home': surface === 'home' && !inline, 'weekend-banner--inline': inline }" aria-label="周末分组倍率">
    <router-link to="/events/weekend" class="weekend-banner__link">
      <span class="weekend-banner__tag">周末限时</span>
      <span class="weekend-banner__message">GPT 分组默认倍率大约少 40%，Claude 正价分组大约少 10%。有专属倍率的不跟着变。</span>
      <span class="weekend-banner__time">距窗口结束 <span class="weekend-banner__clock">{{ countdown }}</span></span>
      <span class="weekend-banner__more">查看倍率 <span aria-hidden="true">↗</span></span>
    </router-link>
    <button v-if="surface !== 'home'" type="button" class="weekend-banner__close" aria-label="关闭本次周末倍率提示" @click="dismissed = true">×</button>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useWeekendRate } from '@/composables/useWeekendRate'
withDefaults(defineProps<{ surface?: 'home' | 'workspace'; inline?: boolean }>(), { surface: 'workspace', inline: false })
const { bannerVisible, countdown } = useWeekendRate()
const dismissed = ref(false)
</script>

<style scoped>
.weekend-banner { position: relative; margin-bottom: 22px; border: 1px solid #8eb6ff55; border-radius: 12px; background: #101820; color: #f4f7fb; }
.weekend-banner__link { display: flex; align-items: center; flex-wrap: wrap; gap: 10px 18px; padding: 15px 42px 15px 18px; text-decoration: none; color: inherit; }
.weekend-banner__tag { color: #b9d4ff; font-size: 13px; font-weight: 600; letter-spacing: .06em; }
.weekend-banner__message { font-size: 13px; line-height: 1.6; }
.weekend-banner__time { margin-left: auto; font-size: 13px; color: #aeb9c9; white-space: nowrap; }
.weekend-banner__clock { color: #d6e6ff; font-variant-numeric: tabular-nums; }
.weekend-banner__more { font-size: 13px; white-space: nowrap; color: #c5d8f5; }
.weekend-banner__close { position: absolute; right: 5px; top: 50%; transform: translateY(-50%); width: 32px; height: 36px; color: #b9c3d3; font-size: 24px; border-radius: 6px; }
.weekend-banner__link:focus-visible, .weekend-banner__close:focus-visible { outline: 2px solid #9ec1ff; outline-offset: 3px; }
.weekend-banner--home { position: fixed; z-index: 46; bottom: 108px; left: 50%; transform: translateX(-50%); width: var(--campaign-home-width, min(1100px, calc(100% - 40px))); margin: 0; background: #0a1220f5; box-shadow: 0 12px 35px #0005; }
.weekend-banner--home .weekend-banner__link { padding-right: 18px; }
.weekend-banner--inline { flex-shrink: 0; width: calc(100% - 24px); margin: 8px auto calc(12px + env(safe-area-inset-bottom, 0px)); }
.weekend-banner--inline .weekend-banner__link { padding-right: 14px; }
@media (max-width: 640px) {
  .weekend-banner__link { gap: 6px 12px; padding: 13px 35px 13px 14px; }
  .weekend-banner__message { flex-basis: 100%; order: 2; font-size: 13px; }
  .weekend-banner__time { margin-left: auto; }
  .weekend-banner__more { order: 3; margin-top: 2px; }
  .weekend-banner--home { bottom: 148px; width: calc(100% - 24px); }
}
</style>
