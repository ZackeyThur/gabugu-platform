<template>
  <div class="inspector" :class="{ 'inspector--podium': variant === 'podium' }" data-testid="arena-inspector">
    <div class="inspector-top">
      <div v-if="variant === 'podium'" class="inspector-title">
        <span class="inspector-rank">#{{ entry.rank }}</span>
        <ArenaOrgIcon :organization="entry.organization" :model-name="entry.model_name" :size="16" :dark="theme === 'dark'" />
        <strong>{{ entry.model_name }}</strong>
        <span class="inspector-org">{{ orgMeta.displayName }}</span>
      </div>
      <div v-else class="inspector-title inspector-title--subtle">
        <span>{{ t('arena.inspector.movement') }}</span>
        <em v-if="rankMovement" :class="rankMovement.cls">{{ rankMovement.text }}</em>
        <em v-if="scoreDeltaText" :class="(entry.score_delta ?? 0) > 0 ? 'is-up' : 'is-down'">{{ scoreDeltaText }}</em>
        <span v-if="!rankMovement && !scoreDeltaText" class="inspector-ci-inline">{{ t('arena.inspector.unchanged') }}</span>
      </div>

      <div class="inspector-actions">
        <button
          type="button"
          class="inspector-action"
          :class="{ 'is-copied': copied }"
          data-testid="arena-copy-model-btn"
          @click.stop="copyModelName"
        >
          <svg v-if="copied" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <rect x="9" y="9" width="11" height="11" rx="2" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
          </svg>
          <span>{{ copied ? t('arena.inspector.copied') : t('arena.inspector.copyModel') }}</span>
        </button>

        <button
          type="button"
          class="inspector-action"
          data-testid="arena-filter-org-btn"
          @click.stop="emit('filter-org', orgMeta.key)"
        >
          <span>{{ t('arena.inspector.filterOrg', { org: orgMeta.displayName }) }}</span>
        </button>

        <button
          type="button"
          class="inspector-close"
          :aria-label="t('arena.inspector.close')"
          :title="t('arena.inspector.close')"
          @click.stop="emit('close')"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path stroke-linecap="round" d="M6 6l12 12M18 6L6 18" />
          </svg>
        </button>
      </div>
    </div>

    <dl class="inspector-grid">
      <div class="inspector-metric">
        <dt>{{ t('arena.inspector.exactScore') }}</dt>
        <dd>
          <strong>{{ formatArenaExactScore(entry.score) }}</strong>
          <em v-if="scoreDeltaText" :class="(entry.score_delta ?? 0) > 0 ? 'is-up' : 'is-down'">{{ scoreDeltaText }}</em>
        </dd>
      </div>
      <div class="inspector-metric">
        <dt>{{ t('arena.inspector.confidenceInterval') }}</dt>
        <dd>
          <template v-if="entry.rating_lower > 0 && entry.rating_upper > 0">
            {{ formatArenaExactScore(entry.rating_lower) }} – {{ formatArenaExactScore(entry.rating_upper) }}
            <small v-if="entry.ci > 0">±{{ entry.ci }}</small>
          </template>
          <template v-else-if="entry.ci > 0">±{{ entry.ci }}</template>
          <template v-else>—</template>
        </dd>
      </div>
      <div class="inspector-metric">
        <dt>{{ t('arena.inspector.gapToLeader') }}</dt>
        <dd>
          <span v-if="entry.rank === 1" class="is-leader">{{ t('arena.inspector.leaderLabel') }}</span>
          <span v-else-if="gapValue < 0.05">{{ t('arena.podium.tiedLeader') }}</span>
          <span v-else>-{{ gapValue.toFixed(1) }}</span>
        </dd>
      </div>
      <div class="inspector-metric">
        <dt>{{ t('arena.inspector.votes') }}</dt>
        <dd>{{ formatArenaNumber(entry.votes) }}</dd>
      </div>
      <div class="inspector-metric">
        <dt>{{ t('arena.inspector.license') }}</dt>
        <dd>
          <span :class="{ 'is-open-text': entry.is_open }">{{ licenseLabel }}</span>
        </dd>
      </div>
    </dl>

    <div class="inspector-cross">
      <span class="inspector-cross-label">{{ t('arena.inspector.crossTrack') }}</span>
      <div v-if="crossTrackMatches.length > 0" class="inspector-cross-chips">
        <button
          v-for="match in crossTrackMatches"
          :key="match.categoryId"
          type="button"
          class="inspector-cross-chip"
          :data-testid="`arena-cross-track-${match.categoryId}`"
          @click.stop="emit('jump-track', match.categoryId, entry.model_name)"
        >
          <span class="inspector-cross-track">{{ t(`arena.categories.${match.categoryId}.label`) }}</span>
          <strong class="inspector-cross-rank">#{{ match.rank }}</strong>
          <span class="inspector-cross-score">{{ formatArenaScore(match.score) }}</span>
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 11L11 5M6.5 5H11v4.5" />
          </svg>
        </button>
      </div>
      <span v-else class="inspector-cross-empty">{{ t('arena.inspector.singleTrack') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ArenaCategoryBoard, ArenaCategoryId, ArenaModelEntry } from '@/api/arena'
import ArenaOrgIcon from './ArenaOrgIcon.vue'
import {
  formatArenaExactScore,
  formatArenaNumber,
  formatArenaScore,
  formatScoreDelta,
  resolveArenaOrgMeta
} from '@/utils/arenaOrg'

const props = withDefaults(
  defineProps<{
    entry: ArenaModelEntry
    activeBoard: ArenaCategoryBoard
    allBoards: ArenaCategoryBoard[]
    theme?: 'dark' | 'light'
    variant?: 'row' | 'podium'
  }>(),
  { theme: 'dark', variant: 'row' }
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'filter-org', orgKey: string): void
  (e: 'jump-track', categoryId: ArenaCategoryId, modelName: string): void
}>()

