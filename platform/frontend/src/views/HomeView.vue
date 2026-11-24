<template>
  <div
    v-if="hasHomeContent"
    class="min-h-screen"
  >
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      :title="siteName"
      class="h-screen w-full border-0"
      sandbox="allow-scripts allow-forms allow-popups allow-popups-to-escape-sandbox"
      referrerpolicy="no-referrer"
      allowfullscreen
    ></iframe>
    <!-- Custom HTML is sanitized lazily with DOMPurify before rendering. -->
    <div
      v-else
      v-html="sanitizedHomeContent"
    ></div>
  </div>
  <CompactHome v-else-if="compactHomeEnabled" />
  <div
    v-else
    ref="landingRoot"
    data-testid="home-page"
    class="gabugu-landing"
    :class="{ staged: surface !== null }"
    :data-paging="pagingMode"
    :lang="isEnglish ? 'en' : 'zh-CN'"
    @focusin="followFocus"
  >
    <EarthStage
      v-if="surface"
      :key="stageRun"
      ref="stage"
      :surface="surface"
      :journey="journey"
      :opening="opening"
      :after="introStartedAt"
      @failed="stageFailed"
    />
    <!-- Where the live Earth cannot be drawn, a still picture of the same sunrise stands behind the page. -->
    <HeroPoster
      v-else
      class="landing-backdrop"
      :anchor="horizonAnchor"
    />
    <!-- As tall as one screen; read by the script, which cannot ask the browser for that height otherwise. -->
    <div
      ref="screenProbe"
      class="landing-screen-probe"
      aria-hidden="true"
    ></div>
    <a
      href="#main-content"
      class="landing-skip"
    >{{ copy.skip }}</a>
    <header
      class="landing-header"
      :class="{ solid: headerSolid }"
    >
      <nav
        class="landing-container landing-nav"
        :aria-label="copy.navigation"
      >
        <router-link
          to="/home"
          class="landing-brand"
          :aria-label="siteName"
          @click="toTop"
        ><LandingBrandMark
            :source="siteLogo"
            alt=""
            width="39"
            height="44"
          /><GabuguWordmark :name="siteName" theme="dark" /></router-link>
        <div class="landing-nav-links">
          <!-- Without a documentation site the integration screen is the documentation. -->
          <a
            v-for="section in headerSections"
            :key="section"
            :href="`#${section}`"
            :data-testid="section === 'connect' && !externalDocsUrl ? 'home-docs-link' : undefined"
            @click.prevent="goTo(section)"
          >{{ copy.screenNames[section] }}</a>
          <a
            v-if="externalDocsUrl"
            :href="externalDocsUrl"
            target="_blank"
            rel="noopener noreferrer"
            data-testid="home-docs-link"
          >{{ copy.docs }}</a>
        </div>
        <div class="landing-nav-actions">
          <LocaleSwitcher />
          <router-link
            v-if="!isAuthenticated"
            to="/login"
            class="landing-sign-in"
          >{{ copy.login }}</router-link>
          <router-link
            :to="entryPath"
            class="landing-button landing-button-small landing-button-blue"
          >{{ isAuthenticated ? copy.console : copy.start }}
            <Icon
              name="arrowRight"
              size="sm"
            />
          </router-link>
        </div>
      </nav>
    </header>
    <main id="main-content" tabindex="-1">
      <!-- 1. What this is. -->
      <section
        class="landing-screen landing-hero"
        data-screen="hero"
        data-earth="below"
      >
        <div class="hero-stage">
          <div class="landing-container hero-content screen-words">
            <div class="hero-eyebrow"><span></span> IDEAS MOVE FURTHER</div>
            <!-- Read as one sentence; drawn in pieces, which rise one after another on load. -->
            <h1 :aria-label="heroHeadline"><span
                v-for="(line, index) in heroLines"
                :key="index"
                class="hero-line"
                aria-hidden="true"
              ><span
                  v-for="(piece, at) in line"
                  :key="at"
                  class="hero-piece"
                >{{ piece }}</span></span></h1>
            <p class="hero-description"><template
                v-for="sentence in heroSentences"
                :key="sentence"
              ><span>{{ sentence }}</span>{{ isEnglish ? ' ' : '' }}</template></p>
            <div class="hero-actions">
              <router-link
                :to="entryPath"
                class="landing-button landing-button-white"
              >{{ isAuthenticated ? copy.console : copy.start }}
                <Icon
                  name="arrowRight"
                  size="sm"
                />
              </router-link>
              <a
                v-if="externalDocsUrl"
                :href="externalDocsUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="landing-text-link"
              >
                {{ copy.readDocs }}<Icon name="arrowRight" size="sm" />
              </a>
              <a
                v-else
                href="#connect"
                class="landing-text-link"
                @click.prevent="goTo('connect')"
              >
                {{ copy.seeHow }}<Icon name="arrowRight" size="sm" />
              </a>
            </div>
          </div>
          <div
            ref="heroGround"
            class="hero-ground"
          >
            <div class="landing-container hero-foot">
              <span data-testid="home-tag-gateway">{{ copy.tagline }}</span>
              <a
                class="hero-scroll"
                href="#models"
                @click.prevent="goTo('models')"
              ><span>{{ copy.explore }}</span>
                <Icon
                  name="arrowDown"
                  size="sm"
                />
              </a>
            </div>
            <BillingCampaignBanner surface="home" inline class="hero-campaign" />
            <WeekendRateBanner surface="home" inline class="hero-campaign" />
          </div>
        </div>
      </section>
      <!-- 2. Which models. -->
      <section
        id="models"
        class="landing-screen"
        data-screen="models"
        data-earth="beside"
        aria-labelledby="landing-models-title"
        tabindex="-1"
      >
        <div class="landing-container screen-column screen-words">
          <div
            class="screen-head"
            data-reveal-part
          >
            <p class="section-kicker"><span>01</span>{{ copy.screenNames.models }}</p>
            <h2 id="landing-models-title">{{ copy.modelTitle }}</h2>
            <p class="section-description">{{ copy.modelDescription }}</p>
          </div>
          <ul
            class="provider-list"
            data-reveal-part
          >
            <li v-for="provider in modelProviders" :key="provider.model">
              <button
                type="button"
                class="provider-example"
                @click="showExample(provider.protocol)"
              >
                <span class="provider-name"><ProviderIcon :provider="provider.icon" :size="24" class="provider-symbol" /><span class="provider-model">{{ provider.model }}</span><span v-if="provider.company" class="provider-company">{{ provider.company }}</span></span>
                <span class="provider-protocol">{{ protocolLabel(provider.protocol) }}</span>
                <span class="provider-action"><span class="landing-visually-hidden">{{ copy.viewExample }}</span><Icon name="arrowRight" size="sm" /></span>
              </button>
            </li>
          </ul>
          <div
            class="screen-foot"
            data-reveal-part
          >
            <router-link
              v-if="catalog"
              :to="catalog.to"
              class="landing-text-link"
            >{{ catalog.label ?? copy.exploreModels }}<Icon name="arrowRight" size="sm" /></router-link>
            <router-link
              v-else
              :to="entryPath"
              class="landing-text-link"
            >{{ isAuthenticated ? copy.console : copy.start }}<Icon name="arrowRight" size="sm" /></router-link>
            <p class="screen-note">{{ copy.modelNote }}</p>
          </div>
        </div>
      </section>
      <!-- 3. How much code changes. -->
      <section
        id="connect"
        class="landing-screen"
        data-screen="connect"
        data-earth="beside"
        aria-labelledby="landing-connect-title"
        tabindex="-1"
      >
        <div class="landing-container screen-column screen-words">
          <div
            class="screen-head"
            data-reveal-part
          >
            <p class="section-kicker"><span>02</span>{{ copy.screenNames.connect }}</p>
            <h2 id="landing-connect-title">{{ copy.connectTitle }}</h2>
            <p class="section-description">{{ connectDescription }}</p>
          </div>
          <div
            class="connection-console"
            data-reveal-part
          >
            <div class="console-bar">
              <div
                class="protocol-tabs"
                role="tablist"
                :aria-label="copy.protocol"
              ><button
                  v-for="protocol in protocols"
                  :id="`protocol-tab-${protocol.id}`"
                  :key="protocol.id"
                  type="button"
                  role="tab"
                  :aria-selected="activeProtocol === protocol.id"
                  :aria-controls="`protocol-panel-${protocol.id}`"
                  :tabindex="activeProtocol === protocol.id ? 0 : -1"
                  :class="{ active: activeProtocol === protocol.id }"
                  @click="activeProtocol = protocol.id"
                  @keydown="handleProtocolKey($event, protocol.id)"
                >{{ protocol.name }}</button></div>
              <span class="console-language">Python</span>
              <button
                class="code-copy"
                type="button"
                @click="copyExample"
              >
                <Icon
                  :name="copied ? 'check' : 'copy'"
                  size="sm"
                />{{ copied ? copy.copied : copy.copy }}
              </button>
              <span
                class="landing-visually-hidden"
                role="status"
              >{{ copied ? copy.copied : '' }}</span>
            </div>
            <!-- All three examples are laid out in the same place, so the panel keeps one height. -->
            <div class="console-panes">
              <div
                v-for="protocol in protocols"
                :id="`protocol-panel-${protocol.id}`"
                :key="protocol.id"
                class="console-pane"
                :class="{ active: activeProtocol === protocol.id }"
                :role="activeProtocol === protocol.id ? 'tabpanel' : undefined"
                :aria-labelledby="`protocol-tab-${protocol.id}`"
                :tabindex="activeProtocol === protocol.id ? 0 : undefined"
              >
                <pre><code><span v-for="(token, index) in exampleTokens[protocol.id]" :key="index" :class="token.kind">{{ token.text }}</span></code></pre>
              </div>
            </div>
            <div
              v-if="copyError"
              class="copy-error"
              role="status"
            >{{ copyError }}</div>
          </div>
          <div
            class="screen-foot"
            data-reveal-part
          >
            <router-link
              to="/keys"
              class="landing-text-link"
            >{{ copy.createKey }}<Icon name="arrowRight" size="sm" /></router-link>
            <p class="screen-note">{{ copy.codeNote }}</p>
          </div>
        </div>
      </section>
      <!-- 4. Whether the visitor's tools can connect. -->
      <section
        id="tools"
        class="landing-screen"
        data-screen="tools"
        data-earth="beside"
        aria-labelledby="landing-tools-title"
        tabindex="-1"
      >
        <div class="landing-container screen-column screen-words">
          <div
            class="screen-head"
            data-reveal-part
          >
            <p class="section-kicker"><span>03</span>{{ copy.screenNames.tools }}</p>
            <h2 id="landing-tools-title">{{ copy.toolsTitle }}</h2>
            <p class="section-description">{{ copy.toolsDescription }}</p>
          </div>
          <ol
            class="tool-steps"
            data-reveal-part
          >
            <li
              v-for="(step, index) in toolSteps"
              :key="step"
            ><span class="step-number">0{{ index + 1 }}</span><span>{{ step }}</span></li>
          </ol>
          <div
            class="screen-foot"
            data-reveal-part
          >
            <p class="tool-names"><span
                v-for="tool in tools"
                :key="tool"
              >{{ tool }}</span><em>{{ copy.toolNamesNote }}</em></p>
            <p class="screen-note">{{ copy.toolGuide }}</p>
          </div>
        </div>
      </section>
      <!-- 5. How a call is priced. -->
      <section
        id="pricing"
        class="landing-screen"
        data-screen="pricing"
        data-earth="below"
        aria-labelledby="landing-pricing-title"
        tabindex="-1"
      >
        <div class="landing-container screen-center screen-words">
          <div
            class="screen-head"
            data-reveal-part
          >
            <p class="section-kicker"><span>04</span>{{ copy.screenNames.pricing }}</p>
            <h2 id="landing-pricing-title">{{ copy.pricingTitle }}</h2>
          </div>
          <div
            class="formula"
            data-reveal-part
          >
            <!-- A sign stays with the term after it where the line has to be folded. -->
            <p class="formula-line"><span class="formula-part formula-cost">{{ copy.formulaCost }}</span><span
                v-for="(term, index) in copy.formulaTerms"
                :key="term.name"
                class="formula-part"
              ><span class="formula-sign">{{ index ? '×' : '=' }}</span><span class="formula-term">{{ term.name }}</span></span></p>
            <dl class="formula-notes">
              <div
                v-for="term in copy.formulaTerms"
                :key="term.name"
                class="formula-note"
              >
                <dt>{{ term.name }}</dt>
                <dd>{{ term.note }}</dd>
              </div>
            </dl>
          </div>
          <div
            class="screen-foot"
            data-reveal-part
          >
            <ul
              v-if="payments.length"
              class="pay-list"
            >
              <li
                v-for="payment in payments"
                :key="payment"
              >{{ payment }}</li>
            </ul>
            <p class="screen-note">{{ copy.pricingNote }}</p>
            <router-link
              v-if="catalog"
              :to="catalog.to"
              class="landing-text-link"
            >{{ catalog.label ?? copy.viewPricing }}<Icon name="arrowRight" size="sm" /></router-link>
          </div>
        </div>
      </section>
      <!-- 6. Whether spending can be held, and where it went. -->
      <section
        id="control"
        class="landing-screen"
        data-screen="control"
        data-earth="beside"
        aria-labelledby="landing-control-title"
        tabindex="-1"
      >
        <div class="landing-container screen-column screen-words">
          <div
            class="screen-head"
            data-reveal-part
          >
            <p class="section-kicker"><span>05</span>{{ copy.screenNames.control }}</p>
            <h2 id="landing-control-title">{{ copy.controlTitle }}</h2>
            <p class="section-description">{{ copy.controlDescription }}</p>
          </div>
          <!-- An illustration of what a key's limits and one usage record look like; labelled as a sample. -->
          <div
            class="sample"
            data-reveal-part
          >
            <p class="sample-note">{{ copy.sampleNote }}</p>
            <div
              class="sample-meter"
              aria-hidden="true"
            ><span></span></div>
            <p class="sample-figures"><span>{{ copy.sampleUsed }}</span><span>{{ copy.sampleCap }}</span></p>
            <ul class="sample-limits">
              <li
                v-for="limit in copy.sampleLimits"
                :key="limit"
              >{{ limit }}</li>
            </ul>
            <dl class="sample-record">
              <div
                v-for="field in copy.sampleRecord"
                :key="field.label"
              >
                <dt>{{ field.label }}</dt>
                <dd>{{ field.value }}</dd>
              </div>
            </dl>
          </div>
          <div
            class="screen-foot"
            data-reveal-part
          >
            <router-link
              to="/key-usage"
              class="landing-text-link"
            >{{ copy.checkUsage }}<Icon name="arrowRight" size="sm" /></router-link>
            <p class="screen-note">{{ copy.usageLine }}</p>
          </div>
        </div>
      </section>
      <!-- 7. What is still unclear. -->
      <section
        id="faq"
        class="landing-screen"
        data-screen="faq"
        data-earth="beside"
        aria-labelledby="landing-faq-title"
        tabindex="-1"
      >
        <div class="landing-container screen-column screen-words">
          <div
            class="screen-head"
            data-reveal-part
          >
            <p class="section-kicker"><span>06</span>{{ copy.screenNames.faq }}</p>
            <h2 id="landing-faq-title">{{ copy.faqTitle }}</h2>
          </div>
          <!-- One answer at a time, all laid out in the same place: the screen keeps its height. -->
          <div
            class="faq"
            data-reveal-part
          >
            <div
              class="faq-questions"
              role="tablist"
              aria-orientation="vertical"
              aria-labelledby="landing-faq-title"
            ><button
                v-for="item in faqItems"
                :id="`faq-tab-${item.id}`"
                :key="item.id"
                type="button"
                role="tab"
                :aria-selected="activeFaq === item.id"
                :aria-controls="`faq-panel-${item.id}`"
                :tabindex="activeFaq === item.id ? 0 : -1"
                :class="{ active: activeFaq === item.id }"
                @click="activeFaq = item.id"
                @keydown="handleFaqKey($event, item.id)"
              ><span>{{ item.question }}</span></button></div>
            <div class="faq-answers">
              <p
                v-for="item in faqItems"
                :id="`faq-panel-${item.id}`"
                :key="item.id"
                class="faq-answer"
                :class="{ active: activeFaq === item.id }"
                :role="activeFaq === item.id ? 'tabpanel' : undefined"
                :aria-labelledby="`faq-tab-${item.id}`"
                :tabindex="activeFaq === item.id ? 0 : undefined"
              ><template
                  v-for="sentence in sentencesOf(item.answer)"
                  :key="sentence"
                ><span>{{ sentence }}</span>{{ isEnglish ? ' ' : '' }}</template></p>
            </div>
          </div>
        </div>
      </section>
      <!-- 8. What to do next. -->
      <section
        class="landing-screen landing-closing"
        data-screen="start"
        data-earth="below"
        aria-labelledby="landing-closing-title"
      >
        <div
          class="landing-container closing-content screen-words"
          data-reveal-part
        ><LandingBrandMark
            :source="siteLogo"
            alt=""
            width="56"
            height="68"
          />
          <h2 id="landing-closing-title">{{ copy.closingLineOne }}<br />{{ copy.closingLineTwo }}</h2>
          <ol class="closing-steps">
            <li
              v-for="step in copy.closingSteps"
              :key="step"
            >{{ step }}</li>
          </ol>
          <router-link
            :to="entryPath"
            class="landing-button landing-button-white"
          >{{ isAuthenticated ? copy.console : copy.start }}
            <Icon
              name="arrowRight"
              size="sm"
            />
          </router-link>
        </div>
      </section>
    </main>
    <!-- Where the visitor is among the screens, and a way to any of them. -->
    <nav
      class="screen-dots"
      :aria-label="copy.screenNav"
    >
      <button
        v-for="(screen, index) in screens"
        :key="screen"
        type="button"
        :class="{ active: currentScreen === index }"
        :aria-current="currentScreen === index ? 'true' : undefined"
        @click="goToScreen(index)"
      ><span class="screen-dot-name">{{ copy.screenNames[screen] }}</span></button>
      <span
        class="screen-count"
        aria-hidden="true"
      >{{ String(currentScreen + 1).padStart(2, '0') }} / {{ String(screens.length).padStart(2, '0') }}</span>
    </nav>
    <footer class="landing-footer">
      <div class="landing-container">
        <div class="footer-top"><router-link
            to="/home"
            class="landing-brand"
            :aria-label="siteName"
            @click="toTop"
          ><LandingBrandMark
              :source="siteLogo"
              alt=""
              width="30"
              height="35"
            /><GabuguWordmark :name="siteName" theme="dark" /></router-link>
          <div><router-link
              v-if="showModelPlazaEntry"
              to="/model-plaza"
            >{{ copy.screenNames.models }}</router-link><router-link
              to="/arena"
            >{{ copy.arenaRankings }}</router-link><a
              v-if="externalDocsUrl"
              :href="externalDocsUrl"
              target="_blank"
              rel="noopener noreferrer"
            >{{ copy.docs }}</a><router-link
              v-for="agreement in agreements"
              :key="agreement.id"
              :to="`/legal/${agreement.id}`"
            >{{ agreement.title }}</router-link><a
              href="https://www.gnu.org/licenses/lgpl-3.0.html"
              target="_blank"
              rel="noopener noreferrer"
            >LGPL-3.0</a></div>
        </div>
        <p class="footer-disclaimer">{{ disclaimer }}<template v-if="contactInfo"> {{ copy.contact }}{{ isEnglish ? ': ' : '：' }}{{ contactInfo }}</template></p>
        <div class="footer-bottom"><span>© {{ currentYear }} {{ siteName }}</span><span class="footer-credits"><a href="https://visibleearth.nasa.gov/collection/1484/blue-marble" target="_blank" rel="noopener noreferrer">{{ copy.earthCredit }}</a><span>{{ copy.cloudCredit }}<a href="https://www.solarsystemscope.com/textures/" target="_blank" rel="noopener noreferrer">Solar System Scope</a>{{ copy.cloudLicence }}<a href="https://creativecommons.org/licenses/by/4.0/" target="_blank" rel="noopener noreferrer">CC BY 4.0</a>{{ copy.cloudChanged }}</span></span></div>
      </div>
    </footer>
  </div>
  <BillingCampaignBanner surface="home" :class="{ 'home-floating-campaign': !hasHomeContent && !compactHomeEnabled }" />
  <WeekendRateBanner surface="home" :class="{ 'home-floating-campaign': !hasHomeContent && !compactHomeEnabled }" />
