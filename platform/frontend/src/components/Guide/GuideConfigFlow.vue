<template>
  <div class="config-diagram-root">
    <figure class="config-figure" :aria-label="t('guide.configDiagram.caption')">
      <div class="config-layout">
        <div class="config-source">
          <div class="config-paper">
            <div class="config-paper-heading"><img src="/logo.svg" alt="" width="24" height="24" /><strong>{{ t('guide.configDiagram.connection') }}</strong><GabuguWordmark class="config-gabugu-wordmark" /></div>
            <dl class="config-fields">
              <div v-for="field in fields" :key="field.id" class="config-field">
                <dt><Icon :name="field.icon" size="sm" /><span>{{ t(field.title) }}</span><code>{{ field.example }}</code></dt>
                <dd>{{ t(`guide.configDiagram.${field.id}Help`) }}</dd>
              </div>
            </dl>
          </div>
          <p class="config-stage-note">{{ t('guide.configDiagram.source') }}</p>
        </div>
        <div class="config-connector config-connector-import">
          <span>{{ t('guide.configDiagram.import') }}</span>
          <svg viewBox="0 0 80 24" fill="none" aria-hidden="true"><path class="config-wire" d="M1 12H76m-5-5 5 5-5 5" /></svg>
          <i class="config-mobile-arrow" aria-hidden="true"></i>
        </div>
        <div class="config-manager">
          <div class="config-manager-panel">
            <div class="config-manager-header"><div><span class="config-manager-kicker">{{ t('guide.configDiagram.managerTitle') }}</span><h4>CC Switch</h4></div><span class="config-manager-state"><i></i> {{ t('guide.configDiagram.active') }}</span></div>
            <div class="config-app-picker">
              <span class="config-app-choice config-app-choice-active"><PlatformIcon platform="openai" size="sm" />Codex</span>
              <span class="config-app-choice"><PlatformIcon platform="anthropic" size="sm" />Claude Code</span>
            </div>
            <span class="config-profile"><img src="/logo.svg" alt="" width="18" height="18" /><span><GabuguWordmark class="config-gabugu-wordmark" /><small>{{ t('guide.configDiagram.profile') }}</small></span><Icon name="check" size="xs" /></span>
          </div>
        </div>
        <div class="config-connector config-connector-apply">
          <span>{{ t('guide.configDiagram.apply') }}</span>
          <svg viewBox="0 0 80 24" fill="none" aria-hidden="true"><path class="config-wire" d="M1 12H76m-5-5 5 5-5 5" /></svg>
          <i class="config-mobile-arrow" aria-hidden="true"></i>
        </div>
        <div class="config-clients">
          <p class="config-clients-label">{{ t('guide.configDiagram.destination') }}</p>
          <div class="config-client-list">
            <div class="config-client"><span class="config-app-icon"><PlatformIcon platform="openai" size="lg" /></span><div><strong>Codex</strong><span>{{ t('guide.configDiagram.desktop') }}</span></div></div>
            <div class="config-client"><span class="config-app-icon config-app-claude"><PlatformIcon platform="anthropic" size="lg" /></span><div><strong>Claude Code</strong><span>{{ t('guide.configDiagram.codingTool') }}</span></div></div>
          </div>
          <p class="config-stage-note">{{ t('guide.configDiagram.clientHelp') }}</p>
        </div>
      </div>
      <figcaption><Icon name="infoCircle" size="sm" /><span>{{ t('guide.configDiagram.caption') }}</span></figcaption>
    </figure>
  </div>
</template>

<script setup lang="ts">
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
const { t } = useI18n()
const fields = [
  { id: 'key', title: 'guide.basics.keyLabel', icon: 'key', example: 'sk-••••' },
  { id: 'address', title: 'guide.basics.addressLabel', icon: 'link', example: 'https://…' },
  { id: 'model', title: 'guide.basics.modelLabel', icon: 'grid', example: 'model' }
] as const
</script>

