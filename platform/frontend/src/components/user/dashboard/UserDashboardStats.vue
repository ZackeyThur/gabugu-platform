<template>
  <div
    class="overview-surface overview-summary-grid overview-command-monolith"
    :class="{ 'overview-summary-single': isSimple }"
    data-star-border
  >
    <!-- Left Chamber: Adaptive Treasury / Subscription Command Dial -->
    <section
      v-if="!isSimple"
      class="overview-account-card is-treasury"
      :class="{ 'is-subscription-hero': activeLeftChamber === 'subscription' }"
      :aria-labelledby="activeLeftChamber === 'subscription' ? 'subscription-hub-title' : 'balance-title'"
    >
      <!-- Subtle Swiss Horology Guilloché Dial Engraving (Whisper-quiet architectural ornament) -->
      <svg class="overview-treasury-guilloche" viewBox="0 0 280 280" width="280" height="280" aria-hidden="true">
        <circle cx="240" cy="40" r="88" fill="none" stroke="currentColor" stroke-width="0.7" stroke-dasharray="2 5" />
        <circle cx="240" cy="40" r="134" fill="none" stroke="currentColor" stroke-width="0.6" />
        <circle cx="240" cy="40" r="184" fill="none" stroke="currentColor" stroke-width="0.5" stroke-dasharray="1 6" />
        <circle cx="240" cy="40" r="228" fill="none" stroke="currentColor" stroke-width="0.5" />
      </svg>

      <div class="overview-card-label">
        <!-- Adaptive borderless hairline switcher when user holds active subscriptions alongside cash balance -->
        <div
          v-if="canSwitchChamber"
          class="overview-chamber-tabs"
          role="tablist"
          :aria-label="t('dashboard.overview.subChamberSwitchAria')"
        >
          <button
            v-for="tab in chamberTabs"
            :id="tab === 'subscription' ? 'subscription-hub-title' : 'balance-title'"
            :key="tab"
            type="button"
            role="tab"
            class="overview-chamber-tab"
            :class="{ 'is-active': activeLeftChamber === tab }"
            :aria-selected="activeLeftChamber === tab"
            @click="selectChamber(tab)"
          >
            <span>{{ tab === 'subscription' ? t('dashboard.overview.subChamberPlans') : t('dashboard.overview.subChamberBalance') }}</span>
            <span v-if="tab === 'subscription' && normalizedSubscriptions.length > 1" class="overview-chamber-tab-count">
              {{ normalizedSubscriptions.length }}
            </span>
          </button>
        </div>
        <span v-else-if="activeLeftChamber === 'subscription'" id="subscription-hub-title">
          {{ t('dashboard.overview.subChamberPlans') }}
        </span>
        <span v-else id="balance-title">
          {{ t('dashboard.overview.availableBalance') }}
        </span>

        <div class="overview-card-label-right">
          <template v-if="activeLeftChamber === 'subscription' && selectedSub">
            <span
              class="overview-runway-pill"
              :data-status="selectedSub.expiryStatus"
              :title="selectedSub.expiresAt ? resetTime(selectedSub.expiresAt) : t('dashboard.overview.subNoExpiry')"
            >
              <span class="overview-status-dot" :data-status="selectedSub.expiryStatus" aria-hidden="true" />
              <span>{{ subscriptionExpiryLabel(selectedSub) }}</span>
            </span>
          </template>
          <template v-else>
            <span
              v-if="!balanceLoading && !balanceError && runway.daysLeft !== null"
              class="overview-runway-pill"
              :data-status="runway.status"
              :title="t('dashboard.overview.runwayTooltip', { daily: overviewMoney(runway.dailyBurn) })"
            >
              <span class="overview-status-dot" :data-status="runway.status" aria-hidden="true" />
              <span>{{ runwayLabel }}</span>
            </span>
            <Icon v-else name="dollar" size="md" />
          </template>
        </div>
      </div>

      <!-- SUBSCRIPTION-FIRST COMMAND HUB -->
      <template v-if="activeLeftChamber === 'subscription' && selectedSub">
        <!-- Multi-Plan Hairline Switcher OR Single-Plan Identity Strip -->
        <div
          v-if="normalizedSubscriptions.length > 1"
          class="overview-sub-plan-tabs"
          role="tablist"
          :aria-label="t('dashboard.overview.subPlanSwitchAria')"
        >
          <button
            v-for="plan in normalizedSubscriptions"
            :key="plan.id"
            type="button"
            role="tab"
            class="overview-sub-plan-tab"
            :class="{ 'is-active': selectedSub.id === plan.id }"
            :aria-selected="selectedSub.id === plan.id"
            @click="selectSubscriptionPlan(plan.id)"
          >
            <span
              v-if="plan.platform"
              class="overview-sub-plan-icon"
              :data-brand="plan.platform"
              aria-hidden="true"
            >
              <PlatformIcon :platform="plan.platform" size="xs" />
            </span>
            <span class="overview-sub-plan-name">{{ plan.groupName }}</span>
            <span
              v-if="plan.rateMultiplier !== null && plan.rateMultiplier !== 1"
              class="overview-sub-plan-rate"
            >
              ×{{ formatMultiplier(plan.rateMultiplier) }}
            </span>
            <span
              class="overview-status-dot overview-sub-plan-dot"
              :data-status="planDotStatus(plan)"
              aria-hidden="true"
            />
          </button>
        </div>
        <div v-else class="overview-sub-plan-single">
          <span
            v-if="selectedSub.platform"
            class="overview-sub-plan-icon"
            :data-brand="selectedSub.platform"
            aria-hidden="true"
          >
            <PlatformIcon :platform="selectedSub.platform" size="sm" />
          </span>
          <strong class="overview-sub-plan-title">{{ selectedSub.groupName }}</strong>
          <span
            v-if="selectedSub.rateMultiplier !== null && selectedSub.rateMultiplier !== 1"
            class="overview-sub-plan-rate"
          >
            ×{{ formatMultiplier(selectedSub.rateMultiplier) }}
          </span>
        </div>

        <div class="overview-balance-hero-row">
          <div class="overview-balance-hero-main">
            <p
              v-if="!selectedSub.isUnlimited && selectedSub.bindingWindow"
              class="overview-amount"
              :class="{ 'is-exhausted': selectedSub.bindingWindow.state === 'exhausted' }"
              :title="preciseMoney(selectedSub.bindingWindow.remaining)"
            >
              <CountUp
                :value="selectedSub.bindingWindow.remaining"
                :formatter="overviewMoney"
                :duration="460"
                :update-duration="200"
                split-decimals
              />
            </p>
            <p v-else class="overview-amount is-unlimited-label">
              {{ t('dashboard.overview.subUnlimitedValue') }}
            </p>

            <p class="overview-card-note" :class="{ 'is-preview': hoveredSubWindow !== null }">
              {{ subHeroNote }}
            </p>
          </div>

          <div class="overview-balance-cta-cluster">
            <template v-if="selectedSub.needsRenew && subscriptionEnabled">
              <router-link
                :to="{ path: '/purchase', query: { tab: 'subscription', group: String(selectedSub.groupId) } }"
                data-magnet
                class="overview-primary-button"
              >
                {{ t('dashboard.overview.renewSubscription') }}
                <Icon name="arrowRight" size="sm" />
              </router-link>
              <router-link to="/subscriptions" class="overview-secondary-button overview-redeem-btn">
                {{ t('dashboard.overview.viewSubscriptions') }}
              </router-link>
            </template>
            <template v-else>
              <router-link to="/subscriptions" data-magnet class="overview-primary-button">
                {{ t('dashboard.overview.viewSubscriptions') }}
                <Icon name="arrowRight" size="sm" />
              </router-link>
              <router-link
                v-if="subscriptionEnabled"
                :to="{ path: '/purchase', query: { tab: 'subscription' } }"
                class="overview-secondary-button overview-redeem-btn"
              >
                {{ t('dashboard.overview.buySubscription') }}
              </router-link>
              <router-link v-else to="/redeem" class="overview-secondary-button overview-redeem-btn">
                {{ t('dashboard.redeemCode') }}
              </router-link>
            </template>
          </div>
        </div>

        <!-- Flush Architectural Multi-Window Quota Ledger (Daily / Weekly / Monthly) -->
        <div
          v-if="!selectedSub.isUnlimited"
          class="overview-sub-windows-grid"
          :style="{ '--sub-win-cols': selectedSub.windows.length }"
          @mouseleave="hoveredSubWindow = null"
        >
          <div
            v-for="win in selectedSub.windows"
            :key="win.key"
            class="overview-sub-window-cell"
            :class="{ 'is-binding': selectedSub.bindingWindow?.key === win.key }"
            :data-state="win.state"
            :data-window="win.key"
            @mouseenter="hoveredSubWindow = win.key"
          >
            <div class="overview-sub-win-top">
              <span class="overview-sub-win-label">{{ subWindowLabel(win.key) }}</span>
              <span class="overview-sub-win-pct" :data-state="win.state">{{ win.percent }}%</span>
            </div>
            <div class="overview-sub-win-usage overview-number" :title="`${preciseMoney(win.used)} / ${preciseMoney(win.limit)}`">
              <strong>{{ compactQuotaMoney(win.used) }}</strong>
              <span>/ {{ compactQuotaMoney(win.limit) }}</span>
            </div>
            <div class="overview-mini-bar" aria-hidden="true">
              <span :style="{ width: Math.min(100, win.ratio * 100) + '%' }" :data-state="win.state" />
            </div>
            <span class="overview-sub-win-reset">{{ subWindowResetText(win) }}</span>
          </div>
        </div>
        <div v-else class="overview-sub-windows-grid is-unlimited">
          <span class="overview-status-dot" data-status="healthy" aria-hidden="true" />
          <span>{{ t('dashboard.overview.subUnlimitedLedger') }}</span>
        </div>

        <!-- Quiet Bottom Rail: Inline Cash Balance + Affiliate Transfer + Top-up / Redeem -->
        <div
          v-if="billingMode !== 'subscription_only' || hasAffiliateQuota"
          class="overview-balance-nudges is-sub-rail"
        >
          <div class="overview-sub-rail-left">
            <span v-if="billingMode !== 'subscription_only'" class="overview-sub-cash-inline">
              <span>{{ t('dashboard.overview.subCashBalanceInline') }}</span>
              <strong class="overview-number">
                {{ balanceLoading ? '…' : balanceError ? '—' : overviewMoney(balance ?? 0) }}
              </strong>
            </span>
            <div v-if="!balanceLoading && !balanceError && hasAffiliateQuota" class="overview-affiliate-nudge">
              <Icon name="gift" size="sm" />
              <span>{{ t('dashboard.overview.subAffiliateInline', { amount: overviewMoney(affiliateQuota) }) }}</span>
              <template v-if="!confirmingTransfer">
                <button type="button" class="overview-link overview-nudge-action" @click="confirmingTransfer = true">
                  {{ t('dashboard.overview.transferToBalance') }}
                </button>
              </template>
              <template v-else>
                <button
                  type="button"
                  class="overview-link overview-nudge-action"
                  :disabled="affiliateTransferring"
                  @click="executeAffiliateTransfer"
                >
                  {{ affiliateTransferring ? t('common.loading') : t('dashboard.overview.confirmTransfer') }}
                </button>
                <button type="button" class="overview-muted-action" :disabled="affiliateTransferring" @click="confirmingTransfer = false">
                  {{ t('common.cancel') }}
                </button>
              </template>
            </div>
          </div>
          <div v-if="billingMode !== 'subscription_only'" class="overview-sub-rail-actions">
            <router-link v-if="paymentEnabled" to="/purchase" class="overview-link">
              {{ t('dashboard.overview.subQuickTopUp') }}
            </router-link>
            <router-link to="/redeem" class="overview-link">
              {{ t('dashboard.redeemCode') }}
            </router-link>
          </div>
        </div>
      </template>

      <!-- BALANCE-FIRST TREASURY HUB (PAYG & Hybrid Default) -->
      <template v-else>
        <div class="overview-balance-hero-row">
          <div class="overview-balance-hero-main">
            <div v-if="balanceLoading" class="overview-skeleton overview-skeleton-amount" :aria-label="t('common.loading')" />
            <div v-else-if="balanceError" class="overview-inline-error" role="alert">
              {{ t('dashboard.overview.balanceError') }}
              <button type="button" class="overview-link" @click="$emit('retryBalance')">{{ t('common.retry') }}</button>
            </div>
            <p v-else class="overview-amount" :title="preciseMoney(balance ?? 0)">
              <CountUp :value="balance ?? 0" :formatter="overviewMoney" split-decimals />
            </p>

            <p class="overview-card-note" :class="{ 'is-preview': Boolean(hoveredQuickPreviewLabel) }">
              <template v-if="hoveredQuickPreviewLabel">
                {{ hoveredQuickPreviewLabel }}
              </template>
              <template v-else-if="!balanceLoading && !balanceError && frozenBalance > 0">
                {{ t('dashboard.overview.frozenBalance') }}
                <span class="overview-number">{{ overviewMoney(frozenBalance) }}</span>
              </template>
              <template v-else-if="!balanceLoading && !balanceError && runway.daysLeft !== null">
                {{ t('dashboard.overview.cashDailyBurnHint', { daily: overviewMoney(runway.dailyBurn) }) }}
              </template>
              <template v-else>
                {{ t('dashboard.overview.balanceHint') }}
              </template>
            </p>
          </div>

          <div class="overview-balance-cta-cluster">
            <router-link
              v-if="paymentEnabled"
              to="/purchase"
              data-magnet
              class="overview-primary-button"
            >
              {{ t('dashboard.overview.recharge') }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <router-link
              v-else-if="billingMode === 'subscription_only'"
              :to="{ path: '/purchase', query: { tab: 'subscription' } }"
              data-magnet
              class="overview-primary-button"
            >
              {{ t('dashboard.overview.buySubscription') }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <router-link to="/redeem" class="overview-secondary-button overview-redeem-btn">
              {{ t('dashboard.redeemCode') }}
            </router-link>
          </div>
        </div>

        <!-- Flush Architectural Quick-Recharge Ledger Strip -->
        <div
          v-if="!balanceLoading && !balanceError && paymentEnabled && quickRechargeOptions.length > 0"
          class="overview-quick-grid"
          :style="{ '--quick-cols': quickRechargeOptions.length }"
          :aria-label="t('dashboard.overview.quickRecharge')"
          @mouseleave="hoveredQuickAmount = null"
        >
          <router-link
            v-for="opt in quickRechargeOptions"
            :key="opt.amount"
            :to="{ path: '/purchase', query: { amount: String(opt.amount) } }"
            class="overview-quick-tile"
            :class="{ 'is-recommended': opt.recommended }"
            @mouseenter="hoveredQuickAmount = opt.amount"
            @focusin="hoveredQuickAmount = opt.amount"
            @focusout="hoveredQuickAmount = null"
          >
            <div class="overview-quick-tile-top">
              <span class="overview-quick-pill-amount">{{ opt.currencySymbol }}{{ formatRechargeBonusNumber(opt.amount) }}</span>
              <span v-if="opt.percent > 0" class="overview-quick-pill-tag">
                {{
                  opt.mode === 'discount'
                    ? t('dashboard.overview.quickDiscountTag', { percent: formatRechargeBonusNumber(opt.percent) })
                    : t('dashboard.overview.quickBonusTag', { percent: formatRechargeBonusNumber(opt.percent) })
                }}
              </span>
            </div>
            <div class="overview-quick-tile-bottom">
              <span class="overview-quick-tile-sub">
                {{
                  opt.mode === 'discount'
                    ? `${opt.currencySymbol}${formatRechargeBonusNumber(opt.payBase)} → $${formatRechargeBonusNumber(opt.credited)}`
                    : `→ $${formatRechargeBonusNumber(opt.credited)}`
                }}
              </span>
              <span class="overview-quick-tile-cue" aria-hidden="true">
                <Icon name="chevronRight" size="xs" />
              </span>
            </div>
          </router-link>
        </div>

        <!-- Compact Footer Rail: Affiliate Quota Transfer / Baseline Note + Low-Balance Notify / Orders -->
        <div
          v-if="!balanceLoading && !balanceError"
          class="overview-balance-nudges"
          :class="{ 'is-standalone-footer': !(paymentEnabled && quickRechargeOptions.length > 0) }"
        >
          <div v-if="hasAffiliateQuota" class="overview-affiliate-nudge">
            <Icon name="gift" size="sm" />
            <span>{{ t('dashboard.overview.affiliateAvailable', { amount: overviewMoney(affiliateQuota) }) }}</span>
            <template v-if="!confirmingTransfer">
              <button type="button" class="overview-link overview-nudge-action" @click="confirmingTransfer = true">
                {{ t('dashboard.overview.transferToBalance') }}
              </button>
            </template>
            <template v-else>
              <button
                type="button"
                class="overview-link overview-nudge-action"
                :disabled="affiliateTransferring"
                @click="executeAffiliateTransfer"
              >
                {{ affiliateTransferring ? t('common.loading') : t('dashboard.overview.confirmTransfer') }}
              </button>
              <button type="button" class="overview-muted-action" :disabled="affiliateTransferring" @click="confirmingTransfer = false">
                {{ t('common.cancel') }}
              </button>
            </template>
          </div>
          <span v-else class="overview-muted">
            {{
              paymentEnabled && quickRechargeOptions.length > 0
                ? t('dashboard.overview.quickRechargeFootnote')
                : paymentEnabled
                  ? t('dashboard.overview.balanceFootnote')
                  : t('dashboard.overview.redeemFootnote')
            }}
          </span>

          <router-link v-if="showLowBalanceNotifyLink" to="/profile" class="overview-link overview-notify-link">
            <Icon name="bell" size="sm" />
            {{ t('dashboard.overview.enableLowBalanceNotify') }}
          </router-link>
          <router-link v-else-if="paymentEnabled" to="/orders" class="overview-link">
            {{ t('nav.myOrders') }}
            <Icon name="chevronRight" size="xs" />
          </router-link>
          <router-link v-else-if="!isSimple" to="/usage" class="overview-link">
            {{ t('nav.usage') }}
            <Icon name="chevronRight" size="xs" />
          </router-link>
        </div>
      </template>
    </section>

    <!-- Right Chamber: Real-Time Pulse & Scrubbable 7-Day Telemetry -->
    <section class="overview-account-card is-pulse" aria-labelledby="today-title">
      <div class="overview-card-label">
        <span id="today-title" class="overview-pulse-heading">
          <span>{{ t('dashboard.todayCost') }}</span>
          <span v-if="activeScrubDay" class="overview-scrub-date-tag">
            · {{ activeScrubDay.date.slice(5).replace('-', '/') }}
          </span>
        </span>
        <span class="overview-live-label" :class="{ 'is-live': isLiveActive }">
          <span v-if="isLiveActive" class="g-live-dot" aria-hidden="true" />
          {{ liveLabel }}
        </span>
      </div>

      <div v-if="loading" class="overview-skeleton overview-skeleton-amount" :aria-label="t('common.loading')" />
      <div v-else-if="error" class="overview-inline-error" role="alert">
        {{ t('dashboard.overview.statsError') }}
        <button type="button" class="overview-link" @click="$emit('retryStats')">{{ t('common.retry') }}</button>
      </div>
      <div v-else-if="stats" class="overview-today-hero">
        <div class="overview-today-headline">
          <p
            class="overview-amount"
            :class="{ 'is-scrubbing': activeScrubDay !== null }"
            :title="activeScrubDay ? preciseMoney(activeScrubDay.amount) : preciseMoney(stats.today_actual_cost)"
          >
            <CountUp
              :value="activeScrubDay ? activeScrubDay.amount : stats.today_actual_cost"
              :formatter="overviewMoney"
              :duration="520"
              :update-duration="0"
              split-decimals
            />
          </p>
          <div class="overview-today-subrow">
            <span v-if="activeScrubDay" class="overview-scrub-readout">
              {{ activeScrubDay.date }} · {{ t('dashboard.overview.trend7dShort') }}
            </span>
            <span
              v-else-if="yesterdayDelta"
              class="overview-delta-badge"
              :data-direction="yesterdayDelta.direction"
              :title="t('dashboard.overview.yesterdaySameTimeTitle', { amount: preciseMoney(yesterdayDelta.baselineCost) })"
            >
              {{ yesterdayDeltaLabel }}
            </span>
            <span v-else-if="stats.today_requests === 0 && stats.today_actual_cost === 0" class="overview-card-note">
              {{ t('dashboard.overview.todayNoUsageYet') }}
            </span>
            <span
              v-if="sparklinePoints.length >= 2"
              class="overview-sparkline-caption"
              :class="{ 'is-hidden': activeScrubDay !== null }"
            >
              {{ t('dashboard.overview.trend7dShort') }} · {{ t('dashboard.overview.periodPeak') }} {{ overviewMoney(sparklineMax) }}
            </span>
          </div>
        </div>

        <!-- Full-Width 1-to-1 Cursor-Scrubbable 7-Day Silk Sparkline -->
        <div
          v-if="sparklinePoints.length >= 2"
          ref="sparklineWrapRef"
          class="overview-sparkline-stage"
          :class="{ 'is-scrubbing': activeScrubPoint !== null }"
          role="slider"
          tabindex="0"
          :aria-label="t('dashboard.overview.sparkline7d')"
          :aria-valuemin="1"
          :aria-valuemax="sparklineDays.length"
          :aria-valuenow="(scrubIndex ?? sparklineDays.length - 1) + 1"
          :aria-valuetext="activeScrubDay ? `${activeScrubDay.date} ${overviewMoney(activeScrubDay.amount)}` : `${t('dashboard.overview.periodPeak')} ${overviewMoney(sparklineMax)}`"
          @pointermove="onSparklinePointerMove"
          @pointerleave="clearSparklineScrub"
          @blur="clearSparklineScrub"
          @click="onSparklineClick"
          @keydown="onSparklineKeydown"
        >
          <!-- Live Floating Magnetic Scrub Callout Tag -->
          <span
            v-if="activeScrubDay && activeScrubPoint"
            class="overview-spark-callout"
            :style="{ left: `${Math.max(14, Math.min(86, (activeScrubPoint.x / 360) * 100))}%` }"
            aria-hidden="true"
          >
            <span class="overview-spark-callout-date">{{ activeScrubDay.date.slice(5).replace('-', '/') }}</span>
            <strong>{{ overviewMoney(activeScrubDay.amount) }}</strong>
            <Icon name="chevronRight" size="xs" class="overview-spark-callout-cue" />
          </span>

          <svg viewBox="0 0 360 66" preserveAspectRatio="none" class="overview-sparkline-svg" aria-hidden="true">
            <defs>
              <linearGradient :id="sparkGradId" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="var(--ov-blue)" stop-opacity="0.24" />
                <stop offset="85%" stop-color="var(--ov-blue)" stop-opacity="0.02" />
                <stop offset="100%" stop-color="var(--ov-blue)" stop-opacity="0" />
              </linearGradient>
              <linearGradient :id="sparkStrokeId" x1="0" y1="0" x2="1" y2="0">
                <stop offset="0%" stop-color="var(--ov-blue)" stop-opacity="0.62" />
                <stop offset="55%" stop-color="var(--ov-blue)" stop-opacity="0.95" />
                <stop offset="100%" stop-color="var(--ov-green)" stop-opacity="0.96" />
              </linearGradient>
            </defs>

            <!-- Subtle 7-day average whisper hairline -->
            <line
              v-if="sparklineMeanY !== null"
              x1="6"
              x2="354"
              :y1="sparklineMeanY"
              :y2="sparklineMeanY"
              class="overview-spark-mean-line"
            />

            <!-- Silk area fill + gradient stroke -->
            <path :d="sparklineAreaPath" :fill="`url(#${sparkGradId})`" class="overview-spark-area" />
            <path
              :d="sparklineLinePath"
              fill="none"
              :stroke="`url(#${sparkStrokeId})`"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              vector-effect="non-scaling-stroke"
              class="overview-spark-line"
            />

            <!-- Subtle 7-day baseline tick markers -->
            <circle
              v-for="(pt, idx) in sparklinePoints"
              :key="`tick-${idx}`"
              :cx="pt.x"
              cy="62"
              r="1.15"
              class="overview-spark-tick"
              :class="{ 'is-active': scrubIndex === idx }"
            />

            <!-- Live 1-to-1 laser hair-beam when scrubbing -->
            <line
              v-if="activeScrubPoint"
              :x1="activeScrubPoint.x"
              :x2="activeScrubPoint.x"
              y1="4"
              y2="60"
              class="overview-spark-beam"
              vector-effect="non-scaling-stroke"
            />
          </svg>

          <!-- Crisp aspect-ratio-safe HTML dot overlay (zero SVG distortion across widths) -->
          <span
            v-if="activeDisplayPoint"
            class="overview-spark-beacon"
            :class="{ 'is-scrubbing': activeScrubPoint !== null }"
            :style="{
              left: `${(activeDisplayPoint.x / 360) * 100}%`,
              top: `${(activeDisplayPoint.y / 66) * 100}%`,
            }"
            aria-hidden="true"
          />
        </div>
      </div>

      <!-- Open 4-Column Architectural Telemetry Strip -->
      <div v-if="stats && !loading && !error" class="overview-today-micro">
        <div>
          <span>{{ t('dashboard.todayRequests') }}</span>
          <strong class="overview-number">{{ stats.today_requests.toLocaleString() }}</strong>
        </div>
        <div>
          <span>{{ t('dashboard.todayTokens') }}</span>
          <strong class="overview-number">
            <template v-if="splitMetricUnit(formatCompactTokens(stats.today_tokens))">
              <span>{{ splitMetricUnit(formatCompactTokens(stats.today_tokens))!.value }}</span><span class="overview-unit-suffix">{{ splitMetricUnit(formatCompactTokens(stats.today_tokens))!.unit }}</span>
            </template>
            <template v-else>{{ formatCompactTokens(stats.today_tokens) }}</template>
          </strong>
        </div>
        <div>
          <span>{{ todayThirdMetric.label }}</span>
          <strong class="overview-number">
            <template v-if="splitMetricUnit(todayThirdMetric.value)">
              <span>{{ splitMetricUnit(todayThirdMetric.value)!.value }}</span><span class="overview-unit-suffix">{{ splitMetricUnit(todayThirdMetric.value)!.unit }}</span>
            </template>
            <template v-else>{{ todayThirdMetric.value }}</template>
          </strong>
        </div>
        <div>
          <span>{{ t('dashboard.overview.todayAvgDuration') }}</span>
          <strong class="overview-number">
            <template v-if="splitMetricUnit(formatDurationCompact(stats.today_average_duration_ms || stats.average_duration_ms))">
              <span>{{ splitMetricUnit(formatDurationCompact(stats.today_average_duration_ms || stats.average_duration_ms))!.value }}</span><span class="overview-unit-suffix">{{ splitMetricUnit(formatDurationCompact(stats.today_average_duration_ms || stats.average_duration_ms))!.unit }}</span>
            </template>
            <template v-else>{{ formatDurationCompact(stats.today_average_duration_ms || stats.average_duration_ms) }}</template>
          </strong>
        </div>
      </div>

      <div class="overview-today-footer">
        <span class="overview-muted">{{ todayCostFooterHint }}</span>
        <button type="button" class="overview-link" :disabled="loading || error || !stats" @click="drawer = 'fees'">
          {{ t('dashboard.overview.feeDetails') }}
          <Icon name="chevronRight" size="sm" />
        </button>
      </div>
    </section>
  </div>

  <slot name="savings" />

  <!-- Hybrid Multi-Plan Subscription Ledger Ribbon (Hidden when Subscription Command Dial is active in Row 1) -->
  <section
    v-if="!isSimple && hasActiveSubs && activeLeftChamber === 'balance'"
    class="overview-surface overview-sub-strip is-ledger"
    :data-state="overallSubBarState"
    :aria-label="t('dashboard.overview.subscriptionStatus')"
  >
    <div class="overview-sub-ledger-head">
      <div class="overview-sub-ledger-head-left">
        <span class="overview-sub-badge">
          <span class="overview-status-dot" :data-status="overallSubDotStatus" aria-hidden="true" />
          <span>{{ t('dashboard.overview.activeSubscription') }}</span>
        </span>
        <span v-if="normalizedSubscriptions.length > 1" class="overview-muted overview-number">
          {{ normalizedSubscriptions.length }}
        </span>
      </div>
      <div class="overview-sub-strip-actions">
        <button
          v-if="canSwitchChamber"
          type="button"
          class="overview-link"
          @click="selectChamber('subscription')"
        >
          {{ t('dashboard.overview.subSwitchToHero') }}
        </button>
        <span v-if="canSwitchChamber" class="overview-sub-action-sep" aria-hidden="true" />
        <router-link to="/subscriptions" class="overview-link">
          {{ t('dashboard.overview.viewSubscriptions') }}
          <Icon name="chevronRight" size="sm" />
        </router-link>
      </div>
    </div>

    <div class="overview-sub-ledger-rows">
      <div
        v-for="plan in visibleLedgerSubscriptions"
        :key="plan.id"
        class="overview-sub-ledger-row"
      >
        <div class="overview-sub-ledger-identity">
          <span
            v-if="plan.platform"
            class="overview-sub-plan-icon"
            :data-brand="plan.platform"
            aria-hidden="true"
          >
            <PlatformIcon :platform="plan.platform" size="sm" />
          </span>
          <div class="overview-sub-ledger-meta">
            <div class="overview-sub-ledger-title-line">
              <strong>{{ plan.groupName }}</strong>
              <span
                v-if="plan.rateMultiplier !== null && plan.rateMultiplier !== 1"
                class="overview-sub-plan-rate"
              >
                ×{{ formatMultiplier(plan.rateMultiplier) }}
              </span>
            </div>
            <span class="overview-sub-ledger-expiry" :data-status="plan.expiryStatus">
              {{ subscriptionExpiryLabel(plan) }}
            </span>
          </div>
        </div>

        <div
          v-if="!plan.isUnlimited"
          class="overview-sub-ledger-windows"
          :style="{ '--ledger-win-cols': plan.windows.length }"
        >
          <div
            v-for="win in plan.windows"
            :key="win.key"
            class="overview-sub-ledger-win"
          >
            <div class="overview-sub-progress-text">
              <span>
                {{ subWindowLabel(win.key) }}
                <span class="overview-sub-pct" :data-state="win.state">{{ win.percent }}%</span>
              </span>
              <strong class="overview-number">{{ compactQuotaMoney(win.used) }} / {{ compactQuotaMoney(win.limit) }}</strong>
            </div>
            <div class="overview-mini-bar" aria-hidden="true">
              <span :style="{ width: Math.min(100, win.ratio * 100) + '%' }" :data-state="win.state" />
            </div>
            <span class="overview-sub-ledger-reset">{{ subWindowResetText(win) }}</span>
          </div>
        </div>
        <div v-else class="overview-sub-ledger-unlimited overview-muted">
          {{ t('dashboard.overview.subUnlimitedLedger') }}
        </div>

        <div class="overview-sub-ledger-cta">
          <router-link
            v-if="plan.needsRenew && subscriptionEnabled"
            :to="{ path: '/purchase', query: { tab: 'subscription', group: String(plan.groupId) } }"
            class="overview-secondary-button overview-sub-cta"
          >
            {{ t('dashboard.overview.renewSubscription') }}
          </router-link>
          <button
            v-else-if="canSwitchChamber"
            type="button"
            class="overview-link"
            @click="focusPlanInHero(plan.id)"
          >
            {{ t('dashboard.overview.posterViewDetail') }}
            <Icon name="chevronRight" size="xs" />
          </button>
        </div>
      </div>
    </div>

    <div v-if="normalizedSubscriptions.length > 3" class="overview-sub-ledger-more">
      <router-link to="/subscriptions" class="overview-link">
        {{ t('dashboard.overview.subMoreCount', { count: normalizedSubscriptions.length - 3 }) }}
        <Icon name="chevronRight" size="xs" />
      </router-link>
    </div>
  </section>

  <section
    v-else-if="!isSimple && !hasActiveSubs && paygPlanNudge"
    class="overview-plan-nudge"
    :aria-label="t('dashboard.overview.subscriptionSuggestion')"
  >
    <div class="overview-plan-nudge-copy">
      <Icon name="layers" size="sm" class="overview-plan-nudge-glyph" aria-hidden="true" />
      <span>
        {{
          t('dashboard.overview.highSpendPlanHint', {
            monthly: overviewMoney(paygPlanNudge.monthlySpend),
            plan: paygPlanNudge.planName,
            price: paygPlanNudge.priceLabel,
          })
        }}
      </span>
    </div>
    <router-link :to="{ path: '/purchase', query: { tab: 'subscription' } }" class="overview-link">
      {{ t('dashboard.overview.comparePlans') }}
      <Icon name="chevronRight" size="sm" />
    </router-link>
  </section>

  <!-- Platform Spending Matrix -->
  <section
    v-if="!isSimple && !hidePlatforms"
    class="overview-platform-section"
    aria-labelledby="platform-heading"
  >
    <div class="overview-section-heading">
      <div>
        <h2 id="platform-heading">{{ t('dashboard.overview.platformSpending') }}</h2>
        <p>{{ t('dashboard.overview.platformHint') }}</p>
      </div>
      <span v-if="stats && !loading && !error" class="overview-muted">
        {{ t('dashboard.platformCount', { count: platformCards.filter(p => p.platform !== '__other__').length }) }}
      </span>
    </div>
    <div v-if="loading" class="overview-surface overview-platform-ribbon" :aria-label="t('common.loading')">
      <div class="overview-platform-grid">
        <div v-for="i in 4" :key="i" class="overview-platform-card">
          <div class="overview-skeleton overview-platform-skeleton" />
        </div>
      </div>
    </div>
    <div v-else-if="error" class="overview-surface overview-empty" role="alert">
      <p>{{ t('dashboard.overview.statsError') }}</p>
      <button type="button" class="overview-retry-btn" @click="$emit('retryStats')">
        <Icon name="refresh" size="xs" />
        {{ t('common.retry') }}
      </button>
    </div>
    <template v-else>
      <p v-if="quotasError" class="overview-quota-warning" role="alert">
        {{ t('dashboard.overview.quotasError') }}
        <button type="button" class="overview-link" @click="$emit('retryQuotas')">{{ t('common.retry') }}</button>
      </p>
      <div v-if="platformCards.length" class="overview-surface overview-platform-ribbon" data-no-spotlight>
        <!-- Flush 3px Proportional Multi-Brand Share Spectrum Bar -->
        <div v-if="platformSpectrum.length > 0" class="overview-platform-spectrum" aria-hidden="true">
          <span
            v-for="seg in platformSpectrum"
            :key="seg.platform"
            class="overview-platform-spectrum-seg"
            :data-brand="seg.platform"
            :class="{
              'is-active': hoveredPlatform === seg.platform,
              'is-dimmed': hoveredPlatform !== null && hoveredPlatform !== seg.platform,
            }"
            :style="{ flexGrow: seg.weight }"
          />
        </div>

        <div
          class="overview-platform-grid"
          :style="{ '--platform-cols': Math.min(4, Math.max(1, platformCards.length)) }"
          @mouseleave="hoveredPlatform = null"
        >
          <article
            v-for="item in platformCards"
            :key="item.platform"
            class="overview-platform-card"
            :class="{ 'is-hovered': hoveredPlatform === item.platform }"
            data-testid="platform-card"
            :data-platform="item.platform"
            @mouseenter="hoveredPlatform = item.platform"
            @click="drawer = item.platform"
          >
            <div class="overview-platform-identity">
              <span class="overview-platform-logo" :data-brand="item.platform" aria-hidden="true">
                <PlatformIcon :platform="item.platform as GroupPlatform" size="lg" />
              </span>
              <h3>{{ label(item.platform) }}</h3>
              <span
                v-if="platformShare(item) !== null"
                class="overview-platform-share"
                :title="t('dashboard.overview.lifetimeShare')"
              >
                {{ platformShare(item) }}
              </span>
            </div>

            <div class="overview-platform-figure">
              <p class="overview-platform-amount" :title="preciseMoney(item.today_actual_cost)">
                <template v-if="splitFormattedDecimal(overviewMoney(item.today_actual_cost))">
                  <span>{{ splitFormattedDecimal(overviewMoney(item.today_actual_cost))!.major }}</span><span class="overview-cents">{{ splitFormattedDecimal(overviewMoney(item.today_actual_cost))!.cents }}</span><span v-if="splitFormattedDecimal(overviewMoney(item.today_actual_cost))!.suffix">{{ splitFormattedDecimal(overviewMoney(item.today_actual_cost))!.suffix }}</span>
                </template>
                <template v-else>{{ overviewMoney(item.today_actual_cost) }}</template>
              </p>
              <p class="overview-lifetime">
                <span>{{ t('dashboard.overview.lifetimeCost') }}</span>
                <strong :title="preciseMoney(item.total_actual_cost)">{{ overviewMoney(item.total_actual_cost) }}</strong>
              </p>
            </div>

            <!-- Integrated Micro Quota Track (or calm unlimited hairline) -->
            <div v-if="!quotasLoading && !quotasError && urgentPlatformQuota(item.quota)" class="overview-platform-quota-block">
              <div class="overview-platform-qmeta">
                <span>{{ t('dashboard.platformQuota.' + urgentPlatformQuota(item.quota)!.window) }}</span>
                <span class="overview-number">
                  {{ overviewMoney(urgentPlatformQuota(item.quota)!.used) }} / {{ overviewMoney(urgentPlatformQuota(item.quota)!.limit) }}
                </span>
              </div>
              <div class="overview-platform-qbar" aria-hidden="true">
                <span
                  :style="{ width: Math.min(100, (urgentPlatformQuota(item.quota)?.ratio ?? 0) * 100) + '%' }"
                  :data-state="urgentPlatformQuota(item.quota)?.state"
                />
              </div>
            </div>
            <div v-else-if="!quotasLoading && !quotasError && item.platform !== '__other__'" class="overview-platform-quota-block is-unlimited">
              <div class="overview-platform-qmeta">
                <span>{{ t('dashboard.platformQuota.title') }}</span>
                <span>{{ t('dashboard.platformQuota.noLimit') }}</span>
              </div>
              <div class="overview-platform-qbar is-unlimited" aria-hidden="true" />
            </div>

            <div class="overview-platform-footer">
              <span v-if="quotasLoading" class="overview-quota-label">{{ t('dashboard.overview.checkingQuota') }}</span>
              <span v-else-if="quotasError" class="overview-quota-label" data-state="near">{{ t('dashboard.overview.quotaUnknown') }}</span>
              <span v-else-if="quotaState(item.quota)" class="overview-quota-label" :data-state="quotaState(item.quota)">
                {{ t('dashboard.overview.quota.' + quotaState(item.quota)) }}
              </span>
              <span v-else class="overview-quota-label is-muted">{{ t('dashboard.overview.quotaUnlimitedHint') }}</span>
              <button
                type="button"
                class="overview-platform-chevron"
                :aria-label="t('dashboard.overview.platformDetails', { platform: label(item.platform) })"
                @click.stop="drawer = item.platform"
              >
                <Icon name="chevronRight" size="sm" />
              </button>
            </div>
          </article>
        </div>
      </div>
      <div v-else class="overview-surface overview-empty">
        <p>{{ t('dashboard.overview.noPlatforms') }}</p>
        <router-link to="/keys" class="overview-link">
          {{ t('dashboard.overview.manageKeys') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </div>
    </template>
  </section>

  <!-- Fees & Platform Details Drawer -->
  <DashboardDrawer :open="drawer !== null" :title="drawer === 'fees' ? t('dashboard.overview.feeDetails') : label(drawer ?? '')" @close="drawer = null">
    <template #title-prefix>
      <span
        v-if="selected && selected.platform !== '__other__'"
        class="overview-platform-logo overview-drawer-brand"
        :data-brand="selected.platform"
        aria-hidden="true"
      >
        <PlatformIcon :platform="selected.platform as GroupPlatform" size="lg" />
      </span>
      <span v-else-if="drawer === 'fees'" class="overview-drawer-icon" aria-hidden="true">
        <Icon name="chartBar" size="sm" />
      </span>
    </template>
    <p v-if="loading" role="status">{{ t('common.loading') }}</p>
    <p v-else-if="error" role="alert">{{ t('dashboard.overview.statsError') }}</p>
    <template v-else-if="drawer === 'fees' && stats">
      <p class="overview-detail-intro">{{ t('dashboard.overview.feeExplanation') }}</p>
      <p class="overview-detail-intro">{{ t('dashboard.overview.subscriptionExplanation') }}</p>
      <dl class="overview-detail-list">
        <div><dt>{{ t('dashboard.overview.todayActual') }}</dt><dd>{{ preciseMoney(stats.today_actual_cost) }}</dd></div>
        <div><dt>{{ t('dashboard.overview.todayStandard') }}</dt><dd>{{ preciseMoney(stats.today_cost) }}</dd></div>
        <div v-if="(stats.today_balance_actual_cost ?? 0) > 0 || (stats.today_subscription_actual_cost ?? 0) > 0">
          <dt>{{ t('dashboard.overview.todayBalanceVsSub') }}</dt>
          <dd>{{ overviewMoney(stats.today_balance_actual_cost ?? 0) }} / {{ overviewMoney(stats.today_subscription_actual_cost ?? 0) }}</dd>
        </div>
        <div v-if="stats.yesterday_actual_cost !== undefined">
          <dt>{{ t('dashboard.overview.yesterdayActual') }}</dt>
          <dd>{{ preciseMoney(stats.yesterday_actual_cost) }}</dd>
        </div>
        <div><dt>{{ t('dashboard.overview.lifetimeCost') }}</dt><dd>{{ preciseMoney(stats.total_actual_cost) }}</dd></div>
        <div><dt>{{ t('dashboard.overview.lifetimeStandard') }}</dt><dd>{{ preciseMoney(stats.total_cost) }}</dd></div>
      </dl>
      <h3 class="overview-detail-title">{{ t('dashboard.overview.usageDetails') }}</h3>
      <dl class="overview-detail-list">
        <div><dt>{{ t('dashboard.todayRequests') }}</dt><dd>{{ stats.today_requests.toLocaleString() }}</dd></div>
        <div><dt>{{ t('dashboard.todayTokens') }}</dt><dd>{{ stats.today_tokens.toLocaleString() }}</dd></div>
        <div><dt>{{ t('dashboard.totalTokens') }}</dt><dd>{{ stats.total_tokens.toLocaleString() }}</dd></div>
        <div><dt>{{ t('dashboard.overview.rpm') }}</dt><dd>{{ formattedRpmDetail }}</dd></div>
        <div><dt>{{ t('dashboard.overview.tpm') }}</dt><dd>{{ stats.tpm.toLocaleString() }}</dd></div>
        <div><dt>{{ t('dashboard.overview.averageDuration') }}</dt><dd>{{ ((stats.today_average_duration_ms || stats.average_duration_ms) / 1000).toFixed(2) }} s</dd></div>
      </dl>
      <router-link to="/usage" class="overview-link">
        {{ t('dashboard.viewUsage') }}
        <Icon name="arrowRight" size="sm" />
      </router-link>
    </template>
    <template v-else-if="selected">
      <p class="overview-detail-intro">{{ t('dashboard.overview.platformDetailHint') }}</p>
      <dl class="overview-detail-list">
        <div><dt>{{ t('dashboard.todayCost') }}</dt><dd>{{ preciseMoney(selected.today_actual_cost) }}</dd></div>
        <div><dt>{{ t('dashboard.overview.lifetimeCost') }}</dt><dd>{{ preciseMoney(selected.total_actual_cost) }}</dd></div>
        <div><dt>{{ t('dashboard.overview.lifetimeRequests') }}</dt><dd>{{ selected.total_requests?.toLocaleString() ?? '—' }}</dd></div>
        <div><dt>{{ t('dashboard.totalTokens') }}</dt><dd>{{ selected.total_tokens?.toLocaleString() ?? '—' }}</dd></div>
      </dl>
      <p v-if="selected.platform === '__other__'" class="overview-detail-intro">{{ t('dashboard.overview.otherExplanation') }}</p>
      <template v-else>
        <h3 class="overview-detail-title">{{ t('dashboard.platformQuota.title') }}</h3>
        <p v-if="quotasLoading">{{ t('common.loading') }}</p>
        <p v-else-if="quotasError" role="alert">
          {{ t('dashboard.overview.quotasError') }}
          <button type="button" class="overview-link" @click="$emit('retryQuotas')">{{ t('common.retry') }}</button>
        </p>
        <p v-else-if="!hasQuota(selected.quota)" class="overview-muted">{{ t('dashboard.platformQuota.noLimit') }}</p>
        <template v-else>
          <template v-for="window in quotaWindows" :key="window">
            <div v-if="selected.quota?.[`${window}_limit_usd`] != null" class="overview-quota-detail">
              <div class="overview-quota-detail-heading">
                <span class="overview-quota-window-name">
                  <span>{{ t('dashboard.platformQuota.' + window) }}</span>
                  <span
                    v-if="(selected.quota?.[`${window}_limit_usd`] ?? 0) > 0"
                    class="overview-quota-pct"
                    :data-state="quotaWindowState(selected.quota?.[`${window}_usage_usd`] ?? 0, selected.quota?.[`${window}_limit_usd`] ?? 0)"
                  >
                    {{ quotaWindowPercent(selected.quota?.[`${window}_usage_usd`] ?? 0, selected.quota?.[`${window}_limit_usd`] ?? 0) }}
                  </span>
                </span>
                <strong v-if="selected.quota?.[`${window}_limit_usd`] === 0">{{ t('dashboard.platformQuota.disabled') }}</strong>
                <span v-else class="overview-number">
                  {{ preciseMoney(selected.quota?.[`${window}_usage_usd`] ?? 0) }} / {{ preciseMoney(selected.quota?.[`${window}_limit_usd`] ?? 0) }}
                </span>
              </div>
              <progress
                :value="selected.quota?.[`${window}_usage_usd`] ?? 0"
                :max="selected.quota?.[`${window}_limit_usd`] || 1"
                :data-state="quotaWindowState(selected.quota?.[`${window}_usage_usd`] ?? 0, selected.quota?.[`${window}_limit_usd`] ?? 0)"
                :aria-label="t('dashboard.platformQuota.' + window)"
              />
              <p v-if="selected.quota?.[`${window}_window_resets_at`]" class="overview-muted">
                {{ t('dashboard.platformQuota.resetsAt', { time: resetTime(selected.quota?.[`${window}_window_resets_at`]) }) }}
              </p>
            </div>
          </template>
        </template>
      </template>

      <div class="overview-drawer-footer">
        <router-link
          :to="selected.platform === '__other__' ? '/usage' : { path: '/usage', query: { platform: selected.platform } }"
          class="overview-primary-button overview-drawer-drill"
        >
          {{ selected.platform === '__other__' ? t('dashboard.viewUsage') : t('dashboard.overview.viewPlatformUsage') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </div>
    </template>
  </DashboardDrawer>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import CountUp from '@/components/common/CountUp.vue'
import DashboardDrawer from './DashboardDrawer.vue'
import type { UserDashboardStats } from '@/api/usage'
import type {
  GroupPlatform,
  PlatformQuotaItem,
  TrendDataPoint,
  UserAffiliateDetail,
  UserSubscription,
} from '@/types'
import type { CheckoutInfoResponse } from '@/types/payment'
import type { SiteBillingMode } from '@/utils/siteBillingMode'
import { formatRechargeBonusNumber } from '@/utils/rechargeBonus'
import {
  buildQuickRechargeOptions,
  calculateBalanceRunway,
  calculateCacheEfficiency,
  calculateYesterdayDelta,
  compactQuotaMoney,
  dailySpending,
  dashboardPlatforms,
  dashboardToday,
  DASHBOARD_TIMEZONE,
  formatCompactTokens,
  formatDurationCompact,
  hasQuota,
  isSubscriptionPrimaryUser,
  mostUrgentQuota,
  normalizeActiveSubscriptions,
  overviewMoney,
  platformName,
  preciseMoney,
  quotaState,
  quotaWindows,
  shiftDate,
  splitFormattedDecimal,
  splitMetricUnit,
  type DashboardPlatform,
  type NormalizedSubscriptionSummary,
  type QuotaWindow,
  type SubscriptionWindowSummary,
} from '@/utils/dashboardOverview'

const props = withDefaults(defineProps<{
  stats: UserDashboardStats | null
  balance: number | null
  frozenBalance?: number
  isSimple: boolean
  platformQuotas?: PlatformQuotaItem[] | null
  loading?: boolean
  error?: boolean
  balanceLoading?: boolean
  balanceError?: boolean
  quotasLoading?: boolean
  quotasError?: boolean
  paymentEnabled?: boolean
  subscriptionEnabled?: boolean
  billingMode?: SiteBillingMode
  checkoutInfo?: CheckoutInfoResponse | null
  affiliateDetail?: UserAffiliateDetail | null
  activeSubscriptions?: UserSubscription[]
  baseline7d?: TrendDataPoint[]
  today?: string
  balanceLowNotifyEnabled?: boolean
  userBalanceNotifyEnabled?: boolean
  hidePlatforms?: boolean
  affiliateTransferring?: boolean
}>(), {
  frozenBalance: 0,
  platformQuotas: null,
  paymentEnabled: true,
  subscriptionEnabled: false,
  billingMode: 'recharge_and_subscription',
  checkoutInfo: null,
  affiliateDetail: null,
  activeSubscriptions: () => [],
  baseline7d: () => [],
  balanceLowNotifyEnabled: false,
  userBalanceNotifyEnabled: false,
  hidePlatforms: false,
  affiliateTransferring: false,
})

const emit = defineEmits<{
  retryStats: []
  retryBalance: []
  retryQuotas: []
  transferAffiliate: []
  drillDay: [date: string]
}>()

const { t } = useI18n()
const drawer = ref<string | null>(null)
const confirmingTransfer = ref(false)
const sparkRand = Math.random().toString(36).slice(2, 8)
const sparkGradId = `ov-spark-${sparkRand}`
const sparkStrokeId = `ov-spark-stroke-${sparkRand}`
const sparklineWrapRef = ref<HTMLElement | null>(null)
const scrubIndex = ref<number | null>(null)
const hoveredPlatform = ref<string | null>(null)
const hoveredQuickAmount = ref<number | null>(null)
const userChamberOverride = ref<'balance' | 'subscription' | null>(null)
const selectedSubId = ref<number | null>(null)
const hoveredSubWindow = ref<QuotaWindow | null>(null)

const platformCards = computed(() =>
  props.stats ? dashboardPlatforms(props.stats, props.platformQuotas ?? []) : []
)
const platformSpectrum = computed(() => {
  const positive = platformCards.value.filter((p) => p.total_actual_cost > 0)
  const total = positive.reduce((sum, p) => sum + p.total_actual_cost, 0)
  if (total <= 0) return []
  return positive.map((p) => ({
    platform: p.platform,
    weight: Math.max(0.015, p.total_actual_cost / total),
  }))
})
const selected = computed(() => platformCards.value.find((p) => p.platform === drawer.value))
const label = (platform: string) =>
  platform === '__other__' ? t('dashboard.platformOther') : platformName(platform)

const runway = computed(() => calculateBalanceRunway(props.balance, props.stats))
const runwayLabel = computed(() => {
  if (runway.value.daysLeft === null) return ''
  if (runway.value.daysLeft < 1) return t('dashboard.overview.runwayLessThanDay')
  if (runway.value.daysLeft > 99) return t('dashboard.overview.runwayOver99Days')
  return t('dashboard.overview.runwayDays', { days: Math.round(runway.value.daysLeft) })
})

const quickRechargeOptions = computed(() =>
  buildQuickRechargeOptions(props.checkoutInfo, runway.value.dailyBurn)
)

const hoveredQuickPreviewLabel = computed(() => {
  if (hoveredQuickAmount.value === null || props.balanceLoading || props.balanceError) return ''
  const opt = quickRechargeOptions.value.find((o) => o.amount === hoveredQuickAmount.value)
  if (!opt) return ''
  const projectedBalance = Math.max(0, (props.balance ?? 0) + opt.credited)
  const balanceText = overviewMoney(projectedBalance)
  if (runway.value.dailyBurn > 0.0001) {
    const projectedDays = projectedBalance / runway.value.dailyBurn
    if (projectedDays > 99) {
      return t('dashboard.overview.rechargePreviewRunwayMax', { balance: balanceText })
    }
    return t('dashboard.overview.rechargePreviewRunway', {
      balance: balanceText,
      days: Math.max(1, Math.round(projectedDays)),
    })
  }
  return t('dashboard.overview.rechargePreviewBalance', { balance: balanceText })
})

const affiliateQuota = computed(() => props.affiliateDetail?.aff_quota ?? 0)
const hasAffiliateQuota = computed(() => affiliateQuota.value > 0.005)
const showLowBalanceNotifyLink = computed(
  () => props.balanceLowNotifyEnabled && !props.userBalanceNotifyEnabled
)

function executeAffiliateTransfer() {
  if (props.affiliateTransferring) return
  confirmingTransfer.value = false
  emit('transferAffiliate')
}

const isLiveActive = computed(() => {
  if (!props.stats || props.loading || props.error) return false
  return (props.stats.rpm_exact ?? 0) > 0 || (props.stats.recent_5m_requests ?? 0) > 0 || props.stats.rpm > 0
})

const liveLabel = computed(() => {
  if (!isLiveActive.value || !props.stats) return t('dashboard.overview.todaySoFar')
  const rpmExact = props.stats.rpm_exact ?? props.stats.rpm
  const rpmText = props.stats.rpm === 0 && rpmExact > 0 ? '<1' : String(props.stats.rpm)
  return `${rpmText} RPM · ${formatCompactTokens(props.stats.tpm)} TPM`
})

const formattedRpmDetail = computed(() => {
  if (!props.stats) return '0'
  if (props.stats.rpm === 0 && (props.stats.rpm_exact ?? 0) > 0) {
    return `${props.stats.rpm_exact!.toFixed(1)} (${props.stats.recent_5m_requests ?? 0} / 5m)`
  }
  return props.stats.rpm.toLocaleString()
})

const yesterdayDelta = computed(() => calculateYesterdayDelta(props.stats))
const yesterdayDeltaLabel = computed(() => {
  const d = yesterdayDelta.value
  if (!d) return ''
  if (d.direction === 'new') return t('dashboard.overview.vsYesterdayNew')
  if (d.direction === 'flat') return t('dashboard.overview.vsYesterdayFlat')
  const pct = Math.min(999, Math.abs(d.percent ?? 0)).toFixed(1)
  return d.direction === 'up'
    ? t('dashboard.overview.vsYesterdayUp', { percent: pct })
    : t('dashboard.overview.vsYesterdayDown', { percent: pct })
})

const todayThirdMetric = computed(() => {
  const s = props.stats
  if (!s) return { label: t('dashboard.overview.todayCacheHit'), value: '—' }
  const cache = calculateCacheEfficiency(
    s.today_input_tokens,
    s.today_cache_creation_tokens,
    s.today_cache_read_tokens
  )
  return {
    label: t('dashboard.overview.todayCacheHit'),
    value: cache.promptTokens > 0 ? `${cache.rate.toFixed(0)}%` : '0%',
  }
})

// Frozen 7-day smooth cubic SVG sparkline geometry (360×66 full-bleed responsive stage)
const sparklineDays = computed(() => {
  if (!props.baseline7d?.length) return []
  const end = props.today || dashboardToday()
  const start = shiftDate(end, -6)
  return dailySpending(props.baseline7d, start, end)
})

const sparklineMax = computed(() => Math.max(...sparklineDays.value.map((d) => d.amount), 0))

const sparklineMean = computed(() => {
  const days = sparklineDays.value
  if (days.length === 0) return 0
  return days.reduce((sum, d) => sum + d.amount, 0) / days.length
})

const sparklinePoints = computed(() => {
  const days = sparklineDays.value
  const max = sparklineMax.value
  if (days.length < 2 || max <= 0) return []
  const positive = days.map((d) => d.amount).filter((v) => v > 0)
  const rawMin = positive.length ? Math.min(...positive) : 0
  // Expressive optical baseline: preserve full visual amplitude while staying grounded above zero
  const domainMin = rawMin > 0 && max > rawMin ? rawMin * 0.56 : 0
  const span = Math.max(1e-9, max * 1.04 - domainMin)
  const width = 348
  const height = 44
  const padX = 6
  const padY = 9
  return days.map((d, i) => {
    const normalized = Math.max(0, Math.min(1, (d.amount - domainMin) / span))
    return {
      x: padX + (days.length > 1 ? (i / (days.length - 1)) * width : width / 2),
      y: padY + height - normalized * height,
    }
  })
})

const sparklineMeanY = computed(() => {
  const days = sparklineDays.value
  const max = sparklineMax.value
  const mean = sparklineMean.value
  if (max <= 0 || mean <= 0) return null
  const positive = days.map((d) => d.amount).filter((v) => v > 0)
  const rawMin = positive.length ? Math.min(...positive) : 0
  const domainMin = rawMin > 0 && max > rawMin ? rawMin * 0.56 : 0
  const span = Math.max(1e-9, max * 1.04 - domainMin)
  const normalized = Math.max(0, Math.min(1, (mean - domainMin) / span))
  return Number((9 + 44 - normalized * 44).toFixed(1))
})

const sparklineLinePath = computed(() => {
  const pts = sparklinePoints.value
  if (pts.length < 2) return ''
  let d = `M${pts[0].x.toFixed(1)},${pts[0].y.toFixed(1)}`
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i]
    const p1 = pts[i + 1]
    const cx = ((p0.x + p1.x) / 2).toFixed(1)
    d += ` C${cx},${p0.y.toFixed(1)} ${cx},${p1.y.toFixed(1)} ${p1.x.toFixed(1)},${p1.y.toFixed(1)}`
  }
  return d
})

const sparklineAreaPath = computed(() => {
  const pts = sparklinePoints.value
  if (pts.length < 2) return ''
  const line = sparklineLinePath.value
  const last = pts[pts.length - 1]
  const first = pts[0]
  return `${line} L${last.x.toFixed(1)},60 L${first.x.toFixed(1)},60 Z`
})

const sparklineLastPoint = computed(() =>
  sparklinePoints.value.length > 0 ? sparklinePoints.value[sparklinePoints.value.length - 1] : null
)

const activeScrubDay = computed(() => {
  const idx = scrubIndex.value
  if (idx === null || idx < 0 || idx >= sparklineDays.value.length) return null
  return sparklineDays.value[idx]
})

const activeScrubPoint = computed(() => {
  const idx = scrubIndex.value
  if (idx === null || idx < 0 || idx >= sparklinePoints.value.length) return null
  return sparklinePoints.value[idx]
})

const activeDisplayPoint = computed(() => activeScrubPoint.value ?? sparklineLastPoint.value)

function onSparklinePointerMove(e: PointerEvent) {
  const el = sparklineWrapRef.value
  const count = sparklineDays.value.length
  if (!el || count < 2) return
  const rect = el.getBoundingClientRect()
  if (rect.width <= 0) return
  const ratio = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width))
  const idx = Math.round(ratio * (count - 1))
  scrubIndex.value = idx
}

