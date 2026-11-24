<template>
  <div class="gabugu-auth">
    <aside
      class="auth-story"
      aria-label="Gabugu"
    >
      <router-link
        to="/home"
        class="auth-brand"
      ><LandingBrandMark
          :source="siteLogo"
          alt=""
          width="40"
          height="46"
          class="landing-brand-mark"
        /><GabuguWordmark :name="siteName" theme="dark" /></router-link>
      <div
        class="auth-story-image"
        :style="storyBackdropStyle"
        aria-hidden="true"
      ></div>
      <div class="auth-story-copy"><span class="auth-overline">IDEAS MOVE FURTHER</span>
        <h1>
          {{ isEnglish ? 'A new connection.' : '每一个想法，' }}<br /><span>{{ isEnglish ? 'A new possibility.' : '都有新的可能。' }}</span>
        </h1>
        <p>
          {{ isEnglish ? 'Models, ideas, and workflows. Bring them together, and move further.' : '连接模型、灵感与工作流，让创造从这里开始。' }}
        </p>
      </div>
      <div class="auth-story-footer"><span>INTELLIGENCE CONNECTS</span><span>A BRIGHTER TOMORROW.</span></div>
    </aside>
    <main class="auth-main">
      <div class="auth-topbar"><router-link
          to="/home"
          class="auth-back"
        >
          <Icon
            name="arrowLeft"
            size="sm"
          />{{ isEnglish ? 'Back to home' : '返回首页' }}
        </router-link>
        <LocaleSwitcher />
      </div>
      <div class="auth-form-shell">
        <router-link
          to="/home"
          class="auth-mobile-brand"
        ><LandingBrandMark
            :source="siteLogo"
            alt=""
            width="32"
            height="40"
            class="landing-brand-mark"
          /><GabuguWordmark :name="siteName" theme="dark" /></router-link>
        <div
          v-if="settingsLoaded"
          class="auth-product-label"
        ><span class="auth-label-line"></span>
          <p data-testid="auth-subtitle">{{ siteSubtitle }}</p>
        </div>
        <div class="auth-form-card">
          <slot />
        </div>
        <div class="auth-form-footer">
          <slot name="footer" />
        </div>
      </div>
      <div class="auth-copyright">© {{ currentYear }} {{ siteName }}<span>IDEAS MOVE FURTHER.</span></div>
    </main>
  </div>
</template>
<script setup lang="ts">
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import {
  computed,
  onMounted
} from 'vue'
import {
  useI18n
} from 'vue-i18n'
import {
  useAppStore
} from '@/stores'
import {
  sanitizeUrl
} from '@/utils/url'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import LandingBrandMark from '@/components/landing/LandingBrandMark.vue'
import authBackdropUrl from '@/assets/brand/auth-backdrop.webp'
const appStore = useAppStore()
const storyBackdropStyle = {
  '--auth-backdrop': `url("${authBackdropUrl}")`
}
const {
  locale
} = useI18n()
const isEnglish = computed(() => !String(locale?.value || 'zh').toLowerCase().startsWith('zh'))
const siteName = computed(() => appStore.siteName || 'Gabugu')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', {
  allowRelative: true,
  allowDataUrl: true
}))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Gabugu API gateway')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)
const currentYear = new Date().getFullYear()
onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
<style scoped>
.gabugu-auth {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(540px, 1fr);
  background: #08080d;
  color: #e5e7eb;
  color-scheme: dark;
  font-family: Inter, "Gabugu Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif;
  font-size: 17px;
  line-height: 1.7;
  letter-spacing: 0
}

