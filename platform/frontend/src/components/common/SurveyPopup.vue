<template>
  <Teleport to="body">
    <!-- 最小化后的 Dynamic-Island 风格深曜石悬浮胶囊（带实时环形进度） -->
    <Transition name="survey-pop">
      <div
        v-if="shouldShowMinimizedPill"
        class="fixed bottom-5 right-5 z-[110]"
      >
        <button
          type="button"
          data-testid="survey-minimized-pill"
          class="survey-dock-pill group flex items-center gap-3 rounded-full border border-white/15 bg-[#0b1220]/95 py-2.5 pl-3 pr-4 text-xs font-medium text-white backdrop-blur-xl transition-all duration-200 hover:-translate-y-0.5 hover:border-primary-400/40 hover:bg-[#0f172a] active:scale-[0.98] dark:border-primary-400/25 dark:bg-[#0f172a]/95"
          @click="restoreFromMinimized"
        >
          <!-- 实时环形进度 SVG -->
          <span class="relative flex h-6 w-6 flex-shrink-0 items-center justify-center">
            <svg class="h-6 w-6 -rotate-90" viewBox="0 0 24 24" aria-hidden="true">
              <circle
                cx="12"
                cy="12"
                r="9.5"
                fill="none"
                stroke="rgba(255,255,255,0.14)"
                stroke-width="2"
              />
              <circle
                cx="12"
                cy="12"
                r="9.5"
                fill="none"
                stroke="url(#surveyDockRingGrad)"
                stroke-width="2.2"
                stroke-linecap="round"
                :stroke-dasharray="59.69"
                :stroke-dashoffset="dockRingDashOffset"
                class="transition-all duration-300"
              />
              <defs>
                <linearGradient id="surveyDockRingGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stop-color="#60a5fa" />
                  <stop offset="100%" stop-color="#34d399" />
                </linearGradient>
              </defs>
            </svg>
            <span class="absolute h-1.5 w-1.5 rounded-full bg-sky-300 shadow-[0_0_8px_#38bdf8]"></span>
          </span>

          <span class="font-semibold tracking-tight text-white/95">
            {{
              activeSurvey?.display_mode === 'micro'
                ? t('surveys.minimizedMicroPill')
                : t('surveys.minimizedPill', {
                    current: Math.min(stepIndex + 1, totalSteps),
                    total: totalSteps
                  })
            }}
          </span>

          <span
            v-if="activeSurvey?.has_reward && activeSurvey.reward_value > 0"
            class="inline-flex items-center gap-1 rounded-full bg-emerald-400/15 px-2 py-0.5 font-mono text-xs font-bold tabular-nums text-emerald-300 ring-1 ring-inset ring-emerald-400/30"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-400"></span>
            +${{ formatRewardAmount(activeSurvey.reward_value) }}
          </span>
        </button>
      </div>
    </Transition>

    <!-- 弹窗主体 -->
    <Transition name="survey-pop">
      <div
        v-if="shouldRenderPopup && activeSurvey"
        :class="wrapperClasses"
        role="dialog"
        aria-modal="false"
        :aria-label="activeSurvey.title"
        data-testid="survey-popup-root"
        @click.self="handleBackdropClick"
      >
        <!-- 双层精工外框（外层金属折光高光边 + 内层光学载体） -->
        <div :class="bezelClasses" @click.stop>
          <!-- 1 题极速快问（micro） -->
          <div
            v-if="activeSurvey.display_mode === 'micro' && activeSurvey.questions.length === 1"
            :class="panelClasses"
            data-testid="survey-micro-card"
            @pointermove="onShellPointerMove"
            @pointerleave="onShellPointerLeave"
          >
            <!-- 光晕层 + 细点阵建筑纹理 + 随光标追踪聚光 -->
            <div class="survey-aurora pointer-events-none absolute inset-x-0 top-0 h-32" aria-hidden="true"></div>
            <div class="survey-dot-grid pointer-events-none absolute inset-x-0 top-0 h-28" aria-hidden="true"></div>
            <div class="survey-spotlight pointer-events-none absolute inset-0" aria-hidden="true"></div>

            <!-- 手机端拖拽把手 -->
            <div class="relative flex justify-center pt-2.5 sm:hidden">
              <span class="h-1 w-10 rounded-full bg-slate-300/80 dark:bg-slate-600/80"></span>
            </div>

            <div class="relative space-y-4 p-5 sm:p-6">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="inline-flex items-center gap-1.5 rounded-full bg-slate-900/[0.04] py-1 pl-1.5 pr-2.5 text-xs font-semibold tracking-wide text-slate-700 ring-1 ring-inset ring-slate-900/[0.08] dark:bg-white/[0.06] dark:text-slate-200 dark:ring-white/[0.1]">
                      <span class="flex h-4 w-4 items-center justify-center rounded-full bg-gradient-to-br from-primary-500 to-sky-500 text-white shadow-sm shadow-primary-500/35">
                        <svg class="h-2.5 w-2.5" viewBox="0 0 24 24" fill="currentColor">
                          <path d="M13 2L4.5 13.5h6L9.5 22 19.5 10.5h-6L13 2z" />
                        </svg>
                      </span>
                      <span>{{ t('surveys.microBadge') }}</span>
                    </span>
                    <span
                      v-if="activeSurvey.has_reward && activeSurvey.reward_value > 0"
                      class="survey-reward-pill inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2.5 py-1 text-xs font-semibold text-emerald-700 ring-1 ring-inset ring-emerald-500/25 dark:bg-emerald-400/15 dark:text-emerald-300 dark:ring-emerald-400/25"
                    >
                      <span class="h-1.5 w-1.5 rounded-full bg-emerald-500 shadow-[0_0_6px_#10b981] dark:bg-emerald-400"></span>
                      <span>{{ t('surveys.rewardBadge', { amount: formatRewardAmount(activeSurvey.reward_value) }) }}</span>
                    </span>
                    <span
                      v-if="preview"
                      class="inline-flex items-center rounded-full bg-amber-500/15 px-2.5 py-1 text-xs font-semibold text-amber-700 ring-1 ring-inset ring-amber-500/25 dark:text-amber-300"
                    >
                      {{ t('surveys.previewTag') }}
                    </span>
                  </div>
                  <h3 class="mt-3.5 text-[19px] font-bold leading-[1.32] tracking-[-0.025em] text-slate-950 dark:text-white">
                    {{ activeSurvey.questions[0].title }}
                  </h3>
                  <p v-if="activeSurvey.description" class="mt-1.5 text-xs leading-relaxed text-slate-500 dark:text-slate-400">
                    {{ activeSurvey.description }}
                  </p>
                </div>
                <button
                  type="button"
                  data-testid="survey-close-btn"
                  class="-mr-1 -mt-1 flex h-7 w-7 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-900/5 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-white/10 dark:hover:text-white"
                  :aria-label="t('common.close')"
                  @click="handleCloseClick"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              <!-- 投票前：渐次入场触感选项卡 -->
              <div v-if="!submitResult" class="space-y-2">
                <button
                  v-for="(opt, idx) in activeSurvey.questions[0].options || []"
                  :key="opt.id"
                  type="button"
                  :disabled="submitting"
                  :data-testid="`survey-micro-opt-${opt.id}`"
                  :style="{ '--stagger': `${idx * 38}ms` }"
                  class="survey-stagger-item group relative flex w-full items-center gap-3.5 overflow-hidden rounded-2xl border border-slate-200/90 bg-slate-50/70 px-4 py-3.5 text-left transition-all duration-200 hover:-translate-y-[1.5px] hover:border-primary-400/80 hover:bg-white hover:shadow-[0_10px_26px_-14px_rgba(37,99,235,0.4)] active:scale-[0.99] disabled:opacity-60 dark:border-white/[0.08] dark:bg-white/[0.03] dark:hover:border-primary-400/60 dark:hover:bg-white/[0.07]"
                  @click="submitMicroChoice(opt.id)"
                >
                  <span class="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full border-[1.5px] border-slate-300 bg-white transition-all duration-200 group-hover:border-primary-500 group-hover:ring-4 group-hover:ring-primary-500/15 dark:border-slate-600 dark:bg-slate-900 dark:group-hover:border-primary-400">
                    <span class="h-2 w-2 scale-0 rounded-full bg-primary-600 transition-transform duration-200 group-hover:scale-100 dark:bg-primary-400"></span>
                  </span>
                  <div class="min-w-0 flex-1">
                    <div class="text-[14px] font-semibold leading-snug text-slate-800 transition-colors group-hover:text-slate-950 dark:text-slate-100 dark:group-hover:text-white">
                      {{ opt.label }}
                    </div>
                    <div v-if="opt.description" class="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
                      {{ opt.description }}
                    </div>
                  </div>
                  <kbd class="hidden h-6 min-w-[24px] flex-shrink-0 items-center justify-center rounded-lg border border-slate-200/90 bg-white px-1.5 font-mono text-xs font-medium text-slate-400 shadow-[inset_0_-1px_0_rgba(15,23,42,0.06)] transition-colors group-hover:border-primary-300 group-hover:text-primary-600 sm:inline-flex dark:border-white/10 dark:bg-white/5 dark:text-slate-400 dark:group-hover:border-primary-400/40 dark:group-hover:text-primary-300">
                    {{ idx + 1 }}
                  </kbd>
                </button>
              </div>

              <!-- 投票后：实时占比动态数字 + 光感渐变条 -->
              <div v-else class="space-y-3">
                <div
                  v-if="microDisplayStats.length > 0"
                  class="space-y-2"
                  data-testid="survey-micro-results"
                >
                  <div
                    v-for="(row, idx) in microDisplayStats"
                    :key="row.option_id"
                    :style="{ '--stagger': `${idx * 45}ms` }"
                    class="survey-stagger-item relative overflow-hidden rounded-2xl border px-4 py-3.5 transition-all"
                    :class="
                      pickedMicroId === row.option_id
                        ? 'border-primary-500/80 bg-white shadow-[0_8px_22px_-12px_rgba(37,99,235,0.4)] ring-1 ring-primary-500/15 dark:border-primary-400/70 dark:bg-slate-900/80'
                        : 'border-slate-200/80 bg-slate-50/50 dark:border-white/[0.08] dark:bg-white/[0.025]'
                    "
                  >
                    <div
                      class="survey-bar-grow pointer-events-none absolute inset-y-0 left-0 rounded-r-xl"
                      :class="
                        pickedMicroId === row.option_id
                          ? 'bg-gradient-to-r from-primary-500/25 via-primary-500/15 to-sky-400/10 dark:from-primary-500/35 dark:via-primary-500/20 dark:to-sky-400/10'
                          : 'bg-slate-200/55 dark:bg-white/[0.06]'
                      "
                      :style="{ width: `${Math.min(100, Math.max(0, row.percent))}%` }"
                    >
                      <span
                        v-if="pickedMicroId === row.option_id"
                        class="absolute inset-y-1.5 right-0 w-[2px] rounded-full bg-primary-500/60 dark:bg-primary-400/70"
                      ></span>
                    </div>
                    <div class="relative flex items-center justify-between gap-2.5">
                      <div class="flex min-w-0 items-center gap-2">
                        <span
                          v-if="pickedMicroId === row.option_id"
                          class="flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-full bg-primary-600 text-white shadow-sm shadow-primary-600/30 dark:bg-primary-500"
                        >
                          <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3.2">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                          </svg>
                        </span>
                        <span
                          class="truncate text-[13.5px]"
                          :class="
                            pickedMicroId === row.option_id
                              ? 'font-semibold text-primary-800 dark:text-primary-200'
                              : 'font-medium text-slate-700 dark:text-slate-200'
                          "
                        >
                          {{ row.label }}
                        </span>
                        <span
                          v-if="microTopPercent > 0 && row.percent === microTopPercent"
                          class="inline-flex flex-shrink-0 items-center gap-0.5 rounded-full bg-primary-600/10 px-2 py-0.5 text-xs font-bold uppercase tracking-wider text-primary-700 ring-1 ring-inset ring-primary-600/15 dark:bg-primary-400/15 dark:text-primary-300 dark:ring-primary-400/25"
                        >
                          <svg class="h-2.5 w-2.5" viewBox="0 0 24 24" fill="currentColor">
                            <path d="M12 2l2.4 6.6L21 11l-6.6 2.4L12 20l-2.4-6.6L3 11l6.6-2.4L12 2z" />
                          </svg>
                          TOP
                        </span>
                      </div>
                      <span
                        class="flex-shrink-0 font-mono text-[13px] font-bold tabular-nums"
                        :class="
                          pickedMicroId === row.option_id
                            ? 'text-primary-700 dark:text-primary-300'
                            : 'text-slate-500 dark:text-slate-400'
                        "
                      >
                        {{ formatMicroPercent(row.option_id, row.percent) }}%
                      </span>
                    </div>
                  </div>
                </div>

                <!-- 快问奖励回馈块 -->
                <div
                  v-if="submitResult.reward_status === 'granted'"
                  class="flex items-center gap-3 rounded-2xl border border-emerald-200/90 bg-gradient-to-br from-emerald-50 to-teal-50/60 p-3.5 text-xs text-emerald-800 dark:border-emerald-500/25 dark:from-emerald-500/15 dark:to-teal-500/10 dark:text-emerald-200"
                >
                  <div class="flex h-10 min-w-[54px] flex-shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 px-2 font-mono text-xs font-bold text-white shadow-sm shadow-emerald-600/30">
                    +${{ formatRewardAmount(submitResult.reward_value) }}
                  </div>
                  <div class="font-semibold leading-snug">
                    {{ t('surveys.rewardGrantedTitle', { amount: formatRewardAmount(submitResult.reward_value) }) }}
                  </div>
                </div>
                <div
                  v-else-if="submitResult.reward_status === 'code_issued' && submitResult.reward_redeem_code"
                  class="space-y-2.5 rounded-2xl border border-primary-200/90 bg-gradient-to-br from-primary-50/80 to-sky-50/40 p-3.5 text-xs dark:border-primary-500/25 dark:from-primary-500/15 dark:to-sky-500/5"
                >
                  <div class="font-semibold text-primary-700 dark:text-primary-300">
                    {{ t('surveys.rewardCodeTitle', { amount: formatRewardAmount(submitResult.reward_value) }) }}
                  </div>
                  <div class="flex items-center gap-2">
                    <code class="flex-1 truncate rounded-xl border border-slate-200/90 bg-white px-3 py-2 text-center font-mono text-xs font-bold tracking-wider text-slate-900 dark:border-white/10 dark:bg-slate-950/70 dark:text-white">
                      {{ submitResult.reward_redeem_code }}
                    </code>
                    <button
                      type="button"
                      class="rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-semibold text-slate-700 transition-colors hover:bg-slate-50 dark:border-white/10 dark:bg-white/5 dark:text-slate-200 dark:hover:bg-white/10"
                      @click="copyRewardCode(submitResult.reward_redeem_code)"
                    >
                      {{ copiedCode ? t('surveys.copiedCode') : t('surveys.copyCode') }}
                    </button>
                  </div>
                  <button
                    v-if="!redeemedInline"
                    type="button"
                    data-testid="survey-micro-redeem-btn"
                    :disabled="redeemingInline"
                    class="flex w-full items-center justify-center gap-1.5 rounded-xl bg-gradient-to-r from-primary-600 to-sky-600 px-3.5 py-2 text-xs font-semibold text-white shadow-md shadow-primary-600/25 transition-all hover:brightness-105 active:scale-[0.99] disabled:opacity-60"
                    @click="redeemIssuedCodeNow(submitResult.reward_redeem_code, submitResult.reward_value)"
                  >
                    {{ redeemingInline ? t('surveys.redeemingNow') : t('surveys.redeemNow') }}
                  </button>
                  <div
                    v-else
                    class="flex items-center justify-center gap-1.5 rounded-xl bg-emerald-500/15 px-3 py-2 text-center text-xs font-semibold text-emerald-700 dark:text-emerald-300"
                  >
                    <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                    <span>{{ t('surveys.redeemSuccess', { amount: formatRewardAmount(submitResult.reward_value) }) }}</span>
                  </div>
                </div>
                <div
                  v-else-if="submitResult.reward_status === 'budget_exhausted'"
                  class="rounded-2xl border border-amber-200/90 bg-amber-50/80 p-3 text-xs text-amber-800 dark:border-amber-500/25 dark:bg-amber-500/10 dark:text-amber-300"
                >
                  {{ t('surveys.rewardBudgetExhausted') }}
                </div>
              </div>

              <div class="flex items-center justify-between border-t border-slate-100/90 pt-3.5 text-xs text-slate-500 dark:border-white/[0.06] dark:text-slate-400">
                <template v-if="!submitResult">
                  <span>{{ t('surveys.microFooterHint') }}</span>
                  <button
                    type="button"
                    data-testid="survey-dismiss-btn"
                    class="font-medium text-slate-400 transition-colors hover:text-slate-700 dark:hover:text-slate-200"
                    @click="handleDismiss"
                  >
                    {{ t('surveys.dismiss') }}
                  </button>
                </template>
                <template v-else>
                  <span class="inline-flex items-center gap-1.5 font-semibold text-emerald-600 dark:text-emerald-400">
                    <span class="flex h-4 w-4 items-center justify-center rounded-full bg-emerald-500/15">
                      <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                    </span>
                    <span>
                      {{
                        microDisplayStats.length > 0
                          ? t('surveys.microVotedSummary', { count: microTotalVotes })
                          : t('surveys.doneTitle')
                      }}
                    </span>
                  </span>
                  <div class="flex items-center gap-2.5">
                    <button
                      v-if="preview"
                      type="button"
                      class="font-medium text-primary-600 hover:underline dark:text-primary-400"
                      @click="resetLocalState"
                    >
                      {{ t('surveys.replayPreview') }}
                    </button>
                    <button
                      type="button"
                      class="rounded-full bg-slate-900 px-3 py-1 text-xs font-semibold text-white transition-colors hover:bg-slate-800 dark:bg-white dark:text-slate-900 dark:hover:bg-slate-100"
                      @click="finishAndClose"
                    >
                      {{ t('common.close') }}
                    </button>
                  </div>
                </template>
              </div>
            </div>
          </div>

          <!-- 多步交互卡片（corner 轻浮窗 / modal 居中焦点弹窗） -->
          <div
            v-else
            :class="panelClasses"
            data-testid="survey-wizard-card"
            @pointermove="onShellPointerMove"
            @pointerleave="onShellPointerLeave"
          >
            <!-- 顶部电光蓝极光晕 + 细建筑点阵 + 鼠标追光层 -->
            <div class="survey-aurora pointer-events-none absolute inset-x-0 top-0 h-40" aria-hidden="true"></div>
            <div class="survey-dot-grid pointer-events-none absolute inset-x-0 top-0 h-32" aria-hidden="true"></div>
            <div class="survey-spotlight pointer-events-none absolute inset-0" aria-hidden="true"></div>

            <!-- 手机端顶部把手 -->
            <div class="relative flex flex-shrink-0 justify-center pt-2.5 sm:hidden">
              <span class="h-1 w-10 rounded-full bg-slate-300/80 dark:bg-slate-600/80"></span>
            </div>

            <!-- 编辑感精工头部：品牌印记 + 翡翠回馈徽章 + 极细 3px 分段轨道 -->
            <div class="relative flex-shrink-0 px-5 pb-3 pt-4 sm:px-6 sm:pt-5">
              <div class="flex items-center justify-between gap-2">
                <div class="flex min-w-0 flex-wrap items-center gap-2">
                  <!-- GABUGU 品牌签名徽标 -->
                  <span class="inline-flex items-center gap-2 rounded-full bg-slate-900/[0.04] py-1 pl-1.5 pr-2.5 text-xs font-semibold tracking-wide text-slate-700 ring-1 ring-inset ring-slate-900/[0.07] dark:bg-white/[0.06] dark:text-slate-200 dark:ring-white/[0.1]">
                    <img
                      src="/logo.svg"
                      alt=""
                      class="h-4 w-4 flex-shrink-0 object-contain"
                      aria-hidden="true"
                    />
                    <span>{{ t('surveys.badge') }}</span>
                  </span>

                  <!-- 珠宝感奖励微标 -->
                  <span
                    v-if="!submitResult && activeSurvey.has_reward && activeSurvey.reward_value > 0"
                    class="survey-reward-pill inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2.5 py-1 text-xs font-semibold text-emerald-700 ring-1 ring-inset ring-emerald-500/25 dark:bg-emerald-400/15 dark:text-emerald-300 dark:ring-emerald-400/25"
                  >
                    <span class="h-1.5 w-1.5 rounded-full bg-emerald-500 shadow-[0_0_6px_#10b981] dark:bg-emerald-400"></span>
                    <span>{{ t('surveys.rewardBadge', { amount: formatRewardAmount(activeSurvey.reward_value) }) }}</span>
                  </span>

                  <span
                    v-if="preview"
                    class="inline-flex items-center rounded-full bg-amber-500/15 px-2.5 py-1 text-xs font-semibold text-amber-700 ring-1 ring-inset ring-amber-500/25 dark:text-amber-300"
                  >
                    {{ t('surveys.previewTag') }}
                  </span>
                </div>

                <div class="flex flex-shrink-0 items-center gap-0.5">
                  <button
                    v-if="!submitResult && activeSurvey.display_mode === 'corner'"
                    type="button"
                    data-testid="survey-minimize-btn"
                    class="hidden h-7 w-7 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-900/5 hover:text-slate-700 sm:inline-flex dark:text-slate-400 dark:hover:bg-white/10 dark:hover:text-white"
                    :title="t('surveys.minimize')"
                    @click="minimizeToPill"
                  >
                    <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19 12H5" />
                    </svg>
                  </button>
                  <button
                    type="button"
                    data-testid="survey-close-btn"
                    class="flex h-7 w-7 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-900/5 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-white/10 dark:hover:text-white"
                    :aria-label="t('common.close')"
                    @click="handleCloseClick"
                  >
                    <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              </div>

              <!-- 极细 3px 精工分段进度轨 + 零补齐杂志感步骤计数 -->
              <div v-if="!submitResult" class="mt-4 flex items-center gap-3">
                <div class="flex flex-1 items-center gap-1.5">
                  <span
                    v-for="idx in totalSteps"
                    :key="idx"
                    class="h-[3px] rounded-full transition-all duration-300 ease-out"
                    :class="
                      idx - 1 < stepIndex
                        ? 'flex-1 bg-primary-500/80 dark:bg-primary-400/85'
                        : idx - 1 === stepIndex
                          ? 'flex-[1.8] bg-gradient-to-r from-primary-600 via-primary-500 to-sky-400 shadow-[0_0_10px_rgba(37,99,235,0.5)]'
                          : 'flex-1 bg-slate-200/85 dark:bg-white/[0.1]'
                    "
                  ></span>
                </div>
                <span class="flex-shrink-0 font-mono text-xs font-semibold tracking-[0.12em] tabular-nums text-slate-400 dark:text-slate-400">
                  {{ formatStepIndex(stepIndex + 1) }} / {{ formatStepIndex(totalSteps) }}
                </span>
              </div>
            </div>

            <!-- 正文区 -->
            <div class="relative flex-1 overflow-y-auto px-5 pb-5 pt-1.5 sm:px-6">
              <!-- 完成感谢页：仪式感庆祝动效 + 金属账本回执 / 带撕口光泽礼券卡 -->
              <div v-if="submitResult" class="space-y-5 py-3 text-center" data-testid="survey-done-view">
                <div class="relative mx-auto flex h-20 w-20 items-center justify-center">
                  <span class="survey-celebrate-ring absolute inset-0 rounded-full bg-emerald-500/15 dark:bg-emerald-400/20"></span>
                  <span class="survey-celebrate-ring-2 absolute inset-1.5 rounded-full border border-emerald-400/40"></span>
                  <span class="survey-confetti survey-confetti-1"></span>
                  <span class="survey-confetti survey-confetti-2"></span>
                  <span class="survey-confetti survey-confetti-3"></span>
                  <span class="survey-confetti survey-confetti-4"></span>
                  <span class="survey-confetti survey-confetti-5"></span>
                  <span class="survey-confetti survey-confetti-6"></span>
                  <div class="relative flex h-14 w-14 items-center justify-center rounded-[18px] bg-gradient-to-br from-emerald-400 via-emerald-500 to-teal-600 text-white shadow-[0_14px_30px_-8px_rgba(16,185,129,0.55)] ring-1 ring-inset ring-white/30">
                    <svg class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.6">
                      <path class="survey-check-draw" stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                </div>

                <div class="space-y-1.5">
                  <h3 class="text-[20px] font-bold tracking-[-0.025em] text-slate-950 dark:text-white">
                    {{ t('surveys.doneTitle') }}
                  </h3>
                  <p class="mx-auto max-w-[310px] text-xs leading-relaxed text-slate-500 dark:text-slate-400">
                    {{ t('surveys.doneSubtitle') }}
                  </p>
                </div>

                <!-- 自动赠送余额：精工金属账本回执卡 -->
                <div
                  v-if="submitResult.reward_status === 'granted'"
                  data-testid="survey-reward-granted"
                  class="survey-voucher-foil relative overflow-hidden rounded-2xl border border-emerald-200/90 bg-gradient-to-br from-emerald-50/90 via-white to-teal-50/70 p-4 text-left shadow-[0_14px_32px_-16px_rgba(16,185,129,0.35)] dark:border-emerald-400/25 dark:from-emerald-500/15 dark:via-[#0d1829] dark:to-teal-500/10"
                >
                  <div class="flex items-center gap-4">
                    <div class="flex flex-col items-center justify-center border-r border-emerald-200/80 pr-4 dark:border-emerald-400/20">
                      <span class="font-mono text-[24px] font-bold leading-none tracking-tight tabular-nums text-emerald-600 dark:text-emerald-300">
                        +${{ formatRewardAmount(submitResult.reward_value) }}
                      </span>
                      <span class="mt-1 inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-xs font-bold text-emerald-700 dark:text-emerald-300">
                        <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                        USD
                      </span>
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="text-[13.5px] font-bold leading-snug text-slate-900 dark:text-white">
                        {{ t('surveys.rewardGrantedTitle', { amount: formatRewardAmount(submitResult.reward_value) }) }}
                      </div>
                      <p class="mt-1 text-xs leading-relaxed text-slate-500 dark:text-slate-400">
                        {{ t('surveys.rewardGrantedDesc') }}
                      </p>
                    </div>
                  </div>
                </div>

                <!-- 发放一次性专属兑换码：带对角折光与撕口的高级数字礼券 -->
                <div
                  v-else-if="submitResult.reward_status === 'code_issued' && submitResult.reward_redeem_code"
                  data-testid="survey-reward-code"
                  class="survey-voucher-foil relative overflow-hidden rounded-2xl border border-primary-200/90 bg-gradient-to-br from-primary-50/85 via-white to-sky-50/50 text-left shadow-[0_16px_36px_-16px_rgba(37,99,235,0.35)] dark:border-primary-400/25 dark:from-primary-500/15 dark:via-[#0d1728] dark:to-sky-500/10"
                >
                  <div class="flex items-center justify-between gap-3 px-4 pb-3.5 pt-4">
                    <div class="min-w-0">
                      <div class="flex items-center gap-1.5 text-[13.5px] font-bold text-slate-900 dark:text-white">
                        <span class="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-md bg-primary-600/10 text-primary-600 dark:bg-primary-400/20 dark:text-primary-300">
                          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M20 12v8a2 2 0 01-2 2H6a2 2 0 01-2-2v-8m16 0H4m16 0a2 2 0 002-2V9a2 2 0 00-2-2h-3.17a3 3 0 00.17-1 3 3 0 00-5.83-1 3 3 0 00-5.83 1c0 .35.06.68.17 1H4a2 2 0 00-2 2v1a2 2 0 002 2m8-5v15" />
                          </svg>
                        </span>
                        <span class="truncate">{{ t('surveys.rewardCodeTitle', { amount: formatRewardAmount(submitResult.reward_value) }) }}</span>
                      </div>
                      <p class="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
                        {{ t('surveys.rewardCodeDesc') }}
                      </p>
                    </div>
                    <span class="flex-shrink-0 rounded-xl bg-gradient-to-br from-primary-600 to-sky-500 px-3 py-1 font-mono text-sm font-bold tabular-nums text-white shadow-sm shadow-primary-600/35">
                      ${{ formatRewardAmount(submitResult.reward_value) }}
                    </span>
                  </div>

                  <!-- 礼券半圆撕口虚线 -->
                  <div class="relative flex items-center px-2">
                    <span class="-ml-4 h-4 w-4 rounded-full border border-primary-200/90 bg-white dark:border-primary-400/25 dark:bg-[#090f1d]"></span>
                    <div class="mx-1.5 flex-1 border-t border-dashed border-primary-300/75 dark:border-primary-400/25"></div>
                    <span class="-mr-4 h-4 w-4 rounded-full border border-primary-200/90 bg-white dark:border-primary-400/25 dark:bg-[#090f1d]"></span>
                  </div>

                  <div class="space-y-2.5 px-4 pb-4 pt-3.5">
                    <div class="flex items-center gap-2">
                      <code class="flex-1 truncate rounded-xl border border-slate-200/90 bg-slate-50/90 px-3.5 py-2.5 text-center font-mono text-xs font-bold tracking-[0.12em] text-slate-900 shadow-inner dark:border-white/10 dark:bg-slate-950/90 dark:text-white">
                        {{ submitResult.reward_redeem_code }}
                      </code>
                      <button
                        type="button"
                        class="inline-flex flex-shrink-0 items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-xs font-semibold text-slate-700 transition-all hover:bg-slate-50 active:scale-[0.97] dark:border-white/10 dark:bg-white/[0.06] dark:text-slate-200 dark:hover:bg-white/10"
                        @click="copyRewardCode(submitResult.reward_redeem_code)"
                      >
                        <svg v-if="copiedCode" class="h-3.5 w-3.5 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.8">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                        <svg v-else class="h-3.5 w-3.5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                        </svg>
                        <span>{{ copiedCode ? t('surveys.copiedCode') : t('surveys.copyCode') }}</span>
                      </button>
                    </div>
                    <button
                      v-if="!redeemedInline"
                      type="button"
                      :disabled="redeemingInline"
                      class="flex w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-primary-600 via-primary-500 to-sky-500 px-4 py-2.5 text-xs font-semibold text-white shadow-[0_8px_22px_-6px_rgba(37,99,235,0.55)] transition-all hover:-translate-y-0.5 hover:brightness-105 active:scale-[0.99] disabled:opacity-60"
                      @click="redeemIssuedCodeNow(submitResult.reward_redeem_code, submitResult.reward_value)"
                    >
                      <span>{{ redeemingInline ? t('surveys.redeemingNow') : t('surveys.redeemNow') }}</span>
                    </button>
                    <div
                      v-else
                      class="flex items-center justify-center gap-1.5 rounded-xl bg-emerald-500/15 px-3 py-2.5 text-center text-xs font-semibold text-emerald-700 ring-1 ring-inset ring-emerald-500/25 dark:text-emerald-300"
                    >
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.6">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                      <span>{{ t('surveys.redeemSuccess', { amount: formatRewardAmount(submitResult.reward_value) }) }}</span>
                    </div>
                  </div>
                </div>

                <!-- 预算名额已领完 -->
                <div
                  v-else-if="submitResult.reward_status === 'budget_exhausted'"
                  class="rounded-2xl border border-amber-200/90 bg-amber-50/80 p-3.5 text-xs text-amber-800 dark:border-amber-500/25 dark:bg-amber-500/10 dark:text-amber-300"
                >
                  {{ t('surveys.rewardBudgetExhausted') }}
                </div>

                <div class="flex items-center justify-center gap-2.5 pt-1">
                  <button
                    type="button"
                    data-testid="survey-done-close"
                    class="rounded-full bg-slate-950 px-6 py-2.5 text-xs font-semibold text-white shadow-md shadow-slate-950/15 transition-all hover:-translate-y-0.5 hover:bg-slate-800 active:scale-[0.98] dark:bg-white dark:text-slate-950 dark:hover:bg-slate-100"
                    @click="finishAndClose"
                  >
                    {{ t('surveys.backToWorkspace') }}
                  </button>
                  <button
                    v-if="preview"
                    type="button"
                    class="rounded-full border border-slate-200 bg-white px-4 py-2.5 text-xs font-semibold text-slate-700 transition-colors hover:bg-slate-50 dark:border-white/10 dark:bg-white/5 dark:text-slate-200 dark:hover:bg-white/10"
                    @click="resetLocalState"
                  >
                    {{ t('surveys.replayPreview') }}
                  </button>
                </div>
              </div>

              <!-- 答题步骤区（方向感知弹性切题 + 逐项微错峰入场） -->
              <div
                v-else-if="currentQuestion"
                :key="`${currentQuestion.id}-${stepIndex}`"
                :class="stepDirection === 'backward' ? 'survey-step-backward' : 'survey-step-forward'"
                class="space-y-4"
              >
                <!-- ① 单选大卡片 -->
                <template v-if="currentQuestion.type === 'single'">
                  <div class="space-y-1.5">
                    <h3 class="text-[19px] font-bold leading-[1.32] tracking-[-0.025em] text-slate-950 sm:text-[20px] dark:text-white">
                      {{ currentQuestion.title }}
                    </h3>
                    <p class="text-xs leading-relaxed text-slate-500 dark:text-slate-400">
                      {{
                        stepIndex === 0 && activeSurvey.description
                          ? activeSurvey.description
                          : currentQuestion.hint || t('surveys.singleHint')
                      }}
                    </p>
                  </div>
                  <div class="space-y-2.5">
                    <button
                      v-for="(opt, idx) in currentQuestion.options || []"
                      :key="opt.id"
                      type="button"
                      :data-testid="`survey-opt-${currentQuestion.id}-${opt.id}`"
                      :style="{ '--stagger': `${idx * 38}ms` }"
                      class="survey-stagger-item group relative flex w-full items-start gap-3.5 overflow-hidden rounded-2xl border p-4 text-left transition-all duration-200 hover:-translate-y-[1.5px] active:scale-[0.99]"
                      :class="
                        answers[currentQuestion.id]?.choice_id === opt.id
                          ? 'border-primary-500 bg-gradient-to-r from-primary-50/95 via-white to-sky-50/50 shadow-[0_10px_28px_-14px_rgba(37,99,235,0.45)] ring-1 ring-primary-500/20 dark:border-primary-400 dark:from-primary-500/20 dark:via-primary-500/10 dark:to-sky-500/10'
                          : 'border-slate-200/90 bg-slate-50/65 hover:border-primary-400/75 hover:bg-white hover:shadow-[0_10px_24px_-14px_rgba(37,99,235,0.3)] dark:border-white/[0.08] dark:bg-white/[0.03] dark:hover:border-primary-400/50 dark:hover:bg-white/[0.065]'
                      "
                      @click="selectSingle(currentQuestion, opt.id)"
                    >
                      <!-- 选中自动跳转时的底部流光确认条 -->
                      <span
                        v-if="advancingOptionId === `${currentQuestion.id}:${opt.id}`"
                        class="survey-advance-sweep pointer-events-none absolute inset-x-0 bottom-0 h-[2.5px] bg-gradient-to-r from-primary-500 via-sky-400 to-emerald-400"
                      ></span>

                      <!-- 左侧精工单选圆环 -->
                      <span
                        class="mt-0.5 flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full border-[1.5px] transition-all duration-200"
                        :class="
                          answers[currentQuestion.id]?.choice_id === opt.id
                            ? 'scale-105 border-primary-600 bg-primary-600 text-white shadow-sm shadow-primary-600/40 dark:border-primary-400 dark:bg-primary-500'
                            : 'border-slate-300 bg-white group-hover:border-primary-500 group-hover:ring-4 group-hover:ring-primary-500/10 dark:border-slate-600 dark:bg-slate-900 dark:group-hover:border-primary-400'
                        "
                      >
                        <svg
                          v-if="answers[currentQuestion.id]?.choice_id === opt.id"
                          class="h-3 w-3"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                          stroke-width="3.2"
                        >
                          <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                      </span>

                      <div class="min-w-0 flex-1">
                        <div class="flex items-center justify-between gap-2">
                          <span
                            class="text-[14.5px] font-semibold leading-snug tracking-[-0.01em]"
                            :class="
                              answers[currentQuestion.id]?.choice_id === opt.id
                                ? 'text-primary-800 dark:text-primary-100'
                                : 'text-slate-800 group-hover:text-slate-950 dark:text-slate-100 dark:group-hover:text-white'
                            "
                          >
                            {{ opt.label }}
                          </span>
                          <kbd
                            class="hidden h-6 min-w-[24px] flex-shrink-0 items-center justify-center rounded-lg border px-1.5 font-mono text-xs font-medium transition-colors sm:inline-flex"
                            :class="
                              answers[currentQuestion.id]?.choice_id === opt.id
                                ? 'border-primary-300 bg-primary-100/80 text-primary-700 dark:border-primary-400/40 dark:bg-primary-500/25 dark:text-primary-200'
                                : 'border-slate-200/90 bg-white text-slate-400 shadow-[inset_0_-1px_0_rgba(15,23,42,0.06)] group-hover:border-primary-300 group-hover:text-primary-600 dark:border-white/10 dark:bg-white/5 dark:text-slate-400 dark:group-hover:border-primary-400/40 dark:group-hover:text-primary-300'
                            "
                          >
                            {{ idx + 1 }}
                          </kbd>
                        </div>
                        <p v-if="opt.description" class="mt-1 text-xs leading-relaxed text-slate-500 dark:text-slate-400">
                          {{ opt.description }}
                        </p>
                      </div>
                    </button>
                  </div>
                </template>

                <!-- ② 多选标签网格 -->
                <template v-else-if="currentQuestion.type === 'multi'">
                  <div class="space-y-1.5">
                    <h3 class="text-[19px] font-bold leading-[1.32] tracking-[-0.025em] text-slate-950 sm:text-[20px] dark:text-white">
                      {{ currentQuestion.title }}
                    </h3>
                    <p class="text-xs leading-relaxed text-slate-500 dark:text-slate-400">
                      {{
                        currentQuestion.hint ||
                          t('surveys.multiHint', {
                            count: answers[currentQuestion.id]?.choice_ids?.length || 0
                          })
                      }}
                    </p>
                  </div>
                  <div class="grid grid-cols-2 gap-2.5">
                    <button
                      v-for="(opt, idx) in currentQuestion.options || []"
                      :key="opt.id"
                      type="button"
                      :data-testid="`survey-opt-${currentQuestion.id}-${opt.id}`"
                      :style="{ '--stagger': `${idx * 30}ms` }"
                      class="survey-stagger-item group flex items-center justify-between gap-2 rounded-2xl border px-3.5 py-3.5 text-left transition-all duration-150 hover:-translate-y-[1.5px] active:scale-[0.98]"
                      :class="
                        isMultiSelected(currentQuestion.id, opt.id)
                          ? 'border-primary-500 bg-gradient-to-br from-primary-50/95 to-sky-50/50 text-primary-800 shadow-[0_8px_20px_-12px_rgba(37,99,235,0.4)] ring-1 ring-primary-500/20 dark:border-primary-400 dark:from-primary-500/20 dark:to-sky-500/10 dark:text-primary-100'
                          : 'border-slate-200/90 bg-slate-50/65 text-slate-800 hover:border-primary-400/75 hover:bg-white dark:border-white/[0.08] dark:bg-white/[0.03] dark:text-slate-200 dark:hover:border-primary-400/50 dark:hover:bg-white/[0.065]'
                      "
                      @click="toggleMulti(currentQuestion.id, opt.id)"
                    >
                      <div class="min-w-0">
                        <div class="truncate text-[13.5px] font-semibold">{{ opt.label }}</div>
                        <div v-if="opt.tag || opt.description" class="mt-0.5 truncate text-xs text-slate-500 dark:text-slate-400">
                          {{ opt.tag || opt.description }}
                        </div>
                      </div>
                      <span
                        class="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-md border-[1.5px] transition-all duration-150"
                        :class="
                          isMultiSelected(currentQuestion.id, opt.id)
                            ? 'scale-105 border-primary-600 bg-primary-600 text-white shadow-sm dark:border-primary-400 dark:bg-primary-500'
                            : 'border-slate-300 bg-white group-hover:border-primary-400 dark:border-slate-600 dark:bg-slate-900'
                        "
                      >
                        <svg
                          v-if="isMultiSelected(currentQuestion.id, opt.id)"
                          class="h-3 w-3"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                          stroke-width="3.2"
                        >
                          <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                      </span>
                    </button>
                  </div>
                </template>

                <!-- ③ 5 档 Dock 联动精工表情评分 -->
                <template v-else-if="currentQuestion.type === 'rating'">
                  <div class="space-y-1.5">
                    <h3 class="text-[19px] font-bold leading-[1.32] tracking-[-0.025em] text-slate-950 sm:text-[20px] dark:text-white">
                      {{ currentQuestion.title }}
                    </h3>
                    <p class="text-xs leading-relaxed text-slate-500 dark:text-slate-400">
                      {{ currentQuestion.hint || t('surveys.ratingHint') }}
                    </p>
                  </div>
                  <div
                    class="grid grid-cols-5 gap-2 pt-1"
                    @mouseleave="hoveredRating = null"
                  >
                    <button
                      v-for="(item, idx) in ratingScale"
                      :key="item.score"
                      type="button"
                      :data-testid="`survey-rate-${currentQuestion.id}-${item.score}`"
                      :style="{ '--stagger': `${idx * 32}ms` }"
                      class="survey-stagger-item group relative flex flex-col items-center gap-2 overflow-hidden rounded-2xl border px-1 py-4 transition-all duration-200 hover:-translate-y-1.5 hover:scale-[1.03] active:scale-[0.97]"
                      :class="[
                        getRatingCardClasses(answers[currentQuestion.id]?.rating === item.score, item.score),
                        hoveredRating !== null && hoveredRating !== item.score && answers[currentQuestion.id]?.rating !== item.score
                          ? 'opacity-55 scale-[0.98]'
                          : 'opacity-100'
                      ]"
                      @mouseenter="hoveredRating = item.score"
                      @focus="hoveredRating = item.score"
                      @blur="hoveredRating = null"
                      @click="selectRating(currentQuestion, item.score)"
                    >
                      <span
                        v-if="advancingOptionId === `${currentQuestion.id}:rate-${item.score}`"
                        class="survey-advance-sweep pointer-events-none absolute inset-x-0 bottom-0 h-[2.5px] bg-white/80"
                      ></span>

                      <svg class="h-6 w-6 transition-transform duration-200 group-hover:scale-110" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                        <circle cx="12" cy="12" r="9.5" />
                        <template v-if="item.score === 1">
                          <path d="M8.5 9.5l1.5 1-1.5 1" />
                          <path d="M15.5 9.5l-1.5 1 1.5 1" />
                          <path d="M8.5 16.5c1-1.6 2.2-2.2 3.5-2.2s2.5.6 3.5 2.2" />
                        </template>
                        <template v-else-if="item.score === 2">
                          <circle cx="9" cy="10" r="0.8" fill="currentColor" />
                          <circle cx="15" cy="10" r="0.8" fill="currentColor" />
                          <path d="M9 16c.9-.9 1.9-1.3 3-1.3s2.1.4 3 1.3" />
                        </template>
                        <template v-else-if="item.score === 3">
                          <circle cx="9" cy="10" r="0.8" fill="currentColor" />
                          <circle cx="15" cy="10" r="0.8" fill="currentColor" />
                          <path d="M9 15h6" />
                        </template>
                        <template v-else-if="item.score === 4">
                          <circle cx="9" cy="10" r="0.8" fill="currentColor" />
                          <circle cx="15" cy="10" r="0.8" fill="currentColor" />
                          <path d="M8.5 14.5c.9 1.3 2.1 2 3.5 2s2.6-.7 3.5-2" />
                        </template>
                        <template v-else>
                          <path d="M8 10.2c.5-.8 1.5-.8 2 0" />
                          <path d="M14 10.2c.5-.8 1.5-.8 2 0" />
                          <path d="M8 14.2c1 1.8 2.4 2.8 4 2.8s3-1 4-2.8" />
                        </template>
                      </svg>
                      <span class="font-mono text-[15px] font-bold leading-none tabular-nums">{{ item.score }}</span>
                      <span
                        class="text-xs font-medium leading-tight"
                        :class="
                          answers[currentQuestion.id]?.rating === item.score
                            ? 'text-white/95'
                            : 'text-slate-500 dark:text-slate-400'
                        "
                      >
                        {{ t(item.labelKey) }}
                      </span>
                    </button>
                  </div>

                  <div
                    v-if="answers[currentQuestion.id]?.rating"
                    class="flex items-center gap-2 rounded-2xl px-3.5 py-2.5 text-xs font-medium"
                    :class="
                      (answers[currentQuestion.id]?.rating || 0) <= 3
                        ? 'bg-amber-50 text-amber-800 ring-1 ring-inset ring-amber-200/80 dark:bg-amber-500/15 dark:text-amber-200 dark:ring-amber-400/20'
                        : 'bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-200/80 dark:bg-emerald-500/15 dark:text-emerald-200 dark:ring-emerald-400/20'
                    "
                  >
                    <svg class="h-4 w-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span>
                      {{
                        (answers[currentQuestion.id]?.rating || 0) <= 3
                          ? t('surveys.ratingFeedbackLow')
                          : t('surveys.ratingFeedbackHigh')
                      }}
                    </span>
                  </div>
                </template>

                <!-- ④ 快捷标签 + 选填文本 -->
                <template v-else-if="currentQuestion.type === 'tags_text'">
                  <div class="space-y-1.5">
                    <div v-if="activeTagsMeta.hasLinkedScore" class="mb-1">
                      <span
                        class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold"
                        :class="
                          activeTagsMeta.isLow
                            ? 'bg-amber-500/15 text-amber-700 ring-1 ring-inset ring-amber-500/25 dark:text-amber-300'
                            : 'bg-primary-500/15 text-primary-700 ring-1 ring-inset ring-primary-500/25 dark:text-primary-300'
                        "
                      >
                        <span
                          class="h-1.5 w-1.5 rounded-full"
                          :class="activeTagsMeta.isLow ? 'bg-amber-500' : 'bg-primary-500'"
                        ></span>
                        <span>{{ activeTagsMeta.isLow ? t('surveys.tagsLowBadge') : t('surveys.tagsHighBadge') }}</span>
                      </span>
                    </div>
                    <h3 class="text-[19px] font-bold leading-[1.32] tracking-[-0.025em] text-slate-950 sm:text-[20px] dark:text-white">
                      {{ activeTagsMeta.title }}
                    </h3>
                    <p class="text-xs leading-relaxed text-slate-500 dark:text-slate-400">
                      {{ currentQuestion.hint || t('surveys.tagsHint') }}
                    </p>
                  </div>

                  <div v-if="activeTagsMeta.chips.length > 0" class="flex flex-wrap gap-2">
                    <button
                      v-for="(chip, idx) in activeTagsMeta.chips"
                      :key="chip"
                      type="button"
                      :data-testid="`survey-chip-${chip}`"
                      :style="{ '--stagger': `${idx * 25}ms` }"
                      class="survey-stagger-item inline-flex items-center gap-1.5 rounded-full border px-3.5 py-1.5 text-xs transition-all duration-150 active:scale-[0.95]"
                      :class="
                        isTagSelected(currentQuestion.id, chip)
                          ? 'border-slate-950 bg-slate-950 font-semibold text-white shadow-sm dark:border-primary-400 dark:bg-primary-500 dark:text-white'
                          : 'border-slate-200/90 bg-slate-50/75 font-medium text-slate-700 hover:border-primary-400 hover:bg-white hover:text-primary-700 dark:border-white/10 dark:bg-white/[0.04] dark:text-slate-200 dark:hover:border-primary-400/60 dark:hover:bg-white/[0.08]'
                      "
                      @click="toggleTag(currentQuestion.id, chip)"
                    >
                      <svg
                        v-if="isTagSelected(currentQuestion.id, chip)"
                        class="h-3.5 w-3.5"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2.8"
                      >
                        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                      <svg
                        v-else
                        class="h-3.5 w-3.5 opacity-55"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2.2"
                      >
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14m-7-7h14" />
                      </svg>
                      <span>{{ chip }}</span>
                    </button>
                  </div>

                  <div class="relative">
                    <textarea
                      :value="answers[currentQuestion.id]?.text || ''"
                      rows="3"
                      maxlength="1000"
                      data-testid="survey-textarea"
                      :placeholder="currentQuestion.placeholder || t('surveys.textPlaceholder')"
                      class="w-full resize-none rounded-2xl border border-slate-200/90 bg-slate-50/60 px-4 py-3 text-base text-slate-900 placeholder-slate-400 transition-all focus:border-primary-500 focus:bg-white focus:outline-none focus:ring-4 focus:ring-primary-500/15 sm:text-sm dark:border-white/10 dark:bg-white/[0.03] dark:text-white dark:placeholder-slate-500 dark:focus:border-primary-400 dark:focus:bg-slate-900/90"
                      @input="onTextInput(currentQuestion.id, ($event.target as HTMLTextAreaElement).value)"
                    ></textarea>
                    <span
                      v-if="(answers[currentQuestion.id]?.text?.length || 0) > 0"
                      class="pointer-events-none absolute bottom-2.5 right-3.5 font-mono text-xs tabular-nums text-slate-400"
                    >
                      {{ answers[currentQuestion.id]?.text?.length || 0 }} / 1000
                    </span>
                  </div>
                </template>

                <p
                  v-if="validationError"
                  class="survey-shake flex items-center gap-1.5 text-xs font-medium text-rose-500"
                >
                  <svg class="h-3.5 w-3.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01M12 3a9 9 0 100 18 9 9 0 000-18z" />
                  </svg>
                  <span>{{ validationError }}</span>
                </p>
              </div>
            </div>

            <!-- 页脚操作区：极简呼吸留白 + 药丸主行动按钮 -->
            <div
              v-if="!submitResult"
              class="relative flex flex-shrink-0 items-center justify-between gap-2 border-t border-slate-100/90 bg-slate-50/60 px-5 py-3.5 sm:px-6 dark:border-white/[0.06] dark:bg-white/[0.02]"
            >
              <div class="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400">
                <button
                  v-if="stepIndex > 0"
                  type="button"
                  data-testid="survey-prev-btn"
                  class="inline-flex items-center gap-1 rounded-full border border-slate-200/90 bg-white px-3.5 py-2 text-xs font-semibold text-slate-700 transition-all hover:bg-slate-50 active:scale-[0.98] dark:border-white/10 dark:bg-white/[0.06] dark:text-slate-200 dark:hover:bg-white/10"
                  @click="goPrev"
                >
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
                  </svg>
                  <span>{{ t('surveys.prev') }}</span>
                </button>
                <template v-else>
                  <button
                    type="button"
                    data-testid="survey-snooze-btn"
                    class="rounded-full px-2 py-1 font-medium text-slate-500 transition-colors hover:bg-slate-200/60 hover:text-slate-800 dark:text-slate-400 dark:hover:bg-white/10 dark:hover:text-slate-200"
                    @click="handleSnooze"
                  >
                    {{ t('surveys.snooze') }}
                  </button>
                  <span class="opacity-30">·</span>
                  <button
                    type="button"
                    data-testid="survey-dismiss-btn"
                    class="rounded-full px-2 py-1 font-medium text-slate-400 transition-colors hover:bg-slate-200/60 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-slate-200"
                    @click="handleDismiss"
                  >
                    {{ t('surveys.dismiss') }}
                  </button>
                </template>
              </div>

              <div class="flex items-center gap-2">
                <button
                  v-if="!isLastStep"
                  type="button"
                  data-testid="survey-next-btn"
                  class="inline-flex items-center gap-1.5 rounded-full bg-gradient-to-r from-primary-600 to-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-[0_6px_18px_-6px_rgba(37,99,235,0.55)] transition-all hover:-translate-y-0.5 hover:brightness-105 active:scale-[0.98]"
                  @click="goNext"
                >
                  <span>{{ hasCurrentStepAnswer || currentQuestion?.required ? t('surveys.next') : t('surveys.skip') }}</span>
                  <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.3">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
                  </svg>
                </button>
                <button
                  v-else
                  type="button"
                  data-testid="survey-submit-btn"
                  :disabled="submitting"
                  class="inline-flex items-center gap-1.5 rounded-full bg-gradient-to-r from-primary-600 via-primary-500 to-emerald-500 px-5 py-2 text-xs font-semibold text-white shadow-[0_8px_22px_-6px_rgba(37,99,235,0.55)] transition-all hover:-translate-y-0.5 hover:brightness-105 active:scale-[0.98] disabled:opacity-60"
                  @click="handleSubmit"
                >
                  <span v-if="submitting">{{ t('surveys.submitting') }}</span>
                  <span v-else-if="activeSurvey.has_reward && activeSurvey.reward_value > 0">
                    {{ t('surveys.submitWithReward') }}
                  </span>
                  <span v-else>{{ t('surveys.submit') }}</span>
                  <svg v-if="!submitting" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.6">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { redeemAPI } from '@/api'
