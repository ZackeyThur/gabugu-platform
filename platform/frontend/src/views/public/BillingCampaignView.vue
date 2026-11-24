<template>
  <main class="blg-page">
    <div class="blg-grid" aria-hidden="true"></div>
    <header class="blg-header">
      <router-link to="/home" class="blg-brand" aria-label="GABUGU首页"><img src="/brand/gabugu-logo-dark.png" width="36" height="40" alt="" /><GabuguWordmark theme="dark" /></router-link>
      <router-link to="/dashboard" class="blg-console">进入控制台 <span aria-hidden="true">↗</span></router-link>
    </header>

    <section class="blg-hero" aria-labelledby="blg-title">
      <p class="blg-eyebrow"><span></span> BLG CHAMPIONS · GABUGU CELEBRATES</p>
      <div class="blg-status">{{ phaseLabel }}</div>
      <h1 id="blg-title">BLG夺冠。<br /><span>说到做到。</span></h1>
      <p class="blg-lead">首日兑现之前的约定，<br class="blg-mobile" />再用一周9折，继续庆祝。</p>
      <template v-if="campaign">
        <div class="blg-clock" v-if="phase !== 'ended' && phase !== 'unavailable'">
          <span>{{ phase === 'scheduled' ? '距离活动开始' : phase === 'championship-day' ? '首日优惠剩余' : '9折活动剩余' }}</span><strong>{{ countdown }}</strong>
        </div>
        <p class="blg-server-note">北京时间 · 优惠资格以服务器受理请求时间为准</p>
      </template>
      <p v-else class="blg-unavailable" role="status">{{ error ? '活动状态暂时无法读取，请稍后刷新；当前价格以模型广场和实际账单为准。' : '正在读取活动安排…' }}</p>
      <div class="blg-actions"><router-link to="/keys" class="blg-button blg-button--primary">使用活动分组 <span aria-hidden="true">↗</span></router-link><a href="#activity-rules" class="blg-button blg-button--quiet">查看完整规则 ↓</a></div>
    </section>

    <section v-if="campaign" class="blg-stages" aria-label="活动安排">
      <article class="blg-day" :class="{ 'is-active': phase === 'championship-day' }">
        <div class="blg-section-top"><span>01 / 首日24小时</span><span v-if="phase === 'championship-day'" class="blg-live">正在进行</span></div>
        <div class="blg-day-price"><span>GPT指定分组</span><strong>2<span>折</span></strong></div>
        <div class="blg-other-price"><span>Claude · DeepSeek · Gemini<br />指定分组</span><strong>5<span>折</span></strong></div>
        <p class="blg-dates">{{ formatCampaignTime(campaign.starts_at) }}<br />至 {{ formatCampaignTime(campaign.day_ends_at) }}</p>
      </article>
      <article class="blg-week" :class="{ 'is-active': phase === 'celebration-week' }">
        <div class="blg-section-top"><span>02 / 随后完整7天</span><span v-if="phase === 'celebration-week'" class="blg-live">正在进行</span></div>
        <p class="blg-week-title">庆祝，接着来。</p>
        <div class="blg-week-price"><strong>9<span>折</span></strong><span>四类指定分组<br />统一享受</span></div>
        <p class="blg-dates">{{ formatCampaignTime(campaign.day_ends_at) }}<br />至 {{ formatCampaignTime(campaign.ends_at) }}</p>
      </article>
    </section>

    <section class="blg-rules" id="activity-rules" aria-labelledby="blg-rules-title">
      <div><p class="blg-eyebrow">THE DETAILS</p><h2 id="blg-rules-title">优惠怎么算，<br />这里写清楚。</h2></div>
      <div class="blg-rule-list">
        <article><h3>仅限指定分组</h3><p>GPT、Claude、DeepSeek、Gemini活动分组参与。其他分组，包括低价分组、AZ GPT、图像生成和订阅套餐，按原规则计费。同一个模型通过其他分组调用也不享受本次优惠。</p><div v-if="campaign" class="blg-groups"><span v-for="group in campaign.groups" :key="group.id">{{ group.name }}</span></div></article>
        <article><h3>按站内常规实付价再打折</h3><p>先按您原有的分组、专属价格和正常服务档计算，再应用本阶段折扣。首日与随后7天的折扣不叠加；9折阶段按常规价的90%计费。</p></article>
        <article><h3>输入、输出和缓存Token费用参与</h3><p>普通文本、推理、编程调用的输入、输出、缓存读取和缓存写入Token费用参与。生图、视频、实时语音、单独计价的搜索工具与缓存存储费用维持原价；包含媒体生成的混合调用不参与Token优惠。</p></article>
        <article><h3>已有余额同享，调用自动优惠</h3><p>新老用户均可参加，无需领券、转发或最低充值。已有API Key绑定活动分组即可直接使用；当前Key属于其他分组时，请在密钥管理中选择活动分组。Codex、Claude Code等客户端按所用Key的分组判断资格。</p></article>
        <article><h3>跨阶段的请求，保持受理时价格</h3><p>首日持续24小时，之后接完整7天9折，总计8天。以服务器受理并确定计价的时间为准；已受理请求的流式输出不中途变价，长连接每一轮新请求分别判断。到期后，新请求恢复常规计费。</p></article>
        <article><h3>每笔优惠，账单可核对</h3><p>用量详情记录活动阶段、原应收、优惠额与实际扣费。价格有疑问时，可通过站内反馈提供请求时间和请求编号，便于核查。</p><router-link to="/usage" class="blg-text-link">查看使用记录 ↗</router-link></article>
      </div>
    </section>
    <footer class="blg-footer"><span>GABUGU · BLG夺冠庆祝活动</span><router-link to="/home">返回官网 ↗</router-link></footer>
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import { useBillingCampaign } from '@/composables/useBillingCampaign'
import { formatCampaignTime } from '@/utils/billingCampaign'
const { campaign, phase, countdown, error } = useBillingCampaign()
const phaseLabel = computed(() => ({ unavailable: '活动安排读取中', scheduled: '即将开始', 'championship-day': '首日优惠正在进行', 'celebration-week': '9折一周正在进行', ended: '本次活动已结束' })[phase.value])
</script>

