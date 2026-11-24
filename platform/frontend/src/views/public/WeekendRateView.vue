<template>
  <main class="weekend-page">
    <header class="weekend-header">
      <router-link to="/home" class="weekend-brand" aria-label="GABUGU首页"><img src="/brand/gabugu-logo-dark.png" width="36" height="40" alt="" /><GabuguWordmark theme="dark" /></router-link>
      <router-link to="/model-plaza" class="weekend-console">模型广场 <span aria-hidden="true">↗</span></router-link>
    </header>

    <section class="weekend-hero" aria-labelledby="weekend-title">
      <p class="weekend-eyebrow"><span></span> WEEKEND RATE · TWO GROUPS ONLY</p>
      <div class="weekend-status" :data-weekend-status="offer.status">{{ statusLabel }}</div>
      <h1 id="weekend-title">这两个分组，<br /><span>默认倍率下调。</span></h1>
      <p class="weekend-lead">即刻起，到 11月23日 00:00（北京时间）。<br class="weekend-break" />不是从当天 0 点追溯生效。</p>
      <div v-if="countdown" class="weekend-clock"><span>调整窗口剩余</span><strong>{{ countdown }}</strong></div>
      <p class="weekend-note">只有读到的分组默认倍率与下面的新数值一致时，才显示正在生效。</p>
    </section>

    <section class="weekend-offers" aria-label="两个分组">
      <article v-for="line in offer.lines" :key="line.id">
        <div class="weekend-kicker"><span>{{ line.label }}</span><strong>{{ line.saveLabel }}</strong></div>
        <p class="weekend-rate"><s>{{ line.fromLabel }}</s><span aria-hidden="true">→</span><b>{{ line.toLabel }}</b></p>
        <p class="weekend-sub">大约少 {{ line.saveLabel.replace('约 ', '') }}，按分组默认倍率计算。</p>
        <p v-if="line.effectiveFrom && line.effectiveTo" class="weekend-effective">含当前活动系数后，大约 {{ line.effectiveFrom }} → {{ line.effectiveTo }}</p>
        <p v-if="line.userRate !== null" class="weekend-personal">您的专属倍率仍是 {{ formatObservedRate(line.userRate) }}，这次不替换它。</p>
      </article>
    </section>

    <section class="weekend-rules" aria-labelledby="weekend-rules-title">
      <div><p class="weekend-eyebrow">THE DETAILS</p><h2 id="weekend-rules-title">范围就这些。</h2></div>
      <div>
        <article><h3>只动两个分组的默认倍率</h3><p>GPT 分组（id 4）0.37 → 0.22，大约少 40%。Claude 正价分组（id 5，模型广场里的 Claude）1.33 → 1.20，大约少 10%。低价、AZ GPT、图像、订阅和其他分组不变。</p></article>
        <article><h3>大约，不是更大的折扣</h3><p>GPT 实际少 40% 出头，不到 41%，不写成四折。Claude 实际不到一成，不写成九折。页面用「约 40%」「约 10%」。</p></article>
        <article><h3>BLG 九折照旧，另外计算</h3><p>夺冠活动的九折持续到 11月23日 23:00，不并进这次分组倍率，也不互相替代。两边同时打九折时，实付大约是 GPT 0.333 → 0.198、Claude 1.197 → 1.08。这个有效倍率只在页面读到相同的活动系数时才出现。</p></article>
        <article><h3>专属倍率和到期</h3><p>账号上的专属倍率仍优先于分组默认值。11月23日 00:00 后，本页不再把这次调整显示为进行中；是否恢复以模型广场的实时倍率为准。</p></article>
      </div>
    </section>
    <footer class="weekend-footer"><span>GABUGU · 周末分组倍率</span><router-link to="/home">返回官网 ↗</router-link></footer>
  </main>
</template>

<script setup lang="ts">
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import { useWeekendRate } from '@/composables/useWeekendRate'
import { formatObservedRate, type WeekendOfferStatus } from '@/utils/weekendRate'
import { computed } from 'vue'

const { offer, countdown } = useWeekendRate()
const statusCopy: Record<WeekendOfferStatus, string> = {
  active: '正在生效。已核对到这两个分组的新默认倍率。',
  'not-yet': '尚未生效。读到的仍是调整前的默认倍率。',
  unexpected: '读到的倍率与这次调整不一致，因此不显示为生效。',
  unverified: '还没核对到实时分组倍率，因此不显示为生效。登录后会自动核对。',
  ended: '已过 11月23日 00:00，不再显示为进行中。'
}
const statusLabel = computed(() => statusCopy[offer.value.status])
</script>