const { t } = useI18n()

const copied = ref(false)
let copyTimer: number | undefined

const orgMeta = computed(() => resolveArenaOrgMeta(props.entry.organization, props.entry.model_name))
const scoreDeltaText = computed(() => formatScoreDelta(props.entry.score_delta))
const rankMovement = computed<{ cls: string; text: string } | null>(() => {
  if (props.entry.is_new) return { cls: 'is-new', text: t('arena.table.newBadge') }
  if (props.entry.rank_delta != null && props.entry.rank_delta > 0) return { cls: 'is-up', text: `▲${props.entry.rank_delta}` }
  if (props.entry.rank_delta != null && props.entry.rank_delta < 0) return { cls: 'is-down', text: `▼${Math.abs(props.entry.rank_delta)}` }
  return null
})

const leaderScore = computed(() => props.activeBoard.entries[0]?.score ?? props.entry.score)
const gapValue = computed(() => Math.max(0, leaderScore.value - props.entry.score))

const licenseLabel = computed(() => {
  const raw = (props.entry.license || '').trim()
  if (!raw || raw.toLowerCase() === 'proprietary') return t('arena.inspector.proprietary')
  return raw
})

const crossTrackMatches = computed(() => {
  const target = props.entry.model_name.trim().toLowerCase()
  const matches: Array<{ categoryId: ArenaCategoryId; rank: number; score: number }> = []
  for (const board of props.allBoards) {
    if (board.id === props.activeBoard.id) continue
    const found = board.entries.find((item) => item.model_name.trim().toLowerCase() === target)
    if (found) {
      matches.push({ categoryId: board.id, rank: found.rank, score: found.score })
    }
  }
  return matches
})

async function copyModelName() {
  try {
    await navigator.clipboard.writeText(props.entry.model_name)
    copied.value = true
    if (copyTimer) window.clearTimeout(copyTimer)
    copyTimer = window.setTimeout(() => {
      copied.value = false
    }, 1800)
  } catch {
    // Clipboard unavailable in insecure contexts; ignore silently.
  }
}

onBeforeUnmount(() => {
  if (copyTimer) window.clearTimeout(copyTimer)
})
</script>

<style scoped>
.inspector {
  padding: 18px 20px 20px;
  border-radius: 12px;
  background: var(--a-inspector-bg);
  border: 1px solid var(--a-line);
}
.inspector--podium {
  margin-top: 12px;
  border-color: var(--a-line-strong);
  box-shadow: var(--a-inspector-shadow);
}