import {
  useAdminComplianceStore,
  useAnnouncementStore,
  useAppStore,
  useAuthStore,
  useSurveyStore
} from '@/stores'
import type {
  MicroPollOptionStat,
  SubmitSurveyResult,
  SurveyAnswers,
  SurveyQuestion,
  UserSurvey
} from '@/types/survey'

const props = withDefaults(
  defineProps<{
    survey?: UserSurvey | null
    preview?: boolean
  }>(),
  {
    survey: null,
    preview: false
  }
)

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const route = useRoute()
const surveyStore = useSurveyStore()
const announcementStore = useAnnouncementStore()
const adminComplianceStore = useAdminComplianceStore()
const authStore = useAuthStore()
const appStore = useAppStore()

const stepIndex = ref(0)
const stepDirection = ref<'forward' | 'backward'>('forward')
const advancingOptionId = ref('')
const hoveredRating = ref<number | null>(null)
const answers = ref<SurveyAnswers>({})
const submitting = ref(false)
const submitResult = ref<SubmitSurveyResult | null>(null)
const pickedMicroId = ref('')
const microAnimatedPercents = ref<Record<string, number>>({})
const validationError = ref('')
const copiedCode = ref(false)
const redeemingInline = ref(false)
const redeemedInline = ref(false)
const previewMinimized = ref(false)
const startedAt = ref(Date.now())

