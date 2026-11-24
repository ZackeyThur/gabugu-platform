<template>
  <section ref="rootRef" class="arena" :data-theme="theme" :class="{ 'arena--embedded': embedded }">
    <header class="arena-hero">
      <div class="arena-eyebrow" data-arena-intro="hero">
        <span class="arena-eyebrow-dot" aria-hidden="true"></span>
        <span>{{ t('arena.eyebrow') }}</span>
      </div>
      <div class="arena-hero-row">
        <div class="arena-hero-copy">
          <h1 v-if="!embedded" class="arena-title" data-arena-intro="hero">{{ t('arena.title') }}</h1>
          <p class="arena-subtitle" data-arena-intro="hero">{{ t('arena.subtitle') }}</p>
        </div>
        <div v-if="response" class="arena-hero-actions" data-arena-intro="hero">
          <button
            type="button"
            class="arena-btn arena-btn--ghost"
            :class="{ 'is-spinning': loading }"
            :disabled="loading"
            @click="emit('refresh')"
          >
            <svg class="arena-btn-icon arena-refresh-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M20 11a8 8 0 10-2.34 5.66M20 4v7h-7" />
            </svg>
            <span>{{ t('arena.status.refresh') }}</span>
          </button>
          <button
            v-if="activeBoard"
            type="button"
            class="arena-btn arena-btn--primary"
            data-testid="open-poster-modal-btn"
            @click="posterOpen = true"
          >
            <svg class="arena-btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v2a2 2 0 002 2h12a2 2 0 002-2v-2M12 4v11m0-11l-4 4m4-4l4 4" />
            </svg>
            <span>{{ t('arena.status.share') }}</span>
          </button>
        </div>
      </div>
    </header>

    <!-- Loading -->
    <div v-if="loading && !response" class="arena-skeleton" aria-busy="true">
      <div class="arena-skeleton-tabs"><span v-for="n in 4" :key="n"></span></div>
      <div class="arena-skeleton-podium"><span v-for="n in 3" :key="n"></span></div>
      <div v-for="n in 6" :key="`r${n}`" class="arena-skeleton-row"></div>
    </div>

    <!-- First sync still running (503) -->
    <div v-else-if="initializing && !response" class="arena-state">
      <h2>{{ t('arena.states.initializingTitle') }}</h2>
      <p>{{ t('arena.states.initializingDesc') }}</p>
      <button type="button" class="arena-btn arena-btn--ghost" @click="emit('refresh')">{{ t('arena.states.retry') }}</button>
    </div>

    <!-- Error -->
    <div v-else-if="error && !response" class="arena-state">
      <h2>{{ t('arena.states.loadFailedTitle') }}</h2>
      <p>{{ t('arena.states.loadFailedDesc') }}</p>
      <button type="button" class="arena-btn arena-btn--ghost" @click="emit('refresh')">{{ t('arena.states.retry') }}</button>
    </div>

    <template v-else-if="activeBoard">
      <!-- Tracks with sliding underline indicator -->
      <nav
        ref="tabsRef"
        class="arena-tabs"
        :class="{ 'has-indicator': tabIndicator.ready }"
        role="tablist"
        aria-label="Arena tracks"
        data-arena-intro="chrome"
      >
        <button
          v-for="board in allBoards"
          :key="board.id"
          type="button"
          role="tab"
          class="arena-tab"
          :class="{ 'is-active': activeCategory === board.id }"
          :aria-selected="activeCategory === board.id"
          :data-testid="`arena-tab-${board.id}`"
          @click="selectCategory(board.id)"
        >
          <svg v-if="board.id === 'text'" class="arena-tab-icon" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 5.5h12M4 10h9M4 14.5h6.5" />
          </svg>
          <svg v-else-if="board.id === 'code'" class="arena-tab-icon" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.8 6.5L3.3 10l3.5 3.5M13.2 6.5L16.7 10l-3.5 3.5M11.2 4.5l-2.4 11" />
          </svg>
          <svg v-else-if="board.id === 'vision'" class="arena-tab-icon" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M2.5 10s2.8-4.8 7.5-4.8S17.5 10 17.5 10s-2.8 4.8-7.5 4.8S2.5 10 2.5 10z" />
            <circle cx="10" cy="10" r="2.2" />
          </svg>
          <svg v-else class="arena-tab-icon" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
            <rect x="3" y="4" width="14" height="12" rx="2.2" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.5 13l2.6-2.8 2 2 2.4-2.7L15 11.5" />
            <circle cx="7.2" cy="7.5" r="0.9" fill="currentColor" stroke="none" />
          </svg>
          <span>{{ t(`arena.categories.${board.id}.label`) }}</span>
          <span class="arena-tab-count">{{ board.total_models }}</span>
        </button>
        <span
          v-if="tabIndicator.ready"
          class="arena-tabs-indicator"
          aria-hidden="true"
          :style="{ transform: `translate3d(${tabIndicator.left}px, 0, 0)`, width: `${tabIndicator.width}px` }"
        ></span>
      </nav>

      <div class="arena-meta" data-arena-intro="chrome">
        <span class="arena-meta-desc">
          {{ t(`arena.categories.${activeBoard.id}.desc`) }}
          <span v-if="activeBoard.style_controlled" class="arena-meta-pill">{{ t('arena.status.styleControl') }}</span>
        </span>
        <span class="arena-meta-live">
          <i class="arena-live-dot" aria-hidden="true"></i>
          {{ t('arena.status.published', { date: formatDay(activeBoard.publish_date) }) }}
          <template v-if="syncedText"> · {{ t('arena.status.synced', { time: syncedText }) }}</template>
        </span>
      </div>

      <!-- Top 3 Architectural Podium -->
      <div v-if="showPodium" class="arena-podium-shell">
        <div class="arena-podium">
          <article
            v-for="(entry, idx) in podium"
            :key="`${activeBoard.id}-${entry.model_name}`"
            class="arena-podium-item"
            :class="[`is-rank-${entry.rank}`, { 'is-selected': selectedModel === entry.model_name }]"
            :style="{ '--org-color': orgMeta(entry).color }"
            data-arena-anim="podium"
            data-testid="arena-podium-card"
            role="button"
            tabindex="0"
            :aria-expanded="selectedModel === entry.model_name"
            @click="toggleModelInspector(entry.model_name)"
            @keydown.enter.prevent="toggleModelInspector(entry.model_name)"
            @keydown.space.prevent="toggleModelInspector(entry.model_name)"
            @pointermove="handleSpotlightMove"
            @pointerleave="handleSpotlightLeave"
          >
            <div class="arena-podium-head">
              <div class="arena-podium-rank">
                <span class="arena-podium-num">{{ String(entry.rank).padStart(2, '0') }}</span>
                <em v-if="rankDelta(entry)" :class="rankDelta(entry)!.cls">{{ rankDelta(entry)!.text }}</em>
              </div>
              <span v-if="podiumGapLabel(entry, idx)" class="arena-podium-gap" :class="{ 'is-leader': idx === 0 }">
                {{ podiumGapLabel(entry, idx) }}
              </span>
            </div>

            <h3 class="arena-podium-name">{{ entry.model_name }}</h3>

            <div class="arena-podium-org">
              <ArenaOrgIcon :organization="entry.organization" :model-name="entry.model_name" :size="16" :dark="theme === 'dark'" />
              <span>{{ orgMeta(entry).displayName }}</span>
              <span v-if="entry.is_open" class="arena-tag-open">{{ t('arena.table.openBadge') }}</span>
            </div>

            <div class="arena-podium-bottom">
              <div class="arena-podium-score">
                <strong>{{ displayPodiumScore(entry, idx) }}</strong>
                <span v-if="entry.ci > 0" class="arena-ci">±{{ entry.ci }}</span>
                <span v-if="scoreDelta(entry)" :class="scoreDelta(entry)!.cls">{{ scoreDelta(entry)!.text }}</span>
              </div>
              <div class="arena-podium-foot">
                <div class="arena-podium-gauge" aria-hidden="true">
                  <span class="arena-podium-gauge-fill" :style="{ width: `${barPercent(entry.score)}%` }"></span>
                  <span
                    v-if="ciGeometry(entry)"
                    class="arena-podium-gauge-ci"
                    :style="{ left: `${ciGeometry(entry)!.left}%`, width: `${ciGeometry(entry)!.width}%` }"
                  ></span>
                </div>
                <div class="arena-podium-meta-row">
                  <span>{{ t('arena.status.votesUnit', { votes: formatArenaNumber(entry.votes) }) }}</span>
                  <span class="arena-podium-inspect-cue">
                    {{ selectedModel === entry.model_name ? t('arena.inspector.close') : t('arena.podium.inspectHint') }}
                  </span>
                </div>
              </div>
            </div>
          </article>
        </div>

        <!-- Inline spotlight inspector for Top-3 selection -->
        <div class="arena-drawer-wrap" :class="{ 'is-open': Boolean(selectedPodiumEntry) }">
          <div class="arena-drawer-inner">
            <ArenaModelInspector
              v-if="selectedPodiumEntry"
              :entry="selectedPodiumEntry"
              :active-board="activeBoard"
              :all-boards="allBoards"
              :theme="theme"
              variant="podium"
              @close="selectedModel = null"
              @filter-org="applyOrgFilter"
              @jump-track="jumpToModelInTrack"
            />
          </div>
        </div>
      </div>

      <!-- What changed (Interactive insights) -->
      <dl v-if="showPodium" class="arena-insights">
        <div
          data-arena-anim="insight"
          :class="{ 'is-clickable': Boolean(activeBoard.highlights.biggest_riser) }"
          :role="activeBoard.highlights.biggest_riser ? 'button' : undefined"
          :tabindex="activeBoard.highlights.biggest_riser ? 0 : undefined"
          :title="activeBoard.highlights.biggest_riser ? t('arena.insights.locate') : undefined"
          @click="activeBoard.highlights.biggest_riser && locateModelInCurrentBoard(activeBoard.highlights.biggest_riser.model_name)"
          @keydown.enter.prevent="activeBoard.highlights.biggest_riser && locateModelInCurrentBoard(activeBoard.highlights.biggest_riser.model_name)"
        >
          <dt>
            <span>{{ t('arena.insights.riser') }}</span>
            <svg v-if="activeBoard.highlights.biggest_riser" class="arena-insight-arrow" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 11L11 5M6.5 5H11v4.5" />
            </svg>
          </dt>
          <dd v-if="activeBoard.highlights.biggest_riser">
            <span class="arena-insight-name">{{ activeBoard.highlights.biggest_riser.model_name }}</span>
            <em class="is-up">▲{{ activeBoard.highlights.biggest_riser.rank_delta }}</em>
            <small>{{ t('arena.insights.rankAt', { rank: activeBoard.highlights.biggest_riser.rank }) }}</small>
          </dd>
          <dd v-else class="is-muted">{{ activeBoard.previous_publish_date ? t('arena.insights.none') : t('arena.insights.noBaseline') }}</dd>
        </div>

        <div data-arena-anim="insight">
          <dt>{{ t('arena.insights.newEntries') }}</dt>
          <dd v-if="activeBoard.previous_publish_date">{{ t('arena.insights.newCount', { count: activeBoard.highlights.new_entries }) }}</dd>
          <dd v-else class="is-muted">{{ t('arena.insights.noBaseline') }}</dd>
        </div>

        <div
          data-arena-anim="insight"
          :class="{ 'is-clickable': Boolean(activeBoard.highlights.top_open_model) }"
          :role="activeBoard.highlights.top_open_model ? 'button' : undefined"
          :tabindex="activeBoard.highlights.top_open_model ? 0 : undefined"
          :title="activeBoard.highlights.top_open_model ? t('arena.insights.locate') : undefined"
          @click="activeBoard.highlights.top_open_model && locateModelInCurrentBoard(activeBoard.highlights.top_open_model)"
          @keydown.enter.prevent="activeBoard.highlights.top_open_model && locateModelInCurrentBoard(activeBoard.highlights.top_open_model)"
        >
          <dt>
            <span>{{ t('arena.insights.topOpen') }}</span>
            <svg v-if="activeBoard.highlights.top_open_model" class="arena-insight-arrow" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 11L11 5M6.5 5H11v4.5" />
            </svg>
          </dt>
          <dd v-if="activeBoard.highlights.top_open_model">
            <span class="arena-insight-name">{{ activeBoard.highlights.top_open_model }}</span>
            <small>{{ t('arena.insights.rankAt', { rank: activeBoard.highlights.top_open_rank }) }}</small>
          </dd>
          <dd v-else class="is-muted">{{ t('arena.insights.none') }}</dd>
        </div>
      </dl>

      <!-- Filters + Organization Quick Strip -->
      <div class="arena-controls">
        <div class="arena-toolbar">
          <label class="arena-search">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.2-5.2M17 10.5a6.5 6.5 0 11-13 0 6.5 6.5 0 0113 0z" />
            </svg>
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="search"
              data-testid="arena-search-input"
              :placeholder="t('arena.filters.search')"
              :aria-label="t('arena.filters.search')"
            />
            <button
              v-if="searchQuery"
              type="button"
              class="arena-search-clear"
              :aria-label="t('arena.filters.clear')"
              @click="searchQuery = ''"
            >
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
                <path stroke-linecap="round" d="M4.5 4.5l7 7M11.5 4.5l-7 7" />
              </svg>
            </button>
            <kbd v-else class="arena-kbd" aria-hidden="true">/</kbd>
          </label>

          <div ref="segmentRef" class="arena-segment" :class="{ 'has-thumb': segmentThumb.ready }" role="group">
            <span
              v-if="segmentThumb.ready"
              class="arena-segment-thumb"
              aria-hidden="true"
              :style="{ transform: `translate3d(${segmentThumb.left}px, 0, 0)`, width: `${segmentThumb.width}px` }"
            ></span>
            <button
              v-for="opt in licenseOptions"
              :key="opt.value"
              type="button"
              :class="{ 'is-active': licenseFilter === opt.value }"
              :data-testid="`arena-license-${opt.value}`"
              @click="licenseFilter = opt.value"
            >
              {{ opt.label }}
            </button>
          </div>

          <span v-if="hasActiveFilters" class="arena-results">
            {{ t('arena.filters.results', { count: filteredEntries.length }) }}
            <button type="button" @click="clearFilters">{{ t('arena.filters.clear') }}</button>
          </span>
        </div>

        <div v-if="topOrgs.length >= 2" class="arena-org-strip" role="group" aria-label="Filter by organization">
          <button
            type="button"
            class="arena-org-chip"
            :class="{ 'is-active': orgFilter === 'all' }"
            data-testid="arena-org-chip-all"
            @click="orgFilter = 'all'"
          >
            <span>{{ t('arena.filters.allOrgs') }}</span>
          </button>
          <button
            v-for="org in topOrgs"
            :key="org.key"
            type="button"
            class="arena-org-chip"
            :class="{ 'is-active': orgFilter === org.key }"
            :data-testid="`arena-org-chip-${org.key}`"
            @click="toggleOrgFilter(org.key)"
          >
            <ArenaOrgIcon :organization="org.rawOrg" :model-name="org.sampleModel" :size="14" :dark="theme === 'dark'" />
            <span>{{ org.displayName }}</span>
            <small>{{ org.count }}</small>
          </button>
        </div>
      </div>

      <!-- Ranking list -->
      <div v-if="filteredEntries.length === 0" class="arena-empty">{{ t('arena.table.empty') }}</div>
      <div v-else class="arena-list" role="table" data-testid="arena-list">
        <div class="arena-row arena-row--head" role="row">
          <button
            type="button"
            role="columnheader"
            class="arena-sort-btn"
            :class="{ 'is-sorted': sortKey === 'rank' }"
            data-testid="arena-sort-rank"
            @click="toggleSort('rank')"
          >
            <span>{{ t('arena.table.rank') }}</span>
            <span class="arena-sort-dir" aria-hidden="true">{{ sortIndicator('rank') }}</span>
          </button>
          <span role="columnheader">{{ t('arena.table.model') }}</span>
          <span role="columnheader" class="c-bar-head">{{ t('arena.table.distribution') }}</span>
          <button
            type="button"
            role="columnheader"
            class="arena-sort-btn arena-sort-btn--right c-score"
            :class="{ 'is-sorted': sortKey === 'score' }"
            data-testid="arena-sort-score"
            @click="toggleSort('score')"
          >
            <span>{{ t('arena.table.score') }}</span>
            <span class="arena-sort-dir" aria-hidden="true">{{ sortIndicator('score') }}</span>
          </button>
          <button
            type="button"
            role="columnheader"
            class="arena-sort-btn arena-sort-btn--right c-delta"
            :class="{ 'is-sorted': sortKey === 'delta' }"
            data-testid="arena-sort-delta"
            @click="toggleSort('delta')"
          >
            <span>{{ t('arena.table.delta') }}</span>
            <span class="arena-sort-dir" aria-hidden="true">{{ sortIndicator('delta') }}</span>
          </button>
          <button
            type="button"
            role="columnheader"
            class="arena-sort-btn arena-sort-btn--right c-votes"
            :class="{ 'is-sorted': sortKey === 'votes' }"
            data-testid="arena-sort-votes"
            @click="toggleSort('votes')"
          >
            <span>{{ t('arena.table.votes') }}</span>
            <span class="arena-sort-dir" aria-hidden="true">{{ sortIndicator('votes') }}</span>
          </button>
          <span class="c-chevron" aria-hidden="true"></span>
        </div>

        <div
          v-for="entry in visibleEntries"
          :key="`${activeBoard.id}-${entry.model_name}`"
          class="arena-row-group"
          :class="{
            'is-open': selectedModel === entry.model_name,
            'is-highlighted': highlightedModel === entry.model_name
          }"
          :data-model-name="entry.model_name"
          data-arena-anim="row"
        >
          <div
            class="arena-row"
            :style="{ '--org-color': orgMeta(entry).color }"
            role="row"
            tabindex="0"
            data-testid="arena-row"
            :aria-expanded="selectedModel === entry.model_name"
            @click="toggleModelInspector(entry.model_name)"
            @keydown.enter.prevent="toggleModelInspector(entry.model_name)"
            @keydown.space.prevent="toggleModelInspector(entry.model_name)"
          >
            <div class="c-rank" role="cell">
              <span class="arena-rank-num" :class="rankMedalClass(entry.rank)">{{ entry.rank }}</span>
              <em v-if="rankDelta(entry)" :class="rankDelta(entry)!.cls">{{ rankDelta(entry)!.text }}</em>
            </div>

            <div class="c-model" role="cell">
              <ArenaOrgIcon :organization="entry.organization" :model-name="entry.model_name" :size="18" :dark="theme === 'dark'" />
              <div class="c-model-text">
                <div class="arena-model-name">
                  <span>{{ entry.model_name }}</span>
                  <span v-if="entry.is_open" class="arena-tag-open">{{ t('arena.table.openBadge') }}</span>
                </div>
                <div class="arena-model-org">{{ orgMeta(entry).displayName }}</div>
              </div>
            </div>

            <!-- Precision Score + 95% CI Whisker Plot -->
            <div
              class="c-bar"
              aria-hidden="true"
              :title="
                entry.rating_lower > 0 && entry.rating_upper > 0
                  ? `${formatArenaExactScore(entry.score)} (${formatArenaExactScore(entry.rating_lower)} – ${formatArenaExactScore(entry.rating_upper)})`
                  : undefined
              "
            >
              <div class="c-bar-track">
                <span class="c-bar-fill" :style="{ width: `${barPercent(entry.score)}%` }"></span>
                <span
                  v-if="ciGeometry(entry)"
                  class="c-bar-ci"
                  :style="{ left: `${ciGeometry(entry)!.left}%`, width: `${ciGeometry(entry)!.width}%` }"
                ></span>
                <span class="c-bar-dot" :style="{ left: `${barPercent(entry.score)}%` }"></span>
              </div>
            </div>

            <div class="c-score" role="cell">
              <span class="c-score-val">{{ formatArenaScore(entry.score) }}</span>
              <span v-if="entry.ci > 0" class="c-score-ci">±{{ entry.ci }}</span>
            </div>

            <div class="c-delta" role="cell">
              <span v-if="scoreDelta(entry)" :class="scoreDelta(entry)!.cls">{{ scoreDelta(entry)!.text }}</span>
              <span v-else class="c-delta-dash">—</span>
            </div>

            <div class="c-votes" role="cell">{{ formatArenaNumber(entry.votes) }}</div>

            <div class="c-chevron" aria-hidden="true">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 6.2L8 9.7l3.5-3.5" />
              </svg>
            </div>
          </div>

          <!-- Smooth expandable row inspector -->
          <div class="arena-drawer-wrap" :class="{ 'is-open': selectedModel === entry.model_name }">
            <div class="arena-drawer-inner">
              <div v-if="selectedModel === entry.model_name" class="arena-row-inspector-pad">
                <ArenaModelInspector
                  :entry="entry"
                  :active-board="activeBoard"
                  :all-boards="allBoards"
                  :theme="theme"
                  variant="row"
                  @close="selectedModel = null"
                  @filter-org="applyOrgFilter"
                  @jump-track="jumpToModelInTrack"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="canExpand" class="arena-more">
        <button type="button" class="arena-btn arena-btn--ghost" data-testid="arena-toggle-expand-btn" @click="expanded = !expanded">
          {{ expanded ? t('arena.table.collapse') : t('arena.table.showAll', { count: filteredEntries.length }) }}
        </button>
      </div>

      <footer class="arena-footer">
        <h2>{{ t('arena.footer.title') }}</h2>
        <p>{{ t('arena.footer.desc') }}</p>
        <div class="arena-footer-links">
          <a :href="response?.source?.site_url || 'https://arena.ai/leaderboard'" target="_blank" rel="noopener noreferrer">{{ t('arena.footer.arena') }} ↗</a>
          <a :href="response?.source?.dataset_url || 'https://huggingface.co/datasets/lmarena-ai/leaderboard-dataset'" target="_blank" rel="noopener noreferrer">{{ t('arena.footer.dataset') }} ↗</a>
        </div>
      </footer>

      <ArenaPosterModal
        :open="posterOpen"
        :theme="theme"
        :active-board="activeBoard"
        :all-boards="allBoards"
        :last-checked-text="posterSyncedText"
        @close="posterOpen = false"
      />
    </template>

    <Transition name="arena-fab">
      <button
        v-if="showBackToTop && !embedded"
        type="button"
        class="arena-back-top"
        :aria-label="t('arena.table.backToTop')"
        :title="t('arena.table.backToTop')"
        @click="scrollToTop"
      >
        <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 15V5m0 0l-4 4m4-4l4 4" />
        </svg>
      </button>
    </Transition>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ArenaCategoryBoard, ArenaCategoryId, ArenaLeaderboardsResponse, ArenaModelEntry } from '@/api/arena'
