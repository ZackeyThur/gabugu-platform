<template>
  <BaseDialog
    :show="show"
    :title="isEditing ? t('surveys.admin.editSurvey') : t('surveys.admin.createSurvey')"
    width="extra-wide"
    @close="emit('close')"
  >
    <form id="survey-editor-form" class="space-y-6" @submit.prevent="handleSubmit">
      <!-- 一键套用交互模板：精工卡片选择 -->
      <div
        v-if="!isEditing"
        class="space-y-2.5 rounded-2xl border border-primary-200/80 bg-gradient-to-br from-primary-50/80 via-white to-sky-50/50 p-4 dark:border-primary-500/20 dark:from-primary-500/15 dark:via-dark-800/80 dark:to-sky-500/8"
      >
        <div class="flex items-center gap-2 text-xs font-semibold text-primary-700 dark:text-primary-300">
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2l2.4 6.6L21 11l-6.6 2.4L12 20l-2.4-6.6L3 11l6.6-2.4L12 2z" />
          </svg>
          <span>{{ t('surveys.admin.templates.label') }}</span>
        </div>
        <div class="grid grid-cols-1 gap-2.5 md:grid-cols-3">
          <button
            type="button"
            data-testid="template-daily-btn"
            class="group flex items-center gap-3 rounded-xl border px-3.5 py-2.5 text-left transition-all"
            :class="
              activeTemplate === 'daily'
                ? 'border-primary-500 bg-white shadow-sm ring-2 ring-primary-500/20 dark:border-primary-400 dark:bg-dark-800'
                : 'border-gray-200/90 bg-white/80 hover:border-primary-400/70 hover:bg-white dark:border-dark-600 dark:bg-dark-900/50 dark:hover:border-primary-400/50'
            "
            @click="applyTemplate('daily')"
          >
            <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-primary-500/15 text-primary-600 dark:bg-primary-500/20 dark:text-primary-300">
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
              </svg>
            </span>
            <span class="text-xs font-semibold leading-snug text-gray-800 dark:text-gray-100">
              {{ t('surveys.admin.templates.daily') }}
            </span>
          </button>

          <button
            type="button"
            data-testid="template-rewarded-btn"
            class="group flex items-center gap-3 rounded-xl border px-3.5 py-2.5 text-left transition-all"
            :class="
              activeTemplate === 'rewarded'
                ? 'border-emerald-500 bg-white shadow-sm ring-2 ring-emerald-500/20 dark:border-emerald-400 dark:bg-dark-800'
                : 'border-gray-200/90 bg-white/80 hover:border-emerald-400/70 hover:bg-white dark:border-dark-600 dark:bg-dark-900/50 dark:hover:border-emerald-400/50'
            "
            @click="applyTemplate('rewarded')"
          >
            <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-emerald-500/15 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-300">
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M20 12v8a2 2 0 01-2 2H6a2 2 0 01-2-2v-8m16 0H4m16 0a2 2 0 002-2V9a2 2 0 00-2-2h-3.17a3 3 0 00.17-1 3 3 0 00-5.83-1 3 3 0 00-5.83 1c0 .35.06.68.17 1H4a2 2 0 00-2 2v1a2 2 0 002 2m8-5v15" />
              </svg>
            </span>
            <span class="text-xs font-semibold leading-snug text-gray-800 dark:text-gray-100">
              {{ t('surveys.admin.templates.rewarded') }}
            </span>
          </button>

          <button
            type="button"
            data-testid="template-micro-btn"
            class="group flex items-center gap-3 rounded-xl border px-3.5 py-2.5 text-left transition-all"
            :class="
              activeTemplate === 'micro'
                ? 'border-sky-500 bg-white shadow-sm ring-2 ring-sky-500/20 dark:border-sky-400 dark:bg-dark-800'
                : 'border-gray-200/90 bg-white/80 hover:border-sky-400/70 hover:bg-white dark:border-dark-600 dark:bg-dark-900/50 dark:hover:border-sky-400/50'
            "
            @click="applyTemplate('micro')"
          >
            <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-sky-500/15 text-sky-600 dark:bg-sky-500/20 dark:text-sky-300">
              <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                <path d="M13 2L4.5 13.5h6L9.5 22 19.5 10.5h-6L13 2z" />
              </svg>
            </span>
            <span class="text-xs font-semibold leading-snug text-gray-800 dark:text-gray-100">
              {{ t('surveys.admin.templates.micro') }}
            </span>
          </button>
        </div>
      </div>

      <!-- 一、基本信息与弹窗形态 -->
      <section class="space-y-4 rounded-2xl border border-gray-200/90 bg-white p-5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.08)] dark:border-dark-700 dark:bg-dark-900/30">
        <div class="flex items-center gap-2.5">
          <span class="inline-flex h-6 w-6 items-center justify-center rounded-lg bg-slate-900 font-mono text-xs font-bold text-white dark:bg-white dark:text-slate-900">01</span>
          <h4 class="text-sm font-bold tracking-tight text-gray-900 dark:text-white">
            {{ t('surveys.admin.form.sectionBasic') }}
          </h4>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label for="survey-title-input" class="input-label">
              {{ t('surveys.admin.form.title') }}
            </label>
            <input
              id="survey-title-input"
              v-model="form.title"
              type="text"
              class="input"
              maxlength="200"
              required
              :placeholder="t('surveys.admin.form.titlePlaceholder')"
            />
          </div>
          <div>
            <label class="input-label">{{ t('surveys.admin.form.status') }}</label>
            <Select v-model="form.status" :options="statusOptions" />
          </div>
        </div>

        <div>
          <label for="survey-desc-input" class="input-label">
            {{ t('surveys.admin.form.description') }}
          </label>
          <input
            id="survey-desc-input"
            v-model="form.description"
            type="text"
            class="input"
            maxlength="500"
            :placeholder="t('surveys.admin.form.descriptionPlaceholder')"
          />
        </div>

        <!-- 弹出形态可视化示意图选择器 -->
        <div>
          <label class="input-label">{{ t('surveys.admin.form.displayMode') }}</label>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
            <button
              v-for="mode in displayModes"
              :key="mode"
              type="button"
              :data-testid="`display-mode-${mode}`"
              class="group flex flex-col gap-2.5 rounded-2xl border p-3.5 text-left transition-all"
              :class="
                form.display_mode === mode
                  ? 'border-primary-500 bg-primary-50/75 shadow-sm ring-2 ring-primary-500/20 dark:border-primary-400 dark:bg-primary-900/20'
                  : 'border-gray-200 bg-white hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-primary-500/40'
              "
              @click="selectDisplayMode(mode)"
            >
              <!-- 迷你浏览器窗口落位示意 SVG -->
              <div
                class="relative h-16 w-full overflow-hidden rounded-xl border transition-colors"
                :class="
                  form.display_mode === mode
                    ? 'border-primary-200 bg-white dark:border-primary-500/30 dark:bg-slate-900'
                    : 'border-gray-200/80 bg-gray-50/90 dark:border-dark-700 dark:bg-slate-950/60'
                "
              >
                <div class="flex items-center gap-1 border-b border-gray-200/70 px-2 py-1 dark:border-white/5">
                  <span class="h-1.5 w-1.5 rounded-full bg-gray-300 dark:bg-slate-600"></span>
                  <span class="h-1.5 w-1.5 rounded-full bg-gray-300 dark:bg-slate-600"></span>
                  <span class="h-1.5 w-1.5 rounded-full bg-gray-300 dark:bg-slate-600"></span>
                </div>
                <!-- 背景骨架线 -->
                <div class="space-y-1.5 p-2 opacity-40">
                  <div class="h-1.5 w-1/2 rounded bg-gray-300 dark:bg-slate-600"></div>
                  <div class="h-1.5 w-2/3 rounded bg-gray-200 dark:bg-slate-700"></div>
                </div>
                <!-- 右下角轻浮窗示意 -->
                <div
                  v-if="mode === 'corner'"
                  class="absolute bottom-1.5 right-1.5 h-8 w-14 rounded-md border border-primary-500/50 bg-gradient-to-br from-primary-500 to-sky-500 p-1 shadow-md shadow-primary-500/30"
                >
                  <div class="h-1 w-6 rounded bg-white/90"></div>
                  <div class="mt-1 h-1 w-10 rounded bg-white/50"></div>
                  <div class="mt-1 h-1 w-8 rounded bg-white/50"></div>
                </div>
                <!-- 居中焦点弹窗示意 -->
                <template v-else-if="mode === 'modal'">
                  <div class="absolute inset-0 top-3.5 bg-slate-900/20 dark:bg-black/45"></div>
                  <div class="absolute left-1/2 top-1/2 h-9 w-20 -translate-x-1/2 -translate-y-1/3 rounded-md border border-primary-400/60 bg-gradient-to-br from-primary-500 to-indigo-500 p-1.5 shadow-lg shadow-primary-500/35">
                    <div class="h-1 w-8 rounded bg-white/90"></div>
                    <div class="mt-1 h-1 w-full rounded bg-white/55"></div>
                    <div class="mt-1 h-1 w-10 rounded bg-white/55"></div>
                  </div>
                </template>
                <!-- 1 题快问示意 -->
                <div
                  v-else
                  class="absolute bottom-1.5 right-1.5 flex h-6 w-16 items-center gap-1 rounded-full border border-sky-400/60 bg-gradient-to-r from-sky-500 to-primary-600 px-2 shadow-md shadow-sky-500/30"
                >
                  <span class="h-2 w-2 rounded-full bg-white"></span>
                  <span class="h-1 flex-1 rounded bg-white/80"></span>
                </div>
              </div>

              <div>
                <div class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t(`surveys.admin.displayMode.${mode}`) }}
                </div>
                <p class="mt-0.5 text-xs leading-relaxed text-gray-500 dark:text-dark-400">
                  {{ t(`surveys.admin.displayModeDesc.${mode}`) }}
                </p>
              </div>
            </button>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('surveys.admin.form.startsAt') }}</label>
            <input v-model="form.starts_at_str" type="datetime-local" max="9999-12-31T23:59" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('surveys.admin.form.endsAt') }}</label>
            <input v-model="form.ends_at_str" type="datetime-local" max="9999-12-31T23:59" class="input" />
          </div>
        </div>
      </section>

      <!-- 二、题目设计 -->
      <section class="space-y-4 rounded-2xl border border-gray-200/90 bg-white p-5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.08)] dark:border-dark-700 dark:bg-dark-900/30">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2.5">
            <span class="inline-flex h-6 w-6 items-center justify-center rounded-lg bg-slate-900 font-mono text-xs font-bold text-white dark:bg-white dark:text-slate-900">02</span>
            <h4 class="text-sm font-bold tracking-tight text-gray-900 dark:text-white">
              {{ t('surveys.admin.form.sectionQuestions') }} ({{ form.questions.length }})
            </h4>
          </div>
          <button
            v-if="form.display_mode !== 'micro'"
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="form.questions.length >= 10"
            @click="addQuestion"
          >
            + {{ t('surveys.admin.form.addQuestion') }}
          </button>
        </div>

        <div class="space-y-4">
          <div
            v-for="(q, qIdx) in form.questions"
            :key="q.id"
            class="space-y-3 rounded-xl border border-gray-200 bg-pearl-bg/60 p-4 dark:border-dark-700 dark:bg-dark-900/35"
          >
            <div class="flex flex-wrap items-center justify-between gap-2">
              <span class="badge badge-primary">Q{{ qIdx + 1 }}</span>
              <div class="flex items-center gap-1.5">
                <button
                  v-if="form.display_mode !== 'micro'"
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="qIdx === 0"
                  @click="moveQuestion(qIdx, -1)"
                >
                  ↑ {{ t('surveys.admin.form.moveUp') }}
                </button>
                <button
                  v-if="form.display_mode !== 'micro'"
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="qIdx === form.questions.length - 1"
                  @click="moveQuestion(qIdx, 1)"
                >
                  ↓ {{ t('surveys.admin.form.moveDown') }}
                </button>
                <button
                  v-if="form.display_mode !== 'micro' && form.questions.length > 1"
                  type="button"
                  class="btn btn-secondary btn-sm text-red-600 dark:text-red-400"
                  @click="removeQuestion(qIdx)"
                >
                  {{ t('surveys.admin.form.removeQuestion') }}
                </button>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
              <div class="md:col-span-2">
                <label class="input-label">{{ t('surveys.admin.form.questionTitle') }}</label>
                <input v-model="q.title" type="text" class="input" maxlength="200" required />
              </div>
              <div>
                <label class="input-label">{{ t('surveys.admin.form.questionType') }}</label>
                <Select
                  :model-value="q.type"
                  :options="questionTypeOptions"
                  :disabled="form.display_mode === 'micro'"
                  @update:model-value="(val) => onQuestionTypeChange(q, val as SurveyQuestionType)"
                />
              </div>
            </div>

            <div class="flex flex-wrap items-center gap-4 text-xs text-gray-700 dark:text-gray-300">
              <label class="flex cursor-pointer items-center gap-2">
                <input v-model="q.required" type="checkbox" class="h-4 w-4 rounded" />
                <span>{{ t('surveys.admin.form.questionRequired') }}</span>
              </label>
              <label
                v-if="q.type === 'single' && form.display_mode !== 'micro'"
                class="flex cursor-pointer items-center gap-2"
              >
                <input v-model="q.auto_advance" type="checkbox" class="h-4 w-4 rounded" />
                <span>{{ t('surveys.admin.form.questionAutoAdvance') }}</span>
              </label>
            </div>

            <!-- 单选 / 多选 选项列表 -->
            <div v-if="q.type === 'single' || q.type === 'multi'" class="space-y-2">
              <div class="flex items-center justify-between">
                <label class="input-label !mb-0">{{ t('surveys.admin.form.optionsLabel') }}</label>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="(q.options?.length || 0) >= 12"
                  @click="addOption(q)"
                >
                  + {{ t('surveys.admin.form.addOption') }}
                </button>
              </div>
              <div
                v-for="(opt, oIdx) in q.options || []"
                :key="opt.id"
                class="grid grid-cols-1 gap-2 sm:grid-cols-12"
              >
                <input
                  v-model="opt.label"
                  type="text"
                  class="input sm:col-span-5"
                  maxlength="100"
                  required
                  :placeholder="t('surveys.admin.form.optionLabelPlaceholder')"
                />
                <input
                  v-if="q.type === 'single'"
                  v-model="opt.description"
                  type="text"
                  class="input sm:col-span-6"
                  maxlength="200"
                  :placeholder="t('surveys.admin.form.optionDescPlaceholder')"
                />
                <input
                  v-else
                  v-model="opt.tag"
                  type="text"
                  class="input sm:col-span-6"
                  maxlength="40"
                  :placeholder="t('surveys.admin.form.optionDescPlaceholder')"
                />
                <button
                  type="button"
                  class="btn btn-secondary btn-sm sm:col-span-1"
                  :disabled="(q.options?.length || 0) <= 2"
                  @click="removeOption(q, oIdx)"
                >
                  ×
                </button>
              </div>
            </div>

            <!-- tags_text 快捷标签与文本配置 -->
            <div v-if="q.type === 'tags_text'" class="space-y-3">
              <p class="text-xs text-gray-500 dark:text-dark-400">
                {{ t('surveys.admin.form.linkedRatingHint', { threshold: q.low_score_threshold || 3 }) }}
              </p>
              <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('surveys.admin.form.lowTitle') }}</label>
                  <input v-model="q.low_title" type="text" class="input" maxlength="200" />
                </div>
                <div>
                  <label class="input-label">{{ t('surveys.admin.form.highTitle') }}</label>
                  <input v-model="q.high_title" type="text" class="input" maxlength="200" />
                </div>
                <div>
                  <label class="input-label">{{ t('surveys.admin.form.lowChips') }}</label>
                  <textarea
                    :value="getChipDraft(q, 'low')"
                    rows="2"
                    class="input text-xs"
                    @input="onChipDraftInput(q, 'low', ($event.target as HTMLTextAreaElement).value)"
                    @blur="commitChipDraft(q, 'low')"
                  ></textarea>
                </div>
                <div>
                  <label class="input-label">{{ t('surveys.admin.form.highChips') }}</label>
                  <textarea
                    :value="getChipDraft(q, 'high')"
                    rows="2"
                    class="input text-xs"
                    @input="onChipDraftInput(q, 'high', ($event.target as HTMLTextAreaElement).value)"
                    @blur="commitChipDraft(q, 'high')"
                  ></textarea>
                </div>
              </div>
              <div>
                <label class="input-label">{{ t('surveys.admin.form.textPlaceholderLabel') }}</label>
                <input v-model="q.placeholder" type="text" class="input" maxlength="200" />
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- 三、奖励回馈配置 -->
      <section class="space-y-4 rounded-2xl border border-gray-200/90 bg-white p-5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.08)] dark:border-dark-700 dark:bg-dark-900/30">
        <div class="flex items-center gap-2.5">
          <span class="inline-flex h-6 w-6 items-center justify-center rounded-lg bg-slate-900 font-mono text-xs font-bold text-white dark:bg-white dark:text-slate-900">03</span>
          <h4 class="text-sm font-bold tracking-tight text-gray-900 dark:text-white">
            {{ t('surveys.admin.form.sectionReward') }}
          </h4>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
          <div>
            <label class="input-label">{{ t('surveys.admin.form.rewardType') }}</label>
            <Select v-model="form.reward_type" :options="rewardTypeOptions" />
          </div>

          <div v-if="form.reward_type !== 'none'">
            <label class="input-label">{{ t('surveys.admin.form.rewardValue') }}</label>
            <input
              v-model.number="form.reward_value"
              type="number"
              min="0.01"
              max="1000"
              step="0.01"
              class="input"
              required
            />
          </div>

          <div v-if="form.reward_type !== 'none'">
            <label class="input-label">{{ t('surveys.admin.form.rewardBudgetLimit') }}</label>
            <input
              v-model="form.reward_budget_limit_str"
              type="number"
              min="1"
              step="1"
              class="input"
              placeholder="200"
            />
          </div>

          <div v-if="form.reward_type === 'redeem_code'">
            <label class="input-label">{{ t('surveys.admin.form.rewardCodeValidDays') }}</label>
            <input
              v-model.number="form.reward_code_valid_days"
              type="number"
              min="1"
              max="365"
              step="1"
              class="input"
            />
          </div>
        </div>

        <div
          v-if="form.reward_type !== 'none'"
          class="space-y-1 rounded-xl border border-emerald-200/80 bg-emerald-50/70 px-3.5 py-2.5 text-xs text-emerald-800 dark:border-emerald-800/40 dark:bg-emerald-900/15 dark:text-emerald-300"
        >
          <div v-if="estimatedMaxBudget !== null" class="font-semibold">
            {{ t('surveys.admin.form.rewardMaxBudgetHint', { total: estimatedMaxBudget.toFixed(2) }) }}
          </div>
          <div>{{ t('surveys.admin.form.rewardGiftNote') }}</div>
        </div>
      </section>

      <!-- 四、定向人群与防打扰策略 -->
      <section class="space-y-4 rounded-2xl border border-gray-200/90 bg-white p-5 shadow-[0_2px_10px_-6px_rgba(15,23,42,0.08)] dark:border-dark-700 dark:bg-dark-900/30">
        <div class="flex items-center gap-2.5">
          <span class="inline-flex h-6 w-6 items-center justify-center rounded-lg bg-slate-900 font-mono text-xs font-bold text-white dark:bg-white dark:text-slate-900">04</span>
          <h4 class="text-sm font-bold tracking-tight text-gray-900 dark:text-white">
            {{ t('surveys.admin.form.sectionDelivery') }}
          </h4>
        </div>

        <AnnouncementTargetingEditor
          v-model="form.targeting"
          :groups="subscriptionGroups"
        />

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <label class="input-label">{{ t('surveys.admin.form.minAccountAgeDays') }}</label>
            <input v-model.number="form.delivery_policy.min_account_age_days" type="number" min="0" max="3650" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('surveys.admin.form.requireRecentUsageDays') }}</label>
            <input v-model.number="form.delivery_policy.require_recent_usage_days" type="number" min="0" max="365" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('surveys.admin.form.snoozeDays') }}</label>
            <input v-model.number="form.delivery_policy.snooze_days" type="number" min="1" max="90" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('surveys.admin.form.delaySeconds') }}</label>
            <input v-model.number="form.delivery_policy.delay_seconds" type="number" min="0" max="60" class="input" />
          </div>
        </div>

        <div class="flex flex-wrap gap-5 text-xs text-gray-700 dark:text-gray-300">
          <label class="flex cursor-pointer items-center gap-2">
            <input v-model="form.delivery_policy.skip_if_announcement_unread" type="checkbox" class="h-4 w-4 rounded" />
            <span>{{ t('surveys.admin.form.skipIfAnnouncementUnread') }}</span>
          </label>
          <label v-if="form.display_mode === 'micro'" class="flex cursor-pointer items-center gap-2">
            <input v-model="form.delivery_policy.show_micro_results" type="checkbox" class="h-4 w-4 rounded" />
            <span>{{ t('surveys.admin.form.showMicroResults') }}</span>
          </label>
        </div>
      </section>
    </form>

    <template #footer>
      <div class="flex flex-wrap items-center justify-end gap-2">
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="!form.title.trim() || form.questions.length === 0"
          @click="emitPreview"
        >
          {{ t('surveys.admin.preview') }}
        </button>
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="survey-editor-form"
          data-testid="survey-save-btn"
          :disabled="saving"
          class="btn btn-primary"
        >
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import AnnouncementTargetingEditor from '@/components/admin/announcements/AnnouncementTargetingEditor.vue'
import type { AdminGroup, AnnouncementTargeting } from '@/types'
import type {
  CreateSurveyRequest,
  Survey,
  SurveyDeliveryPolicy,
  SurveyDisplayMode,
  SurveyQuestion,
  SurveyQuestionType,
  SurveyRewardType,
  SurveyStatus,
  UpdateSurveyRequest,
  UserSurvey
} from '@/types/survey'
import {
  buildDailyExperienceTemplate,
  buildMicroPollTemplate,
  buildRewardedDeepDiveTemplate,
  defaultSurveyDeliveryPolicy
} from '@/utils/surveyTemplates'