let autoAdvanceTimer: ReturnType<typeof setTimeout> | null = null
let microAnimRaf: number | null = null

const ratingScale = [
  { score: 1, labelKey: 'surveys.ratingScale1' },
  { score: 2, labelKey: 'surveys.ratingScale2' },
  { score: 3, labelKey: 'surveys.ratingScale3' },
  { score: 4, labelKey: 'surveys.ratingScale4' },
  { score: 5, labelKey: 'surveys.ratingScale5' }
] as const

function getRatingCardClasses(selected: boolean, score: number): string {
  if (selected) {
    if (score === 1) {
      return 'border-rose-500 bg-gradient-to-b from-rose-500 to-red-600 text-white shadow-lg shadow-rose-500/35'
    }
    if (score === 2) {
      return 'border-amber-500 bg-gradient-to-b from-amber-500 to-orange-600 text-white shadow-lg shadow-amber-500/35'
    }
    if (score === 3) {
      return 'border-slate-700 bg-gradient-to-b from-slate-700 to-slate-900 text-white shadow-lg shadow-slate-700/30 dark:border-slate-500 dark:from-slate-600 dark:to-slate-800'
    }
    if (score === 4) {
      return 'border-primary-500 bg-gradient-to-b from-primary-500 to-primary-600 text-white shadow-lg shadow-primary-500/40'
    }
    return 'border-primary-500 bg-gradient-to-b from-primary-500 via-sky-500 to-emerald-500 text-white shadow-lg shadow-sky-500/40'
  }
  if (score === 1) {
    return 'border-slate-200/90 bg-slate-50/70 text-slate-700 hover:border-rose-400/80 hover:bg-rose-50/70 hover:text-rose-700 dark:border-white/[0.08] dark:bg-white/[0.03] dark:text-slate-200 dark:hover:border-rose-400/50 dark:hover:bg-rose-500/15 dark:hover:text-rose-200'
  }
  if (score === 2) {
    return 'border-slate-200/90 bg-slate-50/70 text-slate-700 hover:border-amber-400/80 hover:bg-amber-50/70 hover:text-amber-700 dark:border-white/[0.08] dark:bg-white/[0.03] dark:text-slate-200 dark:hover:border-amber-400/50 dark:hover:bg-amber-500/15 dark:hover:text-amber-200'
  }
  if (score === 5) {
    return 'border-slate-200/90 bg-slate-50/70 text-slate-700 hover:border-emerald-400/80 hover:bg-emerald-50/70 hover:text-emerald-700 dark:border-white/[0.08] dark:bg-white/[0.03] dark:text-slate-200 dark:hover:border-emerald-400/50 dark:hover:bg-emerald-500/15 dark:hover:text-emerald-200'
  }
  return 'border-slate-200/90 bg-slate-50/70 text-slate-700 hover:border-primary-400/80 hover:bg-primary-50/70 hover:text-primary-700 dark:border-white/[0.08] dark:bg-white/[0.03] dark:text-slate-200 dark:hover:border-primary-400/55 dark:hover:bg-primary-500/15 dark:hover:text-primary-200'
}