import ArenaOrgIcon from './ArenaOrgIcon.vue'
import ArenaModelInspector from './ArenaModelInspector.vue'
import ArenaPosterModal from './ArenaPosterModal.vue'
import { measureIndicator, useArenaMotion } from './useArenaMotion'
import {
  formatArenaExactScore,
  formatArenaNumber,
  formatArenaScore,
  formatScoreDelta,
  resolveArenaOrgMeta,
  type ArenaOrgMeta
} from '@/utils/arenaOrg'

const INITIAL_VISIBLE = 30
type SortKey = 'rank' | 'score' | 'delta' | 'votes'
type SortDir = 'asc' | 'desc'

const props = withDefaults(
  defineProps<{
    response: ArenaLeaderboardsResponse | null
    loading: boolean
    initializing?: boolean
    error?: boolean
    embedded?: boolean
    initialCategory?: ArenaCategoryId
    theme?: 'dark' | 'light'
  }>(),
  { theme: 'dark' }
)

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'update:category', value: ArenaCategoryId): void
}>()

const { t, locale } = useI18n()

const rootRef = ref<HTMLElement | null>(null)
const tabsRef = ref<HTMLElement | null>(null)
const segmentRef = ref<HTMLElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)

const activeCategory = ref<ArenaCategoryId>(props.initialCategory || 'text')
const searchQuery = ref('')
const licenseFilter = ref<'all' | 'proprietary' | 'open'>('all')
const orgFilter = ref<string>('all')
const sortKey = ref<SortKey>('rank')
const sortDir = ref<SortDir>('asc')
const expanded = ref(false)
const posterOpen = ref(false)
const selectedModel = ref<string | null>(null)
const highlightedModel = ref<string | null>(null)
const showBackToTop = ref(false)

