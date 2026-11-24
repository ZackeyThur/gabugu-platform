<template>
  <div class="guide-hero-art" aria-hidden="true">
    <!-- 连接线：工具 → 网关 → 模型，往返方向用双弧表示 -->
    <svg class="hero-connectors" viewBox="0 0 420 300" fill="none">
      <path d="M86 150H150" class="link" />
      <path d="M270 150c34 0 30-52 62-52h18" class="link" />
      <path d="M270 150c34 0 30 52 62 52h18" class="link" />
      <circle cx="118" cy="150" r="2.5" class="node" />
      <circle cx="303" cy="98" r="2.5" class="node" />
      <circle cx="303" cy="202" r="2.5" class="node" />
    </svg>

    <div class="hero-client">
      <GuideSymbol kind="client" />
      <span>{{ t('guide.basics.yourTool') }}</span>
    </div>

    <div class="hero-gateway">
      <img src="/logo.svg" alt="" width="44" height="44" />
      <GabuguWordmark class="hero-gateway-wordmark" />
      <span class="hero-gateway-role">{{ t('guide.basics.gatewayKind') }}</span>
    </div>

    <div class="hero-models">
      <div class="hero-model hero-model-claude">
        <PlatformIcon platform="anthropic" size="lg" />
      </div>
      <div class="hero-model hero-model-openai">
        <PlatformIcon platform="openai" size="lg" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import GabuguWordmark from '@/components/brand/GabuguWordmark.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GuideSymbol from './GuideSymbol.vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
</script>

<style scoped>
/* 容器查询让整幅图按列宽等比缩放：420×300 的设计尺寸映射到任意 100cqw */
.guide-hero-art {
  position: relative;
  container-type: inline-size;
  width: 100%;
  max-width: 560px;
  aspect-ratio: 420 / 300;
  margin-right: auto;
}

.hero-connectors { position: absolute; inset: 0; width: 100%; height: 100%; overflow: visible; }
.hero-connectors .link {
  stroke: var(--guide-border-strong);
  stroke-width: 1.5;
  stroke-dasharray: 4 5;
}
.hero-connectors .node { fill: var(--guide-accent); opacity: 0.55; }

/* 工具 */
.hero-client {
  position: absolute;
  left: 0;
  top: 25.7cqw;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1.4cqw;
  width: 20.5cqw;
  height: 20cqw;
  border: 1px solid var(--guide-border);
  border-radius: 3.3cqw;
  background: var(--guide-surface);
  box-shadow: 0 6px 18px -12px rgb(16 24 40 / 0.2);
}
.hero-client :deep(svg) { width: 7.1cqw; height: 7.1cqw; }
/* 中文不加正字距：方块字本身就是等宽的，加宽只会在字与字之间
   多出肉眼可见的缝，和页面上其它中文的节奏对不上 */
.hero-client span {
  color: var(--guide-muted);
  font-size: 2.74cqw;
  line-height: 1.4;
  letter-spacing: normal;
}

/* 网关：视觉主体 */
.hero-gateway {
  position: absolute;
  left: 35.7cqw;
  top: 14.76cqw;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1.9cqw;
  width: 28.6cqw;
  height: 41.9cqw;
  border: 1px solid var(--guide-border);
  border-radius: 4.8cqw;
  background: var(--guide-surface);
  box-shadow:
    0 1px 2px rgb(16 24 40 / 0.04),
    0 16px 40px -24px rgb(16 24 40 / 0.28);
}
.hero-gateway img { display: block; width: 10.5cqw; height: 10.5cqw; }
.hero-gateway-wordmark { font-size: 3.3cqw; }
/* 「API 网关」中英混排：交给 text-autospace 处理，
   手写字距会把中文那两个字也一起拉开 */
.hero-gateway-role {
  color: var(--guide-faint);
  font-size: 2.6cqw;
  line-height: 1.4;
  letter-spacing: normal;
}

/* 模型：两张小卡上下错落 */
.hero-models {
  position: absolute;
  right: 0;
  top: 14.3cqw;
  display: flex;
  flex-direction: column;
  gap: 10.5cqw;
}
.hero-model {
  display: grid;
  place-items: center;
  width: 14.76cqw;
  height: 14.76cqw;
  border: 1px solid var(--guide-border);
  border-radius: 3.6cqw;
  background: var(--guide-surface);
  box-shadow: 0 6px 18px -12px rgb(16 24 40 / 0.2);
}
.hero-model :deep(svg) { width: 6.7cqw; height: 6.7cqw; }
.hero-model-claude { color: #bf7a58; margin-left: -5.2cqw; }
.hero-model-openai { color: var(--guide-text); margin-left: 5.2cqw; }
</style>