const activeSurvey = computed<UserSurvey | null>(() =>
  props.preview ? props.survey : surveyStore.currentSurvey
)

const isBlockedByOtherModal = computed(() => {
  if (props.preview) return false
  if (route.path.startsWith('/admin')) return true
  if (adminComplianceStore.shouldShow) return true
  if (
    activeSurvey.value?.skip_if_announcement_unread &&
    (announcementStore.loading || announcementStore.currentPopup)
  ) {
    return true
  }
  return false
})

const isMinimized = computed(() =>
  props.preview ? previewMinimized.value : surveyStore.minimized
)

const shouldRenderPopup = computed(
  () => !!activeSurvey.value && !isBlockedByOtherModal.value && !isMinimized.value
)

const shouldShowMinimizedPill = computed(
  () => !!activeSurvey.value && !isBlockedByOtherModal.value && isMinimized.value && !submitResult.value
)

const totalSteps = computed(() => activeSurvey.value?.questions?.length || 1)
const currentQuestion = computed<SurveyQuestion | null>(
  () => activeSurvey.value?.questions?.[stepIndex.value] ?? null
)
const isLastStep = computed(() => stepIndex.value >= totalSteps.value - 1)

const dockRingDashOffset = computed(() => {
  const ratio = Math.min(1, Math.max(0.15, (stepIndex.value + 1) / Math.max(1, totalSteps.value)))
  return 59.69 * (1 - ratio)
})

