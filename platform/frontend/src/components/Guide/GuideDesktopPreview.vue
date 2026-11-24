<template>
  <figure class="desktop-preview" :aria-label="t('guide.desktop.preview.caption')">
    <div class="desktop-window">
      <div class="desktop-titlebar"><span class="window-controls" aria-hidden="true"><i></i><i></i><i></i></span><span>Codex</span><span class="desktop-demo">{{ t('guide.media.schematicLabel') }}</span></div>
      <div class="desktop-workspace">
        <div class="desktop-project" aria-hidden="true"><PlatformIcon platform="openai" size="lg" /><span class="project-label">{{ t('guide.desktop.preview.project') }}</span><span class="project-folder"><Icon name="document" size="sm" />gabugu-start</span><span class="project-new"><Icon name="plus" size="sm" />{{ t('guide.desktop.preview.newChat') }}</span></div>
        <div class="desktop-chat">
          <div class="desktop-chat-toolbar"><span class="desktop-mode"><PlatformIcon platform="openai" size="sm" /> Codex</span><span class="desktop-new-chat"><Icon name="plus" size="xs" />{{ t('guide.desktop.preview.newChat') }}</span></div>
          <h4>{{ t('guide.desktop.preview.heading') }}</h4>
          <p>{{ t('guide.desktop.preview.description') }}</p>
          <div class="desktop-context"><span><Icon name="document" size="xs" />gabugu-start</span><span>{{ t('guide.desktop.preview.model') }}<Icon name="chevronDown" size="xs" /></span></div>
          <div class="desktop-composer"><span>{{ t('guide.desktop.preview.prompt') }}</span><div class="composer-footer"><span>{{ t('guide.desktop.preview.local') }}</span><span class="composer-send" aria-hidden="true"><Icon name="arrowUp" size="sm" /></span></div></div>
        </div>
      </div>
    </div>
    <figcaption>{{ t('guide.desktop.preview.caption') }}</figcaption>
  </figure>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
const { t } = useI18n()
</script>

<style scoped>
.desktop-preview { width: 100%; max-width: var(--guide-panel); margin: 0; }
.desktop-window { overflow: hidden; border: 1px solid var(--guide-border-strong); border-radius: 16px; background: var(--guide-surface); box-shadow: 0 12px 36px -26px #15264266; }
.desktop-titlebar { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid var(--guide-border); font-size: 13px; line-height: 20px; color: var(--guide-muted); }
.window-controls { display: flex; gap: 5px; width: 48px; }.window-controls i { width: 6px; height: 6px; border: 1px solid var(--guide-border-strong); border-radius: 50%; }
.desktop-demo { font-size: 12px; }
.desktop-workspace { display: grid; grid-template-columns: 156px minmax(0,1fr); }
.desktop-project { display: flex; align-items: flex-start; flex-direction: column; gap: 18px; padding: 24px 16px; border-right: 1px solid var(--guide-border); background: var(--guide-canvas); }
.desktop-project>svg { width: 24px; height: 24px; }.project-label { margin-top: 10px; color: var(--guide-muted); font-size: 12px; }
.project-folder,.project-new { display: flex; align-items: center; gap: 7px; font-size: 13px; line-height: 22px; }.project-folder { color: var(--guide-accent-text); }.project-new { color: var(--guide-muted); }
.desktop-chat { min-width: 0; padding: 32px 24px 22px; }
.desktop-mode { display: inline-flex; align-items: center; gap: 7px; font-size: 13px; line-height: 22px; color: var(--guide-muted); }
.desktop-chat-toolbar { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.desktop-new-chat { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; line-height: 22px; color: var(--guide-accent-text); }
.desktop-context { display: flex; flex-wrap: wrap; gap: 8px; margin: 16px 0 10px; }
.desktop-context>span { display: inline-flex; align-items: center; gap: 6px; padding: 3px 8px; border: 1px solid var(--guide-border); border-radius: 6px; color: var(--guide-muted); font-size: 12px; line-height: 20px; }
h4 { margin: 16px 0 8px; font-size: 21px; font-weight: 600; line-height: 30px; }
.desktop-chat>p { margin: 0 0 24px; color: var(--guide-muted); font-size: 14px; line-height: 24px; }
.desktop-composer { padding: 15px; border: 1px solid var(--guide-border-strong); border-radius: 14px; font-size: 14px; line-height: 24px; }
.composer-footer { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-top: 18px; }
.composer-footer>span:first-child { display: inline-flex; gap: 6px; align-items: center; color: var(--guide-muted); font-size: 12px; }
.composer-send { display: grid; place-items: center; width: 27px; height: 27px; background: var(--guide-text); color: var(--guide-surface); border-radius: 50%; }
figcaption { padding-top: 10px; font-size: 13px; line-height: 22px; color: var(--guide-muted); }
@media(max-width:767px) { .desktop-workspace { grid-template-columns: 1fr; }.desktop-project { display: none; }.desktop-chat { padding: 22px 18px 18px; }h4 { font-size: 20px; }.desktop-chat>p { margin-bottom: 18px; } }
</style>