function clearSparklineScrub() {
  scrubIndex.value = null
}

function onSparklineClick() {
  if (activeScrubDay.value?.date) {
    emit('drillDay', activeScrubDay.value.date)
  }
}

function onSparklineKeydown(e: KeyboardEvent) {
  const count = sparklineDays.value.length
  if (count < 2) return
  if (e.key === 'ArrowLeft') {
    e.preventDefault()
    scrubIndex.value = scrubIndex.value === null ? count - 2 : Math.max(0, scrubIndex.value - 1)
  } else if (e.key === 'ArrowRight') {
    e.preventDefault()
    scrubIndex.value = scrubIndex.value === null ? count - 1 : Math.min(count - 1, scrubIndex.value + 1)
  } else if (e.key === 'Home') {
    e.preventDefault()
    scrubIndex.value = 0
  } else if (e.key === 'End') {
    e.preventDefault()
    scrubIndex.value = count - 1
  } else if (e.key === 'Enter' && activeScrubDay.value?.date) {
    e.preventDefault()
    emit('drillDay', activeScrubDay.value.date)
  } else if (e.key === 'Escape') {
    scrubIndex.value = null
  }
}

// Adaptive Subscription-First & Hybrid Multi-Plan State
const normalizedSubscriptions = computed(() =>
  normalizeActiveSubscriptions(props.activeSubscriptions)
)
const hasActiveSubs = computed(() => normalizedSubscriptions.value.length > 0)