const hasCurrentStepAnswer = computed(() => {
  const q = currentQuestion.value
  if (!q) return false
  const val = answers.value[q.id]
  if (!val) return false
  if (q.type === 'single') return !!val.choice_id
  if (q.type === 'multi') return (val.choice_ids?.length || 0) > 0
  if (q.type === 'rating') return typeof val.rating === 'number'
  if (q.type === 'tags_text') return (val.tags?.length || 0) > 0 || !!(val.text && val.text.trim())
  return false
})

const activeTagsMeta = computed(() => {
  const q = currentQuestion.value
  if (!q || q.type !== 'tags_text') {
    return { hasLinkedScore: false, isLow: false, title: '', chips: [] as string[] }
  }
  const threshold = q.low_score_threshold || 3
  let linkedScore: number | undefined
  if (q.linked_rating_question_id) {
    linkedScore = answers.value[q.linked_rating_question_id]?.rating
  } else {
    const firstRatingQ = activeSurvey.value?.questions.find((item) => item.type === 'rating')
    if (firstRatingQ) {
      linkedScore = answers.value[firstRatingQ.id]?.rating
    }
  }
  const hasLinkedScore = typeof linkedScore === 'number'
  const isLow = hasLinkedScore && (linkedScore as number) <= threshold
  const lowChips = q.low_chips || []
  const highChips = q.high_chips || []
  const chips = isLow
    ? (lowChips.length > 0 ? lowChips : highChips)
    : (highChips.length > 0 ? highChips : lowChips)
  const title = isLow
    ? (q.low_title || q.title)
    : (hasLinkedScore ? (q.high_title || q.title) : q.title)
  return { hasLinkedScore, isLow, title, chips }
})

