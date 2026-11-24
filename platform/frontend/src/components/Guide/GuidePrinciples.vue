<template>
  <div class="principles">
    <GuideRequestFlow />
    <div class="config-section">
      <div class="config-explanation"><h3>{{ t('guide.basics.configTitle') }}</h3><p>{{ t('guide.basics.configIntro') }}</p></div>
      <GuideConfigFlow />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import GuideRequestFlow from './GuideRequestFlow.vue'
import GuideConfigFlow from './GuideConfigFlow.vue'
const { t } = useI18n()
</script>

<style scoped>

.principles { color: var(--guide-text); }

/* 配置字段：标题在上、示意图在下通栏，避免左侧被挤成窄纸条 */
.config-section {
  display: flex;
  flex-direction: column;
  gap: 24px;
  margin-top: 48px;
}
.config-explanation h3 {
  margin: 0 0 12px;
  font-size: 22px;
  line-height: 1.35;
  font-weight: 600;
  letter-spacing: normal;
}
.config-explanation p {
  max-width: 34em;
  margin: 0;
  color: var(--guide-muted);
  font-size: 16px;
  line-height: 27px;
}
@media (max-width: 1599px) {
  .config-section { grid-template-columns: 1fr; gap: 28px; }
  .config-explanation { max-width: 38em; }
}
@media (max-width: 767px) {

  .config-explanation h3 { font-size: 20px; }
  .config-explanation p { font-size: 15.5px; line-height: 26px; }
}
</style>