.inspector-top {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--a-line);
}
.inspector-title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: 14px;
}
.inspector-title--subtle {
  color: var(--a-muted);
  font-size: 13px;
}
.inspector-title--subtle strong {
  color: var(--a-text);
  font-size: 15px;
}
.inspector-title--subtle em {
  font-style: normal;
  font-size: 13px;
  font-weight: 600;
}
.inspector-title--subtle em.is-up {
  color: var(--a-up);
}
.inspector-title--subtle em.is-down {
  color: var(--a-down);
}
.inspector-title--subtle em.is-new {
  color: var(--a-new);
  font-size: 13px;
  letter-spacing: 0.04em;
}
.inspector-rank {
  color: var(--a-gold);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.06em;
}
.inspector-org {
  color: var(--a-muted);
  font-size: 13px;
}
.inspector-ci-inline {
  color: var(--a-faint);
  font-size: 13px;
}

.inspector-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.inspector-action {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 11px;
  border: 1px solid var(--a-line-strong);
  border-radius: 999px;
  background: var(--a-chip-bg);
  color: var(--a-text-2);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.16s, border-color 0.16s, color 0.16s, transform 0.12s;
}
.inspector-action:hover {
  background: var(--a-hover-strong);
  color: var(--a-text);
}
.inspector-action:active {
  transform: scale(0.97);
}
.inspector-action.is-copied {
  border-color: color-mix(in srgb, var(--a-up) 45%, transparent);
  color: var(--a-up);
}
.inspector-action svg {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
}
.inspector-close {
  display: inline-flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: none;
  color: var(--a-muted);
  cursor: pointer;
  transition: background-color 0.15s, color 0.15s;
}
.inspector-close:hover {
  background: var(--a-hover-strong);
  color: var(--a-text);
}
.inspector-close svg {
  width: 15px;
  height: 15px;
}

.inspector-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 16px;
  margin: 14px 0 0;
}
.inspector-metric dt {
  color: var(--a-faint);
  font-size: 13px;
  font-weight: 500;
}
.inspector-metric dd {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
  margin: 5px 0 0;
  color: var(--a-text);
  font-size: 14px;
  font-weight: 500;
}
.inspector-metric dd strong {
  font-size: 15px;
  font-weight: 600;
}
.inspector-metric dd small {
  color: var(--a-faint);
  font-size: 13px;
  font-weight: 400;
}
.inspector-metric em {
  font-style: normal;
  font-size: 13px;
  font-weight: 600;
}
.inspector-metric em.is-up,
.is-open-text {
  color: var(--a-up);
}
.inspector-metric em.is-down {
  color: var(--a-down);
}
.is-leader {
  color: var(--a-gold);
  font-weight: 600;
}

.inspector-cross {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 12px;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid var(--a-line);
}
.inspector-cross-label {
  color: var(--a-muted);
  font-size: 13px;
  font-weight: 500;
}
.inspector-cross-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.inspector-cross-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--a-line);
  border-radius: 999px;
  background: var(--a-chip-bg);
  color: var(--a-text-2);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
  transition: border-color 0.16s, background-color 0.16s, color 0.16s, transform 0.14s;
}
.inspector-cross-chip:hover {
  border-color: var(--a-accent-border);
  background: var(--a-accent-soft);
  color: var(--a-text);
  transform: translateY(-1px);
}
.inspector-cross-rank {
  color: var(--a-accent);
  font-weight: 600;
}
.inspector-cross-score {
  color: var(--a-muted);
}
.inspector-cross-chip svg {
  width: 12px;
  height: 12px;
  color: var(--a-faint);
  transition: transform 0.16s, color 0.16s;
}
.inspector-cross-chip:hover svg {
  color: var(--a-accent);
  transform: translate(1px, -1px);
}
.inspector-cross-empty {
  color: var(--a-faint);
  font-size: 13px;
}

@media (max-width: 820px) {
  .inspector-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px 16px;
  }
}

@media (max-width: 640px) {
  .inspector {
    padding: 14px 14px 16px;
  }
  .inspector-top {
    gap: 10px;
  }
}
</style>
