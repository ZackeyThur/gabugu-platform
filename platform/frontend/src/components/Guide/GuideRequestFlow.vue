<template>
  <div class="flow-diagram-root">
    <figure class="request-flow" :aria-label="t('guide.basics.flowCaption')">
      <div class="flow-layout">
        <template v-for="(role, index) in roles" :key="role.id">
          <div class="flow-node" :class="`flow-node-${role.id}`">
            <div class="flow-visual" aria-hidden="true">
              <div v-if="role.id === 'client'" class="flow-app-window">
                <div class="app-chrome"><PlatformIcon platform="openai" size="sm" /><span>Codex</span><span class="app-window-dot"></span></div>
                <div class="app-message"><span></span><span></span></div>
                <div class="app-composer"><span></span><Icon name="arrowUp" size="xs" /></div>
              </div>
              <div v-else-if="role.id === 'gateway'" class="flow-gateway-mark">
                <img src="/logo.svg" alt="" width="48" height="48" />
              </div>
              <div v-else class="flow-providers">
                <div class="provider-mark provider-openai"><PlatformIcon platform="openai" size="lg" /><span>OpenAI</span></div>
                <div class="provider-mark provider-anthropic"><PlatformIcon platform="anthropic" size="lg" /><span>Anthropic</span></div>
              </div>
            </div>
            <div class="flow-copy">
              <h3><GabuguWordmark v-if="role.id === 'gateway'" /><template v-else>{{ t(role.title) }}</template></h3>
              <p>{{ t(role.body) }}</p>
            </div>
          </div>
          <div v-if="index < roles.length - 1" class="flow-connections">
            <div class="flow-link flow-link-request"><span>{{ t('guide.basics.request') }}</span><i aria-hidden="true"></i></div>
            <div class="flow-link flow-link-reply"><i aria-hidden="true"></i><span>{{ t('guide.basics.reply') }}</span></div>
          </div>
        </template>
      </div>
      <figcaption><Icon name="infoCircle" size="sm" /><span>{{ t('guide.basics.flowCaption') }}</span></figcaption>
    </figure>
  </div>
</template>

<script setup lang="ts">
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'

const { t } = useI18n()
const roles = [
  { id: 'client', title: 'guide.basics.yourTool', body: 'guide.basics.clientRole' },
  { id: 'gateway', title: 'guide.basics.gateway', body: 'guide.basics.gatewayRole' },
  { id: 'models', title: 'guide.basics.models', body: 'guide.basics.modelRole' }
] as const
</script>

<style scoped>
.flow-diagram-root { container-type: inline-size; }
.request-flow {
  --flow-blue: var(--guide-accent-text);
  --flow-node-width: calc((100cqw - 186px) / 3);
  --flow-client-width: 154px;
  --flow-providers-width: 148px;
  margin: 0;
  padding: 26px 32px 0;
  border: 1px solid var(--guide-border);
  border-radius: 20px;
  background: var(--guide-surface);
  color: var(--guide-text);
}
.flow-layout { display: grid; grid-template-columns: minmax(0,1fr) 60px minmax(0,1fr) 60px minmax(0,1fr); align-items: start; }
.flow-node { min-width: 0; }
.flow-visual { display: flex; justify-content: center; align-items: center; height: 148px; }
.flow-copy { max-width: 240px; margin: 6px auto 0; }
.flow-copy h3 { margin: 0 0 10px; font-size: 19px; line-height: 28px; font-weight: 600; letter-spacing: normal; }
.flow-copy p { margin: 0; color: var(--guide-muted); font-size: 15px; line-height: 26px; letter-spacing: normal; }
.flow-node-gateway .flow-copy h3 { color: var(--flow-blue); }

/* A miniature desktop window makes the client recognizable without a terminal symbol. */
.flow-app-window { width: 154px; height: 106px; border: 1px solid var(--guide-border-strong); border-radius: 12px; background: var(--guide-canvas); box-shadow: 0 10px 24px -16px rgb(0 0 0 / .28); overflow: hidden; }
.app-chrome { display: flex; align-items: center; gap: 6px; padding: 9px 11px; border-bottom: 1px solid var(--guide-border); font-size: 11px; line-height: 14px; }
.app-chrome :deep(svg) { width: 13px; height: 13px; }
.app-window-dot { width: 4px; height: 4px; margin-left: auto; border-radius: 50%; background: var(--guide-faint); opacity: .6; }
.app-message { display: grid; gap: 5px; margin: 12px 14px; }
.app-message span { width: 85%; height: 3px; background: var(--guide-border-strong); border-radius: 2px; }
.app-message span + span { width: 58%; opacity: .7; }
.app-composer { display: flex; align-items: center; gap: 12px; margin: 0 9px; padding: 6px; border: 1px solid var(--guide-border); border-radius: 6px; background: var(--guide-surface); }
.app-composer span { width: 68%; height: 3px; border-radius: 2px; background: var(--guide-border-strong); }
.app-composer :deep(svg) { width: 16px; height: 16px; margin-left: auto; padding: 2px; border-radius: 50%; background: var(--flow-blue); color: var(--guide-surface); }