const subscriptionPrimary = computed(() =>
  isSubscriptionPrimaryUser({
    balance: props.balance,
    activeSubscriptions: props.activeSubscriptions,
    stats: props.stats,
    billingMode: props.billingMode,
    paymentEnabled: props.paymentEnabled,
  })
)

const activeLeftChamber = computed<'balance' | 'subscription'>(() => {
  if (!hasActiveSubs.value) return 'balance'
  if (props.billingMode === 'subscription_only') return 'subscription'
  return userChamberOverride.value ?? (subscriptionPrimary.value ? 'subscription' : 'balance')
})

const canSwitchChamber = computed(
  () => hasActiveSubs.value && props.billingMode !== 'subscription_only'
)

const chamberTabs = computed<Array<'subscription' | 'balance'>>(() =>
  subscriptionPrimary.value ? ['subscription', 'balance'] : ['balance', 'subscription']
)

const selectedSub = computed<NormalizedSubscriptionSummary | null>(() => {
  const list = normalizedSubscriptions.value
  if (list.length === 0) return null
  return list.find((s) => s.id === selectedSubId.value) ?? list[0]
})

const visibleLedgerSubscriptions = computed(() => normalizedSubscriptions.value.slice(0, 3))

const overallSubBarState = computed<'active' | 'near' | 'exhausted'>(() => {
  if (normalizedSubscriptions.value.some((s) => s.barState === 'exhausted')) return 'exhausted'
  if (normalizedSubscriptions.value.some((s) => s.barState === 'near')) return 'near'
  return 'active'
})

