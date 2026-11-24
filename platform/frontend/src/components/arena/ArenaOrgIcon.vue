<template>
  <span class="arena-org" :class="{ 'arena-org--invert': dark && isMonochrome }" :style="{ width: `${size}px`, height: `${size}px` }" :title="meta.displayName">
    <ModelIcon v-if="meta.iconKey" :model="iconModelHint" :size="`${size}px`" />
    <span
      v-else
      class="arena-org-badge"
      :style="{ color: meta.color, background: `${meta.color}24`, fontSize: `${Math.max(9, Math.round(size * 0.46))}px` }"
    >{{ meta.shortBadge }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { resolveArenaOrgMeta } from '@/utils/arenaOrg'

const props = withDefaults(
  defineProps<{
    organization?: string
    modelName?: string
    size?: number
    dark?: boolean
  }>(),
  { organization: '', modelName: '', size: 18, dark: false }
)

const meta = computed(() => resolveArenaOrgMeta(props.organization, props.modelName))

// These marks are drawn in black by ModelIcon and disappear on a dark page.
const isMonochrome = computed(() => ['openai', 'xai', 'moonshot', 'midjourney'].includes(meta.value.iconKey ?? ''))

const ICON_HINTS: Record<string, string> = {
  claude: 'claude',
  openai: 'gpt',
  gemini: 'gemini',
  xai: 'grok',
  deepseek: 'deepseek',
  qwen: 'qwen',
  zhipu: 'glm',
  moonshot: 'moonshot',
  meta: 'llama',
  mistral: 'mistral',
  minimax: 'minimax',
  doubao: 'doubao',
  wenxin: 'ernie',
  hunyuan: 'hunyuan',
  cohere: 'command',
  midjourney: 'midjourney'
}
const iconModelHint = computed(() => ICON_HINTS[meta.value.iconKey ?? ''] ?? (props.modelName || props.organization))
</script>

<style scoped>
.arena-org { display: inline-flex; flex-shrink: 0; align-items: center; justify-content: center; }
.arena-org :deep(svg), .arena-org :deep(img) { display: block; }
.arena-org--invert :deep(svg) { filter: invert(1); }
.arena-org-badge { display: inline-flex; width: 100%; height: 100%; align-items: center; justify-content: center; border-radius: 50%; font-weight: 600; letter-spacing: -0.02em; line-height: 1; }
</style>