const props = defineProps<{
  show: boolean
  survey: Survey | null
  saving: boolean
  subscriptionGroups: AdminGroup[]
}>()

const emit = defineEmits<{
  close: []
  save: [payload: { id?: number; data: CreateSurveyRequest | UpdateSurveyRequest }]
  preview: [survey: UserSurvey]
}>()

const { t } = useI18n()

const displayModes: SurveyDisplayMode[] = ['corner', 'modal', 'micro']

const isEditing = computed(() => Boolean(props.survey?.id))

const form = reactive<{
  title: string
  description: string
  status: SurveyStatus
  display_mode: SurveyDisplayMode
  questions: SurveyQuestion[]
  targeting: AnnouncementTargeting
  delivery_policy: SurveyDeliveryPolicy
  reward_type: SurveyRewardType
  reward_value: number
  reward_budget_limit_str: string
  reward_code_valid_days: number
  starts_at_str: string
  ends_at_str: string
}>({
  title: '',
  description: '',
  status: 'draft',
  display_mode: 'corner',
  questions: [],
  targeting: { any_of: [] },
  delivery_policy: defaultSurveyDeliveryPolicy(),
  reward_type: 'none',
  reward_value: 0.5,
  reward_budget_limit_str: '',
  reward_code_valid_days: 30,
  starts_at_str: '',
  ends_at_str: ''
})