<style scoped>
.blg-page { --gold: #e3c18a; position: relative; overflow: clip; min-height: 100vh; background: #080e18; color: #eaf0f7; font-family: Inter, 'Gabugu Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif; }
.blg-grid { pointer-events: none; position: absolute; inset: 0; opacity: .3; background: linear-gradient(90deg, transparent calc(100% - 1px), #98acc814 0); background-size: 12.5% 100%; }
.blg-header,.blg-hero,.blg-stages,.blg-rules,.blg-footer { position: relative; max-width: 1120px; margin-inline: auto; }
.blg-header { display: flex; align-items: center; justify-content: space-between; padding: 32px 0; border-bottom: 1px solid #ffffff14; }
.blg-brand { display: inline-flex; align-items: center; gap: 13px; text-decoration: none; font-size: 21px; }
.blg-brand img { object-fit: contain; }
.blg-console,.blg-text-link { font-size: 14px; color: #c0cede; text-decoration: none; }
.blg-console span { margin-left: 14px; color: var(--gold); }
.blg-hero { padding-block: 90px 74px; }
.blg-hero::before { content: ''; pointer-events: none; position: absolute; z-index: 0; top: 0; bottom: -20px; right: -80px; width: 70%; background: url('/brand/blg/championship-background-v1.webp') 75% center / cover no-repeat; mask-image: linear-gradient(90deg,transparent 0%,#000 60%); opacity: .9; }
.blg-hero>* { position: relative; }
.blg-eyebrow { display: flex; align-items: center; gap: 10px; font-size: 13px; letter-spacing: .16em; color: #8796a9; }
.blg-eyebrow span { width: 24px; height: 1px; background: var(--gold); }
.blg-status { margin-top: 30px; font-size: 13px; color: var(--gold); }
.blg-hero h1 { margin: 18px 0 24px; font-size: clamp(52px,7vw,90px); line-height: 1.16; font-weight: 600; letter-spacing: -.04em; }
.blg-hero h1 span { color: var(--gold); }
.blg-lead { font-size: 19px; color: #adbacb; line-height: 1.8; }
.blg-clock { display: flex; align-items: center; gap: 22px; margin-top: 30px; }
.blg-clock>span { font-size: 13px; color: #95a4b8; }
.blg-clock strong { font-size: 24px; font-variant-numeric: tabular-nums; font-weight: 500; }
.blg-server-note { margin-top: 9px; font-size: 13px; color: #6f829b; }
.blg-actions { display: flex; gap: 16px; flex-wrap: wrap; margin-top: 28px; }
.blg-button { display: inline-flex; align-items: center; justify-content: center; gap: 24px; min-height: 48px; padding: 12px 24px; font-size: 14px; text-decoration: none; border-radius: 6px; }
.blg-button--primary { background: var(--gold); color: #101724; font-weight: 600; }
.blg-button--quiet { color: #d1dbe8; border: 1px solid #e4c28a38; }
.blg-stages { display: grid; grid-template-columns: 1.1fr 1fr; gap: 18px; }
.blg-day,.blg-week { padding: 32px; border: 1px solid #ccd9ec22; border-radius: 12px; background: #0e1725; }
.blg-stages .is-active { border-color: #e4c28a85; }
.blg-section-top { display: flex; align-items: center; justify-content: space-between; font-size: 13px; letter-spacing: .06em; color: #b9c8db; }
.blg-live { font-size: 13px; color: var(--gold); background: #e4c28a12; padding: 4px 8px; border-radius: 4px; }
.blg-day-price,.blg-other-price { display: flex; align-items: center; justify-content: space-between; margin-top: 24px; }
.blg-day-price>span,.blg-other-price>span { font-size: 15px; line-height: 1.6; }
.blg-day-price strong,.blg-other-price strong,.blg-week-price strong { font-size: 82px; line-height: 1; font-weight: 500; letter-spacing: -.07em; color: var(--gold); }
.blg-day-price strong span,.blg-other-price strong span,.blg-week-price strong span { font-size: 22px; margin-left: 8px; letter-spacing: 0; }
.blg-other-price { padding-top: 24px; border-top: 1px solid #ffffff0e; }
.blg-other-price strong { font-size: 62px; color: #e6edf7; }
.blg-week-title { margin-top: 31px; font-size: 24px; }
.blg-week-price { display: flex; gap: 28px; align-items: flex-end; margin-top: 28px; }
.blg-week-price strong { font-size: 124px; }
.blg-week-price>span { font-size: 14px; line-height: 1.8; color: #a3b3c8; padding-bottom: 10px; }
.blg-dates { color: #94a5ba; font-size: 13px; line-height: 1.9; margin-top: 28px; }
.blg-rules { display: grid; grid-template-columns: .9fr 1.4fr; gap: 70px; padding-block: 90px; }
.blg-rules h2 { font-size: 32px; line-height: 1.5; font-weight: 500; margin-top: 16px; letter-spacing: -.02em; }
.blg-rule-list article { border-top: 1px solid #ffffff18; padding: 23px 0; }
.blg-rule-list h3 { font-size: 15px; font-weight: 500; margin-bottom: 10px; }
.blg-rule-list p { font-size: 13px; line-height: 1.9; color: #97a8be; }
.blg-groups { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 13px; }
.blg-groups span { border: 1px solid #e4c28a30; color: #d8bf96; padding: 4px 11px; font-size: 13px; border-radius: 4px; }
.blg-text-link { display: inline-block; margin-top: 14px; color: var(--gold); font-size: 13px; }
.blg-footer { display: flex; align-items: center; justify-content: space-between; padding: 25px 0 36px; border-top: 1px solid #ffffff18; font-size: 13px; color: #71839c; }
.blg-footer a { color: #adbed3; }
.blg-unavailable { margin-top: 24px; color: #c3ac83; font-size: 13px; }
.blg-mobile { display: none; }
.blg-page a:focus-visible { outline: 2px solid var(--gold); outline-offset: 5px; }
@media(max-width:1200px) { .blg-header,.blg-hero,.blg-stages,.blg-rules,.blg-footer { margin-inline: 40px; } }
@media(max-width:640px) { .blg-header,.blg-hero,.blg-stages,.blg-rules,.blg-footer { margin-inline: 22px; } .blg-header { padding-block: 23px; } .blg-brand { font-size: 16px; gap: 9px; } .blg-brand img { width: 28px; height: 32px; } .blg-console { font-size: 13px; } .blg-hero { padding-block: 55px 40px; } .blg-eyebrow { font-size: 13px; letter-spacing: .1em; } .blg-hero h1 { font-size: 57px; margin-top: 17px; } .blg-lead { font-size: 16px; } .blg-mobile { display: block; } .blg-clock { gap: 14px; margin-top: 24px; } .blg-clock strong { font-size: 22px; } .blg-actions { gap: 10px; } .blg-button { padding-inline: 16px; font-size: 13px; gap: 12px; } .blg-stages { grid-template-columns: 1fr; gap: 14px; } .blg-day,.blg-week { padding: 25px; } .blg-day-price strong { font-size: 74px; } .blg-week-price strong { font-size: 96px; } .blg-rules { grid-template-columns: 1fr; gap: 28px; padding-block: 55px; } .blg-rules h2 { font-size: 28px; } .blg-footer { align-items: flex-start; gap: 18px; } }
</style>
