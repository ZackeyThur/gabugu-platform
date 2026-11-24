<template>
  <svg class="guide-symbol" viewBox="0 0 48 48" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <template v-if="kind === 'client'">
      <rect x="5" y="7" width="38" height="30" rx="7" class="symbol-surface" />
      <path d="M5 16h38M17 43h14M24 37v6" class="symbol-muted" />
      <circle cx="11" cy="11.5" r="1" fill="currentColor" stroke="none" />
      <circle cx="15" cy="11.5" r="1" fill="currentColor" stroke="none" opacity=".45" />
      <path d="m13 22 5 4-5 4M23 30h11" class="symbol-accent" />
    </template>
    <template v-else-if="kind === 'switch'">
      <path d="M8 14h29l-5-5M40 34H11l5 5" class="symbol-accent" />
      <path d="M37 14v9M11 34v-9" />
      <rect x="5" y="7" width="9" height="14" rx="4.5" class="symbol-surface" />
      <rect x="34" y="27" width="9" height="14" rx="4.5" class="symbol-surface" />
      <circle cx="9.5" cy="11.5" r="1.5" fill="currentColor" stroke="none" />
      <circle cx="38.5" cy="36.5" r="1.5" fill="currentColor" stroke="none" />
    </template>
    <template v-else-if="kind === 'book'">
      <path d="M24 12c-5-5-12-5-18-3v29c6-2 13-2 18 3 5-5 12-5 18-3V9c-6-2-13-2-18 3Z" class="symbol-surface" />
      <path d="M24 12v29M12 16c2-.4 4-.2 6 .8M12 22c2-.4 4-.2 6 .8M12 28c2-.4 4-.2 6 .8" class="symbol-muted" />
      <path d="m29 23 3 3 6-7" class="symbol-accent" />
    </template>
    <!--
      AI 服务与模型：一个汇聚核心与六个外部节点，表示"多个模型由同一网关调度"。
      线条与节点使用同一 1.6 描边体系，避免看起来像芯片示意图。
    -->
    <template v-else>
      <path
        d="M24 24 24 9M24 24 37 16.5M24 24 37 31.5M24 24 24 39M24 24 11 31.5M24 24 11 16.5"
        class="symbol-muted"
        stroke-width="1.3"
        opacity=".5"
      />
      <circle cx="24" cy="24" r="6.5" class="symbol-surface" />
      <circle cx="24" cy="24" r="6.5" class="symbol-accent" stroke-width="1.6" />
      <circle cx="24" cy="24" r="2" class="symbol-dot" />
      <circle cx="24" cy="9" r="2.6" class="symbol-node" />
      <circle cx="37" cy="16.5" r="2.6" class="symbol-node" />
      <circle cx="37" cy="31.5" r="2.6" class="symbol-node" />
      <circle cx="24" cy="39" r="2.6" class="symbol-node" />
      <circle cx="11" cy="31.5" r="2.6" class="symbol-node" />
      <circle cx="11" cy="16.5" r="2.6" class="symbol-node" />
    </template>
  </svg>
</template>

<script setup lang="ts">
defineProps<{ kind: 'client' | 'switch' | 'book' | 'model' }>()
</script>

<style scoped>
.guide-symbol { display: block; width: 48px; height: 48px; flex-shrink: 0; }
.symbol-surface { fill: var(--guide-surface, var(--g-surface)); }
.symbol-muted { stroke: var(--guide-muted, var(--g-muted)); opacity: .65; }
.symbol-accent { stroke: var(--guide-accent, var(--g-accent)); stroke-width: 2; }
/* 模型图：外圈节点实心，中心点用强调色，负空间保持呼吸 */
.symbol-node { fill: var(--guide-surface, var(--g-surface)); stroke: var(--guide-muted, var(--g-muted)); stroke-width: 1.4; }
.symbol-dot { fill: var(--guide-accent, var(--g-accent)); stroke: none; }
</style>
