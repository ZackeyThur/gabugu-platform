<template>
  <AppLayout :auto-start-tour="false" class="guide-layout">
    <div class="beginner-guide" data-testid="beginner-guide">
      <header class="guide-hero">
        <div class="guide-hero-copy">
          <p class="guide-eyebrow">{{ t('guide.nav') }}</p>
          <h2>{{ t('guide.headline') }}</h2>
          <p class="guide-intro">{{ t('guide.intro') }}</p>
          <div class="guide-hero-actions">
            <RouterLink :to="chapterLink('setup')" class="guide-button guide-button-primary"><span>{{ t('guide.startLearning') }}</span><Icon name="arrowRight" size="sm" /></RouterLink>
            <RouterLink :to="chapterLink('basics')" class="guide-text-link"><span>{{ t('guide.jumpToSetup') }}</span></RouterLink>
          </div>
        </div>
        <GuideHeroArt />
      </header>

      <ol class="guide-journey" :aria-label="t('guide.readGuide')">
        <li v-for="(item, index) in journey" :key="item"><span class="journey-number">{{ String(index + 1).padStart(2, '0') }}</span><span class="journey-label">{{ t(`guide.journey.${item}`) }}</span><Icon v-if="index < 3" name="chevronRight" size="sm" /></li>
      </ol>

      <nav class="guide-mobile-nav" :aria-label="t('guide.toc')">
        <RouterLink v-for="chapter in guideChapters" :key="chapter" :to="chapterLink(chapter)" :aria-current="activeChapter === chapter ? 'location' : undefined">{{ t(`guide.chapters.${chapter}`) }}</RouterLink>
      </nav>

      <div class="guide-body">
        <div class="guide-content">
          <section id="basics" class="guide-section" aria-labelledby="basics-title">
            <div class="guide-section-heading"><p class="guide-eyebrow"><span>01</span>{{ t('guide.chapters.basics') }}</p><h2 id="basics-title">{{ t('guide.basics.title') }}</h2><p>{{ t('guide.basics.intro') }}</p></div>
            <GuidePrinciples />
            <details class="guide-terms-disclosure"><summary>{{ t('guide.basics.termsTitle') }}<Icon name="chevronDown" size="sm" /></summary>
              <div class="guide-glossary">
              <details v-for="term in guideTerms" :key="term"><summary>{{ t(`guide.terms.${term}.title`) }}<Icon name="plus" size="sm" /></summary><p>{{ t(`guide.terms.${term}.body`) }}</p></details>
              </div>
            </details>
          </section>

          <section id="setup" class="guide-section" aria-labelledby="setup-title">
            <div class="guide-section-heading"><p class="guide-eyebrow"><span>02</span>{{ t('guide.chapters.setup') }}</p><h2 id="setup-title">{{ t('guide.setup.title') }}</h2><p>{{ t('guide.setup.intro') }}</p></div>
            <div class="guide-selectors">
              <fieldset><legend>{{ t('guide.setup.osLabel') }}</legend><div class="guide-segmented"><button v-for="system in systems" :key="system" type="button" :aria-pressed="os === system" :data-testid="`os-${system}`" @click="os = system">{{ t(`guide.setup.${system}`) }}</button></div></fieldset>
              <fieldset><legend>{{ t('guide.setup.clientLabel') }}</legend><div class="guide-segmented"><button v-for="tool in clients" :key="tool" type="button" :aria-pressed="client === tool" :data-testid="`client-${tool}`" @click="client = tool"><PlatformIcon :platform="tool === 'claude' ? 'anthropic' : 'openai'" size="lg" aria-hidden="true" /><span>{{ t(`guide.setup.${tool}`) }}</span></button></div></fieldset>
            </div>
            <div class="guide-route-summary" aria-live="polite"><span class="guide-route-icon"><PlatformIcon :platform="client === 'codex' ? 'openai' : 'anthropic'" size="lg" /></span><div><strong>{{ clientName }}</strong><p>{{ t(`guide.setup.${client}Description`) }}</p></div><span v-if="client === 'codex'" class="guide-route-badge">{{ t('guide.desktop.badge') }}</span></div>

            <ol class="guide-steps">
              <li v-for="(step, index) in currentSteps" :key="step" class="guide-step" :data-testid="`step-${step}`">
                <div class="guide-step-header"><span class="guide-step-number">{{ String(index + 1).padStart(2, '0') }}</span><h3>{{ stepText(step, 'title') }}</h3></div>
                <div class="guide-step-content">
                  <p>{{ stepText(step, 'body') }}</p>

                  <template v-if="step === 'prepare'">
                    <p>{{ preparation }}</p>
                    <div class="guide-actions"><RouterLink v-for="link in prepareLinks" :key="link.id" :to="link.path" class="guide-link-button">{{ t(`guide.actions.${link.label}`) }}<Icon name="arrowRight" size="xs" /></RouterLink></div>
                  </template>
                  <template v-else-if="step === 'choose'">
                    <div class="guide-selected-tool"><span class="guide-tool-mark" :class="client"><PlatformIcon :platform="client === 'claude' ? 'anthropic' : 'openai'" size="lg" aria-hidden="true" /></span><div><strong>{{ clientName }}</strong><p>{{ t(`guide.setup.${client}Description`) }}</p></div></div>
                    <RouterLink to="/keys" class="guide-text-link">{{ t('guide.actions.keys') }}<Icon name="arrowRight" size="xs" /></RouterLink>
                  </template>
                  <template v-else-if="step === 'installSwitch'">
                    <div class="guide-install-note"><Icon name="download" size="md" /><p>{{ t(`guide.steps.installSwitch.${os}`) }}</p></div>
                    <a :href="guideResources[0].href" target="_blank" rel="noopener noreferrer" class="guide-link-button">{{ t('guide.actions.download') }} CC Switch<Icon name="externalLink" size="xs" /></a>
                  </template>
                  <template v-else-if="step === 'installClient'">
                    <template v-if="client === 'codex'">
                      <p>{{ t(`guide.desktop.install.${os}`) }}</p>
                      <div class="guide-actions"><a :href="codexDesktopResources.download" class="guide-link-button" data-testid="codex-desktop-download" target="_blank" rel="noopener noreferrer"><Icon name="download" size="sm" />{{ t('guide.desktop.download') }}<Icon name="externalLink" size="xs" /></a><a :href="clientDocs" class="guide-text-link" target="_blank" rel="noopener noreferrer">{{ t('guide.desktop.installDocs') }}<Icon name="externalLink" size="xs" /></a></div>
                      <p class="guide-small-note">{{ t('guide.desktop.sourceNotice') }}</p>
                    </template>
                    <template v-else>
                      <p>{{ t('guide.steps.installClient.instruction') }}</p>
                      <a :href="clientDocs" target="_blank" rel="noopener noreferrer" class="guide-text-link">{{ clientName }} · {{ t('guide.actions.officialDocs') }}<Icon name="externalLink" size="xs" /></a>
                      <GuideMedia src="/guide/ccs-apps.webp" :width="2144" :height="1422" :title="t('guide.media.ccsAppsTitle')" :alt="t('guide.media.ccsAppsAlt')" source="https://github.com/farion1231/cc-switch/blob/main/assets/release-notes/v4.0.0/apps-zh.png" version="4.0.0" />
                    </template>
                  </template>
                  <template v-else-if="step === 'connect'">
                    <div class="guide-callout" :data-testid="manualOnly ? 'manual-route' : 'import-route'"><Icon name="infoCircle" size="md" /><p>{{ connectText(manualOnly ? 'manual' : 'import') }}</p></div>
                    <RouterLink to="/keys" class="guide-link-button">{{ t('guide.actions.keys') }}<Icon name="arrowRight" size="xs" /></RouterLink>
                    <div class="guide-media-pair">
                      <GuideMedia v-if="client === 'codex'" src="/guide/keys-create-codex-detail.webp" full-src="/guide/keys-create-codex.webp" :width="472" :height="343" :title="t('guide.media.keysCodexTitle')" :alt="t('guide.media.keysCodexAlt')" fit="contain" />
                      <GuideMedia v-else src="/guide/keys-create.webp" :width="512" :height="768" :title="t('guide.media.keysCreateTitle')" :alt="t('guide.media.keysCreateAlt')" />
                      <div v-if="!manualOnly || client === 'codex'" class="guide-connect-tools">
                        <GuideMedia v-if="!manualOnly" src="/guide/keys-actions-focused.webp" :width="300" :height="136" :title="t('guide.media.keysActionsTitle')" :alt="t('guide.media.keysActionsAlt')" fit="contain" compact />
                        <details v-if="client === 'codex'" class="guide-config-help"><summary>{{ t('guide.desktop.connect.pathTitle') }}<Icon name="chevronDown" size="sm" /></summary><p>{{ t(os === 'windows' ? 'guide.desktop.connect.pathWindows' : 'guide.desktop.connect.pathMac') }}</p><div class="guide-path"><code>{{ configFolder }}</code><button type="button" class="guide-copy" @click="copyToClipboard(configFolder)"><Icon name="copy" size="sm" />{{ t('guide.desktop.connect.copyPath') }}</button></div><p>{{ t('guide.desktop.connect.restart') }}</p><a :href="codexDesktopResources.gateway" target="_blank" rel="noopener noreferrer" class="guide-text-link">{{ t('guide.desktop.gatewayDocs') }}<Icon name="externalLink" size="xs" /></a></details>
                      </div>
                    </div>
                  </template>
                  <template v-else-if="step === 'launch'">
                    <template v-if="client === 'codex'">
                      <ol class="guide-desktop-actions"><li v-for="(action, actionIndex) in desktopActions" :key="action"><span>{{ actionIndex + 1 }}</span><div><strong>{{ t(`guide.desktop.launch.${action}`) }}</strong><p>{{ t(action === 'restart' ? `guide.desktop.launch.restartBody${os === 'windows' ? 'Windows' : 'Mac'}` : `guide.desktop.launch.${action}Body`) }}</p></div></li></ol>
                      <GuideDesktopPreview />
                    </template>
                    <template v-else>
                    <figure class="guide-provider-example" :aria-label="t('guide.media.ccsProviderAlt')">
                      <figcaption><span>{{ t('guide.media.ccsProviderTitle') }}</span><span>{{ t('guide.media.schematicLabel') }}</span></figcaption>
                      <div class="provider-example-row"><span class="guide-tool-mark" :class="client"><PlatformIcon :platform="client === 'claude' ? 'anthropic' : 'openai'" size="lg" aria-hidden="true" /></span><div><strong>{{ clientName }}</strong><small>GABUGU</small></div><Icon name="arrowRight" size="md" /><span class="provider-enabled"><Icon name="checkCircle" size="sm" />{{ t('guide.steps.launch.enabled') }}</span></div>
                      <p>{{ t('guide.steps.launch.direct') }}</p>
                    </figure>
                    <p>{{ t(os === 'windows' ? 'guide.setup.terminalWindows' : 'guide.setup.terminalMac') }}</p>
                    <p>{{ t(`guide.steps.launch.${os}`) }}</p>
                    <div class="guide-terminal">
                      <div class="guide-terminal-header"><span class="terminal-dots" aria-hidden="true">● ● ●</span><span>{{ os === 'windows' ? 'PowerShell' : 'Terminal' }}</span><span>{{ t('guide.media.schematicLabel') }}</span></div>
                      <pre data-testid="launch-command"><code>{{ launchCommand }}</code></pre>
                      <button type="button" class="guide-copy" data-testid="copy-command" @click="copyToClipboard(launchCommand || '')"><Icon name="copy" size="sm" />{{ t('guide.setup.copyCommand') }}</button>
                    </div>
                    </template>
                  </template>
                  <template v-else-if="step === 'verify'">
                    <div class="guide-prompt"><Icon name="chat" size="md" /><p>{{ t('guide.steps.verify.prompt') }}</p><button type="button" class="guide-copy" data-testid="copy-prompt" @click="copyToClipboard(t('guide.steps.verify.prompt'))"><Icon name="copy" size="sm" />{{ t('guide.setup.copyPrompt') }}</button></div>
                    <p class="guide-small-note">{{ t('guide.steps.verify.costNote') }}</p>
                    <RouterLink v-if="usageLink" :to="usageLink.path" class="guide-link-button">{{ t('guide.actions.usage') }}<Icon name="arrowRight" size="xs" /></RouterLink>
                    <p v-else>{{ t('guide.steps.verify.usageUnavailable') }}</p>
                    <GuideMedia v-if="usageLink" src="/guide/usage.webp" :width="1184" :height="1570" :title="t('guide.media.usageTitle')" :alt="t('guide.media.usageAlt')" />
                  </template>

                  <div class="guide-step-result"><Icon name="checkCircle" size="sm" /><p><span>{{ t('guide.setup.completeLabel') }}</span>{{ resultText(step) }}</p></div>
                </div>
              </li>
            </ol>
          </section>

          <section id="resources" class="guide-section" aria-labelledby="resources-title">
            <div class="guide-section-heading"><p class="guide-eyebrow"><span>03</span>{{ t('guide.chapters.resources') }}</p><h2 id="resources-title">{{ t('guide.resources.title') }}</h2><p>{{ t('guide.resources.intro') }}</p></div>
            <h3 class="guide-subtitle">{{ t('guide.resources.externalTitle') }}</h3>
            <div class="guide-resources">
              <a v-for="resource in guideResources" :key="resource.id" :href="resource.href" class="guide-resource" target="_blank" rel="noopener noreferrer"><span class="guide-resource-mark" :class="resource.id"><PlatformIcon v-if="resource.id === 'claude' || resource.id === 'codex'" :platform="resource.id === 'claude' ? 'anthropic' : 'openai'" size="lg" aria-hidden="true" /><GuideSymbol v-else :kind="resource.id === 'ccswitch' ? 'switch' : 'book'" /></span><strong>{{ t(`guide.resources.${resource.id}Title`) }}</strong><p>{{ t(`guide.resources.${resource.id}Description`) }}</p><span class="guide-resource-action">{{ t(`guide.actions.${resource.action}`) }}<Icon name="externalLink" size="sm" /></span></a>
            </div>
            <p class="guide-small-note">{{ t('guide.resources.linux') }} <a :href="guideResources[0].href" target="_blank" rel="noopener noreferrer">{{ t('guide.actions.download') }} CC Switch ↗</a></p>
            <h3 class="guide-subtitle">{{ t('guide.resources.sdkTitle') }}</h3>
            <p class="guide-small-note">{{ t('guide.resources.sdkIntro') }}</p>
            <div class="guide-protocols" data-testid="guide-protocol-reference">
              <article v-for="proto in protocolEntries" :key="proto.id" class="guide-protocol-card">
                <div class="guide-protocol-head">
                  <strong>{{ t(`guide.resources.protocols.${proto.id}.title`) }}</strong>
                  <span>{{ t(`guide.resources.protocols.${proto.id}.models`) }}</span>
                </div>
                <div class="guide-path">
                  <code>{{ proto.endpoint }}</code>
                  <button type="button" class="guide-copy" @click="copyToClipboard(proto.endpoint)"><Icon name="copy" size="sm" />{{ t('guide.resources.copyEndpoint') }}</button>
                </div>
                <p>{{ t(`guide.resources.protocols.${proto.id}.auth`) }}</p>
              </article>
            </div>
            <h3 class="guide-subtitle">{{ t('guide.resources.internalTitle') }}</h3>
            <div class="guide-internal-links"><RouterLink v-for="link in internalLinks" :key="link.id" :to="link.path"><Icon :name="link.icon" size="md" /><span><strong>{{ t(`guide.actions.${link.label}`) }}</strong><small>{{ t(`guide.resources.${link.id}Description`) }}</small></span><Icon name="arrowRight" size="sm" /></RouterLink></div>
          </section>

          <section id="faq" class="guide-section" aria-labelledby="faq-title">
            <div class="guide-section-heading"><p class="guide-eyebrow"><span>04</span>{{ t('guide.chapters.faq') }}</p><h2 id="faq-title">{{ t('guide.faq.title') }}</h2><p>{{ t('guide.faq.intro') }}</p></div>
            <div class="guide-faq"><details v-for="(item, index) in guideFaqs" :key="item"><summary><span>{{ String(index + 1).padStart(2, '0') }}</span><span class="guide-faq-question">{{ faqText(item, 'question') }}</span><Icon name="plus" size="sm" /></summary><p>{{ faqText(item, 'answer') }}</p><RouterLink :to="chapterLink(['key', 'model'].includes(item) ? 'basics' : 'setup')" class="guide-text-link">{{ t('guide.actions.openGuide') }}<Icon name="arrowRight" size="xs" /></RouterLink></details></div>
          </section>

          <footer class="guide-footer"><div><h2>{{ t('guide.footer.title') }}</h2><p>{{ t('guide.footer.body') }}</p></div><RouterLink to="/keys" class="guide-button guide-button-primary">{{ t('guide.actions.keys') }}<Icon name="arrowRight" size="sm" /></RouterLink><small>{{ t('guide.footer.maintenance') }}</small></footer>
        </div>

        <aside class="guide-toc"><nav :aria-label="t('guide.toc')"><p>{{ t('guide.toc') }}</p><RouterLink v-for="(chapter, index) in guideChapters" :key="chapter" :to="chapterLink(chapter)" :aria-current="activeChapter === chapter ? 'location' : undefined"><span>0{{ index + 1 }}</span>{{ t(`guide.chapters.${chapter}`) }}</RouterLink><RouterLink to="/guide" class="guide-toc-top"><Icon name="arrowUp" size="xs" />{{ t('guide.backToTop') }}</RouterLink></nav></aside>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import GuideMedia from '@/components/Guide/GuideMedia.vue'