const overallSubDotStatus = computed<'healthy' | 'warning' | 'critical'>(() => {
  if (overallSubBarState.value === 'exhausted' || normalizedSubscriptions.value.some((s) => s.expiryStatus === 'critical')) {
    return 'critical'
  }
  if (overallSubBarState.value === 'near' || normalizedSubscriptions.value.some((s) => s.expiryStatus === 'warning')) {
    return 'warning'
  }
  return 'healthy'
})

function selectChamber(tab: 'balance' | 'subscription') {
  userChamberOverride.value = tab
  hoveredQuickAmount.value = null
  hoveredSubWindow.value = null
}

function selectSubscriptionPlan(id: number) {
  selectedSubId.value = id
  hoveredSubWindow.value = null
}

function focusPlanInHero(id: number) {
  selectedSubId.value = id
  hoveredSubWindow.value = null
  userChamberOverride.value = 'subscription'
}

function formatMultiplier(value: number): string {
  return Number(value.toFixed(2)).toString()
}

function planDotStatus(plan: NormalizedSubscriptionSummary): 'healthy' | 'warning' | 'critical' {
  if (plan.barState === 'exhausted' || plan.expiryStatus === 'critical') return 'critical'
  if (plan.barState === 'near' || plan.expiryStatus === 'warning') return 'warning'
  return 'healthy'
}