const tabIndicator = ref({ left: 0, width: 0, ready: false })
const segmentThumb = ref({ left: 0, width: 0, ready: false })
const animatedPodiumScores = ref<number[]>([])

const { playIntro, playBoardSwitch, tweenNumbers } = useArenaMotion(rootRef)

let hasPlayedIntro = false
let highlightTimer: number | undefined
let resizeObserver: ResizeObserver | null = null

watch(
  () => props.initialCategory,
  (next) => {
    if (next && next !== activeCategory.value) {
      activeCategory.value = next
      orgFilter.value = 'all'
      selectedModel.value = null
    }
  }
)

const allBoards = computed<ArenaCategoryBoard[]>(() => props.response?.categories ?? [])
const activeBoard = computed<ArenaCategoryBoard | null>(() => {
  const boards = allBoards.value
  return boards.find((b) => b.id === activeCategory.value) ?? boards[0] ?? null
})

function selectCategory(id: ArenaCategoryId) {
  if (activeCategory.value === id) return
  activeCategory.value = id
  expanded.value = false
  orgFilter.value = 'all'
  selectedModel.value = null
  emit('update:category', id)
}

const licenseOptions = computed(() => [
  { value: 'all' as const, label: t('arena.filters.all') },
  { value: 'proprietary' as const, label: t('arena.filters.proprietary') },
  { value: 'open' as const, label: t('arena.filters.open') }
])

function orgMeta(entry: ArenaModelEntry): ArenaOrgMeta {
  return resolveArenaOrgMeta(entry.organization, entry.model_name)
}

const topOrgs = computed(() => {
  const entries = activeBoard.value?.entries ?? []
  const counts = new Map<string, { key: string; displayName: string; rawOrg: string; sampleModel: string; count: number }>()
  for (const entry of entries) {
    const meta = orgMeta(entry)
    const existing = counts.get(meta.key)
    if (existing) {
      existing.count += 1
    } else {
      counts.set(meta.key, {
        key: meta.key,
        displayName: meta.displayName,
        rawOrg: entry.organization,
        sampleModel: entry.model_name,
        count: 1
      })
    }
  }
  return Array.from(counts.values())
    .sort((a, b) => b.count - a.count || a.displayName.localeCompare(b.displayName))
    .slice(0, 8)
})

const isDefaultSort = computed(() => sortKey.value === 'rank' && sortDir.value === 'asc')

const hasActiveFilters = computed(
  () =>
    searchQuery.value.trim() !== '' ||
    licenseFilter.value !== 'all' ||
    orgFilter.value !== 'all' ||
    !isDefaultSort.value
)

function clearFilters() {
  searchQuery.value = ''
  licenseFilter.value = 'all'
  orgFilter.value = 'all'
  sortKey.value = 'rank'
  sortDir.value = 'asc'
}

function toggleOrgFilter(orgKey: string) {
  orgFilter.value = orgFilter.value === orgKey ? 'all' : orgKey
}

function applyOrgFilter(orgKey: string) {
  orgFilter.value = orgKey
  selectedModel.value = null
}