<style scoped>
.config-gabugu-wordmark { font-size: 12px; }
.config-diagram-root { container-type: inline-size; }
.config-figure { --config-blue: var(--guide-accent-text); margin: 0; padding: 36px 32px 0; border: 1px solid var(--guide-border); border-radius: 20px; background: var(--guide-surface); color: var(--guide-text); }
.config-layout { display: grid; grid-template-columns: minmax(225px,1.25fr) 56px minmax(185px,.95fr) 56px minmax(170px,.9fr); gap: 14px; align-items: center; }
.config-source,.config-manager,.config-clients { min-width: 0; }
/* The fields form one configuration document, with a shared visual hierarchy. */
.config-paper { overflow: hidden; border: 1px solid var(--guide-border-strong); border-radius: 14px; background: var(--guide-canvas); box-shadow: 0 8px 22px -18px rgb(0 0 0 / .35); }
.config-paper-heading { display: flex; align-items: center; gap: 9px; padding: 16px 17px; border-bottom: 1px solid var(--guide-border); }
.config-paper-heading img { width: 24px; height: 24px; flex-shrink: 0; }
.config-paper-heading strong { font-size: 15px; line-height: 24px; font-weight: 550; }
.config-paper-heading>.config-gabugu-wordmark { margin-left: auto; font-size: 11px; line-height: 20px; }
.config-fields { margin: 0; padding: 0 17px; }
.config-field { padding: 13px 0; }
.config-field+.config-field { border-top: 1px solid var(--guide-border); }
.config-field dt { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; font-size: 14px; line-height: 22px; font-weight: 500; }
.config-field dt svg { width: 16px; height: 16px; flex-shrink: 0; color: var(--guide-muted); }
.config-field code { margin-left: auto; color: var(--guide-faint); font-size: 11px; line-height: 22px; font-weight: 400; font-family: ui-monospace, 'SFMono-Regular', Consolas, monospace; }
.config-field dd { margin: 3px 0 0 24px; color: var(--guide-muted); font-size: 12px; line-height: 21px; }
.config-figure .config-stage-note { margin: 13px 0 0; color: var(--guide-muted); font-size: 12px; line-height: 22px; }
.config-source>.config-stage-note { padding-left: 2px; }
/* Show the tool selection and active profile, not an abstract switch glyph. */
.config-manager-panel { min-width: 0; padding: 18px 16px 16px; border: 1px solid color-mix(in srgb, var(--config-blue) 24%, var(--guide-border)); border-radius: 14px; background: linear-gradient(145deg, var(--guide-soft), var(--guide-canvas)); box-shadow: 0 10px 28px -24px color-mix(in srgb, var(--config-blue) 65%, transparent); }
.config-manager-header { display: flex; align-items: flex-end; justify-content: space-between; gap: 5px; }
.config-manager-kicker { display: block; margin-bottom: 3px; color: var(--guide-muted); font-size: 11px; line-height: 17px; }
.config-manager h4 { margin: 0; font-size: 19px; line-height: 27px; font-weight: 600; letter-spacing: normal; }
.config-manager-state { display: inline-flex; align-items: center; gap: 5px; padding: 3px 7px; margin-bottom: 3px; border: 1px solid color-mix(in srgb, var(--guide-success) 28%, var(--guide-border)); border-radius: 20px; color: var(--guide-success); font-size: 10px; line-height: 16px; white-space: nowrap; }
.config-manager-state i { width: 6px; height: 6px; border-radius: 50%; background: var(--guide-success); }
.config-app-picker { display: grid; grid-template-columns: minmax(0,1fr) minmax(0,1fr); gap: 3px; padding: 3px; margin-top: 15px; border: 1px solid var(--guide-border); border-radius: 9px; background: var(--guide-canvas); }
.config-app-choice { display: inline-flex; align-items: center; justify-content: center; gap: 6px; min-width: 0; padding: 7px 4px; border: 1px solid transparent; border-radius: 6px; color: var(--guide-muted); font-size: 11px; line-height: 17px; white-space: nowrap; }
.config-app-choice :deep(svg) { width: 15px; height: 15px; flex-shrink: 0; }
.config-app-choice-active { border-color: var(--guide-border); background: var(--guide-surface); color: var(--guide-text); box-shadow: 0 1px 2px rgb(16 24 40 / .07); }
.config-profile { display: flex; align-items: center; gap: 8px; margin-top: 11px; padding: 9px 9px; border: 1px solid color-mix(in srgb, var(--config-blue) 25%, var(--guide-border)); border-radius: 9px; color: var(--guide-text); background: var(--guide-surface); }
.config-profile img { width: 19px; height: 19px; flex-shrink: 0; }
.config-profile>span { display: grid; gap: 1px; min-width: 0; }
.config-profile strong { font-size: 12px; line-height: 17px; font-weight: 550; }
.config-profile small { color: var(--guide-muted); font-size: 10px; line-height: 15px; }
.config-profile svg { width: 14px; height: 14px; margin-left: auto; color: var(--config-blue); flex-shrink: 0; }
.config-figure .config-clients-label { margin: 0 0 22px; font-size: 13px; line-height: 22px; color: var(--guide-muted); }
.config-client-list { display: grid; gap: 28px; }
.config-client { display: flex; align-items: center; gap: 13px; min-width: 0; }
.config-app-icon { display: grid; place-items: center; width: 50px; height: 50px; border: 1px solid var(--guide-border-strong); border-radius: 14px; background: var(--guide-canvas); flex-shrink: 0; }
.config-app-icon :deep(svg) { width: 27px; height: 27px; }
.config-app-claude { color: #c58869; }
.config-client>div { display: grid; gap: 4px; min-width: 0; }
.config-client strong { font-size: 16px; line-height: 24px; font-weight: 550; }
.config-client>div>span { font-size: 12px; line-height: 20px; color: var(--guide-muted); }
.config-clients>.config-stage-note { max-width: 17em; }
.config-connector { display: grid; grid-template-rows: 39px 28px; justify-items: center; align-content: center; align-self: center; height: 112px; color: var(--config-blue); }
.config-connector>span { z-index: 1; align-self: end; margin-bottom: 7px; padding: 2px 7px; border: 1px solid var(--guide-border); border-radius: 99px; background: var(--guide-surface); color: var(--guide-text); font-size: 11px; line-height: 17px; white-space: nowrap; }
.config-connector>svg { width: 100%; height: 24px; overflow: visible; }
.config-wire { stroke: var(--config-blue); stroke-width: 1.15; stroke-linecap: round; stroke-linejoin: round; }
.config-mobile-arrow { display: none; }
.config-figure figcaption { display: flex; gap: 9px; align-items: center; margin-top: 28px; padding: 18px 0; border-top: 1px solid var(--guide-border); color: var(--guide-muted); font-size: 12px; line-height: 23px; }
.config-figure figcaption svg { flex-shrink: 0; width: 15px; height: 15px; color: var(--guide-muted); }
@container (max-width: 900px) {
  .config-figure { padding: 28px 24px 0; }
  .config-layout { grid-template-columns: minmax(185px,1.15fr) 50px minmax(155px,.95fr) 50px minmax(150px,.9fr); gap: 10px; }
  .config-paper-heading { padding: 14px 13px; }
  .config-paper-heading>span { display: none; }
  .config-fields { padding-inline: 13px; }
  .config-field code { display: none; }
  .config-manager h4 { font-size: 18px; white-space: nowrap; }
  .config-app-choice { font-size: 10px; gap: 4px; }
  .config-manager-state { font-size: 9px; padding-inline: 5px; }
  .config-app-picker { grid-template-columns: 1fr; gap: 1px; }
  .config-app-choice { justify-content: flex-start; padding-inline: 8px; }
  .config-client { gap: 10px; }
  .config-app-icon { width: 43px; height: 43px; border-radius: 12px; }
  .config-app-icon :deep(svg) { width: 25px; height: 25px; }
  .config-client strong { font-size: 15px; }
}
@container (max-width: 650px) {
  .config-figure { padding: 22px 20px 0; border-radius: 16px; }
  .config-layout { grid-template-columns: minmax(0,1fr); gap: 0; }
  .config-paper-heading>span { display: inline; }
  .config-paper-heading { padding: 14px 16px; }
  .config-fields { padding-inline: 16px; }
  .config-field code { display: block; }
  .config-field { padding: 11px 0; }
  .config-connector { display: flex; align-items: center; justify-content: center; gap: 10px; height: 54px; margin: 0; }
  .config-connector>svg { order: -1; width: 28px; height: 24px; transform: rotate(90deg); }
  .config-connector>span { position: static; margin: 0; padding: 2px 8px; }
  .config-mobile-arrow { display: none; }
  .config-manager { padding-block: 2px; }
  .config-manager-panel { padding: 16px; }
  .config-manager-kicker { font-size: 12px; }
  .config-manager h4 { font-size: 19px; }
  .config-manager-state { font-size: 10px; padding-inline: 8px; }
  .config-app-choice { font-size: 12px; padding-block: 8px; }
  .config-app-choice :deep(svg) { width: 16px; height: 16px; }
  .config-profile { margin-top: 11px; padding: 10px; }
  .config-figure .config-clients-label { margin-bottom: 14px; }
  .config-client-list { grid-template-columns: minmax(0,1fr) minmax(0,1fr); gap: 20px; }
  .config-client { flex-direction: column; align-items: flex-start; gap: 10px; }
  .config-app-icon { width: 46px; height: 46px; }
  .config-client strong { font-size: 15px; }
  .config-client>div { gap: 2px; }
  .config-clients>.config-stage-note { max-width: none; }
  .config-figure figcaption { align-items: flex-start; margin-top: 22px; line-height: 22px; }
  .config-figure figcaption svg { margin-top: 3px; }
}
</style>