<style scoped>
.weekend-page { min-height: 100vh; background: #070b12; color: #e7eef8; font-family: Inter, 'Gabugu Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif; font-synthesis: none; }
.weekend-header, .weekend-hero, .weekend-offers, .weekend-rules, .weekend-footer { position: relative; max-width: 1120px; margin-inline: auto; }
.weekend-header { display: flex; align-items: center; justify-content: space-between; padding: 32px 0; border-bottom: 1px solid #ffffff14; }
.weekend-brand { display: inline-flex; align-items: center; gap: 13px; text-decoration: none; font-size: 21px; }
.weekend-brand img { object-fit: contain; }
.weekend-console { font-size: 14px; color: #c5d4e6; text-decoration: none; }
.weekend-console span { margin-left: 14px; color: #9ec1ff; }
.weekend-hero { padding-block: 72px 36px; }
.weekend-eyebrow { display: flex; align-items: center; gap: 10px; font-size: 13px; letter-spacing: .16em; color: #8ea0b8; }
.weekend-eyebrow span { width: 24px; height: 1px; background: #9ec1ff; }
.weekend-status { margin-top: 28px; max-width: 640px; font-size: 15px; line-height: 1.6; color: #d5e4ff; }
.weekend-hero h1 { margin: 16px 0 18px; font-size: clamp(42px, 6vw, 72px); line-height: 1.16; font-weight: 600; letter-spacing: -.03em; }
.weekend-hero h1 span { color: #9ec1ff; }
.weekend-lead { font-size: 18px; line-height: 1.7; color: #b7c6d8; }
.weekend-clock { display: flex; align-items: center; gap: 18px; margin-top: 26px; }
.weekend-clock span { font-size: 13px; color: #8ea0b8; }
.weekend-clock strong { font-size: 24px; font-weight: 500; font-variant-numeric: tabular-nums; }
.weekend-note { margin-top: 12px; font-size: 13px; color: #8ea0b8; }
.weekend-offers { display: grid; grid-template-columns: 1fr 1fr; gap: 18px; }
.weekend-offers article { padding: 28px; border: 1px solid #9ec1ff33; border-radius: 12px; background: #101820; }
.weekend-kicker { display: flex; justify-content: space-between; gap: 12px; align-items: baseline; font-size: 16px; }
.weekend-kicker strong { color: #9ec1ff; font-size: 28px; font-weight: 560; letter-spacing: -.03em; }
.weekend-rate { display: flex; align-items: baseline; gap: 16px; margin-top: 18px; font-size: 42px; font-weight: 520; letter-spacing: -.04em; }
.weekend-rate s { color: #8ea0b8; font-size: 28px; text-decoration-thickness: 2px; }
.weekend-rate span { color: #9ec1ff; font-size: 28px; }
.weekend-sub, .weekend-effective, .weekend-personal { margin-top: 12px; font-size: 13px; line-height: 1.7; color: #a9b9cc; }
.weekend-personal { color: #d5e4ff; }
.weekend-rules { display: grid; grid-template-columns: .8fr 1.4fr; gap: 48px; padding-block: 72px; }
.weekend-rules h2 { margin-top: 14px; font-size: 32px; font-weight: 560; }
.weekend-rules article { border-top: 1px solid #ffffff18; padding: 20px 0; }
.weekend-rules h3 { margin-bottom: 8px; font-size: 15px; font-weight: 560; }
.weekend-rules p { font-size: 13px; line-height: 1.85; color: #9aabbf; }
.weekend-footer { display: flex; justify-content: space-between; padding: 24px 0 36px; border-top: 1px solid #ffffff18; font-size: 13px; color: #7d8fa6; }
.weekend-footer a { color: #c5d4e6; text-decoration: none; }
.weekend-break { display: none; }
.weekend-page a:focus-visible { outline: 2px solid #9ec1ff; outline-offset: 4px; }
@media (max-width: 1200px) { .weekend-header, .weekend-hero, .weekend-offers, .weekend-rules, .weekend-footer { margin-inline: 32px; } }
@media (max-width: 640px) {
  .weekend-header, .weekend-hero, .weekend-offers, .weekend-rules, .weekend-footer { margin-inline: 20px; }
  .weekend-header { padding-block: 22px; }
  .weekend-brand { font-size: 16px; }
  .weekend-hero { padding-top: 42px; }
  .weekend-hero h1 { font-size: 40px; }
  .weekend-break { display: block; }
  .weekend-offers, .weekend-rules { grid-template-columns: 1fr; }
  .weekend-rate { font-size: 36px; }
  .weekend-rules { gap: 20px; padding-top: 48px; }
}
</style>