function toggleSort(key: SortKey) {
  if (sortKey.value === key) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortDir.value = key === 'rank' ? 'asc' : 'desc'
  }
}

function sortIndicator(key: SortKey): string {
  if (sortKey.value !== key) return ''
  return sortDir.value === 'asc' ? '↑' : '↓'
}

const filteredEntries = computed(() => {
  const entries = activeBoard.value?.entries ?? []
  const q = searchQuery.value.trim().toLowerCase()
  const filtered = entries.filter((e) => {
    if (licenseFilter.value === 'open' && !e.is_open) return false
    if (licenseFilter.value === 'proprietary' && e.is_open) return false
    if (orgFilter.value !== 'all' && orgMeta(e).key !== orgFilter.value) return false
    if (q && !`${e.model_name} ${e.organization} ${orgMeta(e).displayName}`.toLowerCase().includes(q)) return false
    return true
  })

  if (isDefaultSort.value) return filtered

  const factor = sortDir.value === 'asc' ? 1 : -1
  return [...filtered].sort((a, b) => {
    if (sortKey.value === 'rank') return (a.rank - b.rank) * factor
    if (sortKey.value === 'score') return (a.score - b.score) * factor || a.rank - b.rank
    if (sortKey.value === 'votes') return (a.votes - b.votes) * factor || a.rank - b.rank
    const da = a.score_delta ?? Number.NEGATIVE_INFINITY
    const db = b.score_delta ?? Number.NEGATIVE_INFINITY
    if (da === db) return a.rank - b.rank
    if (da === Number.NEGATIVE_INFINITY) return 1
    if (db === Number.NEGATIVE_INFINITY) return -1
    return (da - db) * factor
  })
})

// The first three get their own podium stage on the default view; the list continues from #4.
const showPodium = computed(() => !hasActiveFilters.value && filteredEntries.value.length >= 3)
const podium = computed(() => (showPodium.value ? filteredEntries.value.slice(0, 3) : []))
const listEntries = computed(() => (showPodium.value ? filteredEntries.value.slice(3) : filteredEntries.value))
const canExpand = computed(() => listEntries.value.length > INITIAL_VISIBLE - podium.value.length)
const visibleEntries = computed(() =>
  expanded.value || hasActiveFilters.value
    ? listEntries.value
    : listEntries.value.slice(0, INITIAL_VISIBLE - podium.value.length)
)

const selectedPodiumEntry = computed<ArenaModelEntry | null>(() => {
  if (!selectedModel.value || !showPodium.value) return null
  return podium.value.find((e) => e.model_name === selectedModel.value) ?? null
})

function toggleModelInspector(modelName: string) {
  selectedModel.value = selectedModel.value === modelName ? null : modelName
}

async function locateModelInCurrentBoard(modelName: string) {
  const board = activeBoard.value
  if (!board) return
  const target = board.entries.find((e) => e.model_name.trim().toLowerCase() === modelName.trim().toLowerCase())
  if (!target) return
  clearFilters()
  if (target.rank > INITIAL_VISIBLE) {
    expanded.value = true
  }
  selectedModel.value = target.model_name
  triggerHighlight(target.model_name)
  await nextTick()
  const rowEl = Array.from(rootRef.value?.querySelectorAll<HTMLElement>('.arena-row-group[data-model-name]') ?? []).find(
    (el) => el.dataset.modelName === target.model_name
  )
  if (rowEl && typeof rowEl.scrollIntoView === 'function') {
    rowEl.scrollIntoView({ behavior: 'smooth', block: 'center' })
  } else if (target.rank <= 3) {
    rootRef.value?.querySelector('.arena-podium-shell')?.scrollIntoView?.({ behavior: 'smooth', block: 'center' })
  }
}

async function jumpToModelInTrack(categoryId: ArenaCategoryId, modelName: string) {
  if (activeCategory.value !== categoryId) {
    activeCategory.value = categoryId
    emit('update:category', categoryId)
  }
  await nextTick()
  await locateModelInCurrentBoard(modelName)
}

function triggerHighlight(modelName: string) {
  highlightedModel.value = modelName
  if (highlightTimer) window.clearTimeout(highlightTimer)
  highlightTimer = window.setTimeout(() => {
    highlightedModel.value = null
  }, 1900)
}

// Statistical range across active board for score bar + 95% CI whisker alignment
const scoreBounds = computed(() => {
  const entries = activeBoard.value?.entries ?? []
  if (entries.length === 0) return { min: 0, max: 1 }
  let min = Number.POSITIVE_INFINITY
  let max = Number.NEGATIVE_INFINITY
  for (const e of entries) {
    const lo = e.rating_lower > 0 ? e.rating_lower : e.score - (e.ci || 0)
    const hi = e.rating_upper > 0 ? e.rating_upper : e.score + (e.ci || 0)
    if (lo < min) min = lo
    if (hi > max) max = hi
  }
  if (!Number.isFinite(min) || !Number.isFinite(max) || max <= min) {
    return { min: entries[entries.length - 1].score - 10, max: entries[0].score + 10 }
  }
  return { min, max }
})

function barPercent(score: number): number {
  const { min, max } = scoreBounds.value
  const ratio = (score - min) / Math.max(1, max - min)
  return Number((6 + Math.max(0, Math.min(1, ratio)) * 90).toFixed(2))
}

function ciGeometry(entry: ArenaModelEntry): { left: number; width: number } | null {
  const lo = entry.rating_lower > 0 ? entry.rating_lower : entry.ci > 0 ? entry.score - entry.ci : 0
  const hi = entry.rating_upper > 0 ? entry.rating_upper : entry.ci > 0 ? entry.score + entry.ci : 0
  if (lo <= 0 || hi <= lo) return null
  const left = barPercent(lo)
  const right = barPercent(hi)
  return { left, width: Math.max(1.2, Number((right - left).toFixed(2))) }
}

function podiumGapLabel(entry: ArenaModelEntry, idx: number): string {
  if (idx === 0) {
    const runner = podium.value[1]
    if (!runner) return ''
    const gap = Math.round(entry.score - runner.score)
    if (gap <= 0) return t('arena.podium.tiedLeader')
    return t('arena.podium.leadBy', { gap })
  }
  const leader = podium.value[0]
  if (!leader) return ''
  const gap = Math.round(leader.score - entry.score)
  if (gap <= 0) return t('arena.podium.tiedLeader')
  return t('arena.podium.behindLeader', { gap })
}

function displayPodiumScore(entry: ArenaModelEntry, idx: number): string {
  const animated = animatedPodiumScores.value[idx]
  if (typeof animated === 'number' && Number.isFinite(animated) && animated > 0) {
    return Math.round(animated).toString()
  }
  return formatArenaScore(entry.score)
}

function rankMedalClass(rank: number): string {
  if (rank === 1) return 'is-medal-1'
  if (rank === 2) return 'is-medal-2'
  if (rank === 3) return 'is-medal-3'
  return ''
}

function rankDelta(entry: ArenaModelEntry): { cls: string; text: string } | null {
  if (entry.is_new) return { cls: 'is-new', text: t('arena.table.newBadge') }
  if (entry.rank_delta != null && entry.rank_delta > 0) return { cls: 'is-up', text: `▲${entry.rank_delta}` }
  if (entry.rank_delta != null && entry.rank_delta < 0) return { cls: 'is-down', text: `▼${Math.abs(entry.rank_delta)}` }
  return null
}

function scoreDelta(entry: ArenaModelEntry): { cls: string; text: string } | null {
  const text = formatScoreDelta(entry.score_delta)
  if (!text) return null
  return { cls: (entry.score_delta ?? 0) > 0 ? 'is-up' : 'is-down', text }
}

// Pointer-tracking specular spotlight on podium cards
function handleSpotlightMove(e: PointerEvent) {
  const el = e.currentTarget as HTMLElement | null
  if (!el) return
  const rect = el.getBoundingClientRect()
  el.style.setProperty('--mx', `${e.clientX - rect.left}px`)
  el.style.setProperty('--my', `${e.clientY - rect.top}px`)
  el.style.setProperty('--spot-opacity', '1')
}

function handleSpotlightLeave(e: PointerEvent) {
  const el = e.currentTarget as HTMLElement | null
  if (!el) return
  el.style.setProperty('--spot-opacity', '0')
}

function syncIndicators() {
  tabIndicator.value = measureIndicator(tabsRef.value, '.arena-tab.is-active')
  segmentThumb.value = measureIndicator(segmentRef.value, 'button.is-active')
}

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function handleScroll() {
  showBackToTop.value = typeof window !== 'undefined' && window.scrollY > 560
}

function handleGlobalKeydown(e: KeyboardEvent) {
  if (posterOpen.value) return
  const tag = (e.target as HTMLElement | null)?.tagName?.toLowerCase()
  const isTyping = tag === 'input' || tag === 'textarea' || (e.target as HTMLElement | null)?.isContentEditable

  if (e.key === '/' && !isTyping && !e.metaKey && !e.ctrlKey && !e.altKey) {
    e.preventDefault()
    searchInputRef.value?.focus()
    return
  }

  if (e.key === 'Escape') {
    if (document.activeElement === searchInputRef.value) {
      if (searchQuery.value) {
        searchQuery.value = ''
      } else {
        searchInputRef.value?.blur()
      }
      return
    }
    if (selectedModel.value) {
      selectedModel.value = null
    }
  }
}