/* The gateway anchors the diagram between the two request/response paths. */
.flow-gateway-mark { position: relative; display: grid; place-items: center; width: 96px; height: 106px; border: 1px solid color-mix(in srgb, var(--flow-blue) 30%, var(--guide-border)); border-radius: 24px; background: linear-gradient(145deg, var(--guide-soft), var(--guide-canvas)); box-shadow: 0 8px 24px -16px color-mix(in srgb, var(--flow-blue) 50%, transparent), inset 0 1px 0 color-mix(in srgb, var(--flow-blue) 9%, transparent); }
.flow-gateway-mark img { width: 48px; height: 48px; }

.flow-providers { display: flex; gap: 10px; }
.provider-mark { display: flex; align-items: center; justify-content: center; flex-direction: column; gap: 9px; width: 69px; height: 88px; border: 1px solid var(--guide-border-strong); border-radius: 14px; background: var(--guide-canvas); box-shadow: 0 8px 20px -16px rgb(0 0 0 / .25); }
.provider-mark :deep(svg) { width: 29px; height: 29px; }
.provider-mark span { color: var(--guide-muted); font-size: 10px; line-height: 15px; }
.provider-openai { margin-top: -12px; }.provider-anthropic { margin-top: 12px; }
.provider-anthropic :deep(svg) { color: #c58869; }

.flow-connections { display: grid; align-content: center; gap: 20px; height: 148px; padding-bottom: 1px; }
.flow-connections:nth-child(2) { margin-left: calc((var(--flow-client-width) - var(--flow-node-width)) / 2 + 8px); margin-right: calc((96px - var(--flow-node-width)) / 2 + 8px); }
.flow-connections:nth-child(4) { margin-left: calc((96px - var(--flow-node-width)) / 2 + 8px); margin-right: calc((var(--flow-providers-width) - var(--flow-node-width)) / 2 + 8px); }
.flow-link { display: flex; flex-direction: column; gap: 8px; font-size: 12px; line-height: 18px; text-align: center; color: var(--flow-blue); }
.flow-link i { display: block; position: relative; height: 1px; width: 100%; background: currentColor; }
.flow-link i::after { position: absolute; right: 0; top: -3px; width: 7px; height: 7px; border-top: 1px solid currentColor; border-right: 1px solid currentColor; transform: rotate(45deg); content: ''; }
.flow-link-reply { color: var(--guide-faint); }
.flow-link-reply i::after { right: auto; left: 0; transform: rotate(-135deg); }
.request-flow figcaption { display: flex; align-items: center; gap: 9px; padding: 18px 0; margin-top: 26px; border-top: 1px solid var(--guide-border); color: var(--guide-muted); font-size: 13px; line-height: 23px; }
.request-flow figcaption svg { flex-shrink: 0; width: 15px; height: 15px; color: var(--flow-blue); }

@container (max-width: 760px) {
  .request-flow { --flow-node-width: calc((100cqw - 122px) / 3); --flow-client-width: 132px; --flow-providers-width: 128px; padding: 20px 24px 0; }
  .flow-layout { grid-template-columns: minmax(0,1fr) 36px minmax(0,1fr) 36px minmax(0,1fr); }
  .flow-app-window { width: 132px; }
  .provider-mark { width: 59px; }
  .flow-copy h3 { font-size: 18px; }
  .flow-copy p { font-size: 14px; line-height: 25px; }
}
@container (max-width: 600px) {
  .request-flow { padding: 24px 20px 0; border-radius: 16px; }
  .flow-layout { grid-template-columns: minmax(0,1fr); }
  .flow-node { display: grid; grid-template-columns: 92px minmax(0,1fr); column-gap: 22px; align-items: center; }
  .flow-visual { height: 112px; }
  .flow-copy { max-width: none; margin: 0; }
  .flow-copy h3 { font-size: 17px; line-height: 25px; margin-bottom: 8px; }
  .flow-copy p { font-size: 14px; line-height: 24px; }
  .flow-app-window { width: 90px; height: 86px; border-radius: 9px; }
  .app-chrome { padding: 6px 7px; font-size: 10px; gap: 5px; }
  .app-chrome :deep(svg) { width: 11px; height: 11px; }
  .app-message { margin: 9px; gap: 4px; }
  .app-composer { margin: 0 6px; padding: 4px; gap: 6px; }
  .app-composer :deep(svg) { width: 14px; height: 14px; flex-shrink: 0; }
  .flow-gateway-mark { width: 76px; height: 80px; border-radius: 19px; }
  .flow-gateway-mark img { width: 40px; height: 40px; }
  .flow-providers { display: grid; gap: 8px; width: 92px; }
  .provider-mark { flex-direction: row; width: 92px; height: 42px; gap: 7px; border-radius: 10px; margin: 0; }
  .provider-mark :deep(svg) { width: 21px; height: 21px; }
  .provider-mark span { font-size: 10px; line-height: 15px; }
  .flow-connections { display: flex; align-items: center; gap: 18px; height: 54px; margin: 4px 0; padding: 0 0 0 35px; }
  .flow-connections:nth-child(2), .flow-connections:nth-child(4) { margin-inline: 0; }
  .flow-link { flex-direction: row; align-items: center; gap: 9px; text-align: left; }
  .flow-link i { order: -1; width: 1px; height: 26px; }
  .flow-link i::after { top: auto; bottom: 0; left: -3px; right: auto; transform: rotate(135deg); }
  .flow-link-reply i::after { top: 0; bottom: auto; transform: rotate(-45deg); }
  .request-flow figcaption { margin-top: 20px; align-items: flex-start; font-size: 12px; line-height: 22px; }
  .request-flow figcaption svg { margin-top: 3px; }
}
</style>