import GuidePrinciples from '@/components/Guide/GuidePrinciples.vue'
import GuideHeroArt from '@/components/Guide/GuideHeroArt.vue'
import GuideDesktopPreview from '@/components/Guide/GuideDesktopPreview.vue'
import GuideSymbol from '@/components/Guide/GuideSymbol.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { codexDesktopResources, detectGuideOs, getGuideLinks, guideChapters, guideDesktopSteps, guideFaqs, guideLaunchCommands, guideResources, guideSteps, guideTerms, type GuideClient, type GuideOs } from '@/data/beginnerGuide'

const { t } = useI18n()
const app = useAppStore()
const auth = useAuthStore()
const { copyToClipboard } = useClipboard()
const systems: GuideOs[] = ['windows', 'macos']
const clients: GuideClient[] = ['codex', 'claude']
const journey = ['understand', 'prepare', 'configure', 'verify'] as const
const os = ref<GuideOs>(detectGuideOs(typeof navigator === 'undefined' ? '' : navigator.platform))
const client = ref<GuideClient>('codex')
const currentSteps = computed(() => client.value === 'codex' ? guideDesktopSteps : guideSteps)
const desktopActions = ['restart', 'folder', 'chat'] as const
const activeChapter = ref<string>('basics')
const internalLinks = computed(() => getGuideLinks(app.cachedPublicSettings, auth.isSimpleMode))
const prepareLinks = computed(() => internalLinks.value.filter(link => ['dashboard', 'modelPlaza', 'channels', 'purchase', 'subscriptions'].includes(link.id)))
const usageLink = computed(() => internalLinks.value.find(link => link.id === 'usage'))
const apiOrigin = computed(() => {
  const configured = app.cachedPublicSettings?.api_base_url?.trim() || app.apiBaseUrl?.trim()
  if (configured) return configured.replace(/\/+$/, '')
  if (typeof window !== 'undefined' && window.location?.origin && window.location.origin !== 'null') {
    return window.location.origin.replace(/\/+$/, '')
  }
  return 'https://gabugu.com'
})
const protocolEntries = computed(() => [
  { id: 'openai', endpoint: `${apiOrigin.value}/v1` },
  { id: 'anthropic', endpoint: apiOrigin.value },
  { id: 'gemini', endpoint: apiOrigin.value },
] as const)
const manualOnly = computed(() => app.cachedPublicSettings?.hide_ccs_import_button === true)
const clientName = computed(() => t(`guide.setup.${client.value}`))
const clientDocs = computed(() => client.value === 'codex' ? codexDesktopResources.setup : guideResources.find(resource => resource.id === 'claude')!.href)
const launchCommand = computed(() => guideLaunchCommands(os.value, client.value))
const configFolder = computed(() => os.value === 'windows' ? '%USERPROFILE%\\.codex' : '~/.codex')
const preparation = computed(() => {
  if (!internalLinks.value.some(link => link.id === 'purchase' || link.id === 'subscriptions')) return t('guide.steps.prepare.unavailable')
  const mode = resolveSiteBillingMode(app.cachedPublicSettings)
  return t(`guide.steps.prepare.${mode === 'recharge_only' ? 'recharge' : mode === 'subscription_only' ? 'subscription' : 'both'}`)
})
const chapterLink = (chapter: string) => ({ path: '/guide', hash: `#${chapter}` })
function resultText(step: typeof guideSteps[number]) {
  if (step === 'connect') return connectText(manualOnly.value ? 'manualResult' : 'importResult')
  if (step === 'verify' && !usageLink.value) return t('guide.steps.verify.simpleResult')
  return stepText(step, 'result')
}
function stepText(step: typeof guideSteps[number], field: 'title' | 'body' | 'result') {
  if (field === 'body' && step === 'verify' && !usageLink.value) return t('guide.steps.verify.simpleBody')
  if (field === 'body' && client.value === 'codex' && manualOnly.value && ['installSwitch', 'connect'].includes(step)) return t(`guide.desktop.steps.${step}.manualBody`)
  return t(`guide.${client.value === 'codex' ? 'desktop.' : ''}steps.${step}.${field}`)
}
function connectText(field: 'import' | 'manual' | 'importResult' | 'manualResult') {
  return t(`guide.${client.value === 'codex' ? 'desktop' : 'steps'}.connect.${field}`)
}
function faqText(item: typeof guideFaqs[number], field: 'question' | 'answer') {
  if (client.value === 'codex' && item === 'command') return t(`guide.desktop.faq.command${field === 'question' ? 'Question' : 'Answer'}`)
  if (client.value === 'codex' && item === 'login' && field === 'answer') return t('guide.desktop.faq.loginAnswer')
  return t(`guide.faq.${item}.${field}`)
}
let observer: IntersectionObserver | undefined
onMounted(() => {
  if (typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver(entries => {
    for (const entry of entries) if (entry.isIntersecting) activeChapter.value = entry.target.id
  }, { rootMargin: '-130px 0px -55% 0px', threshold: 0 })
  guideChapters.forEach(id => {
    const element = document.getElementById(id)
    if (element) observer?.observe(element)
  })
})
onBeforeUnmount(() => observer?.disconnect())
</script>

<style scoped src="../../styles/beginner-guide.css"></style>