// Trigger entrance / track-switch choreography & score rolling
watch(
  () => [activeBoard.value?.id, activeBoard.value?.publish_date] as const,
  async ([boardId], prev) => {
    const prevId = prev?.[0]
    if (!boardId) return
    await nextTick()
    syncIndicators()

    const targetScores = podium.value.map((e) => e.score)
    if (!hasPlayedIntro) {
      hasPlayedIntro = true
      void playIntro()
      void tweenNumbers(targetScores, (vals) => {
        animatedPodiumScores.value = vals
      })
    } else if (prevId && prevId !== boardId) {
      const prevScores = [...animatedPodiumScores.value]
      void playBoardSwitch()
      void tweenNumbers(
        targetScores,
        (vals) => {
          animatedPodiumScores.value = vals
        },
        prevScores
      )
    } else {
      animatedPodiumScores.value = targetScores
    }
  },
  { immediate: true }
)

watch([licenseFilter, locale], async () => {
  await nextTick()
  syncIndicators()
})

onMounted(() => {
  void nextTick().then(syncIndicators)
  window.addEventListener('scroll', handleScroll, { passive: true })
  window.addEventListener('keydown', handleGlobalKeydown)
  window.addEventListener('resize', syncIndicators, { passive: true })

  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(syncIndicators)
    if (tabsRef.value) resizeObserver.observe(tabsRef.value)
    if (segmentRef.value) resizeObserver.observe(segmentRef.value)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('keydown', handleGlobalKeydown)
  window.removeEventListener('resize', syncIndicators)
  resizeObserver?.disconnect()
  if (highlightTimer) window.clearTimeout(highlightTimer)
})

const isZh = computed(() => String(locale.value).toLowerCase().startsWith('zh'))
const pad = (n: number) => String(n).padStart(2, '0')

function formatDay(date?: string): string {
  if (!date) return '—'
  const [y, m, d] = date.split('-').map(Number)
  if (!y || !m || !d) return date
  return isZh.value
    ? `${m}月${d}日`
    : new Date(y, m - 1, d).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

const syncedAt = computed(() => {
  const iso = props.response?.last_checked_at
  const d = iso ? new Date(iso) : null
  return d && !Number.isNaN(d.getTime()) ? d : null
})

const syncedText = computed(() => {
  const d = syncedAt.value
  if (!d) return ''
  const time = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  const now = new Date()
  if (d.toDateString() === now.toDateString()) return time
  return isZh.value ? `${d.getMonth() + 1}月${d.getDate()}日 ${time}` : `${d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} ${time}`
})

const posterSyncedText = computed(() => {
  const d = syncedAt.value
  if (!d) return ''
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
})
</script>

<style scoped>
.arena {
  --a-ease: cubic-bezier(0.16, 1, 0.3, 1);
  --a-bg: #07080d;
  --a-surface: rgba(255, 255, 255, 0.025);
  --a-inspector-bg: rgba(15, 18, 28, 0.78);
  --a-inspector-shadow: 0 20px 44px rgba(0, 0, 0, 0.36);
  --a-text: #f3f5f9;
  --a-text-2: #c7ccd8;
  --a-muted: #8b92a1;
  --a-faint: #5d6473;
  --a-line: rgba(255, 255, 255, 0.08);
  --a-line-strong: rgba(255, 255, 255, 0.16);
  --a-hover: rgba(255, 255, 255, 0.032);
  --a-hover-strong: rgba(255, 255, 255, 0.07);
  --a-field: rgba(255, 255, 255, 0.048);
  --a-chip-bg: rgba(255, 255, 255, 0.032);
  --a-seg-active: rgba(255, 255, 255, 0.11);
  --a-bar: rgba(255, 255, 255, 0.075);
  --a-bar-fill: rgba(168, 186, 224, 0.44);
  --a-ci-line: rgba(243, 245, 249, 0.52);
  --a-dot: #f3f5f9;
  --a-accent: #8fb4ff;
  --a-accent-soft: rgba(91, 140, 255, 0.12);
  --a-accent-border: rgba(143, 180, 255, 0.34);
  --a-up: #3dd68c;
  --a-down: #ff6b70;
  --a-new: #78a9ff;
  --a-open-bg: rgba(61, 214, 140, 0.11);
  --a-gold: #e6c68a;
  --a-gold-soft: rgba(230, 198, 138, 0.11);
  --a-silver: #cfd5e0;
  --a-bronze: #cf9f78;
  --a-btn-bg: #f3f5f9;
  --a-btn-text: #08090f;
  --a-spot: rgba(143, 180, 255, 0.085);
  --a-spot-gold: rgba(230, 198, 138, 0.11);
  color: var(--a-text);
  font-family: Inter, 'Gabugu Arena SC', 'Gabugu Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-variant-numeric: tabular-nums;
  -webkit-font-smoothing: antialiased;
}

.arena[data-theme='light'] {
  --a-bg: transparent;
  --a-surface: rgba(255, 255, 255, 0.72);
  --a-inspector-bg: rgba(255, 255, 255, 0.92);
  --a-inspector-shadow: 0 16px 36px rgba(15, 23, 42, 0.07);
  --a-text: #0b1220;
  --a-text-2: #334155;
  --a-muted: #5f6c80;
  --a-faint: #8e9aaf;
  --a-line: rgba(15, 23, 42, 0.08);
  --a-line-strong: rgba(15, 23, 42, 0.16);
  --a-hover: rgba(15, 23, 42, 0.028);
  --a-hover-strong: rgba(15, 23, 42, 0.06);
  --a-field: rgba(15, 23, 42, 0.045);
  --a-chip-bg: rgba(255, 255, 255, 0.76);
  --a-seg-active: #ffffff;
  --a-bar: rgba(15, 23, 42, 0.07);
  --a-bar-fill: rgba(71, 85, 105, 0.34);
  --a-ci-line: rgba(15, 23, 42, 0.48);
  --a-dot: #0f172a;
  --a-accent: #2563eb;
  --a-accent-soft: rgba(37, 99, 235, 0.08);
  --a-accent-border: rgba(37, 99, 235, 0.28);
  --a-up: #059669;
  --a-down: #e11d48;
  --a-new: #2563eb;
  --a-open-bg: rgba(5, 150, 105, 0.09);
  --a-gold: #a67222;
  --a-gold-soft: rgba(166, 114, 34, 0.085);
  --a-silver: #5b6678;
  --a-bronze: #995c32;
  --a-btn-bg: #0f172a;
  --a-btn-text: #ffffff;
  --a-spot: rgba(37, 99, 235, 0.055);
  --a-spot-gold: rgba(166, 114, 34, 0.07);
}

/* Hero */
.arena-hero {
  padding: 68px 0 42px;
}
.arena--embedded .arena-hero {
  padding: 4px 0 28px;
}
.arena-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  color: var(--a-muted);
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.06em;
}
.arena-eyebrow-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--a-accent);
  box-shadow: 0 0 0 3px var(--a-accent-soft);
}
.arena-hero-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 32px;
}
.arena-title {
  margin: 14px 0 0;
  font-size: clamp(42px, 4.8vw, 58px);
  line-height: 1.06;
  font-weight: 600;
  letter-spacing: -0.03em;
  background: linear-gradient(180deg, #ffffff 15%, #c8d2e5 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}
.arena[data-theme='light'] .arena-title {
  background: linear-gradient(180deg, #0b1220 20%, #334155 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}
.arena-subtitle {
  margin: 14px 0 0;
  max-width: 580px;
  color: var(--a-muted);
  font-size: 17px;
  line-height: 1.72;
  text-wrap: pretty;
}
.arena--embedded .arena-subtitle {
  margin-top: 8px;
  font-size: 15px;
}
.arena-hero-actions {
  display: flex;
  flex-shrink: 0;
  gap: 10px;
}

.arena-btn {
  position: relative;
  display: inline-flex;
  height: 40px;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 0 18px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: none;
  color: inherit;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  overflow: hidden;
  transition: background-color 0.18s ease, border-color 0.18s ease, color 0.18s ease, transform 0.14s var(--a-ease), box-shadow 0.2s ease, opacity 0.16s ease;
}
.arena-btn:active:not(:disabled) {
  transform: scale(0.97);
}
.arena-btn:disabled {
  opacity: 0.55;
  cursor: default;
}
.arena-btn-icon {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
}
.arena-btn.is-spinning .arena-refresh-icon {
  animation: arena-spin 0.85s linear infinite;
}
@keyframes arena-spin {
  to {
    transform: rotate(360deg);
  }
}
.arena-btn--primary {
  background: var(--a-btn-bg);
  color: var(--a-btn-text);
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.25) inset, 0 8px 22px rgba(0, 0, 0, 0.22);
}
.arena[data-theme='light'] .arena-btn--primary {
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.18) inset, 0 8px 20px rgba(15, 23, 42, 0.16);
}
.arena-btn--primary::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(110deg, transparent 30%, rgba(255, 255, 255, 0.28) 50%, transparent 70%);
  transform: translateX(-130%);
  transition: transform 0.65s var(--a-ease);
  pointer-events: none;
}
.arena-btn--primary:hover:not(:disabled) {
  opacity: 0.94;
}
.arena-btn--primary:hover:not(:disabled)::after {
  transform: translateX(130%);
}
.arena-btn--ghost {
  border-color: var(--a-line-strong);
  color: var(--a-text-2);
  background: var(--a-surface);
}
.arena-btn--ghost:hover:not(:disabled) {
  background: var(--a-hover-strong);
  border-color: var(--a-line-strong);
  color: var(--a-text);
}

/* Tabs with sliding indicator */
.arena-tabs {
  position: relative;
  display: flex;
  gap: 30px;
  border-bottom: 1px solid var(--a-line);
  overflow-x: auto;
  scrollbar-width: none;
}
.arena-tabs::-webkit-scrollbar {
  display: none;
}
.arena-tab {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  flex-shrink: 0;
  padding: 0 2px 15px;
  border: 0;
  background: none;
  color: var(--a-muted);
  font: inherit;
  font-size: 15.5px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.18s ease;
}
.arena-tab:hover {
  color: var(--a-text-2);
}
.arena-tab.is-active {
  color: var(--a-text);
}
.arena-tabs:not(.has-indicator) .arena-tab.is-active::after {
  content: '';
  position: absolute;
  right: 0;
  bottom: -1px;
  left: 0;
  height: 2px;
  border-radius: 2px;
  background: var(--a-text);
}
.arena-tab-icon {
  width: 15px;
  height: 15px;
  opacity: 0.65;
  transition: opacity 0.18s ease, color 0.18s ease;
}
.arena-tab.is-active .arena-tab-icon {
  opacity: 1;
  color: var(--a-accent);
}
.arena-tab-count {
  margin-left: 1px;
  color: var(--a-faint);
  font-size: 13px;
  font-weight: 400;
  transition: color 0.18s ease;
}
.arena-tab.is-active .arena-tab-count {
  color: var(--a-muted);
}
.arena-tabs-indicator {
  position: absolute;
  bottom: 0;
  left: 0;
  height: 2px;
  border-radius: 2px;
  background: var(--a-text);
  box-shadow: 0 0 12px color-mix(in srgb, var(--a-accent) 60%, transparent);
  transition: transform 0.36s var(--a-ease), width 0.36s var(--a-ease);
  pointer-events: none;
}

.arena-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px 16px;
  margin-top: 16px;
  color: var(--a-muted);
  font-size: 13px;
}
.arena-meta-desc {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.arena-meta-pill {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 8px;
  border: 1px solid var(--a-line);
  border-radius: 999px;
  background: var(--a-surface);
  color: var(--a-text-2);
  font-size: 13px;
}
.arena-meta-live {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
}
.arena-live-dot {
  position: relative;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--a-up);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--a-up) 18%, transparent);
}
.arena-live-dot::after {
  content: '';
  position: absolute;
  inset: -3px;
  border-radius: 50%;
  border: 1px solid var(--a-up);
  opacity: 0;
  animation: arena-live-ping 2.6s var(--a-ease) infinite;
}
@keyframes arena-live-ping {
  0% {
    transform: scale(0.8);
    opacity: 0.65;
  }
  70%,
  100% {
    transform: scale(2.1);
    opacity: 0;
  }
}

