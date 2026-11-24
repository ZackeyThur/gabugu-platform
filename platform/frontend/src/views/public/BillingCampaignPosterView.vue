<template>
  <main class="blg-poster">
    <div class="poster-art" aria-hidden="true"></div>
    <div class="poster-content">
      <div class="poster-brand"><img src="/brand/gabugu-logo-dark.png" alt="" /><GabuguWordmark theme="dark" /></div>
      <p class="poster-kicker">BLG CHAMPIONS · GABUGU CELEBRATES</p>
      <h1>BLG夺冠。<br /><span>说到做到。</span></h1>
      <div class="poster-offers">
        <div class="poster-offer"><span>首日GPT指定分组</span><strong>2<em>折</em></strong></div>
        <div class="poster-offer"><span>Claude / DeepSeek / Gemini<br />指定分组，首日</span><strong>5<em>折</em></strong></div>
      </div>
      <p class="poster-week">随后四类指定分组 <strong>9折一周</strong></p>
      <div class="poster-dates" v-if="campaign"><p>首日24小时：{{ formatCampaignTime(campaign.starts_at) }} — {{ formatCampaignTime(campaign.day_ends_at) }}</p><p>随后完整7天：{{ formatCampaignTime(campaign.day_ends_at) }} — {{ formatCampaignTime(campaign.ends_at) }}</p></div>
      <p v-else class="poster-dates">具体活动时间请查看官网活动页</p>
      <div class="poster-bottom"><div><strong>gabugu.com/events/blg</strong><p>新老用户 · 已有余额同享 · 调用自动优惠</p></div><div class="poster-qr" ref="qrRoot"><canvas ref="qrCanvas" aria-label="活动规则二维码"></canvas></div></div>
      <p class="poster-fine">北京时间。按站内常规实付价打折，仅限活动指定分组的文本Token费用。两阶段不叠加。<br />固定订阅、生图、视频、语音、独立工具及缓存存储费用按原规则；完整范围见活动页。</p>
    </div>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import QRCode from 'qrcode'
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import { useBillingCampaign } from '@/composables/useBillingCampaign'
import { formatCampaignTime } from '@/utils/billingCampaign'
const { campaign } = useBillingCampaign()
const qrCanvas = ref<HTMLCanvasElement>()
const qrRoot = ref<HTMLElement>()
onMounted(async () => {
  if (qrCanvas.value) await QRCode.toCanvas(qrCanvas.value, 'https://gabugu.com/events/blg', { margin: 1, width: 120, color: { dark: '#0c1421', light: '#f2dfbd' } })
})
</script>

<style scoped>
.blg-poster { position: relative; min-height: 100vh; overflow: hidden; background: #080e18; color: #ecf1f8; font-family: Inter, 'Gabugu Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif; }
.poster-art { position: absolute; inset: 0; background: url('/brand/blg/championship-background-v1.webp') 76% center/cover no-repeat; mask-image: linear-gradient(90deg,transparent,#000 70%); }
.poster-content { position: relative; padding: 6vw 6.5vw 4vw; max-width: 100%; }
.poster-brand { display: flex; align-items: center; gap: 17px; font-size: 31px; }
.poster-brand img { width: 51px; height: 57px; object-fit: contain; }
.poster-kicker { margin-top: 45px; color: #8294ac; font-size: 12px; letter-spacing: .18em; }
h1 { margin-top: 30px; font-size: 100px; line-height: 1.15; font-weight: 600; letter-spacing: -.035em; }
h1>span { color: #e3c18a; }
.poster-offers { display: flex; gap: 56px; margin-top: 42px; max-width: 690px; }
.poster-offer { display: flex; flex-direction: column; gap: 12px; }
.poster-offer>span { font-size: 20px; color: #d5dfec; line-height: 1.5; min-height: 60px; }
.poster-offer strong { font-size: 118px; line-height: 1; font-weight: 500; color: #e3c18a; letter-spacing: -.07em; }
.poster-offer em { font-size: 30px; margin-left: 12px; font-weight: 400; font-style: normal; }
.poster-week { margin-top: 38px; font-size: 25px; color: #c1cedf; }
.poster-week strong { font-weight: 500; color: #f2dfbd; }
.poster-dates { font-size: 14px; color: #9bacbf; line-height: 1.9; margin-top: 22px; }
.poster-bottom { display: flex; align-items: center; gap: 38px; margin-top: 40px; }
.poster-bottom strong { font-size: 25px; font-weight: 500; letter-spacing: .01em; }
.poster-bottom p { font-size: 15px; color: #aebcd0; margin-top: 10px; }
.poster-qr { padding: 6px; border: 1px solid #e3c18a40; border-radius: 4px; background: #f2dfbd; }
.poster-qr canvas { display: block; width: 93px !important; height: 93px !important; }
.poster-fine { font-size: 12px; color: #899bb3; line-height: 1.8; margin-top: 27px; max-width: 750px; }
@media(min-aspect-ratio: 13/10) { .poster-content { padding: 38px 78px 28px; } .poster-brand { font-size: 22px; gap: 12px; } .poster-brand img { width: 37px; height: 42px; } .poster-kicker { margin-top: 24px; font-size: 9px; } h1 { font-size: 64px; margin-top: 16px; } h1 br { display: none; } .poster-offers { margin-top: 20px; gap: 50px; max-width: 830px; } .poster-offer { flex-direction: row; align-items: center; gap: 22px; } .poster-offer>span { min-height: auto; font-size: 15px; } .poster-offer strong { font-size: 83px; } .poster-offer em { font-size: 23px; } .poster-week { font-size: 23px; margin-top: 18px; } .poster-dates { font-size: 11px; margin-top: 12px; } .poster-bottom { margin-top: 15px; gap: 25px; } .poster-bottom strong { font-size: 18px; } .poster-bottom p { font-size: 11px; margin-top: 4px; } .poster-qr canvas { width: 66px !important; height: 66px !important; } .poster-fine { margin-top: 11px; font-size: 10px; } }
@media(min-aspect-ratio: 9/10) and (max-aspect-ratio: 13/10) { .poster-content { padding: 54px 65px 35px; } .poster-brand { font-size: 27px; } .poster-brand img { width: 44px; height: 48px; } .poster-kicker { margin-top: 34px; } h1 { font-size: 88px; margin-top: 20px; } .poster-offers { margin-top: 26px; } .poster-offer strong { font-size: 99px; } .poster-offer>span { font-size: 18px; min-height: 54px; } .poster-week { margin-top: 25px; font-size: 23px; } .poster-dates { font-size: 12px; margin-top: 18px; } .poster-bottom { margin-top: 28px; } .poster-bottom strong { font-size: 23px; } .poster-bottom p { font-size: 13px; } .poster-qr canvas { width: 78px !important; height: 78px !important; } .poster-fine { font-size: 11px; margin-top: 22px; } }
</style>