</template>
<script setup lang="ts">
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import { ref, shallowRef, computed, nextTick, onMounted, onBeforeUnmount, watch } from 'vue'
import DOMPurify from 'dompurify'
import { useI18n } from 'vue-i18n'
import type { RouteLocationRaw } from 'vue-router'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import CompactHome from '@/components/landing/CompactHome.vue'
import BillingCampaignBanner from '@/components/common/BillingCampaignBanner.vue'
import WeekendRateBanner from '@/components/common/WeekendRateBanner.vue'
import EarthStage from '@/components/landing/EarthStage.vue'
import HeroPoster from '@/components/landing/HeroPoster.vue'
import LandingBrandMark from '@/components/landing/LandingBrandMark.vue'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import Icon from '@/components/landing/LandingIcon.vue'
import ProviderIcon from '@/components/user/monitor/ProviderIcon.vue'
import { sanitizeUrl } from '@/utils/url'
import { useLandingCopy } from '@/components/landing/useLandingCopy'
import { useLandingMotion, type LandingOpening } from '@/components/landing/useLandingMotion'
import { useLandingPager } from '@/components/landing/useLandingPager'
import { choosePagingMode, type PagingMode } from '@/components/landing/paging'
import { createEarthJourney, createEarthSurface, earthBeat, type EarthSurface } from '@/components/landing/earth/context'
import { journeyAt, type JourneyStop } from '@/components/landing/journey'