/* Architectural Top-3 Podium */
.arena-podium-shell {
  margin-top: 26px;
}
.arena-podium {
  position: relative;
  display: grid;
  grid-template-columns: 1.18fr 1fr 1fr;
  border: 1px solid var(--a-line);
  border-radius: 16px;
  background: var(--a-surface);
  overflow: hidden;
}
.arena-podium::before {
  content: '';
  position: absolute;
  top: 0;
  left: 8%;
  right: 8%;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.24), transparent);
  pointer-events: none;
  z-index: 2;
}
.arena[data-theme='light'] .arena-podium {
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 12px 32px rgba(15, 23, 42, 0.035);
}
.arena[data-theme='light'] .arena-podium::before {
  background: linear-gradient(90deg, transparent, rgba(37, 99, 235, 0.22), transparent);
}

.arena-podium-item {
  --mx: 50%;
  --my: 0%;
  --spot-opacity: 0;
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 28px 28px 24px;
  cursor: pointer;
  outline: none;
  transition: background-color 0.2s ease;
}
.arena-podium-item + .arena-podium-item {
  border-left: 1px solid var(--a-line);
}
.arena-podium-item::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(320px circle at var(--mx) var(--my), var(--a-spot), transparent 72%);
  opacity: var(--spot-opacity);
  transition: opacity 0.28s ease;
  pointer-events: none;
}
.arena-podium-item.is-rank-1::before {
  background: radial-gradient(360px circle at var(--mx) var(--my), var(--a-spot-gold), transparent 72%);
}
.arena-podium-item.is-rank-1 {
  background: linear-gradient(155deg, color-mix(in srgb, var(--a-gold) 7%, transparent) 0%, transparent 56%);
}
.arena-podium-item.is-rank-1::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, var(--a-gold), color-mix(in srgb, var(--a-gold) 25%, transparent) 78%, transparent);
  opacity: 0.85;
  pointer-events: none;
}
.arena-podium-item:hover {
  background-color: var(--a-hover);
}
.arena-podium-item:focus-visible {
  box-shadow: inset 0 0 0 1.5px var(--a-accent);
}
.arena-podium-item.is-selected {
  background-color: var(--a-accent-soft);
}
.arena-podium-item.is-rank-1.is-selected {
  background-color: var(--a-gold-soft);
}

.arena-podium-head {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.arena-podium-rank {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.14em;
}
.is-rank-1 .arena-podium-num {
  color: var(--a-gold);
}
.is-rank-2 .arena-podium-num {
  color: var(--a-silver);
}
.is-rank-3 .arena-podium-num {
  color: var(--a-bronze);
}
.arena-podium-rank em {
  font-size: 13px;
  letter-spacing: 0;
}
.arena-podium-gap {
  font-size: 13px;
  font-weight: 500;
  color: var(--a-muted);
  white-space: nowrap;
}
.arena-podium-gap.is-leader {
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--a-gold-soft);
  color: var(--a-gold);
}

.arena-podium-name {
  position: relative;
  margin: 16px 0 0;
  font-size: 22px;
  line-height: 1.28;
  font-weight: 600;
  letter-spacing: -0.02em;
  overflow-wrap: anywhere;
}
.is-rank-1 .arena-podium-name {
  font-size: 25px;
}
.arena-podium-org {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 9px;
  color: var(--a-muted);
  font-size: 13px;
}

.arena-podium-bottom {
  position: relative;
  margin-top: auto;
  padding-top: 26px;
}
.arena-podium-score {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 13px;
}
.arena-podium-score strong {
  font-size: 38px;
  line-height: 1;
  font-weight: 600;
  letter-spacing: -0.025em;
}
.is-rank-1 .arena-podium-score strong {
  font-size: 46px;
}
.arena-ci {
  color: var(--a-faint);
}

.arena-podium-foot {
  margin-top: 14px;
}
.arena-podium-gauge {
  position: relative;
  height: 3px;
  border-radius: 999px;
  background: var(--a-bar);
  overflow: visible;
}
.arena-podium-gauge-fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: color-mix(in srgb, var(--org-color, var(--a-accent)) 55%, var(--a-bar-fill));
  transition: width 0.55s var(--a-ease);
}
.is-rank-1 .arena-podium-gauge-fill {
  background: linear-gradient(90deg, color-mix(in srgb, var(--a-gold) 45%, transparent), var(--a-gold));
}
.arena-podium-gauge-ci {
  position: absolute;
  top: -1.5px;
  height: 6px;
  border-radius: 3px;
  border-left: 1.5px solid var(--a-ci-line);
  border-right: 1.5px solid var(--a-ci-line);
  background: color-mix(in srgb, var(--a-text) 14%, transparent);
  transition: left 0.55s var(--a-ease), width 0.55s var(--a-ease);
}
.arena-podium-meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 10px;
  color: var(--a-faint);
  font-size: 13px;
}
.arena-podium-inspect-cue {
  color: var(--a-muted);
  opacity: 0;
  transform: translateX(-4px);
  transition: opacity 0.2s ease, transform 0.2s var(--a-ease), color 0.16s ease;
}
.arena-podium-item:hover .arena-podium-inspect-cue,
.arena-podium-item.is-selected .arena-podium-inspect-cue,
.arena-podium-item:focus-visible .arena-podium-inspect-cue {
  opacity: 1;
  transform: translateX(0);
}
.arena-podium-item.is-selected .arena-podium-inspect-cue {
  color: var(--a-accent);
}

/* Smooth CSS Grid accordion drawer */
.arena-drawer-wrap {
  display: grid;
  grid-template-rows: 0fr;
  opacity: 0;
  transition: grid-template-rows 0.34s var(--a-ease), opacity 0.24s ease;
}
.arena-drawer-wrap.is-open {
  grid-template-rows: 1fr;
  opacity: 1;
}
.arena-drawer-inner {
  overflow: hidden;
  min-height: 0;
}