const microDisplayStats = computed<MicroPollOptionStat[]>(() => {
  if (submitResult.value?.micro_results?.length) {
    return submitResult.value.micro_results
  }
  return []
})

const microTotalVotes = computed(() =>
  microDisplayStats.value.reduce((acc, row) => acc + row.count, 0)
)

const microTopPercent = computed(() =>
  microDisplayStats.value.reduce((max, row) => Math.max(max, row.percent), 0)
)

const wrapperClasses = computed(() => {
  const mode = activeSurvey.value?.display_mode || 'corner'
  if (mode === 'modal') {
    return 'fixed inset-0 z-[115] flex items-end justify-center bg-slate-950/55 p-0 backdrop-blur-[8px] sm:items-center sm:p-5 dark:bg-black/75'
  }
  return 'fixed inset-0 z-[115] flex items-end justify-center bg-slate-950/45 p-0 backdrop-blur-[5px] sm:pointer-events-none sm:items-end sm:justify-end sm:bg-transparent sm:p-6 sm:backdrop-blur-none'
})

const bezelClasses = computed(() => {
  const mode = activeSurvey.value?.display_mode || 'corner'
  const widthClass =
    mode === 'modal'
      ? 'w-full sm:max-w-[512px]'
      : mode === 'micro'
        ? 'w-full sm:w-[382px]'
        : 'w-full sm:w-[418px]'
  return [
    widthClass,
    'survey-bezel pointer-events-auto rounded-t-[29px] p-[1px] sm:rounded-[25px]'
  ].join(' ')
})

const panelClasses = computed(() =>
  'survey-shell relative flex max-h-[88vh] w-full flex-col overflow-hidden rounded-t-[28px] bg-white sm:max-h-[84vh] sm:rounded-[24px] dark:bg-[#090f1d]'
)

function formatRewardAmount(val: number): string {
  return Number(val || 0).toFixed(2).replace(/\.00$/, '')
}

function formatStepIndex(n: number): string {
  return n < 10 ? `0${n}` : String(n)
}

function formatMicroPercent(optionId: string, fallback: number): string {
  const val = optionId in microAnimatedPercents.value ? microAnimatedPercents.value[optionId] : fallback
  return Number(val.toFixed(1)).toString()
}

function onShellPointerMove(e: PointerEvent) {
  if (e.pointerType !== 'mouse') return
  const el = e.currentTarget as HTMLElement | null
  if (!el) return
  const rect = el.getBoundingClientRect()
  el.style.setProperty('--mx', `${Math.round(e.clientX - rect.left)}px`)
  el.style.setProperty('--my', `${Math.round(e.clientY - rect.top)}px`)
  el.style.setProperty('--spot-opacity', '1')
}

function onShellPointerLeave(e: PointerEvent) {
  const el = e.currentTarget as HTMLElement | null
  if (!el) return
  el.style.setProperty('--spot-opacity', '0')
}

function stopMicroAnim() {
  if (microAnimRaf !== null && typeof cancelAnimationFrame === 'function') {
    cancelAnimationFrame(microAnimRaf)
    microAnimRaf = null
  }
}

function startMicroPercentAnimation(rows: MicroPollOptionStat[]) {
  stopMicroAnim()
  if (!rows.length) {
    microAnimatedPercents.value = {}
    return
  }
  const reduced =
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (reduced || typeof requestAnimationFrame !== 'function') {
    const finalMap: Record<string, number> = {}
    for (const r of rows) finalMap[r.option_id] = r.percent
    microAnimatedPercents.value = finalMap
    return
  }
  const duration = 420
  const start = performance.now()
  const tick = (now: number) => {
    const progress = Math.min(1, (now - start) / duration)
    const eased = 1 - Math.pow(1 - progress, 3)
    const next: Record<string, number> = {}
    for (const r of rows) {
      next[r.option_id] = Math.round(r.percent * eased * 10) / 10
    }
    microAnimatedPercents.value = next
    if (progress < 1) {
      microAnimRaf = requestAnimationFrame(tick)
    } else {
      microAnimRaf = null
    }
  }
  microAnimRaf = requestAnimationFrame(tick)
}