.auth-story {
  position: sticky;
  top: 0;
  height: 100vh;
  height: 100svh;
  padding: 42px clamp(36px, 4vw, 72px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid #ffffff12;
  background: #080a10;
  isolation: isolate
}

.auth-brand {
  display: inline-flex;
  align-items: center;
  gap: 13px;
  width: fit-content;
  color: #f2f5fc
}

.auth-brand :deep(.landing-brand-mark),
.auth-mobile-brand :deep(.landing-brand-mark) {
  object-fit: contain
}

.auth-brand>span {
  font-size: 23px;
  letter-spacing: .08em;
  font-weight: 650
}

.auth-story-image {
  position: absolute;
  inset: 0;
  z-index: -1;
  pointer-events: none;
  background-image:
    linear-gradient(180deg, #07090f 0%, #07090fe6 28%, #07090e38 48%, #07090e00 72%, #07090ed9 100%),
    var(--auth-backdrop);
  background-size: cover;
  background-position: center 14%;
  opacity: .98;
  animation: auth-horizon 1.4s ease both
}

.auth-story-copy {
  margin-top: clamp(36px, 7vh, 84px);
  margin-bottom: auto;
  padding: 0 0 180px
}

.auth-overline {
  font-size: 12px;
  letter-spacing: .2em;
  color: #a7b6cf;
  display: block;
  margin-bottom: 22px
}

.auth-story-copy h1 {
  font-size: clamp(42px, 3.7vw, 64px);
  line-height: 1.4;
  letter-spacing: 0;
  font-weight: 500;
  margin: 0
}

.auth-story-copy h1>span {
  color: #c0cce0
}

.auth-story-copy p {
  max-width: 26em;
  font-size: 22px;
  color: #b6c2d6;
  line-height: 1.8;
  text-wrap: pretty;
  margin-top: 28px
}

.auth-story-footer {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  font-size: 11px;
  letter-spacing: .1em;
  color: #9ba9bf
}

.auth-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 35px 48px 27px;
  background: radial-gradient(ellipse at 85% 0%, #1a263b22, transparent 45%), #0c0d13
}

.auth-topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px
}

.auth-back {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 44px;
  font-size: 15px;
  color: #aab6ca;
  transition: color .2s
}

.auth-back:hover {
  color: #e7efff
}

.auth-topbar :deep(button) {
  min-height: 44px;
  font-size: 15px;
  color: #b7c2d5
}

.auth-topbar :deep(button > .text-base) {
  display: none
}

.auth-topbar :deep(button > .hidden) {
  display: inline
}

.auth-topbar :deep(button:hover) {
  background: #1c2230
}

.auth-topbar :deep(.absolute) {
  background: #171b25;
  border-color: #2b3243
}

.auth-form-shell {
  width: 100%;
  max-width: 460px;
  margin: auto;
  padding: 64px 0 72px;
  animation: auth-form-enter .7s .1s both
}

.auth-product-label {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 29px;
  color: #a9b8cf;
  font-size: 15px;
  line-height: 1.6;
  letter-spacing: 0
}

.auth-product-label p {
  margin: 0
}

.auth-label-line {
  width: 19px;
  height: 1px;
  background: #5c8cdd
}

.auth-mobile-brand {
  display: none
}

.auth-form-card :deep(h2) {
  text-align: left;
  font-size: 36px;
  line-height: 1.4;
  font-weight: 600;
  letter-spacing: 0;
  color: #edf1f9
}

.auth-form-card :deep(.text-center) {
  text-align: left
}

.auth-form-card :deep(.text-center>p) {
  margin-top: 12px;
  line-height: 1.75;
  font-size: 18px
}

/* Slots retain their original behaviour, but share one readable type scale. */
.auth-form-card :deep(.text-sm) {
  font-size: 17px;
  line-height: 1.7
}

.auth-form-card :deep(.text-xs),
.auth-form-card :deep(.input-error-text) {
  font-size: 15px;
  line-height: 1.7
}

.auth-form-card :deep(.text-center > p.text-sm) {
  font-size: 18px;
  line-height: 1.75
}

.auth-form-card :deep(form) {
  padding-top: 12px
}

.auth-form-card :deep(.input) {
  border: 1px solid #2b3141;
  background: #11141d;
  border-radius: 10px;
  min-height: 56px;
  padding-block: 13px;
  box-shadow: inset 0 1px 2px #0002;
  color: #e5eaf5;
  font-size: 17px;
  line-height: 1.6
}

.auth-form-card :deep(.input.text-xl) {
  font-size: 22px
}

.auth-form-card :deep(.input.pl-11) {
  padding-left: 44px
}

.auth-form-card :deep(.input.pr-11) {
  padding-right: 44px
}

.auth-form-card :deep(.input:focus) {
  border-color: #578fe6;
  box-shadow: 0 0 0 3px #3b82f61c;
  outline: none
}

.auth-form-card :deep(.input::placeholder) {
  color: #98a5bc
}

.auth-form-card :deep(.input:disabled) {
  opacity: .6
}

.auth-form-card :deep(.input-label) {
  color: #c6cfdf;
  font-size: 16px;
  line-height: 1.5;
  font-weight: 500;
  margin-bottom: 10px
}

.auth-form-card :deep(.input-hint) {
  font-size: 15px;
  color: #a9b6cb;
  line-height: 1.7
}

.auth-form-card :deep(.btn-primary) {
  min-height: 56px;
  background: linear-gradient(120deg, #f2f5fc, #dce6f8);
  border: 1px solid #e9effc;
  color: #101827;
  font-size: 17px;
  line-height: 1.5;
  font-weight: 600;
  box-shadow: 0 3px 18px #aacaff13;
  border-radius: 10px
}

.auth-form-card :deep(.btn-primary:hover:not(:disabled)) {
  background: #fff;
  transform: translateY(-1px);
  box-shadow: 0 5px 24px #aacaff20
}

.auth-form-card :deep(.btn-primary:disabled) {
  opacity: .45
}

.auth-form-card :deep(.btn-primary svg) {
  color: currentColor
}

.auth-form-card :deep(.btn-secondary) {
  border: 1px solid #313a4c;
  background: #161b27;
  color: #d1dbee;
  border-radius: 10px;
  font-size: 16px;
  line-height: 1.5;
  min-height: 52px
}

.auth-form-card :deep(.text-gray-900),
.auth-form-card :deep(.text-gray-800) {
  color: #e5ebf7
}

.auth-form-card :deep(.text-gray-700),
.auth-form-card :deep(.text-gray-600) {
  color: #a9b4c9
}

.auth-form-card :deep(.text-gray-500),
.auth-form-card :deep(.text-gray-400) {
  color: #a9b6cb
}

.auth-form-card :deep(.text-primary-600),
.auth-form-card :deep(.text-primary-500) {
  color: #8eb5f4
}

.auth-form-card :deep(.bg-white) {
  background: #121621
}

.auth-form-card :deep(.bg-gray-50),
.auth-form-card :deep(.bg-gray-100) {
  background: #161b27
}

.auth-form-card :deep(.border-gray-200),
.auth-form-card :deep(.border-gray-300) {
  border-color: #303a4e
}

.auth-form-card :deep(.bg-amber-50) {
  background: rgba(245, 158, 11, 0.12)
}

.auth-form-card :deep(.border-amber-200) {
  border-color: rgba(245, 158, 11, 0.3)
}

.auth-form-card :deep(.text-amber-700),
.auth-form-card :deep(.text-amber-800) {
  color: #fde68a
}

.auth-form-card :deep(.bg-green-50) {
  background: rgba(16, 185, 129, 0.12)
}

.auth-form-card :deep(.bg-green-100) {
  background: rgba(16, 185, 129, 0.2)
}

.auth-form-card :deep(.border-green-200) {
  border-color: rgba(16, 185, 129, 0.3)
}

.auth-form-card :deep(.text-green-700),
.auth-form-card :deep(.text-green-800),
.auth-form-card :deep(.text-green-600) {
  color: #86efac
}

.auth-form-card :deep(.bg-red-50) {
  background: rgba(239, 68, 68, 0.12)
}

.auth-form-card :deep(.border-red-200) {
  border-color: rgba(239, 68, 68, 0.32)
}

.auth-form-card :deep(.text-red-700),
.auth-form-card :deep(.text-red-800),
.auth-form-card :deep(.text-red-600) {
  color: #fca5a5
}

.auth-form-card :deep(.bg-blue-50),
.auth-form-card :deep(.bg-primary-50\/70) {
  background: rgba(59, 130, 246, 0.12)
}

.auth-form-card :deep(.border-blue-200),
.auth-form-card :deep(.border-primary-100) {
  border-color: rgba(59, 130, 246, 0.28)
}

.auth-form-card :deep(.text-primary-900),
.auth-form-card :deep(.text-primary-700) {
  color: #dbeafe
}

.auth-form-footer {
  margin-top: 28px;
  font-size: 17px;
  line-height: 1.8;
  text-align: center;
  color: #afbbce
}

.auth-form-footer :deep(p) {
  font-size: inherit;
  line-height: inherit
}

.auth-form-footer :deep(.text-gray-500) {
  color: #afbbce
}

.auth-form-footer :deep(a),
.auth-form-footer :deep(button) {
  color: #aac9ff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-inline-start: 6px;
  padding-block: 6px;
  font-weight: 500
}

.auth-copyright {
  display: flex;
  justify-content: space-between;
  gap: 18px;
  font-size: 12px;
  color: #94a2b9;
  letter-spacing: 0
}

.auth-copyright>span {
  font-size: 11px;
  letter-spacing: .08em
}

.gabugu-auth a:focus-visible,
.gabugu-auth :deep(button:focus-visible) {
  outline: 2px solid #74a8fd;
  outline-offset: 4px
}

@keyframes auth-horizon {
  from {
    opacity: 0;
    transform: scale(1.035)
  }

  to {
    opacity: .96;
    transform: scale(1)
  }
}

@keyframes auth-form-enter {
  from {
    opacity: 0;
    transform: translateY(14px)
  }

  to {
    opacity: 1;
    transform: translateY(0)
  }
}

@media(max-width:1100px) {
  .gabugu-auth {
    grid-template-columns: minmax(0, .9fr) minmax(460px, 1fr)
  }

  .auth-story {
    padding: 35px
  }

  .auth-main {
    padding-inline: 35px
  }

  .auth-story-copy h1 {
    font-size: 42px
  }

  .auth-story-copy p {
    font-size: 18px
  }

  .auth-story-footer {
    flex-direction: column;
    gap: 9px
  }
}

@media(max-width:820px) {
  .gabugu-auth {
    display: block
  }

  .auth-story {
    display: none
  }

  .auth-main {
    min-height: 100vh;
    min-height: 100svh;
    padding: 24px;
    background: radial-gradient(ellipse at 50% 0%, #18356033, transparent 48%), #0c0d13
  }

  .auth-form-shell {
    max-width: 460px;
    padding-top: 44px;
    padding-bottom: 48px
  }

  .auth-mobile-brand {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 10px;
    margin-bottom: 34px;
    font-size: 21px;
    font-weight: 650;
    letter-spacing: .08em;
    color: #f0f4fc
  }

  .auth-product-label {
    margin-bottom: 20px;
    font-size: 15px
  }

  .auth-form-card :deep(h2) {
    font-size: 32px
  }

  .auth-form-card :deep(.input) {
    min-height: 54px;
    font-size: 17px
  }

  .auth-form-card :deep(.btn-primary) {
    min-height: 54px;
    font-size: 17px
  }

  .auth-form-footer {
    font-size: 17px
  }

  .auth-copyright {
    flex-wrap: wrap;
    gap: 8px 16px;
    font-size: 12px
  }

  .auth-copyright>span {
    display: none
  }
}

@media(max-width:360px) {
  .auth-main {
    padding-inline: 20px
  }
}

@media(prefers-reduced-motion:reduce) {

  .gabugu-auth *,
  .gabugu-auth :deep(*) {
    animation: none !important;
    transition: none !important
  }

  .auth-form-card :deep(.btn-primary:hover:not(:disabled)) {
    transform: none
  }
}
</style>