/* Insights */
.arena-insights {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin: 18px 0 0;
  border-top: 1px solid var(--a-line);
  border-bottom: 1px solid var(--a-line);
}
.arena-insights > div {
  min-width: 0;
  padding: 18px 28px;
  transition: background-color 0.18s ease;
}
.arena-insights > div:first-child {
  padding-left: 4px;
}
.arena-insights > div + div {
  border-left: 1px solid var(--a-line);
}
.arena-insights > div.is-clickable {
  cursor: pointer;
}
.arena-insights > div.is-clickable:hover {
  background: var(--a-hover);
}
.arena-insights dt {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--a-muted);
  font-size: 13px;
}
.arena-insight-arrow {
  width: 13px;
  height: 13px;
  color: var(--a-faint);
  opacity: 0.4;
  transition: opacity 0.18s ease, transform 0.18s var(--a-ease), color 0.18s ease;
}
.arena-insights > div.is-clickable:hover .arena-insight-arrow {
  opacity: 1;
  color: var(--a-accent);
  transform: translate(1.5px, -1.5px);
}
.arena-insights dd {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin: 6px 0 0;
  font-size: 15px;
  font-weight: 500;
  white-space: nowrap;
}
.arena-insights dd.is-muted {
  color: var(--a-faint);
  font-weight: 400;
}
.arena-insights small {
  color: var(--a-muted);
  font-size: 13px;
  font-weight: 400;
}
.arena-insights em {
  font-size: 13px;
}
.arena-insight-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Deltas & tags */
em.is-up,
span.is-up {
  color: var(--a-up);
  font-style: normal;
  font-weight: 600;
}
em.is-down,
span.is-down {
  color: var(--a-down);
  font-style: normal;
  font-weight: 600;
}
em.is-new {
  color: var(--a-new);
  font-style: normal;
  font-size: 13px !important;
  font-weight: 700;
  letter-spacing: 0.04em;
}
.arena-tag-open {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  height: 18px;
  padding: 0 6px;
  border-radius: 4px;
  background: var(--a-open-bg);
  color: var(--a-up);
  font-size: 13px;
  font-weight: 500;
}

/* Controls & Toolbar */
.arena-controls {
  margin-top: 32px;
}
.arena-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.arena-search {
  display: flex;
  flex: 0 1 320px;
  align-items: center;
  gap: 10px;
  height: 40px;
  padding: 0 12px 0 14px;
  border: 1px solid var(--a-line);
  border-radius: 10px;
  background: var(--a-field);
  color: var(--a-faint);
  transition: border-color 0.18s ease, box-shadow 0.18s ease, background-color 0.18s ease;
}
.arena-search:focus-within {
  border-color: var(--a-accent-border);
  box-shadow: 0 0 0 3px var(--a-accent-soft);
}
.arena-search svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.arena-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: none;
  color: var(--a-text);
  font: inherit;
  font-size: 14px;
}
.arena-search input::placeholder {
  color: var(--a-faint);
}
.arena-search input::-webkit-search-cancel-button {
  display: none;
}
.arena-kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  border: 1px solid var(--a-line-strong);
  border-radius: 5px;
  color: var(--a-faint);
  font-family: inherit;
  font-size: 13px;
  line-height: 1;
}
.arena-search-clear {
  display: inline-flex;
  width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: var(--a-hover-strong);
  color: var(--a-muted);
  cursor: pointer;
}
.arena-search-clear:hover {
  color: var(--a-text);
}
.arena-search-clear svg {
  width: 12px;
  height: 12px;
}

/* Sliding segmented control */
.arena-segment {
  position: relative;
  display: inline-flex;
  padding: 3px;
  border: 1px solid var(--a-line);
  border-radius: 10px;
  background: var(--a-field);
}
.arena-segment-thumb {
  position: absolute;
  top: 3px;
  bottom: 3px;
  left: 0;
  border-radius: 7px;
  background: var(--a-seg-active);
  transition: transform 0.28s var(--a-ease), width 0.28s var(--a-ease);
  pointer-events: none;
}
.arena[data-theme='light'] .arena-segment-thumb {
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}
.arena-segment button {
  position: relative;
  z-index: 1;
  height: 34px;
  padding: 0 14px;
  border: 0;
  border-radius: 7px;
  background: none;
  color: var(--a-muted);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.18s ease;
}
.arena-segment button:hover {
  color: var(--a-text-2);
}
.arena-segment button.is-active {
  color: var(--a-text);
}
.arena-segment:not(.has-thumb) button.is-active {
  background: var(--a-seg-active);
}

.arena-results {
  margin-left: auto;
  color: var(--a-muted);
  font-size: 13px;
}
.arena-results button {
  margin-left: 8px;
  border: 0;
  background: none;
  color: var(--a-accent);
  font: inherit;
  text-decoration: underline;
  text-underline-offset: 3px;
  cursor: pointer;
}

/* Compact Lab Quick-Filter Strip */
.arena-org-strip {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
  padding-bottom: 2px;
  overflow-x: auto;
  scrollbar-width: none;
}
.arena-org-strip::-webkit-scrollbar {
  display: none;
}
.arena-org-chip {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 11px;
  border: 1px solid var(--a-line);
  border-radius: 999px;
  background: var(--a-chip-bg);
  color: var(--a-muted);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: border-color 0.16s ease, background-color 0.16s ease, color 0.16s ease, transform 0.12s ease;
}
.arena-org-chip:hover {
  border-color: var(--a-line-strong);
  background: var(--a-hover-strong);
  color: var(--a-text-2);
}
.arena-org-chip:active {
  transform: scale(0.97);
}
.arena-org-chip.is-active {
  border-color: var(--a-accent-border);
  background: var(--a-accent-soft);
  color: var(--a-text);
}
.arena-org-chip small {
  color: var(--a-faint);
  font-size: 13px;
  font-weight: 400;
}
.arena-org-chip.is-active small {
  color: var(--a-accent);
}

/* Ranking List */
.arena-list {
  margin-top: 14px;
}
.arena-row-group {
  border-bottom: 1px solid var(--a-line);
  transition: background-color 0.25s ease;
}
.arena-row-group.is-highlighted {
  animation: arena-row-flash 1.8s var(--a-ease);
}
@keyframes arena-row-flash {
  0%,
  25% {
    background-color: var(--a-accent-soft);
  }
  100% {
    background-color: transparent;
  }
}

.arena-row {
  position: relative;
  display: grid;
  grid-template-columns: 68px minmax(0, 1fr) 204px 86px 64px 86px 24px;
  column-gap: 16px;
  align-items: center;
  min-height: 62px;
  padding: 0 10px;
  transition: background-color 0.16s ease;
}
.arena-row:not(.arena-row--head) {
  cursor: pointer;
  outline: none;
}
.arena-row:not(.arena-row--head)::before {
  content: '';
  position: absolute;
  top: 12px;
  bottom: 12px;
  left: 0;
  width: 2px;
  border-radius: 2px;
  background: var(--org-color, var(--a-accent));
  opacity: 0;
  transform: scaleY(0.5);
  transition: opacity 0.18s ease, transform 0.22s var(--a-ease);
}
.arena-row:not(.arena-row--head):hover,
.arena-row:not(.arena-row--head):focus-visible {
  background: var(--a-hover);
}
.arena-row:not(.arena-row--head):hover::before,
.arena-row-group.is-open .arena-row::before {
  opacity: 0.9;
  transform: scaleY(1);
}
.arena-row-group.is-open .arena-row {
  background: var(--a-hover);
}

.arena-row--head {
  min-height: 40px;
  border-bottom: 1px solid var(--a-line);
  color: var(--a-faint);
  font-size: 13px;
  font-weight: 500;
}
.c-bar-head {
  color: var(--a-faint);
}
.arena-sort-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 0;
  border: 0;
  background: none;
  color: var(--a-faint);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.15s ease;
}
.arena-sort-btn:hover,
.arena-sort-btn.is-sorted {
  color: var(--a-text);
}
.arena-sort-btn--right {
  justify-content: flex-end;
  width: 100%;
}
.arena-sort-dir {
  color: var(--a-accent);
  font-size: 13px;
}

.c-rank {
  display: flex;
  align-items: baseline;
  gap: 7px;
}
.c-rank em {
  font-size: 13px;
}
.arena-rank-num {
  min-width: 24px;
  color: var(--a-text-2);
  font-size: 15px;
  font-weight: 500;
}
.arena-rank-num.is-medal-1 {
  color: var(--a-gold);
  font-weight: 600;
}
.arena-rank-num.is-medal-2 {
  color: var(--a-silver);
  font-weight: 600;
}
.arena-rank-num.is-medal-3 {
  color: var(--a-bronze);
  font-weight: 600;
}

.c-model {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.c-model-text {
  min-width: 0;
}
.arena-model-name {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: 15px;
  font-weight: 500;
}
.arena-model-name > span:first-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.arena-model-org {
  margin-top: 2px;
  color: var(--a-muted);
  font-size: 13px;
}

/* Precision Score + 95% CI Whisker Track */
.c-bar {
  display: flex;
  align-items: center;
  height: 20px;
}
.c-bar-track {
  position: relative;
  width: 100%;
  height: 4px;
  border-radius: 999px;
  background: var(--a-bar);
}
.c-bar-fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: color-mix(in srgb, var(--org-color, var(--a-accent)) 36%, var(--a-bar-fill));
  transition: width 0.48s var(--a-ease);
}
.c-bar-ci {
  position: absolute;
  top: -3px;
  height: 10px;
  border-left: 1.5px solid var(--a-ci-line);
  border-right: 1.5px solid var(--a-ci-line);
  background: color-mix(in srgb, var(--a-text) 14%, transparent);
  border-radius: 2px;
  transition: left 0.48s var(--a-ease), width 0.48s var(--a-ease);
}
.c-bar-dot {
  position: absolute;
  top: 50%;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--a-dot);
  transform: translate(-50%, -50%);
  box-shadow: 0 0 0 2px var(--a-bg);
  transition: left 0.48s var(--a-ease), transform 0.16s ease;
}
.arena-row:hover .c-bar-dot {
  transform: translate(-50%, -50%) scale(1.25);
}