watch(microDisplayStats, (rows) => {
  startMicroPercentAnimation(rows)
})

function clearAutoAdvance() {
  if (autoAdvanceTimer) {
    clearTimeout(autoAdvanceTimer)
    autoAdvanceTimer = null
  }
  advancingOptionId.value = ''
}

function resetLocalState() {
  clearAutoAdvance()
  stopMicroAnim()
  stepIndex.value = 0
  stepDirection.value = 'forward'
  hoveredRating.value = null
  answers.value = {}
  submitting.value = false
  submitResult.value = null
  pickedMicroId.value = ''
  microAnimatedPercents.value = {}
  validationError.value = ''
  copiedCode.value = false
  redeemingInline.value = false
  redeemedInline.value = false
  previewMinimized.value = false
  startedAt.value = Date.now()
}

watch(
  () => activeSurvey.value?.id,
  () => {
    resetLocalState()
  },
  { immediate: true }
)

function selectSingle(q: SurveyQuestion, optionId: string) {
  validationError.value = ''
  answers.value = {
    ...answers.value,
    [q.id]: { choice_id: optionId }
  }
  clearAutoAdvance()
  if (q.auto_advance !== false && !isLastStep.value) {
    advancingOptionId.value = `${q.id}:${optionId}`
    autoAdvanceTimer = setTimeout(() => {
      advancingOptionId.value = ''
      if (currentQuestion.value?.id === q.id && !isLastStep.value) {
        stepDirection.value = 'forward'
        stepIndex.value++
      }
    }, 280)
  }
}

function isMultiSelected(qId: string, optId: string): boolean {
  return Boolean(answers.value[qId]?.choice_ids?.includes(optId))
}

function toggleMulti(qId: string, optId: string) {
  validationError.value = ''
  const existing = [...(answers.value[qId]?.choice_ids || [])]
  const idx = existing.indexOf(optId)
  if (idx >= 0) existing.splice(idx, 1)
  else existing.push(optId)
  answers.value = {
    ...answers.value,
    [qId]: { choice_ids: existing }
  }
}

function selectRating(q: SurveyQuestion, score: number) {
  validationError.value = ''
  const prevScore = answers.value[q.id]?.rating
  answers.value = {
    ...answers.value,
    [q.id]: { rating: score }
  }
  if (typeof prevScore === 'number' && activeSurvey.value) {
    for (const item of activeSurvey.value.questions) {
      if (item.type !== 'tags_text') continue
      const threshold = item.low_score_threshold || 3
      const prevLow = prevScore <= threshold
      const nextLow = score <= threshold
      if (prevLow !== nextLow && answers.value[item.id]?.tags?.length) {
        answers.value = {
          ...answers.value,
          [item.id]: { ...answers.value[item.id], tags: [] }
        }
      }
    }
  }
  clearAutoAdvance()
  if (!isLastStep.value) {
    advancingOptionId.value = `${q.id}:rate-${score}`
    autoAdvanceTimer = setTimeout(() => {
      advancingOptionId.value = ''
      if (currentQuestion.value?.id === q.id && !isLastStep.value) {
        stepDirection.value = 'forward'
        hoveredRating.value = null
        stepIndex.value++
      }
    }, 360)
  }
}

function isTagSelected(qId: string, tag: string): boolean {
  return Boolean(answers.value[qId]?.tags?.includes(tag))
}

function toggleTag(qId: string, tag: string) {
  validationError.value = ''
  const existing = [...(answers.value[qId]?.tags || [])]
  const idx = existing.indexOf(tag)
  if (idx >= 0) existing.splice(idx, 1)
  else existing.push(tag)
  answers.value = {
    ...answers.value,
    [qId]: { ...answers.value[qId], tags: existing }
  }
}

function onTextInput(qId: string, text: string) {
  validationError.value = ''
  answers.value = {
    ...answers.value,
    [qId]: { ...answers.value[qId], text }
  }
}

function validateCurrentStep(): boolean {
  const q = currentQuestion.value
  if (!q || !q.required) return true
  if (!hasCurrentStepAnswer.value) {
    validationError.value = t('surveys.requiredAlert')
    return false
  }
  return true
}

function goPrev() {
  clearAutoAdvance()
  validationError.value = ''
  hoveredRating.value = null
  if (stepIndex.value > 0) {
    stepDirection.value = 'backward'
    stepIndex.value--
  }
}

function goNext() {
  clearAutoAdvance()
  if (!validateCurrentStep()) return
  hoveredRating.value = null
  if (stepIndex.value < totalSteps.value - 1) {
    stepDirection.value = 'forward'
    stepIndex.value++
  }
}

function buildPreviewSubmitResult(choiceId?: string): SubmitSurveyResult {
  const s = activeSurvey.value!
  const microResults: MicroPollOptionStat[] = []
  if (s.display_mode === 'micro' && s.show_micro_results && s.questions[0]?.options?.length) {
    const opts = s.questions[0].options
    const fakeCounts = opts.map((opt, i) => 18 + ((i * 11 + 7) % 27) + (opt.id === choiceId ? 1 : 0))
    const total = fakeCounts.reduce((a, b) => a + b, 0)
    opts.forEach((opt, i) => {
      microResults.push({
        option_id: opt.id,
        label: opt.label,
        count: fakeCounts[i],
        percent: Math.round((fakeCounts[i] * 1000) / Math.max(1, total)) / 10
      })
    })
  }
  const hasReward = s.has_reward && s.reward_value > 0
  return {
    response_id: 0,
    survey_id: s.id,
    reward_status: !hasReward
      ? 'none'
      : s.reward_type === 'balance'
        ? 'granted'
        : 'code_issued',
    reward_type: hasReward ? s.reward_type : 'none',
    reward_value: hasReward ? s.reward_value : 0,
    reward_redeem_code: hasReward && s.reward_type === 'redeem_code' ? 'SRV-PREVIEW-DEMO' : undefined,
    micro_results: microResults,
    submitted_at: new Date().toISOString()
  }
}

async function submitMicroChoice(optionId: string) {
  if (!activeSurvey.value || submitting.value) return
  const q = activeSurvey.value.questions[0]
  pickedMicroId.value = optionId
  answers.value = { [q.id]: { choice_id: optionId } }

  if (props.preview) {
    submitResult.value = buildPreviewSubmitResult(optionId)
    return
  }

  try {
    submitting.value = true
    const duration = Math.max(1, Math.round((Date.now() - startedAt.value) / 1000))
    const res = await surveyStore.submitCurrent(answers.value, duration)
    if (res) {
      submitResult.value = res
      if (res.reward_status === 'granted') {
        authStore.refreshUser().catch(() => {})
      }
    }
  } catch (err: any) {
    appStore.showError(err?.response?.data?.message || t('surveys.submitFailed'))
  } finally {
    submitting.value = false
  }
}

async function handleSubmit() {
  clearAutoAdvance()
  if (!activeSurvey.value || submitting.value) return
  if (!validateCurrentStep()) return

  if (props.preview) {
    submitResult.value = buildPreviewSubmitResult()
    return
  }

  try {
    submitting.value = true
    const duration = Math.max(1, Math.round((Date.now() - startedAt.value) / 1000))
    const res = await surveyStore.submitCurrent(answers.value, duration)
    if (res) {
      submitResult.value = res
      if (res.reward_status === 'granted') {
        authStore.refreshUser().catch(() => {})
      }
    }
  } catch (err: any) {
    appStore.showError(err?.response?.data?.message || t('surveys.submitFailed'))
  } finally {
    submitting.value = false
  }
}

async function copyRewardCode(code: string) {
  try {
    await navigator.clipboard.writeText(code)
    copiedCode.value = true
    setTimeout(() => {
      copiedCode.value = false
    }, 2000)
  } catch {
    // fallback ignore
  }
}

async function redeemIssuedCodeNow(code: string, amount: number) {
  if (props.preview) {
    redeemedInline.value = true
    appStore.showSuccess(t('surveys.redeemSuccess', { amount: formatRewardAmount(amount) }))
    return
  }
  try {
    redeemingInline.value = true
    await redeemAPI.redeem(code)
    redeemedInline.value = true
    appStore.showSuccess(t('surveys.redeemSuccess', { amount: formatRewardAmount(amount) }))
    authStore.refreshUser().catch(() => {})
  } catch (err: any) {
    appStore.showError(err?.response?.data?.message || t('surveys.redeemFailed'))
  } finally {
    redeemingInline.value = false
  }
}

function minimizeToPill() {
  clearAutoAdvance()
  if (props.preview) {
    previewMinimized.value = true
    return
  }
  surveyStore.setMinimized(true)
}

function restoreFromMinimized() {
  if (props.preview) {
    previewMinimized.value = false
    return
  }
  surveyStore.setMinimized(false)
}

function handleBackdropClick() {
  if (submitResult.value) {
    finishAndClose()
    return
  }
  if (props.preview) {
    emit('close')
    return
  }
  if (activeSurvey.value?.display_mode === 'corner') {
    minimizeToPill()
  } else {
    handleSnooze()
  }
}

function handleCloseClick() {
  if (submitResult.value) {
    finishAndClose()
    return
  }
  if (props.preview) {
    emit('close')
    return
  }
  if (activeSurvey.value?.display_mode === 'corner') {
    minimizeToPill()
  } else {
    handleSnooze()
  }
}

async function handleSnooze() {
  clearAutoAdvance()
  const days = activeSurvey.value?.snooze_days || 3
  if (props.preview) {
    appStore.showInfo(t('surveys.snoozeToast', { days }))
    emit('close')
    return
  }
  await surveyStore.snoozeCurrent()
  appStore.showInfo(t('surveys.snoozeToast', { days }))
}

async function handleDismiss() {
  clearAutoAdvance()
  if (props.preview) {
    appStore.showInfo(t('surveys.dismissToast'))
    emit('close')
    return
  }
  await surveyStore.dismissCurrent()
  appStore.showInfo(t('surveys.dismissToast'))
}

function finishAndClose() {
  clearAutoAdvance()
  if (props.preview) {
    emit('close')
    return
  }
  surveyStore.closeAfterDone()
}