const { locale } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
/** The site's public settings. The backend writes them into the page, so they are there from the first paint. */
const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || appStore.siteName || 'Gabugu')
const siteLogo = computed(() => sanitizeUrl(settings.value?.site_logo || appStore.siteLogo ||
  '', {
    allowRelative: true,
    allowDataUrl: true
  }))
const externalDocsUrl = computed(() => sanitizeUrl(settings.value?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => settings.value?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => settings.value?.compact_home_enabled === true)
const isHomeContentUrl = computed(() => /^https?:\/\//.test(homeContent.value.trim()))
const sanitizedHomeContent = computed(() => {
  if (!hasHomeContent.value || isHomeContentUrl.value) return ''
  return DOMPurify.sanitize(homeContent.value)
})
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const entryPath = computed(() => isAuthenticated.value ? dashboardPath.value : '/login')
const currentYear = new Date().getFullYear()
const isEnglish = computed(() => !String(locale?.value || 'zh').toLowerCase().startsWith('zh'))
const copy = useLandingCopy(isEnglish)
const named = (text: string) => text.replace('{site}', siteName.value)
const connectDescription = computed(() => named(copy.value.connectDescription))
const disclaimer = computed(() => named(copy.value.disclaimer))
const contactInfo = computed(() => (settings.value?.contact_info || '').trim())
/** Agreements the site has published; one that has no text yet gets no link. */
const agreements = computed(() => (settings.value?.login_agreement_documents ?? [])
  .filter(agreement => agreement.content_md?.trim() && agreement.title?.trim()))

// Each sentence is kept whole where it fits, so a line never ends one or two characters into the next.
const sentencesOf = (text: string) => text.match(/[^。！？.!?]+[。！？.!?]*/g)?.map(sentence => sentence.trim()) ?? [text]
const heroSentences = computed(() => sentencesOf(copy.value.heroDescription))
/** The headline in the pieces that rise one after another: characters in Chinese, words in English. */
const heroLines = computed(() => [copy.value.heroLineOne, copy.value.heroLineTwo]
  .map(line => isEnglish.value ? line.match(/\S+\s*/g) ?? [line] : Array.from(line)))
const heroHeadline = computed(() => `${copy.value.heroLineOne}${isEnglish.value ? ' ' : ''}${copy.value.heroLineTwo}`)

/** The screens of the page, in order. All but the first and the last are sections a link can lead to. */
const screens = ['hero', 'models', 'connect', 'tools', 'pricing', 'control', 'faq', 'start'] as const
const sections = ['models', 'connect', 'tools', 'pricing', 'control', 'faq'] as const
type Section = typeof sections[number]
const headerSections: Section[] = ['models', 'connect', 'pricing', 'faq']

/**
 * The model catalog and how a visitor gets to it. A site can keep the catalog for signed-in users; a
 * visitor is then sent to sign up, or to sign in where the site takes no new accounts. Null: no catalog.
 */
const plazaEnabled = computed(() => resolveFeatureFlag(settings.value, FeatureFlags.modelPlaza))
const showModelPlazaEntry = computed(() => plazaEnabled.value && (isAuthenticated.value || settings.value?.model_plaza_require_auth !== true))
const catalog = computed<{ to: RouteLocationRaw, label?: string } | null>(() => {
  if (!plazaEnabled.value) return null
  if (showModelPlazaEntry.value) return { to: '/model-plaza' }
  return settings.value?.registration_enabled === true && settings.value.backend_mode_enabled !== true
    ? { to: '/register', label: copy.value.signUpForCatalog }
    : { to: { path: '/login', query: { redirect: '/model-plaza' } }, label: copy.value.signInForCatalog }
})

const protocols = [{
  id: 'openai',
  name: 'OpenAI'
}, {
  id: 'anthropic',
  name: 'Anthropic'
}, {
  id: 'gemini',
  name: 'Gemini'
}] as const
type Protocol = typeof protocols[number]['id']
const protocolLabel = (id: Protocol) => copy.value.protocolOf.replace('{name}', protocols.find(protocol => protocol.id === id)?.name ?? '')
/**
 * The model families the site offers, each with the protocol its groups are called by. Grok and DeepSeek
 * groups are served by the OpenAI-compatible endpoints (backend: domain/platforms.go).
 */
const modelProviders: { model: string, company: string, icon: string, protocol: Protocol }[] = [
  { model: 'GPT', company: 'OpenAI', icon: 'openai', protocol: 'openai' },
  { model: 'Claude', company: 'Anthropic', icon: 'anthropic', protocol: 'anthropic' },
  { model: 'Gemini', company: 'Google', icon: 'gemini', protocol: 'gemini' },
  { model: 'Grok', company: 'xAI', icon: 'grok', protocol: 'openai' },
  { model: 'DeepSeek', company: '', icon: 'deepseek', protocol: 'openai' }
]
const activeProtocol = ref<Protocol>('openai')
/** Where requests go: the address the site gives for its API, or else the address of this page. */
const apiRoot = computed(() => (sanitizeUrl(settings.value?.api_base_url || '') || window.location.origin)
  .replace(/\/v1\/?$/, '').replace(/\/+$/, ''))

// Each official Python SDK, pointed at this site. The two values a visitor changes are marked: the address
// and the key. Checked against the SDKs: OpenAI's takes the address with /v1, Anthropic's without, and
// Google's needs the API version named, because its default is not the one the gateway serves.
type CodeToken = { text: string, kind?: 'code-address' | 'code-flag' | 'code-placeholder' }
const API_KEY: CodeToken = { text: 'YOUR_API_KEY', kind: 'code-placeholder' }
const word = (text: string): CodeToken => ({ text, kind: 'code-flag' })
const exampleTokens = computed<Record<Protocol, CodeToken[]>>(() => {
  const address = (path = ''): CodeToken => ({ text: `${apiRoot.value}${path}`, kind: 'code-address' })
  return {
    openai: [word('from'), { text: ' openai ' }, word('import'), { text: ' OpenAI\n\nclient = OpenAI(\n    base_url="' },
      address('/v1'), { text: '",\n    api_key="' }, API_KEY, { text: '",\n)' }],
    anthropic: [word('from'), { text: ' anthropic ' }, word('import'), { text: ' Anthropic\n\nclient = Anthropic(\n    base_url="' },
      address(), { text: '",\n    api_key="' }, API_KEY, { text: '",\n)' }],
    gemini: [word('from'), { text: ' google ' }, word('import'), { text: ' genai\n\nclient = genai.Client(\n    api_key="' },
      API_KEY, { text: '",\n    http_options={"base_url": "' }, address(), { text: '",\n                  "api_version": "v1beta"},\n)' }]
  }
})

function handleProtocolKey(event: KeyboardEvent, protocol: Protocol) {
  const index = protocols.findIndex(item => item.id === protocol)
  let nextIndex = index
  if (event.key === 'ArrowRight') nextIndex = (index + 1) % protocols.length
  else if (event.key === 'ArrowLeft') nextIndex = (index - 1 + protocols.length) % protocols.length
  else if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = protocols.length - 1
  else return
  event.preventDefault()
  activeProtocol.value = protocols[nextIndex].id
  document.getElementById(`protocol-tab-${activeProtocol.value}`)?.focus()
}
const copied = ref(false)
const copyError = ref('')
let copyTimer: ReturnType<typeof setTimeout> | undefined
async function copyExample() {
  copyError.value = ''
  try {
    await navigator.clipboard.writeText(exampleTokens.value[activeProtocol.value].map(token => token.text).join(''))
    copied.value = true
    clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    copyError.value = copy.value.copyFailed
  }
}

/** The tools a key's "Use Key" dialog has configurations for; which of them depends on the key's group. */
const tools = ['Claude Code', 'Codex', 'Gemini CLI', 'Grok CLI', 'OpenCode']
const toolSteps = computed(() => copy.value.toolSteps.map((step, index, steps) =>
  index === steps.length - 1 && settings.value?.hide_ccs_import_button !== true ? step + copy.value.toolImport : step))

/** The ways to pay that the site has switched on. */
const payments = computed(() => {
  const open = settings.value
  if (open?.payment_enabled !== true) return []
  return [
    ...(open.payment_balance_disabled === true ? [] : [copy.value.payBalance]),
    ...(open.subscription_enabled === false ? [] : [copy.value.paySubscription])
  ]
})

const faqIds = ['pricing', 'plans', 'keys', 'nocode'] as const
type FaqId = typeof faqIds[number]
/** The question that compares the two ways to pay is asked only where the site has both. */
const faqItems = computed(() => faqIds
  .filter(id => id !== 'plans' || payments.value.length === 2)
  .map(id => ({ id, ...copy.value.faq[id] })))
const activeFaq = ref<FaqId>('pricing')
watch(faqItems, items => {
  if (!items.some(item => item.id === activeFaq.value)) activeFaq.value = items[0].id
})
function handleFaqKey(event: KeyboardEvent, id: FaqId) {
  const ids = faqItems.value.map(item => item.id)
  const index = ids.indexOf(id)
  let nextIndex = index
  if (event.key === 'ArrowDown') nextIndex = (index + 1) % ids.length
  else if (event.key === 'ArrowUp') nextIndex = (index - 1 + ids.length) % ids.length
  else if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = ids.length - 1
  else return
  event.preventDefault()
  activeFaq.value = ids[nextIndex]
  document.getElementById(`faq-tab-${activeFaq.value}`)?.focus()
}

/**
 * How the first screen appears. A first visit sees the sunrise in full; a later visit in the same session,
 * or a reload, sees it briefly. A visitor who arrives at a section or comes back with the browser's Back
 * button finds everything in place.
 */
const SEEN_KEY = 'gabugu-landing-seen'
/** The visitor has been on from this entry of the history and is back on it; the router restores their place. */
function cameBack() {
  try {
    return (window.history.state as { forward?: string | null } | null)?.forward != null
  } catch {
    return false
  }
}
function chooseOpening(): LandingOpening {
  try {
    // The router keeps where a history entry led to and how far it was scrolled in the entry's state.
    const state = window.history.state as { scroll?: { top?: number } | null | false } | null
    const linked = sections.some(section => window.location.hash === `#${section}`)
    const returned = cameBack() || (state?.scroll ? (state.scroll.top ?? 0) > 0 : false)
    if (linked || returned) return 'none'
    return window.sessionStorage.getItem(SEEN_KEY) ? 'short' : 'full'
  } catch {
    return 'full'
  }
}

const landingRoot = ref<HTMLElement | null>(null)
const opening = chooseOpening()
const { motionReduced, introStartedAt, arrive } = useLandingMotion(landingRoot, activeProtocol, opening)

/**
 * The live Earth behind the page (EarthStage). A visitor who asked for less motion or for less data, and a
 * browser that cannot draw the scene, get the still picture instead. The page is laid out the same over
 * either, so nothing moves when one takes the other's place.
 */
const surface = shallowRef<EarthSurface | null>(null)
const stage = ref<InstanceType<typeof EarthStage> | null>(null)
const stageRun = ref(0)
const journey = createEarthJourney()
const landingShown = computed(() => !hasHomeContent.value && !compactHomeEnabled.value)
/** Set once the full version has given up on this machine. */
let lightOnly = false

const matches = (query: string) => typeof window.matchMedia === 'function' && window.matchMedia(query).matches
const REDUCED_MEDIA = '(prefers-reduced-motion: reduce)'
function liveWanted() {
  const saving = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection?.saveData === true
  return !matches(REDUCED_MEDIA) && !saving
}
function openStage() {
  if (surface.value || !landingShown.value || !liveWanted()) return
  stageRun.value++
  surface.value = createEarthSurface(lightOnly ? 'low' : undefined)
}
/** The scene gave up. The full version may be followed by the light one, once; after that the still picture stands. */
function stageFailed() {
  const failed = surface.value
  surface.value = null
  if (failed?.quality === 'high' && !lightOnly) {
    lightOnly = true
    openStage()
  }
  void nextTick(layoutChanged)
}
openStage()
watch(landingShown, shown => {
  if (shown) openStage()
  else surface.value = null
})
watch(motionReduced, reduced => {
  if (reduced) surface.value = null
  void nextTick(layoutChanged)
})

const heroGround = ref<HTMLElement | null>(null)
const screenProbe = ref<HTMLElement | null>(null)
/** Pixels from the top of the first screen to where the planet may begin. */
const horizonAnchor = ref(0)
const headerSolid = ref(false)
/** How the page moves from screen to screen; decided from what was measured (see paging.ts). */
const pagingMode = ref<PagingMode>('free')

/** The same condition as in landing.css: a screen's words stand above or below the planet, not beside it. */
const TALL_MEDIA = '(max-width: 1020px), (max-aspect-ratio: 21/20)'
/** Where the finger scrolls the page, the browser's own snapping turns the screens. */
const TOUCH_MEDIA = '(pointer: coarse)'
/** A screen's words and its ground are gone when the page is this far from it, in screens. */
const NEAR_REACH = 0.5
/** While the page turns, a screen's words move at this share of the scroll's speed: they drift, and fade on the way. */
const DRIFT = 0.42
/** A screen's entrance begins when its upper edge has come this far up the window, in screens. */
const ARRIVE_REACH = 0.55
/** Room kept between the last of a screen's words and the planet's edge under them, in pixels. */
const GROUND_GAP = 28

let screenElements: HTMLElement[] = []
let screenTops: number[] = []
/** Scroll positions the page rests at: the top of each screen, then the end of the page. */
let pageStops: number[] = []
/** Scroll positions at which the journey has reached each screen. */
let journeyStops: JourneyStop[] = []
let screenHeight = 0
let pageEnd = 0
const nearness: number[] = []
const shifts: number[] = []

const pager = useLandingPager({
  stops: () => pageStops,
  screen: () => screenHeight,
  mode: pagingMode,
  active: () => landingRoot.value !== null,
  reducedMotion: () => motionReduced.value,
  moved: () => trackScroll()
})
/** The screen the page is on; at the page's end that is still the last one. */
const currentScreen = computed(() => Math.min(screens.length - 1, pager.current.value))

/** Where an element stands on the page, read from the layout, so that an entrance in progress does not count. */
function pagePlace(element: HTMLElement) {
  let top = 0
  let left = 0
  for (let node: HTMLElement | null = element; node; node = node.offsetParent as HTMLElement | null) {
    top += node.offsetTop
    left += node.offsetLeft
  }
  return { top, left }
}

function setPagingMode(mode: PagingMode) {
  pagingMode.value = mode
  // Where the pager moves the page evenly, the picture keeps close to it; a wheel's steps are smoothed more.
  journey.lag = mode === 'paged' ? 70 : 170
  // Snapping belongs to the element that scrolls, which is the document itself.
  document.documentElement.style.scrollSnapType = mode === 'snap' ? 'y mandatory' : ''
}

/** Reads from the page where its screens lie, what the picture has to keep clear of, and how the page turns. */
function measure() {
  const root = landingRoot.value
  const ground = heroGround.value
  if (!root || !ground) return
  horizonAnchor.value = pagePlace(ground).top
  // The picture is as tall as the stage, which on a phone can differ from the window by a toolbar.
  const stageElement = stage.value?.$el as HTMLElement | undefined
  const view = stageElement?.clientHeight || window.innerHeight
  const header = root.querySelector<HTMLElement>('.landing-header')?.offsetHeight ?? 0
  const tall = matches(TALL_MEDIA)
  screenHeight = screenProbe.value?.offsetHeight || window.innerHeight
  pageEnd = Math.max(0, document.documentElement.scrollHeight - window.innerHeight)

  screenElements = Array.from(root.querySelectorAll<HTMLElement>('.landing-screen'))
  screenTops = screenElements.map(screen => pagePlace(screen).top)
  const tops = screenTops.map(top => Math.min(pageEnd, top))
  pageStops = pageEnd > (tops[tops.length - 1] ?? 0) + 1 ? [...tops, pageEnd] : tops
  journeyStops = [
    ...tops.map((at, index) => ({ at, progress: earthBeat(index, tops.length) })),
    { at: pageEnd, progress: 1 }
  ]

  // What each screen tells the picture (see EarthJourney.marks).
  let column = 0
  const marks = screenElements.map((screen, index) => {
    const words = screen.querySelector<HTMLElement>('.screen-words')
    if (!words || !view) return NaN
    const place = pagePlace(words)
    if (screen.dataset.earth === 'below') {
      const ending = index === 0 ? horizonAnchor.value : place.top + words.offsetHeight + GROUND_GAP
      return (ending - screenTops[index]) / view
    }
    if (tall) return (place.top - screenTops[index]) / view
    const edge = (place.left + words.offsetWidth) / (root.clientWidth || 1)
    column = Math.max(column, edge)
    return edge
  })
  journey.tall = tall
  journey.clear = view ? header / view : 0
  journey.marks = marks
  // Each screen's ground is dark as far as its words' column reaches (see landing.css).
  if (column) root.style.setProperty('--landing-words', `${(column * 100).toFixed(1)}%`)

  setPagingMode(choosePagingMode({
    fits: screenElements.every(screen => screen.offsetHeight <= screenHeight + 1),
    reducedMotion: matches(REDUCED_MEDIA),
    touch: matches(TOUCH_MEDIA)
  }))
}

/**
 * Where the page turns screen by screen, a screen's words fade as the page leaves it and come up as the
 * page reaches it, so that between two screens the picture has the page to itself for a moment. Where the
 * pager moves the page, they also drift: they move slower than the scroll (see landing.css).
 */
function fadeScreens(scroll: number) {
  const linked = pagingMode.value !== 'free' && !motionReduced.value && screenHeight > 0
  // Only the pager's own moves are drawn in step with the scroll; a finger's are not, and a drift that
  // trails the finger by a frame would shake.
  const drifting = linked && pagingMode.value === 'paged'
  const reach = screenHeight * NEAR_REACH
  screenElements.forEach((screen, index) => {
    const delta = scroll - screenTops[index]
    if (delta > -ARRIVE_REACH * screenHeight) arrive(index)
    let near = 1
    let shift = 0
    if (linked) {
      // A screen fades over the half screen by which the page has left it, the last one as it gives
      // way to the footer. The distance does not depend on how tall the footer is, and the scroll
      // back up finds the same opacity as the scroll down left.
      const away = Math.abs(delta)
      near = Math.round(Math.max(0, 1 - away / reach) * 50) / 50
      if (drifting && near > 0) shift = Math.round(delta * (1 - DRIFT) * 2) / 2
    }
    if (near !== nearness[index]) {
      nearness[index] = near
      if (near === 1) screen.style.removeProperty('--screen-near')
      else screen.style.setProperty('--screen-near', String(near))
    }
    if (shift !== shifts[index]) {
      shifts[index] = shift
      if (shift) screen.style.setProperty('--screen-shift', `${shift}px`)
      else screen.style.removeProperty('--screen-shift')
    }
  })
}

function trackScroll() {
  const scroll = window.scrollY
  // Where the page scrolls freely, words pass under the header and it needs a ground of its own.
  headerSolid.value = pagingMode.value === 'free' && scroll > 24
  fadeScreens(scroll)
  if (!surface.value) return
  journey.progress = journeyAt(scroll, journeyStops)
  journey.touched = performance.now()
  stage.value?.wake()
}

/** A mouse turns the camera a little. */
function trackPointer(event: PointerEvent) {
  if (event.pointerType !== 'mouse' || !surface.value) return
  journey.pointerX = event.clientX / window.innerWidth * 2 - 1
  journey.pointerY = event.clientY / window.innerHeight * 2 - 1
  journey.touched = performance.now()
  stage.value?.wake()
}

let measureFrame = 0
function layoutChanged() {
  cancelAnimationFrame(measureFrame)
  const was = pagingMode.value
  const on = pager.current.value
  measure()
  if (pagingMode.value === 'paged') {
    // A screen is as tall as the window: when that changes, the page is set on the screen it was on —
    // or on the one a turn was heading for, which the new stops no longer lead to.
    if (pager.turning()) pager.retarget()
    else if (was === 'paged') pager.moveTo(on, true)
    else pager.settle()
  }
  pager.sync()
  trackScroll()
}
/** The page's height changes as fonts arrive and as its content is laid out; measured once per frame at most. */
function layoutChanging() {
  cancelAnimationFrame(measureFrame)
  measureFrame = requestAnimationFrame(layoutChanged)
}
let rootObserver: ResizeObserver | undefined
// Watched rather than read once on mount: the Landing can replace another home layout after settings load.
watch(landingRoot, root => {
  rootObserver?.disconnect()
  rootObserver = undefined
  if (!root) return setPagingMode('free')
  layoutChanged()
  if (typeof ResizeObserver !== 'function') return
  rootObserver = new ResizeObserver(layoutChanging)
  rootObserver.observe(root)
}, { flush: 'post' })
// What the page shows depends on the language and on the site's settings; both can change its measures.
watch([isEnglish, settings, isAuthenticated], () => { void nextTick(layoutChanging) })

function showScreen(index: number, instant = false) {
  pager.moveTo(index, instant)
}
/**
 * The dots at the right go to a screen and take the reader with them: scrolling alone moves nothing the
 * keyboard or a screen reader follows, so the screen itself receives the focus (and is announced, with its
 * heading). `preventScroll` because the pager has already put the page where the screen is.
 */
function goToScreen(index: number) {
  showScreen(index)
  const name = screens[index]
  const screen = screenElements[index] ?? (sections.includes(name as Section) ? document.getElementById(name) : null)
  if (screen) screen.focus({ preventScroll: true })
  window.history.replaceState(window.history.state, '', sections.includes(name as Section) ? `#${name}` : window.location.pathname + window.location.search)
}
/**
 * In-page links scroll here instead of navigating. To the router a new hash is a new location, and its
 * scrollBehavior answers every new location with the top of the page. `takeFocus` is false for a second
 * pass over the same link after the fonts have arrived: by then the visitor may be somewhere else.
 */
function goTo(section: Section, instant = false, takeFocus = true) {
  const target = document.getElementById(section)
  if (!target) return
  showScreen(screens.indexOf(section), instant)
  if (takeFocus) target.focus({ preventScroll: true })
  window.history.replaceState(window.history.state, '', `#${section}`)
}
/** The router does not see the hashes set above, so the brand link returns to the top itself. */
function toTop() {
  showScreen(0)
  if (window.location.hash) {
    window.history.replaceState(window.history.state, '', window.location.pathname + window.location.search)
  }
}
/** A model's row leads to the integration screen, with that model's protocol chosen. */
function showExample(protocol: Protocol) {
  activeProtocol.value = protocol
  showScreen(screens.indexOf('connect'))
  document.getElementById(`protocol-tab-${protocol}`)?.focus({ preventScroll: true })
}
/**
 * Focus that moves to another screen (Tab, a screen reader's cursor) takes the page there. The browser
 * scrolls what has the focus into view by itself, only not to a screen's top; so the page is set a frame
 * later, from wherever the browser has put it.
 */
let focusFrame = 0
function followFocus(event: FocusEvent) {
  if (pagingMode.value !== 'paged') return
  const target = event.target instanceof Element ? event.target : null
  const screen = target?.closest<HTMLElement>('.landing-screen')
  const index = screen ? screenElements.indexOf(screen) : target?.closest('.landing-footer') ? pageStops.length - 1 : -1
  if (index < 0) return
  cancelAnimationFrame(focusFrame)
  focusFrame = requestAnimationFrame(() => {
    if (pagingMode.value !== 'paged' || pager.heading() === index) return
    if (pager.heading() >= 0 || Math.abs(window.scrollY - pageStops[index]) > 1) pager.moveTo(index)
  })
}

let touchQuery: MediaQueryList | undefined
onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()
  window.addEventListener('scroll', trackScroll, { passive: true })
  window.addEventListener('resize', layoutChanged)
  window.addEventListener('pointermove', trackPointer, { passive: true })
  touchQuery = typeof window.matchMedia === 'function' ? window.matchMedia(TOUCH_MEDIA) : undefined
  touchQuery?.addEventListener?.('change', layoutChanged)
  if (landingRoot.value) {
    try {
      window.sessionStorage.setItem(SEEN_KEY, '1')
    } catch {
      // Storage can be unavailable; the opening then plays in full each time.
    }
  }
  // A visitor who comes back with the Back button is put where they were, not at the section in the address.
  const linked = cameBack() ? undefined : sections.find(section => window.location.hash === `#${section}`)
  if (linked) {
    requestAnimationFrame(() => {
      goTo(linked, true)
      // Web fonts that arrive later can change where the section lies, so align once more — without
      // taking the focus from wherever the visitor has put it by then.
      void document.fonts?.ready.then(() => {
        if (landingRoot.value && !pager.turning()) goTo(linked, true, false)
      })
    })
  }
  // The measures are final only once the fonts are in.
  void document.fonts?.ready.then(() => {
    if (landingRoot.value) layoutChanged()
  })
})
onBeforeUnmount(() => {
  clearTimeout(copyTimer)
  cancelAnimationFrame(measureFrame)
  cancelAnimationFrame(focusFrame)
  rootObserver?.disconnect()
  touchQuery?.removeEventListener?.('change', layoutChanged)
  window.removeEventListener('scroll', trackScroll)
  window.removeEventListener('resize', layoutChanged)
  window.removeEventListener('pointermove', trackPointer)
  document.documentElement.style.scrollSnapType = ''
})
</script>
<style scoped src="@/components/landing/landing.css"></style>