function subscriptionExpiryLabel(plan: NormalizedSubscriptionSummary): string {
  if (!plan.expiresAt) return t('dashboard.overview.subNoExpiry')
  if (plan.expiryDuration?.unit === 'hoursMinutes') {
    const { hours, minutes } = plan.expiryDuration
    const text = hours > 0 ? `${hours}h ${minutes}m` : `${Math.max(1, minutes)}m`
    return t('dashboard.overview.subHoursRemaining', { time: text })
  }
  return t('dashboard.overview.subDaysRemaining', { days: plan.daysRemaining ?? 0 })
}

function subWindowLabel(key: QuotaWindow): string {
  if (key === 'daily') return t('dashboard.overview.subWindowDaily')
  if (key === 'weekly') return t('dashboard.overview.subWindowWeekly')
  return t('dashboard.overview.subWindowMonthly')
}

function subWindowRemainingLabel(key: QuotaWindow): string {
  if (key === 'daily') return t('dashboard.overview.subRemainingDaily')
  if (key === 'weekly') return t('dashboard.overview.subRemainingWeekly')
  return t('dashboard.overview.subRemainingMonthly')
}

function subWindowResetText(win: SubscriptionWindowSummary): string {
  if (!win.windowActive) return t('dashboard.overview.subWindowReady')
  if (!win.resetDurationText) return t('dashboard.overview.subWindowReady')
  return win.isOneTime
    ? t('dashboard.overview.subExpiresIn', { time: win.resetDurationText })
    : t('dashboard.overview.subResetsIn', { time: win.resetDurationText })
}