.c-score {
  display: flex;
  align-items: baseline;
  justify-content: flex-end;
  gap: 5px;
  text-align: right;
}
.c-score-val {
  font-size: 16px;
  font-weight: 600;
}
.c-score-ci {
  color: var(--a-faint);
  font-size: 13px;
  font-weight: 400;
}
.c-delta {
  text-align: right;
  font-size: 13px;
}
.c-delta-dash {
  color: var(--a-faint);
  opacity: 0.45;
}
.c-votes {
  text-align: right;
  color: var(--a-muted);
  font-size: 13px;
}
.c-chevron {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--a-faint);
  transition: transform 0.26s var(--a-ease), color 0.16s ease;
}
.c-chevron svg {
  width: 15px;
  height: 15px;
}
.arena-row:hover .c-chevron {
  color: var(--a-text-2);
}
.arena-row-group.is-open .c-chevron {
  transform: rotate(180deg);
  color: var(--a-accent);
}
.arena-row-inspector-pad {
  padding: 2px 10px 16px;
}

.arena-empty {
  padding: 64px 0;
  color: var(--a-muted);
  font-size: 14px;
  text-align: center;
}
.arena-more {
  display: flex;
  justify-content: center;
  margin-top: 28px;
}

/* Footer */
.arena-footer {
  margin-top: 80px;
  padding: 28px 0 76px;
  border-top: 1px solid var(--a-line);
}
.arena--embedded .arena-footer {
  margin-top: 48px;
  padding-bottom: 24px;
}
.arena-footer h2 {
  margin: 0;
  color: var(--a-text-2);
  font-size: 13px;
  font-weight: 500;
}
.arena-footer p {
  margin: 8px 0 0;
  max-width: 720px;
  color: var(--a-muted);
  font-size: 13px;
  line-height: 1.8;
}
.arena-footer-links {
  display: flex;
  gap: 20px;
  margin-top: 12px;
  font-size: 13px;
}
.arena-footer-links a {
  color: var(--a-text-2);
  text-decoration: none;
  transition: color 0.15s ease;
}
.arena-footer-links a:hover {
  color: var(--a-text);
}

/* Floating back-to-top */
.arena-back-top {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 25;
  display: inline-flex;
  width: 42px;
  height: 42px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--a-line-strong);
  border-radius: 999px;
  background: color-mix(in srgb, var(--a-bg) 82%, transparent);
  color: var(--a-text-2);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  box-shadow: 0 10px 28px rgba(0, 0, 0, 0.25);
  cursor: pointer;
  transition: transform 0.18s var(--a-ease), color 0.16s ease, border-color 0.16s ease;
}
.arena[data-theme='light'] .arena-back-top {
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.1);
}
.arena-back-top:hover {
  color: var(--a-text);
  transform: translateY(-2px);
}
.arena-back-top svg {
  width: 18px;
  height: 18px;
}
.arena-fab-enter-active,
.arena-fab-leave-active {
  transition: opacity 0.22s ease, transform 0.22s var(--a-ease);
}
.arena-fab-enter-from,
.arena-fab-leave-to {
  opacity: 0;
  transform: translateY(10px) scale(0.9);
}

/* States & Skeleton */
.arena-state {
  padding: 96px 0;
  text-align: center;
}
.arena-state h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}
.arena-state p {
  margin: 8px 0 24px;
  color: var(--a-muted);
  font-size: 14px;
}
.arena-skeleton span,
.arena-skeleton-row {
  display: block;
  border-radius: 10px;
  background: var(--a-field);
  animation: arena-pulse 1.4s ease-in-out infinite;
}
.arena-skeleton-tabs {
  display: flex;
  gap: 32px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--a-line);
}
.arena-skeleton-tabs span {
  width: 78px;
  height: 20px;
}
.arena-skeleton-podium {
  display: grid;
  grid-template-columns: 1.18fr 1fr 1fr;
  gap: 16px;
  margin: 26px 0;
}
.arena-skeleton-podium span {
  height: 210px;
}
.arena-skeleton-row {
  height: 46px;
  margin-top: 12px;
}
@keyframes arena-pulse {
  50% {
    opacity: 0.45;
  }
}

@media (max-width: 1024px) {
  .arena-row {
    grid-template-columns: 60px minmax(0, 1fr) 82px 58px 80px 22px;
  }
  .arena-row .c-bar,
  .arena-row .c-bar-head {
    display: none;
  }
  .arena-podium {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .arena-podium-item {
    padding: 24px 20px 20px;
  }
  .arena-podium-name,
  .is-rank-1 .arena-podium-name {
    font-size: 20px;
  }
  .arena-podium-score strong,
  .is-rank-1 .arena-podium-score strong {
    font-size: 34px;
  }
  .arena-insights > div {
    padding: 16px 20px;
  }
}

@media (max-width: 640px) {
  .arena-hero {
    padding: 36px 0 26px;
  }
  .arena-title {
    margin-top: 10px;
    font-size: 36px;
  }
  .arena-subtitle {
    font-size: 15px;
  }
  .arena-hero-row {
    flex-direction: column;
    align-items: stretch;
    gap: 18px;
  }
  .arena-hero-actions .arena-btn {
    flex: 1;
  }
  .arena-tabs {
    gap: 20px;
  }
  .arena-tab {
    font-size: 14.5px;
  }
  .arena-tab-icon,
  .arena-tab-count {
    display: none;
  }
  .arena-meta {
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
  }

  .arena-podium-shell {
    margin-top: 20px;
  }
  .arena-podium {
    grid-template-columns: 1fr;
    border-radius: 14px;
  }
  .arena-podium-item {
    display: grid;
    grid-template-columns: 36px minmax(0, 1fr) auto;
    grid-template-areas:
      'rank name score'
      'rank org score';
    column-gap: 12px;
    align-items: center;
    padding: 16px;
  }
  .arena-podium-item + .arena-podium-item {
    border-left: 0;
    border-top: 1px solid var(--a-line);
  }
  .arena-podium-head {
    grid-area: rank;
    flex-direction: column;
    align-items: flex-start;
  }
  .arena-podium-rank {
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    font-size: 15px;
    letter-spacing: 0.04em;
  }
  .arena-podium-gap {
    display: none;
  }
  .arena-podium-name,
  .is-rank-1 .arena-podium-name {
    grid-area: name;
    margin: 0;
    font-size: 16px;
    line-height: 1.35;
  }
  .arena-podium-org {
    grid-area: org;
    margin-top: 4px;
    font-size: 13px;
  }
  .arena-podium-bottom {
    grid-area: score;
    margin: 0;
    padding: 0;
  }
  .arena-podium-score {
    flex-direction: column;
    align-items: flex-end;
    gap: 2px;
    font-size: 13px;
  }
  .arena-podium-score strong,
  .is-rank-1 .arena-podium-score strong {
    font-size: 24px;
  }
  .arena-podium-score .arena-ci,
  .arena-podium-foot {
    display: none;
  }

  .arena-insights {
    grid-template-columns: 1fr;
  }
  .arena-insights > div {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 16px;
    padding: 12px 2px;
  }
  .arena-insights > div + div {
    border-left: 0;
    border-top: 1px solid var(--a-line);
  }
  .arena-insights dd {
    margin: 0;
    font-size: 14px;
  }

  .arena-controls {
    margin-top: 24px;
  }
  .arena-search {
    flex: 1 1 100%;
  }
  .arena-search input {
    font-size: 16px;
  }
  .arena-kbd {
    display: none;
  }
  .arena-results {
    margin-left: 0;
  }

  .arena-row {
    grid-template-columns: 36px minmax(0, 1fr) auto 18px;
    column-gap: 10px;
    padding: 10px 4px;
    min-height: 58px;
  }
  .arena-row--head {
    display: none;
  }
  .arena-row .c-delta,
  .arena-row .c-votes,
  .c-score-ci {
    display: none;
  }
  .c-rank {
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
  }
  .arena-model-name {
    font-size: 14px;
  }
  .arena-model-org {
    font-size: 13px;
  }
  .c-score-val {
    font-size: 15px;
  }
  .arena-row-inspector-pad {
    padding: 2px 0 14px;
  }
  .arena-footer {
    margin-top: 56px;
    padding-bottom: 48px;
  }
  .arena-back-top {
    right: 16px;
    bottom: 16px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .arena-live-dot::after,
  .arena-btn.is-spinning .arena-refresh-icon,
  .arena-row-group.is-highlighted {
    animation: none;
  }
  .arena-tabs-indicator,
  .arena-segment-thumb,
  .arena-drawer-wrap,
  .c-bar-fill,
  .c-bar-ci,
  .c-bar-dot,
  .arena-podium-gauge-fill,
  .arena-podium-gauge-ci,
  .arena-btn::after {
    transition: none;
  }
}
</style>