const statusOptions = computed(() => [
  { value: 'draft', label: t('surveys.admin.status.draft') },
  { value: 'active', label: t('surveys.admin.status.active') },
  { value: 'archived', label: t('surveys.admin.status.archived') }
])

const questionTypeOptions = computed(() => [
  { value: 'single', label: t('surveys.admin.form.questionTypes.single') },
  { value: 'multi', label: t('surveys.admin.form.questionTypes.multi') },
  { value: 'rating', label: t('surveys.admin.form.questionTypes.rating') },
  { value: 'tags_text', label: t('surveys.admin.form.questionTypes.tags_text') }
])

const rewardTypeOptions = computed(() => [
  { value: 'none', label: t('surveys.admin.rewardType.none') },
  { value: 'balance', label: t('surveys.admin.rewardType.balance') },
  { value: 'redeem_code', label: t('surveys.admin.rewardType.redeem_code') }
])

const estimatedMaxBudget = computed<number | null>(() => {
  if (form.reward_type === 'none') return null
  const limit = Number(form.reward_budget_limit_str)
  if (!Number.isFinite(limit) || limit <= 0) return null
  if (!Number.isFinite(form.reward_value) || form.reward_value <= 0) return null
  return limit * form.reward_value
})

function formatDateTimeLocal(iso?: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function parseDateTimeLocalToUnix(str: string): number {
  if (!str.trim()) return 0
  const d = new Date(str)
  if (Number.isNaN(d.getTime())) return 0
  return Math.floor(d.getTime() / 1000)
}

function cloneQuestions(list: SurveyQuestion[]): SurveyQuestion[] {
  return JSON.parse(JSON.stringify(list || []))
}

const chipDrafts = reactive<Record<string, string>>({})

function clearChipDrafts() {
  for (const key of Object.keys(chipDrafts)) {
    delete chipDrafts[key]
  }
}

function getChipDraft(q: SurveyQuestion, kind: 'low' | 'high'): string {
  const key = `${q.id}:${kind}`
  if (key in chipDrafts) return chipDrafts[key]
  const list = kind === 'low' ? q.low_chips : q.high_chips
  return (list || []).join('，')
}

function onChipDraftInput(q: SurveyQuestion, kind: 'low' | 'high', value: string) {
  chipDrafts[`${q.id}:${kind}`] = value
  const parsed = parseChips(value)
  if (kind === 'low') {
    q.low_chips = parsed
  } else {
    q.high_chips = parsed
  }
}

function commitChipDraft(q: SurveyQuestion, kind: 'low' | 'high') {
  delete chipDrafts[`${q.id}:${kind}`]
}

function loadFromRequest(req: CreateSurveyRequest) {
  clearChipDrafts()
  form.title = req.title
  form.description = req.description || ''
  form.status = req.status || 'draft'
  form.display_mode = req.display_mode || 'corner'
  form.questions = cloneQuestions(req.questions)
  form.targeting = JSON.parse(JSON.stringify(req.targeting || { any_of: [] }))
  form.delivery_policy = {
    ...defaultSurveyDeliveryPolicy(),
    ...(req.delivery_policy || {})
  }
  form.reward_type = req.reward_type || 'none'
  form.reward_value = req.reward_value || 0.5
  form.reward_budget_limit_str = req.reward_budget_limit ? String(req.reward_budget_limit) : ''
  form.reward_code_valid_days = req.reward_code_valid_days || 30
  form.starts_at_str = ''
  form.ends_at_str = ''
}

const activeTemplate = ref<'daily' | 'rewarded' | 'micro' | null>('daily')

function applyTemplate(kind: 'daily' | 'rewarded' | 'micro') {
  activeTemplate.value = kind
  if (kind === 'daily') loadFromRequest(buildDailyExperienceTemplate())
  else if (kind === 'rewarded') loadFromRequest(buildRewardedDeepDiveTemplate())
  else loadFromRequest(buildMicroPollTemplate())
}

watch(
  () => [props.show, props.survey],
  () => {
    if (!props.show) return
    if (props.survey) {
      activeTemplate.value = null
      clearChipDrafts()
      form.title = props.survey.title
      form.description = props.survey.description || ''
      form.status = props.survey.status
      form.display_mode = props.survey.display_mode
      form.questions = cloneQuestions(props.survey.questions)
      form.targeting = JSON.parse(JSON.stringify(props.survey.targeting || { any_of: [] }))
      form.delivery_policy = {
        ...defaultSurveyDeliveryPolicy(),
        ...(props.survey.delivery_policy || {})
      }
      form.reward_type = props.survey.reward_type || 'none'
      form.reward_value = props.survey.reward_value || 0.5
      form.reward_budget_limit_str = props.survey.reward_budget_limit
        ? String(props.survey.reward_budget_limit)
        : ''
      form.reward_code_valid_days = props.survey.reward_code_valid_days || 30
      form.starts_at_str = formatDateTimeLocal(props.survey.starts_at)
      form.ends_at_str = formatDateTimeLocal(props.survey.ends_at)
    } else {
      activeTemplate.value = 'daily'
      loadFromRequest(buildDailyExperienceTemplate())
    }
  },
  { immediate: true }
)

function selectDisplayMode(mode: SurveyDisplayMode) {
  form.display_mode = mode
  if (mode === 'micro') {
    const firstSingle = form.questions.find((q) => q.type === 'single')
    if (firstSingle) {
      form.questions = [{ ...firstSingle, required: true, auto_advance: true }]
    } else {
      form.questions = cloneQuestions(buildMicroPollTemplate().questions)
    }
  }
}

function addQuestion() {
  const idx = form.questions.length + 1
  form.questions.push({
    id: `q${Date.now()}_${idx}`,
    type: 'single',
    title: '',
    required: true,
    auto_advance: true,
    options: [
      { id: `opt_${Date.now()}_1`, label: '' },
      { id: `opt_${Date.now()}_2`, label: '' }
    ]
  })
}

function removeQuestion(index: number) {
  if (form.questions.length <= 1) return
  form.questions.splice(index, 1)
}

function moveQuestion(index: number, delta: number) {
  const target = index + delta
  if (target < 0 || target >= form.questions.length) return
  const [item] = form.questions.splice(index, 1)
  form.questions.splice(target, 0, item)
}

function onQuestionTypeChange(q: SurveyQuestion, nextType: SurveyQuestionType) {
  q.type = nextType
  if ((nextType === 'single' || nextType === 'multi') && (!q.options || q.options.length < 2)) {
    q.options = [
      { id: `opt_${Date.now()}_1`, label: '' },
      { id: `opt_${Date.now()}_2`, label: '' }
    ]
  }
  if (nextType === 'rating') {
    q.scale_min = 1
    q.scale_max = 5
  }
  if (nextType === 'tags_text') {
    q.low_score_threshold = q.low_score_threshold || 3
    if (!q.low_chips) q.low_chips = []
    if (!q.high_chips) q.high_chips = []
  }
}

function addOption(q: SurveyQuestion) {
  if (!q.options) q.options = []
  q.options.push({
    id: `opt_${Date.now()}_${q.options.length + 1}`,
    label: ''
  })
}

function removeOption(q: SurveyQuestion, idx: number) {
  if (!q.options || q.options.length <= 2) return
  q.options.splice(idx, 1)
}

function parseChips(raw: string): string[] {
  return raw
    .split(/[,，\n]+/)
    .map((s) => s.trim())
    .filter(Boolean)
    .slice(0, 12)
}

function emitPreview() {
  const hasReward = form.reward_type !== 'none' && Number(form.reward_value) > 0
  emit('preview', {
    id: props.survey?.id || 0,
    title: form.title.trim(),
    description: form.description.trim(),
    display_mode: form.display_mode,
    questions: cloneQuestions(form.questions),
    has_reward: hasReward,
    reward_type: hasReward ? form.reward_type : 'none',
    reward_value: hasReward ? Number(form.reward_value) : 0,
    snooze_days: form.delivery_policy.snooze_days || 3,
    skip_if_announcement_unread: form.delivery_policy.skip_if_announcement_unread,
    delay_seconds: form.delivery_policy.delay_seconds || 0,
    show_micro_results: form.delivery_policy.show_micro_results,
    created_at: new Date().toISOString()
  })
}

function handleSubmit() {
  const budgetNum = Number(form.reward_budget_limit_str)
  const hasBudget = form.reward_type !== 'none' && Number.isFinite(budgetNum) && budgetNum > 0
  const startsUnix = parseDateTimeLocalToUnix(form.starts_at_str)
  const endsUnix = parseDateTimeLocalToUnix(form.ends_at_str)

  const basePayload = {
    title: form.title.trim(),
    description: form.description.trim(),
    status: form.status,
    display_mode: form.display_mode,
    questions: cloneQuestions(form.questions),
    targeting: form.targeting,
    delivery_policy: { ...form.delivery_policy },
    reward_type: form.reward_type,
    reward_value: form.reward_type === 'none' ? 0 : Number(form.reward_value) || 0,
    reward_code_valid_days: Number(form.reward_code_valid_days) || 30
  }

  if (isEditing.value && props.survey) {
    const updatePayload: UpdateSurveyRequest = {
      ...basePayload,
      reward_budget_limit: hasBudget ? Math.floor(budgetNum) : 0,
      starts_at: startsUnix,
      ends_at: endsUnix
    }
    emit('save', { id: props.survey.id, data: updatePayload })
  } else {
    const createPayload: CreateSurveyRequest = {
      ...basePayload,
      reward_budget_limit: hasBudget ? Math.floor(budgetNum) : null,
      starts_at: startsUnix > 0 ? startsUnix : undefined,
      ends_at: endsUnix > 0 ? endsUnix : undefined
    }
    emit('save', { data: createPayload })
  }
}
</script>