const subHeroNote = computed(() => {
  const plan = selectedSub.value
  if (!plan) return ''
  if (plan.isUnlimited) return t('dashboard.overview.subUnlimitedNote')
  const win =
    (hoveredSubWindow.value
      ? plan.windows.find((w) => w.key === hoveredSubWindow.value)
      : null) ?? plan.bindingWindow
  if (!win) return t('dashboard.overview.subUnlimitedNote')
  const parts = [
    hoveredSubWindow.value
      ? `${subWindowRemainingLabel(win.key)} ${overviewMoney(win.remaining)}`
      : subWindowRemainingLabel(win.key),
    t('dashboard.overview.subUsedOfLimit', {
      used: compactQuotaMoney(win.used),
      limit: compactQuotaMoney(win.limit),
    }),
  ]
  const reset = subWindowResetText(win)
  if (reset) parts.push(reset)
  return parts.join(' · ')
})

const todayCostFooterHint = computed(() => {
  const s = props.stats
  if (!s || props.loading || props.error) return t('dashboard.overview.actualCostHint')
  const subCost = s.today_subscription_actual_cost ?? 0
  const cashCost = s.today_balance_actual_cost ?? 0
  if (subCost > 0.0001) {
    if (cashCost > 0.0001) {
      return t('dashboard.overview.todaySplitSubAndCash', {
        sub: overviewMoney(subCost),
        cash: overviewMoney(cashCost),
      })
    }
    return t('dashboard.overview.todayPureSubHint')
  }
  return t('dashboard.overview.actualCostHint')
})