function handleKeydown(e: KeyboardEvent) {
  if (!shouldRenderPopup.value || !activeSurvey.value) return
  const tag = (document.activeElement?.tagName || '').toLowerCase()
  if (tag === 'textarea' || tag === 'input') return

  if (e.key === 'Escape') {
    handleCloseClick()
    return
  }
  if (submitResult.value) return

  const num = Number(e.key)
  if (activeSurvey.value.display_mode === 'micro' && activeSurvey.value.questions.length === 1) {
    const opts = activeSurvey.value.questions[0].options || []
    if (num >= 1 && num <= opts.length) {
      submitMicroChoice(opts[num - 1].id)
    }
    return
  }

  const q = currentQuestion.value
  if (!q) return
  if (q.type === 'single') {
    const opts = q.options || []
    if (num >= 1 && num <= opts.length) {
      selectSingle(q, opts[num - 1].id)
    }
  } else if (q.type === 'rating' && num >= 1 && num <= 5) {
    selectRating(q, num)
  } else if (e.key === 'Enter') {
    if (isLastStep.value) handleSubmit()
    else goNext()
  } else if (e.key === 'ArrowLeft' && stepIndex.value > 0) {
    goPrev()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  clearAutoAdvance()
  stopMicroAnim()
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
/* 双层金属精工外框：浅色微珍珠高光边 / 深色电光蓝玻璃高光边 */
.survey-bezel {
  background: linear-gradient(
    145deg,
    rgba(255, 255, 255, 0.95) 0%,
    rgba(148, 163, 184, 0.28) 45%,
    rgba(37, 99, 235, 0.2) 100%
  );
  box-shadow:
    0 28px 68px -16px rgba(15, 23, 42, 0.22),
    0 10px 28px -8px rgba(37, 99, 235, 0.14);
}

:global(.dark) .survey-bezel {
  background: linear-gradient(
    145deg,
    rgba(147, 197, 253, 0.34) 0%,
    rgba(255, 255, 255, 0.12) 38%,
    rgba(37, 99, 235, 0.14) 100%
  );
  box-shadow:
    0 36px 84px -14px rgba(0, 0, 0, 0.8),
    0 14px 36px -10px rgba(37, 99, 235, 0.28);
}

.survey-shell {
  --mx: 50%;
  --my: 0%;
  --spot-opacity: 0;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.9);
}

:global(.dark) .survey-shell {
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.08);
}

/* 顶部大气极光晕 */
.survey-aurora {
  background:
    radial-gradient(ellipse 80% 95% at 14% -20%, rgba(37, 99, 235, 0.14), transparent 72%),
    radial-gradient(ellipse 65% 80% at 88% -15%, rgba(14, 165, 233, 0.11), transparent 70%);
}

:global(.dark) .survey-aurora {
  background:
    radial-gradient(ellipse 80% 95% at 14% -20%, rgba(59, 130, 246, 0.28), transparent 72%),
    radial-gradient(ellipse 65% 80% at 88% -15%, rgba(56, 189, 248, 0.18), transparent 70%);
}

/* 细微建筑点阵纹理，自然渐隐 */
.survey-dot-grid {
  background-image: radial-gradient(rgba(15, 23, 42, 0.08) 1px, transparent 1px);
  background-size: 14px 14px;
  mask-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.55) 0%, transparent 100%);
  -webkit-mask-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.55) 0%, transparent 100%);
}

:global(.dark) .survey-dot-grid {
  background-image: radial-gradient(rgba(226, 232, 240, 0.12) 1px, transparent 1px);
}

/* 随鼠标追踪的柔和聚光层 */
.survey-spotlight {
  opacity: var(--spot-opacity);
  transition: opacity 0.25s ease;
  background: radial-gradient(
    260px circle at var(--mx) var(--my),
    rgba(37, 99, 235, 0.06),
    transparent 70%
  );
}

:global(.dark) .survey-spotlight {
  background: radial-gradient(
    280px circle at var(--mx) var(--my),
    rgba(96, 165, 250, 0.11),
    transparent 70%
  );
}

/* Dynamic-Island 风格悬浮胶囊阴影 */
.survey-dock-pill {
  box-shadow:
    0 20px 42px -10px rgba(15, 23, 42, 0.45),
    0 6px 18px -4px rgba(37, 99, 235, 0.35);
}

/* 奖励微标流光 */
.survey-reward-pill {
  position: relative;
  overflow: hidden;
}

.survey-reward-pill::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(110deg, transparent 22%, rgba(255, 255, 255, 0.65) 50%, transparent 78%);
  transform: translateX(-150%);
  animation: surveySheen 4.4s ease-in-out infinite;
}

/* 礼券与账本卡对角金属微折光 */
.survey-voucher-foil::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(
    125deg,
    rgba(255, 255, 255, 0.35) 0%,
    transparent 38%,
    transparent 68%,
    rgba(56, 189, 248, 0.08) 100%
  );
}

:global(.dark) .survey-voucher-foil::before {
  background: linear-gradient(
    125deg,
    rgba(255, 255, 255, 0.07) 0%,
    transparent 42%,
    transparent 70%,
    rgba(56, 189, 248, 0.09) 100%
  );
}

@keyframes surveySheen {
  0%,
  68% {
    transform: translateX(-150%);
  }
  100% {
    transform: translateX(150%);
  }
}

/* 弹窗出入场物理弹性 */
.survey-pop-enter-active {
  transition:
    opacity 0.26s ease-out,
    transform 0.34s cubic-bezier(0.16, 1, 0.3, 1);
}
.survey-pop-leave-active {
  transition:
    opacity 0.18s ease-in,
    transform 0.18s ease-in;
}
.survey-pop-enter-from,
.survey-pop-leave-to {
  opacity: 0;
  transform: translateY(18px) scale(0.96);
}

/* 方向感知切题动效 */
@keyframes surveyForwardIn {
  from {
    opacity: 0;
    transform: translateX(16px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
@keyframes surveyBackwardIn {
  from {
    opacity: 0;
    transform: translateX(-16px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
.survey-step-forward {
  animation: surveyForwardIn 0.26s cubic-bezier(0.22, 1, 0.36, 1) both;
}
.survey-step-backward {
  animation: surveyBackwardIn 0.26s cubic-bezier(0.22, 1, 0.36, 1) both;
}

/* 选项逐项微错峰弹入 */
@keyframes surveyStaggerIn {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
.survey-stagger-item {
  animation: surveyStaggerIn 0.28s cubic-bezier(0.22, 1, 0.36, 1) var(--stagger, 0ms) both;
}

/* 选中自动跳转底部流光确认条 */
@keyframes surveyAdvanceSweep {
  from {
    transform: scaleX(0);
    transform-origin: left;
  }
  to {
    transform: scaleX(1);
    transform-origin: left;
  }
}
.survey-advance-sweep {
  animation: surveyAdvanceSweep 0.28s cubic-bezier(0.22, 1, 0.36, 1) both;
}

/* 校验未通过轻抖提示 */
@keyframes surveyShake {
  0%, 100% { transform: translateX(0); }
  20%, 60% { transform: translateX(-4px); }
  40%, 80% { transform: translateX(4px); }
}
.survey-shake {
  animation: surveyShake 0.32s ease-in-out both;
}

/* 投票结果条生长 */
@keyframes surveyBarGrow {
  from {
    width: 0%;
  }
}
.survey-bar-grow {
  animation: surveyBarGrow 0.52s cubic-bezier(0.16, 1, 0.3, 1) both;
}

/* 完成态 SVG 勾画与双层光环庆祝 */
@keyframes surveyCheckDraw {
  from {
    stroke-dasharray: 28;
    stroke-dashoffset: 28;
  }
  to {
    stroke-dasharray: 28;
    stroke-dashoffset: 0;
  }
}
.survey-check-draw {
  animation: surveyCheckDraw 0.44s cubic-bezier(0.22, 1, 0.36, 1) 0.12s both;
}

@keyframes surveyRingPop {
  0% {
    transform: scale(0.7);
    opacity: 0.95;
  }
  100% {
    transform: scale(1.45);
    opacity: 0;
  }
}
.survey-celebrate-ring {
  animation: surveyRingPop 1.25s cubic-bezier(0.16, 1, 0.3, 1) 0.04s both;
}
.survey-celebrate-ring-2 {
  animation: surveyRingPop 1.35s cubic-bezier(0.16, 1, 0.3, 1) 0.18s both;
}

.survey-confetti {
  position: absolute;
  border-radius: 9999px;
  opacity: 0;
}
.survey-confetti-1 {
  width: 6px; height: 6px; top: 4px; left: 10px; background: #3b82f6;
  animation: surveyConfetti1 0.85s cubic-bezier(0.16, 1, 0.3, 1) 0.08s both;
}
.survey-confetti-2 {
  width: 5px; height: 5px; top: 6px; right: 8px; background: #10b981;
  animation: surveyConfetti2 0.85s cubic-bezier(0.16, 1, 0.3, 1) 0.12s both;
}
.survey-confetti-3 {
  width: 5px; height: 5px; bottom: 8px; left: 6px; background: #38bdf8;
  animation: surveyConfetti3 0.85s cubic-bezier(0.16, 1, 0.3, 1) 0.15s both;
}
.survey-confetti-4 {
  width: 6px; height: 6px; bottom: 4px; right: 12px; background: #34d399;
  animation: surveyConfetti4 0.85s cubic-bezier(0.16, 1, 0.3, 1) 0.18s both;
}
.survey-confetti-5 {
  width: 4px; height: 4px; top: 50%; left: 0px; background: #818cf8;
  animation: surveyConfetti5 0.85s cubic-bezier(0.16, 1, 0.3, 1) 0.14s both;
}
.survey-confetti-6 {
  width: 4px; height: 4px; top: 45%; right: 0px; background: #fbbf24;
  animation: surveyConfetti6 0.85s cubic-bezier(0.16, 1, 0.3, 1) 0.16s both;
}

@keyframes surveyConfetti1 {
  0% { opacity: 1; transform: translate(18px, 18px) scale(0.3); }
  100% { opacity: 0; transform: translate(-14px, -16px) scale(1.1); }
}
@keyframes surveyConfetti2 {
  0% { opacity: 1; transform: translate(-16px, 16px) scale(0.3); }
  100% { opacity: 0; transform: translate(16px, -14px) scale(1.1); }
}
@keyframes surveyConfetti3 {
  0% { opacity: 1; transform: translate(16px, -14px) scale(0.3); }
  100% { opacity: 0; transform: translate(-16px, 14px) scale(1.1); }
}
@keyframes surveyConfetti4 {
  0% { opacity: 1; transform: translate(-14px, -16px) scale(0.3); }
  100% { opacity: 0; transform: translate(14px, 16px) scale(1.1); }
}
@keyframes surveyConfetti5 {
  0% { opacity: 1; transform: translate(20px, 0) scale(0.3); }
  100% { opacity: 0; transform: translate(-18px, -4px) scale(1); }
}
@keyframes surveyConfetti6 {
  0% { opacity: 1; transform: translate(-20px, 0) scale(0.3); }
  100% { opacity: 0; transform: translate(18px, 4px) scale(1); }
}

@media (prefers-reduced-motion: reduce) {
  .survey-pop-enter-active,
  .survey-pop-leave-active,
  .survey-step-forward,
  .survey-step-backward,
  .survey-stagger-item,
  .survey-advance-sweep,
  .survey-shake,
  .survey-bar-grow,
  .survey-check-draw,
  .survey-celebrate-ring,
  .survey-celebrate-ring-2,
  .survey-confetti,
  .survey-reward-pill::after {
    animation: none !important;
    transition: none !important;
  }
  .survey-spotlight {
    display: none;
  }
}
</style>