// Gentle high-spend PAYG -> subscription suggestion
const paygPlanNudge = computed(() => {
  if (!props.subscriptionEnabled || props.activeSubscriptions.length > 0) return null
  const plans = (props.checkoutInfo?.plans ?? []).filter((p) => p.for_sale && p.price > 0)
  if (!plans.length) return null
  const monthlySpend = runway.value.dailyBurn * 30
  if (monthlySpend <= 0) return null
  const cheapest = [...plans].sort((a, b) => a.price - b.price)[0]
  if (!cheapest || monthlySpend < cheapest.price * 1.15) return null
  const sym = cheapest.currency && cheapest.currency !== 'CNY' ? `${cheapest.currency} ` : '¥'
  return {
    monthlySpend,
    planName: cheapest.name,
    priceLabel: `${sym}${cheapest.price}`,
  }
})

function platformShare(item: DashboardPlatform): string | null {
  const total = props.stats?.total_actual_cost ?? 0
  if (total <= 0 || item.total_actual_cost <= 0) return null
  const pct = (item.total_actual_cost / total) * 100
  return pct < 0.1 ? '<0.1%' : `${pct.toFixed(1)}%`
}

function urgentPlatformQuota(q?: PlatformQuotaItem) {
  return mostUrgentQuota(q)
}

function quotaWindowPercent(used: number, limit: number): string {
  if (limit <= 0) return '0%'
  const pct = Math.min(999, (used / limit) * 100)
  return `${pct.toFixed(1)}%`
}

function quotaWindowState(used: number, limit: number): 'active' | 'near' | 'exhausted' | 'disabled' {
  if (limit <= 0) return 'disabled'
  const ratio = used / limit
  if (ratio >= 1) return 'exhausted'
  if (ratio >= 0.75) return 'near'
  return 'active'
}

function resetTime(value?: string | null) {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime())
    ? '—'
    : new Intl.DateTimeFormat('zh-CN', {
        timeZone: DASHBOARD_TIMEZONE,
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      }).format(date) + ' UTC+8'
}
</script>
