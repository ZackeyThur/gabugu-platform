<template>
  <div
    class="studio-root"
    :data-studio-theme="studioTheme"
    :class="{
      'has-result': hasResult,
      'fav-only': favOnly,
      'is-scrolled': isScrolled,
    }"
  >
    <!-- 顶栏 -->
    <header class="st-topbar">
      <div class="st-topbar-left">
        <router-link to="/dashboard" class="st-top-btn st-back-btn" title="返回控制台" aria-label="返回控制台">
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M10 3.5 5.5 8l4.5 4.5" />
          </svg>
          <span class="st-back-lbl">控制台</span>
        </router-link>

        <div class="st-brand" aria-label="GABUGU 画室">
          <img
            class="st-brand-icon"
            :src="studioTheme === 'dark' ? '/brand/gabugu-logo-dark.png' : '/brand/gabugu-logo-light.png'"
            alt=""
            width="30"
            height="30"
          />
          <svg
            class="st-brand-wordmark"
            viewBox="0 0 932 154"
            role="img"
            aria-label="GABUGU"
            focusable="false"
            fill-rule="evenodd"
          >
            <path d="M 63.500 6.482 C 26.248 12.462, 2.711 39.765, 2.712 77 C 2.712 121.561, 33.695 148.938, 84.178 148.987 C 113.664 149.015, 132.420 143.170, 146.132 129.679 C 154.800 121.150, 155 120.363, 155 94.740 C 155 78.698, 154.662 72.062, 153.800 71.200 C 152.921 70.321, 144.683 70, 122.974 70 L 93.349 70 91.180 72.308 C 89.988 73.578, 86.555 78.190, 83.552 82.558 L 78.092 90.500 101.498 91 C 115.968 91.309, 125.308 91.904, 125.961 92.559 C 127.766 94.367, 127.417 106.318, 125.438 110.489 C 121.413 118.970, 109.906 123.633, 90.685 124.569 C 70.183 125.567, 55.637 121.595, 44.772 112.031 C 21.861 91.864, 26.989 50.281, 54.051 36.782 C 64.827 31.406, 68.441 31, 105.519 31 C 131.239 31, 140.029 30.693, 141.290 29.750 C 143.667 27.973, 157.925 6.687, 157.355 5.765 C 156.519 4.412, 72.393 5.055, 63.500 6.482" />
            <path d="M 64.378 5.765 C 63.254 6.502, 58.640 16.752, 37.266 66 C 8.981 131.172, 3.050 145.272, 3.567 146.108 C 4.522 147.653, 27.734 147.189, 30.882 145.561 C 32.992 144.470, 34.839 141.513, 38.529 133.311 C 41.205 127.365, 43.811 121.938, 44.320 121.250 C 45.025 120.297, 52.645 119.996, 76.372 119.985 C 93.493 119.976, 107.994 119.639, 108.598 119.235 C 109.202 118.831, 111.605 115.462, 113.938 111.750 C 116.271 108.037, 118.476 105.004, 118.840 105.010 C 119.203 105.015, 123.063 113.678, 127.417 124.260 C 131.771 134.842, 136.003 144.287, 136.822 145.250 C 138.580 147.318, 163.691 147.909, 165.670 145.930 C 166.717 144.883, 161.945 133.129, 122.054 38.500 C 115.330 22.550, 109.182 8.488, 108.391 7.250 C 106.976 5.034, 106.637 5, 86.227 5.015 C 74.827 5.024, 64.995 5.361, 64.378 5.765 M 81.096 35.250 C 80.088 37.587, 73.752 52.100, 67.016 67.500 C 60.281 82.900, 54.548 96.063, 54.278 96.750 C 53.891 97.734, 60.412 98, 84.893 98 C 102.002 98, 116 97.817, 116 97.594 C 116 96.608, 88.681 33.369, 87.771 32.250 C 85.677 29.673, 82.978 30.886, 81.096 35.250" transform="translate(162 0)" />
            <path d="M 3.571 6.571 C 1.304 8.839, 1.163 144.060, 3.426 145.939 C 5.966 148.046, 99.884 146.830, 107.500 144.591 C 121.368 140.514, 131.656 132.303, 136.786 121.218 C 140.945 112.231, 140.732 97.962, 136.306 88.981 C 132.438 81.132, 127.422 76.078, 119.714 72.259 L 113.928 69.393 117.899 66.202 C 127.494 58.490, 131.063 42.018, 126.021 28.711 C 122.012 18.133, 111.190 9.907, 96.825 6.521 C 87.173 4.245, 5.853 4.290, 3.571 6.571 M 31.065 29.878 C 29.503 32.798, 29.678 56.178, 31.279 58.367 C 33.346 61.193, 87.967 60.424, 92.819 57.500 C 102.628 51.589, 102.614 35.660, 92.794 30.299 C 89.882 28.708, 86.176 28.465, 60.815 28.194 C 33.039 27.898, 32.096 27.951, 31.065 29.878 M 31.733 84.743 C 30.227 86.407, 30 88.968, 30 104.294 C 30 121.230, 30.084 121.975, 32.117 123.062 C 33.605 123.859, 43.028 124.081, 63.867 123.811 C 92.362 123.441, 93.702 123.336, 98.773 121.080 C 114.765 113.964, 115.068 94.090, 99.301 86.348 L 93.500 83.500 63.483 83.164 C 34.949 82.845, 33.379 82.923, 31.733 84.743" transform="translate(340 0)" />
            <path d="M 13.538 7.706 C 12.095 9.243, 9.107 12.827, 6.897 15.670 L 2.880 20.841 3.207 62.670 L 3.534 104.500 6.095 111.120 C 16.432 137.831, 37.262 149.896, 71 148.714 C 101.393 147.649, 119.877 135.343, 128.751 110.269 L 131.500 102.500 131.500 54 L 131.500 5.500 121.327 5.216 L 111.154 4.932 108.577 8.310 L 106 11.689 106 51.795 C 106 77.682, 105.601 93.780, 104.876 97.201 C 100.907 115.907, 87.845 124.670, 65.211 123.813 C 48.568 123.183, 39.693 118.214, 33.785 106.217 L 30.500 99.547 30 52.524 L 29.500 5.500 22.831 5.206 C 16.641 4.933, 15.974 5.113, 13.538 7.706" transform="translate(490 0)" />
            <path d="M 63.500 6.482 C 26.248 12.462, 2.711 39.765, 2.712 77 C 2.712 121.561, 33.695 148.938, 84.178 148.987 C 113.664 149.015, 132.420 143.170, 146.132 129.679 C 154.800 121.150, 155 120.363, 155 94.740 C 155 78.698, 154.662 72.062, 153.800 71.200 C 152.921 70.321, 144.683 70, 122.974 70 L 93.349 70 91.180 72.308 C 89.988 73.578, 86.555 78.190, 83.552 82.558 L 78.092 90.500 101.498 91 C 115.968 91.309, 125.308 91.904, 125.961 92.559 C 127.766 94.367, 127.417 106.318, 125.438 110.489 C 121.413 118.970, 109.906 123.633, 90.685 124.569 C 70.183 125.567, 55.637 121.595, 44.772 112.031 C 21.861 91.864, 26.989 50.281, 54.051 36.782 C 64.827 31.406, 68.441 31, 105.519 31 C 131.239 31, 140.029 30.693, 141.290 29.750 C 143.667 27.973, 157.925 6.687, 157.355 5.765 C 156.519 4.412, 72.393 5.055, 63.500 6.482" transform="translate(630 0)" />
            <path d="M 13.538 7.706 C 12.095 9.243, 9.107 12.827, 6.897 15.670 L 2.880 20.841 3.207 62.670 L 3.534 104.500 6.095 111.120 C 16.432 137.831, 37.262 149.896, 71 148.714 C 101.393 147.649, 119.877 135.343, 128.751 110.269 L 131.500 102.500 131.500 54 L 131.500 5.500 121.327 5.216 L 111.154 4.932 108.577 8.310 L 106 11.689 106 51.795 C 106 77.682, 105.601 93.780, 104.876 97.201 C 100.907 115.907, 87.845 124.670, 65.211 123.813 C 48.568 123.183, 39.693 118.214, 33.785 106.217 L 30.500 99.547 30 52.524 L 29.500 5.500 22.831 5.206 C 16.641 4.933, 15.974 5.113, 13.538 7.706" transform="translate(796 0)" />
          </svg>
          <span class="st-brand-name">画室</span>
        </div>
      </div>

      <div class="st-topbar-right">
        <button
          v-if="hasResult"
          type="button"
          class="st-top-btn st-fav-btn"
          :class="{ 'has-favs': totalFavs > 0 }"
          :aria-pressed="favOnly"
          :title="favOnly ? '点击恢复查看全部作品' : totalFavs > 0 ? `只看已收藏的 ${totalFavs} 张作品` : '只看已收藏的作品'"
          @click="toggleFavFilter"
        >
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" aria-hidden="true">
            <path d="m8 2.1 1.75 3.55 3.92.57-2.84 2.76.67 3.9L8 11.04 4.5 12.88l.67-3.9-2.84-2.76 3.92-.57L8 2.1Z" />
          </svg>
          <span class="st-fav-lbl">收藏</span>
          <span v-if="totalFavs > 0">{{ totalFavs }}</span>
        </button>

        <button
          type="button"
          class="st-wallet"
          :class="{ flash: walletFlash }"
          title="点击刷新当前账户余额"
          @click="refreshBalance(true)"
        >
          <span class="st-wallet-lbl">账户余额</span>
          <b>{{ balanceLoaded ? formatWalletUsd(balance) : '加载中…' }}</b>
        </button>

        <button
          type="button"
          class="st-top-btn st-new-btn"
          title="新画布"
          aria-label="新画布"
          @click="handleNewCanvasClick"
        >
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" aria-hidden="true">
            <path d="M8 3.2v9.6M3.2 8h9.6" />
          </svg>
          <span>新画布</span>
        </button>

        <button
          type="button"
          class="st-top-btn icon-only"
          :class="{ spinning: themeSpinning }"
          :title="studioTheme === 'dark' ? '切换到白天暖米色' : '切换到夜间暖炭灰'"
          :aria-label="studioTheme === 'dark' ? '切换到白天暖米色' : '切换到夜间暖炭灰'"
          @click="toggleStudioTheme"
        >
          <svg v-if="studioTheme === 'dark'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" aria-hidden="true">
            <circle cx="8" cy="8" r="3" />
            <path d="M8 1.8v1.4M8 12.8v1.4M1.8 8h1.4M12.8 8h1.4M3.6 3.6l1 1M11.4 11.4l1 1M12.4 3.6l-1 1M4.6 11.4l-1 1" />
          </svg>
          <svg v-else viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M13.2 9.8A5.5 5.5 0 0 1 6.2 2.8a5.5 5.5 0 1 0 7 7Z" />
          </svg>
        </button>
      </div>
    </header>

    <!-- 主舞台 -->
    <main class="st-stage">
      <!-- 中央滚轮历史流 -->
      <section v-if="hasResult" ref="feedStreamRef" class="st-feed-stream" aria-label="历史生成记录">
        <template v-for="(v, verIdx) in versions" :key="v.id">
          <article
            v-show="!favOnly || versionHasFav(v)"
            :ref="(el) => setCardRef(verIdx, el)"
            class="st-feed-card"
            :class="{
              'active-card': currentVersionIdx === verIdx,
              'focus-flash': flashCardIdx === verIdx,
            }"
            :data-ver="verIdx"
          >
            <!-- 卡片头 -->
            <div class="st-feed-head">
              <div class="st-feed-prompt-row">
                <div class="st-feed-prompt-main">
                  <p class="st-feed-prompt">{{ v.prompt }}</p>
                  <button
                    type="button"
                    class="st-copy-prompt-btn"
                    title="复制提示词"
                    aria-label="复制提示词"
                    @click="copyText(v.prompt, `已复制第 ${verIdx + 1} 版提示词`)"
                  >
                    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                      <rect x="5.5" y="5.5" width="8" height="8" rx="1.8" />
                      <path d="M10.5 5.5V3.8A1.3 1.3 0 0 0 9.2 2.5H3.8A1.3 1.3 0 0 0 2.5 3.8v5.4a1.3 1.3 0 0 0 1.3 1.3h1.7" />
                    </svg>
                  </button>
                </div>
                <div class="st-feed-head-right">
                  <span class="st-ver-index">第 {{ verIdx + 1 }} 版</span>
                  <template v-if="v.costKnown">
                    <span class="st-meta-dot">·</span>
                    <span>约 {{ formatStudioUsd(v.cost) }}</span>
                  </template>
                </div>
              </div>

              <div class="st-feed-meta-line">
                <span v-if="v.branchLabel" class="st-branch-badge">{{ v.branchLabel }}</span>
                <span class="st-feed-meta-model">
                  <StudioVendorIcon :vendor="getStudioModel(v.model).protocol" :size="13" />
                  <span>{{ getStudioModel(v.model).name }}</span>
                </span>
                <template v-for="(tag, tIdx) in buildVersionMetaTags(v)" :key="tIdx">
                  <span class="st-meta-dot">·</span>
                  <span>{{ tag }}</span>
                </template>
              </div>
            </div>

            <!-- 画作展示 -->
            <div
              class="st-feed-gallery"
              :class="{
                'layout-grid2': v.images.length === 4 && v.w / Math.max(1, v.h) > 1.35,
                'layout-strip': v.images.length === 4 && v.w / Math.max(1, v.h) <= 1.35,
              }"
              :data-count="v.images.length"
            >
              <div
                v-for="(img, imgIdx) in v.images"
                :key="imgIdx"
                class="st-slot"
                :class="{
                  sel: v.selectedIdx === imgIdx,
                  'is-fav': v.favs[imgIdx],
                  'is-transparent': v.background === 'transparent',
                }"
                :style="{ animationDelay: `${imgIdx * 45}ms` }"
                @click="handleSlotClick(verIdx, imgIdx)"
                @dblclick="openZoom(verIdx, imgIdx)"
              >
                <img :src="img.objectUrl" :alt="`第 ${verIdx + 1} 版 第 ${imgIdx + 1} 张`" loading="lazy" />

                <span v-if="v.images.length > 1" class="st-slot-idx">{{ imgIdx + 1 }}</span>

                <div class="st-slot-tr">
                  <button
                    type="button"
                    class="st-slot-icon-btn st-slot-zoom-btn"
                    title="放大查看"
                    :aria-label="`放大查看第 ${imgIdx + 1} 张`"
                    @click.stop="openZoom(verIdx, imgIdx)"
                  >
                    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                      <circle cx="7.2" cy="7.2" r="4.5" />
                      <path d="m10.7 10.7 2.8 2.8M7.2 5.2v4M5.2 7.2h4" />
                    </svg>
                  </button>
                  <button
                    type="button"
                    class="st-slot-icon-btn st-slot-star-btn"
                    :class="{
                      'is-fav': v.favs[imgIdx],
                      popping: poppingStarKey === `${verIdx}:${imgIdx}`,
                    }"
                    :title="v.favs[imgIdx] ? '取消收藏' : '收藏这张图'"
                    :aria-label="`${v.favs[imgIdx] ? '取消收藏' : '收藏'}第 ${imgIdx + 1} 张`"
                    :aria-pressed="v.favs[imgIdx]"
                    @click.stop="toggleFav(verIdx, imgIdx)"
                  >
                    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" aria-hidden="true">
                      <path d="m8 2.1 1.75 3.55 3.92.57-2.84 2.76.67 3.9L8 11.04 4.5 12.88l.67-3.9-2.84-2.76 3.92-.57L8 2.1Z" />
                    </svg>
                  </button>
                </div>

                <button
                  type="button"
                  class="st-slot-quick-edit"
                  title="将这张图加入下方参考槽继续改图"
                  @click.stop="addVersionRef(verIdx, imgIdx)"
                >
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <path d="M11.3 2.7a1.6 1.6 0 0 1 2.2 2.2L6 12.4 3 13l.6-3 7.7-7.3Z" />
                  </svg>
                  <span>以这张改</span>
                </button>
              </div>
            </div>

            <!-- 底部操作栏 -->
            <div class="st-feed-actions">
              <div class="st-act-group">
                <button
                  type="button"
                  class="st-act act-edit-primary"
                  @click="addVersionRef(verIdx, v.selectedIdx)"
                >
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <path d="M11.3 2.7a1.6 1.6 0 0 1 2.2 2.2L6 12.4 3 13l.6-3 7.7-7.3Z" />
                  </svg>
                  <span>从这张改</span>
                </button>
                <button type="button" class="st-act" @click="reuseVersion(verIdx)">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <path d="M13 8a5 5 0 1 1-1.3-3.4M13 2.8v2.7h-2.7" />
                  </svg>
                  <span>复用设置</span>
                </button>
                <button
                  v-if="versions.length >= 2"
                  type="button"
                  class="st-act"
                  @click="openCompareModal(verIdx)"
                >
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <rect x="2" y="3" width="12" height="10" rx="2" />
                    <path d="M8 2v12" />
                  </svg>
                  <span>对比</span>
                </button>
              </div>

              <div class="st-act-group">
                <span v-if="v.images.length > 1" class="st-feed-sel-hint">
                  已选第 {{ v.selectedIdx + 1 }} / {{ v.images.length }} 张
                </span>
                <button type="button" class="st-act" @click="downloadVersionImage(verIdx, v.selectedIdx)">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <path d="M8 2.5v8M4.8 7.5 8 10.7l3.2-3.2M3 13h10" />
                  </svg>
                  <span>下载</span>
                </button>
                <button type="button" class="st-act" @click="openZoom(verIdx, v.selectedIdx)">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <circle cx="7.2" cy="7.2" r="4.5" />
                    <path d="m10.7 10.7 2.8 2.8M7.2 5.2v4M5.2 7.2h4" />
                  </svg>
                  <span>放大</span>
                </button>
                <button type="button" class="st-act" @click="openCodeModal(verIdx)">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                    <path d="M5.5 4.5 2 8l3.5 3.5M10.5 4.5 14 8l-3.5 3.5M9 3 7 13" />
                  </svg>
                  <span>复制代码</span>
                </button>
              </div>
            </div>
          </article>
        </template>

        <!-- 收藏筛选空态 -->
        <div v-if="favOnly && versions.length > 0 && visibleVersionCount === 0" class="st-fav-empty-state">
          <strong>还没有收藏的作品</strong>
          <span>把鼠标移到任意画面右上角点亮星标，或在放大预览里点收藏。</span>
          <button type="button" class="st-btn-sm primary" @click="favOnly = false">查看全部版本</button>
        </div>

        <!-- 正在生成占位卡 -->
        <article v-if="pendingJob" ref="genCardRef" class="st-feed-card" aria-live="polite">
          <div class="st-feed-head">
            <div class="st-feed-prompt-row">
              <p class="st-feed-prompt">{{ pendingJob.prompt }}</p>
              <div class="st-feed-head-right">
                <span class="st-ver-index">第 {{ versions.length + 1 }} 版</span>
                <template v-if="pendingJob.costKnown">
                  <span class="st-meta-dot">·</span>
                  <span>预计 {{ formatStudioUsd(pendingJob.cost) }}</span>
                </template>
              </div>
            </div>
            <div class="st-feed-meta-line">
              <span v-if="pendingJob.branchLabel" class="st-branch-badge">{{ pendingJob.branchLabel }}</span>
              <span class="st-feed-meta-model">
                <StudioVendorIcon :vendor="getStudioModel(pendingJob.model).protocol" :size="13" />
                <span>{{ getStudioModel(pendingJob.model).name }}</span>
              </span>
              <span class="st-meta-dot">·</span>
              <span>{{ pendingJob.ratioCode }} · {{ pendingJob.tier }}</span>
            </div>
          </div>

          <div
            class="st-gen-skeleton"
            :style="{ width: `${pendingSkeletonSize.w}px`, height: `${pendingSkeletonSize.h}px`, maxWidth: '100%' }"
          >
            <div class="st-gen-skeleton-inner">
              <span class="st-spinner" style="width: 18px; height: 18px; border-width: 2px"></span>
              <span>{{ pendingStatusText }}</span>
              <span class="st-gen-skeleton-sub">
                {{ getStudioModel(pendingJob.model).shortName }} · {{ pendingJob.ratioCode }} · {{ pendingJob.tier }} · {{ pendingJob.count }} 张
              </span>
            </div>
          </div>
        </article>
      </section>

      <!-- 中央输入卡 / 即梦式底部悬浮栏 -->
      <div ref="composerDockRef" class="st-composer-dock">
        <button
          v-if="hasResult"
          type="button"
          class="st-jump-latest"
          :class="{ show: showJumpLatest }"
          aria-label="回到最新生成"
          @click="scrollCardIntoView(versions.length - 1, true)"
        >
          ↓ 回到最新
        </button>

        <div
          class="st-composer"
          :class="{
            'dock-pulse': dockPulse,
            nudge: composerNudge,
            'drag-over': dragOver,
          }"
          @dragenter="onDragEnter"
          @dragover="onDragOver"
          @dragleave="onDragLeave"
          @drop="onDrop"
        >
          <div class="st-drop-overlay" aria-hidden="true">
            <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
              <rect x="2" y="2.5" width="12" height="11" rx="2" />
              <circle cx="5.8" cy="6.2" r="1.2" />
              <path d="m14 10.6-3-3a1 1 0 0 0-1.4 0L4.5 13" />
            </svg>
            <span>松手添加参考图</span>
          </div>

          <div class="st-composer-main">
            <textarea
              ref="promptRef"
              v-model="promptText"
              class="st-prompt-input"
              rows="3"
              :placeholder="promptPlaceholder"
              spellcheck="false"
              @compositionstart="isComposing = true"
              @compositionend="isComposing = false"
              @keydown="onPromptKeydown"
              @input="autoGrowPrompt"
              @paste="onPromptPaste"
            ></textarea>
          </div>

          <!-- 多图参考槽 -->
          <div v-if="refs.length > 0" class="st-composer-refs" aria-label="已选参考图">
            <div v-for="r in refs" :key="r.id" class="st-ref-tile" :title="r.tip">
              <img :src="r.thumbUrl" :alt="r.tip" />
              <span class="st-ref-tag">{{ r.tag }}</span>
              <button
                type="button"
                class="st-ref-rm"
                aria-label="移除参考图"
                title="移除"
                @click.stop="removeRef(r.id)"
              >
                ×
              </button>
            </div>

            <button
              v-if="refs.length < currentModel.maxRefImages"
              type="button"
              class="st-ref-add-tile"
              :title="`继续添加参考图（当前模型最多 ${currentModel.maxRefImages} 张）`"
              aria-label="继续添加参考图"
              @click="triggerRefFilePicker"
            >
              +
            </button>

            <div class="st-ref-meta">
              <span>参考 {{ refs.length }}/{{ currentModel.maxRefImages }}</span>
              <button type="button" class="st-ref-clear-all" @click="clearAllRefs">清空</button>
            </div>
          </div>

          <!-- 内联确认条（切换模型冲突 / 复用覆盖 / 新画布清空） -->
          <div v-if="confirmState" class="st-confirm" role="alert">
            <div>
              <strong v-if="confirmState.title">{{ confirmState.title }}</strong>
              <span>{{ confirmState.body }}</span>
            </div>
            <div class="st-confirm-actions">
              <button
                v-for="(act, idx) in confirmState.actions"
                :key="idx"
                type="button"
                class="st-btn-sm"
                :class="{ primary: act.primary }"
                @click="runConfirmAction(act)"
              >
                {{ act.label }}
              </button>
            </div>
          </div>

          <!-- 无图像分组权限提示 -->
          <div v-else-if="contextLoaded && !imageGroup" class="st-error-banner" role="alert">
            <span>当前账号暂未开通「图像生成」分组权限，暂无法直接出图。可前往「可用渠道」查看已开通分组或联系管理员。</span>
          </div>

          <!-- 真实错误条（保留草稿与参考图） -->
          <div v-if="inlineError" class="st-error-banner" role="alert">
            <span>{{ inlineError }}</span>
            <button type="button" class="st-error-close" aria-label="关闭提示" @click="inlineError = ''">×</button>
          </div>

          <!-- 生成进度行 -->
          <div v-if="generating" class="st-pending" aria-live="polite">
            <span class="st-spinner"></span>
            <span>{{ pendingStatusText }}</span>
          </div>

          <!-- 框内工具栏 -->
          <div class="st-composer-bar">
            <div class="st-tools">
              <input
                ref="refFileInputRef"
                type="file"
                accept="image/*"
                multiple
                hidden
                @change="onRefFileChange"
              />
              <button
                type="button"
                class="st-chip st-chip-icon-btn"
                :class="{ 'has-ref': refs.length > 0 }"
                :disabled="generating"
                :title="`添加参考图（当前模型最多 ${currentModel.maxRefImages} 张，支持多选、粘贴或拖入）`"
                aria-label="添加参考图"
                @click="triggerRefFilePicker"
              >
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <rect x="2" y="2.5" width="12" height="11" rx="2" />
                  <circle cx="5.8" cy="6.2" r="1.1" />
                  <path d="m14 10.6-3-3a1 1 0 0 0-1.4 0L4.5 13" />
                </svg>
              </button>

              <!-- 模型 -->
              <button
                ref="btnModelRef"
                type="button"
                class="st-chip"
                :disabled="generating"
                aria-haspopup="menu"
                :aria-expanded="openPopover === 'model'"
                @click="togglePopover('model', btnModelRef)"
              >
                <span class="st-model-chip-icon" aria-hidden="true">
                  <StudioVendorIcon :vendor="currentModel.protocol" :size="15" />
                </span>
                <span>{{ isMobile ? currentModel.shortName : currentModel.name }}</span>
                <svg class="st-chev" viewBox="0 0 10 10" fill="none" aria-hidden="true">
                  <path d="M2 3.8 5 6.8l3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </button>

              <!-- 比例 + 横竖 -->
              <button
                ref="btnRatioRef"
                type="button"
                class="st-chip"
                :disabled="generating"
                aria-haspopup="menu"
                :aria-expanded="openPopover === 'ratio'"
                @click="togglePopover('ratio', btnRatioRef)"
              >
                <span
                  class="st-swatch"
                  :class="{ 'is-auto': currentRatioCode === 'auto' }"
                  :style="{ width: `${ratioSwatchBox.w}px`, height: `${ratioSwatchBox.h}px` }"
                ></span>
                <span>{{ currentRatioCode === 'auto' ? '自动' : currentRatioCode }}</span>
                <span v-if="ratioDirLabel" class="st-ratio-dir-tag">{{ ratioDirLabel }}</span>
                <svg class="st-chev" viewBox="0 0 10 10" fill="none" aria-hidden="true">
                  <path d="M2 3.8 5 6.8l3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </button>

              <!-- 清晰度 -->
              <button
                ref="btnSizeRef"
                type="button"
                class="st-chip"
                :disabled="generating"
                :title="currentDims.isAuto ? '当前比例为自动（auto），由模型决定最终出图尺寸并按实际档位计费' : '选择输出清晰度档位'"
                aria-haspopup="menu"
                :aria-expanded="openPopover === 'size'"
                @click="togglePopover('size', btnSizeRef)"
              >
                <span>{{ currentDims.isAuto ? '自适应' : TIER_META[tier].label }}</span>
                <svg class="st-chev" viewBox="0 0 10 10" fill="none" aria-hidden="true">
                  <path d="M2 3.8 5 6.8l3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </button>

              <!-- 更多参数 -->
              <button
                ref="btnMoreRef"
                type="button"
                class="st-chip"
                :disabled="generating"
                aria-haspopup="menu"
                :aria-expanded="openPopover === 'more'"
                :title="customOptionTags.length ? `已调参数：${customOptionTags.join(' · ')}` : '更多模型参数'"
                @click="togglePopover('more', btnMoreRef)"
              >
                <span>更多</span>
                <span v-if="customOptionTags.length > 0" class="st-count-badge">
                  {{ customOptionTags.length === 1 ? customOptionTags[0] : `${customOptionTags.length} 项` }}
                </span>
                <svg class="st-chev" viewBox="0 0 10 10" fill="none" aria-hidden="true">
                  <path d="M2 3.8 5 6.8l3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
              </button>
            </div>

            <span class="st-spacer"></span>

            <div class="st-composer-actions">
              <span class="st-est-price" :class="{ 'price-pop': pricePop }">
                <template v-if="currentDims.isAuto && costRange.known">
                  {{ effectiveCount > 1 ? `${effectiveCount} 张 · 按出图 ` : '按出图 ' }}<strong>{{ formatStudioUsd(costRange.minTotalPrice) }}–{{ formatStudioUsd(costRange.maxTotalPrice) }}</strong>
                </template>
                <template v-else-if="costEstimate.known">
                  {{ effectiveCount > 1 ? `${effectiveCount} 张 · 预计 ` : '预计 ' }}<strong>{{ formatStudioUsd(costEstimate.totalPrice) }}</strong>
                </template>
                <template v-else>按实际用量结算</template>
              </span>

              <button
                type="button"
                class="st-gen-btn"
                :class="{ warn: isInsufficientBalance }"
                :disabled="generating || (contextLoaded && !imageGroup)"
                @click="handleGenButtonClick"
              >
                <template v-if="generating">
                  <span class="st-btn-spinner"></span>
                  <span>处理中…</span>
                </template>
                <template v-else-if="isInsufficientBalance">余额不足 · 去充值</template>
                <template v-else-if="refs.length > 1">参考生图</template>
                <template v-else-if="refs.length === 1">生成改图</template>
                <template v-else>生成</template>
              </button>
            </div>
          </div>
        </div>

        <p class="st-hint-line">
          <span>Enter 生成，Shift + Enter 换行</span>
          <span class="st-hint-sep">·</span>
          <span>
            {{
              historyPersisted
                ? '历史记录自动保存在本机浏览器，换设备前请下载原图'
                : '当前浏览器隐私模式不支持本地保存，刷新前请下载原图'
            }}
          </span>
          <template v-if="sessionSpent > 0">
            <span class="st-hint-sep">·</span>
            <span>累计约用 {{ formatStudioUsd(sessionSpent) }}</span>
          </template>
        </p>
      </div>

      <!-- 右侧微缩定位条 -->
      <aside v-if="versions.length > 0" class="st-rail" aria-label="本次记录定位">
        <div class="st-rail-head">
          {{ favOnly ? `${visibleVersionCount}/${versions.length}` : `${versions.length} 版` }}
        </div>
        <div class="st-rail-list">
          <template v-for="(v, idx) in versions" :key="v.id">
            <button
              v-show="!favOnly || versionHasFav(v)"
              type="button"
              class="st-ver"
              :class="{ active: currentVersionIdx === idx }"
              :title="`第 ${idx + 1} 版 · ${getStudioModel(v.model).shortName} · ${v.ratioCode}\n${v.prompt}`"
              @click="scrollCardIntoView(idx, true)"
            >
              <img :src="v.images[0]?.thumbUrl || v.images[0]?.objectUrl" :alt="`第 ${idx + 1} 版`" />
              <span class="st-vnum">{{ idx + 1 }}</span>
              <span v-if="v.refCount > 0" class="st-vbranch-dot" title="参考图改图版本"></span>
              <span v-if="versionHasFav(v)" class="st-vfav-dot" title="含已收藏作品"></span>
            </button>
          </template>
        </div>
      </aside>
    </main>

    <!-- 弹层 1：模型选择（固定顶栏 + 厂商 Logo 筛选） -->
    <div
      v-if="openPopover === 'model'"
      ref="popoverRef"
      class="st-pop model-pop"
      role="menu"
      :style="popoverStyle"
    >
      <div class="st-model-pop-top">
        <div class="st-pop-head">
          <span>选择生图模型</span>
          <span class="st-pop-head-sub">按厂商快速查阅</span>
        </div>
        <div class="st-vendor-filter-bar" role="group" aria-label="按厂商筛选模型">
          <button
            type="button"
            class="st-vendor-pill"
            :aria-pressed="modelVendorFilter === 'all'"
            @click="modelVendorFilter = 'all'"
          >
            <span>全部</span>
          </button>
          <button
            v-for="vk in VENDOR_ORDER"
            :key="vk"
            type="button"
            class="st-vendor-pill"
            :aria-pressed="modelVendorFilter === vk"
            @click="modelVendorFilter = vk"
          >
            <StudioVendorIcon :vendor="vk" :size="13" />
            <span>{{ VENDOR_META[vk].name }}</span>
          </button>
        </div>
      </div>

      <div class="st-pop-scroll">
        <template v-if="modelVendorFilter !== 'all'">
          <div class="st-pop-head">
            <span>{{ VENDOR_META[modelVendorFilter].name }}</span>
            <span class="st-pop-head-sub">{{ filteredVendorModels.length }} 款模型</span>
          </div>
          <button
            v-for="m in filteredVendorModels"
            :key="m.id"
            type="button"
            class="st-item"
            :class="{ selected: modelId === m.id }"
            role="menuitemradio"
            :aria-checked="modelId === m.id"
            @click="selectModelFromMenu(m.id)"
          >
            <span class="st-vendor-badge" :class="`vendor-${m.protocol}`">
              <StudioVendorIcon :vendor="m.protocol" :size="16" />
            </span>
            <span class="st-item-body">
              <span class="st-item-name-row">
                <span class="st-item-name">{{ m.name }}</span>
                <span v-if="modelMinPriceLabel(m)" class="st-item-right">{{ modelMinPriceLabel(m) }}</span>
              </span>
              <span class="st-item-sub">{{ m.summary }}</span>
            </span>
            <svg class="st-item-tick" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M2.5 7.2 5.6 10.3 11.5 3.8" />
            </svg>
          </button>
        </template>

        <template v-else>
          <div class="st-pop-head">
            <span>主力推荐</span>
            <span class="st-pop-head-sub">按线上真实通道能力约束</span>
          </div>
          <button
            v-for="m in featuredModels"
            :key="m.id"
            type="button"
            class="st-item"
            :class="{ selected: modelId === m.id }"
            role="menuitemradio"
            :aria-checked="modelId === m.id"
            @click="selectModelFromMenu(m.id)"
          >
            <span class="st-vendor-badge" :class="`vendor-${m.protocol}`">
              <StudioVendorIcon :vendor="m.protocol" :size="16" />
            </span>
            <span class="st-item-body">
              <span class="st-item-name-row">
                <span class="st-item-name">{{ m.name }}</span>
                <span v-if="modelMinPriceLabel(m)" class="st-item-right">{{ modelMinPriceLabel(m) }}</span>
              </span>
              <span class="st-item-sub">{{ m.summary }}</span>
            </span>
            <svg class="st-item-tick" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M2.5 7.2 5.6 10.3 11.5 3.8" />
            </svg>
          </button>

          <div class="st-pop-sep"></div>

          <template v-if="showAllModels || !currentModel.featured">
            <div class="st-pop-head">
              <span>更多模型</span>
              <span class="st-pop-head-sub">{{ otherModels.length }} 款</span>
            </div>
            <button
              v-for="m in otherModels"
              :key="m.id"
              type="button"
              class="st-item"
              :class="{ selected: modelId === m.id }"
              role="menuitemradio"
              :aria-checked="modelId === m.id"
              @click="selectModelFromMenu(m.id)"
            >
              <span class="st-vendor-badge" :class="`vendor-${m.protocol}`">
                <StudioVendorIcon :vendor="m.protocol" :size="16" />
              </span>
              <span class="st-item-body">
                <span class="st-item-name-row">
                  <span class="st-item-name">{{ m.name }}</span>
                  <span v-if="modelMinPriceLabel(m)" class="st-item-right">{{ modelMinPriceLabel(m) }}</span>
                </span>
                <span class="st-item-sub">{{ m.summary }}</span>
              </span>
              <svg class="st-item-tick" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path d="M2.5 7.2 5.6 10.3 11.5 3.8" />
              </svg>
            </button>
          </template>
          <button v-else type="button" class="st-pop-toggle" @click="showAllModels = true">
            <span>展开全部模型（共 {{ STUDIO_MODEL_LIST.length }} 款，含 2.5 / GPT Image 2 / Flash Lite）</span>
            <span>▾</span>
          </button>
        </template>
      </div>
    </div>

    <!-- 弹层 2：画面比例与横竖 -->
    <div
      v-if="openPopover === 'ratio'"
      ref="popoverRef"
      class="st-pop ratio-pop"
      role="menu"
      :style="popoverStyle"
    >
      <div class="st-pop-head">
        <span>画面比例与方向</span>
        <span class="st-pop-head-sub">{{ currentModel.shortName }}</span>
      </div>

      <div class="st-ratio-header">
        <div class="st-ratio-stage-wrap">
          <div
            class="st-ratio-stage-box"
            :class="{ 'is-auto': previewBaseRatio === 'auto' }"
            :style="{ width: `${ratioPreviewBox.w}px`, height: `${ratioPreviewBox.h}px` }"
          >
            {{ ratioPreviewCode }}
          </div>
        </div>
        <div class="st-ratio-dir-col">
          <div class="st-seg full">
            <button
              type="button"
              :aria-pressed="orientation === 'landscape'"
              @click="setOrientation('landscape')"
            >
              横版
            </button>
            <button
              type="button"
              :disabled="!modelSupportsPortrait"
              :aria-pressed="orientation === 'portrait'"
              @click="setOrientation('portrait')"
            >
              竖版
            </button>
          </div>
          <div class="st-ratio-stage-cap">{{ ratioPreviewCaption }}</div>
          <div class="st-ratio-stage-sub">{{ ratioPreviewSub }}</div>
        </div>
      </div>

      <div class="st-ratio-grid">
        <button
          v-for="entry in currentModel.ratios"
          :key="entry.base"
          type="button"
          class="st-ratio-tile"
          :class="{ selected: baseRatio === entry.base }"
          @mouseenter="hoveredRatio = entry.base"
          @focus="hoveredRatio = entry.base"
          @mouseleave="hoveredRatio = null"
          @click="selectRatioTile(entry.base)"
        >
          <span class="st-rt-code">{{ ratioTileCode(entry) }}</span>
          <span class="st-rt-name">{{ ratioTileName(entry) }}</span>
        </button>
      </div>
    </div>

    <!-- 弹层 3：清晰度档位 -->
    <div
      v-if="openPopover === 'size'"
      ref="popoverRef"
      class="st-pop"
      role="menu"
      :style="popoverStyle"
    >
      <div class="st-pop-head">
        <span>清晰度档位</span>
        <span class="st-pop-head-sub">当前 {{ currentRatioCode === 'auto' ? '自动比例' : currentRatioCode }}</span>
      </div>

      <button
        v-for="tId in currentModel.tiers"
        :key="tId"
        type="button"
        class="st-item"
        :class="{ selected: tier === tId }"
        role="menuitemradio"
        :aria-checked="tier === tId"
        @click="selectTier(tId)"
      >
        <span class="st-item-body">
          <span class="st-item-name-row">
            <span class="st-item-name">
              {{ TIER_META[tId].label }}
              <span class="st-item-note">{{ TIER_META[tId].note }}</span>
            </span>
            <span class="st-item-right">{{ tierRightMeta(tId) }}</span>
          </span>
        </span>
        <svg class="st-item-tick" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M2.5 7.2 5.6 10.3 11.5 3.8" />
        </svg>
      </button>

      <div class="st-pop-footnote">
        <template v-if="currentDims.isAuto">
          当前比例为自动（auto），尺寸与计费档位将由模型按实际出图决定；切换为固定比例（如 1:1 / 16:9）后可精确锁定档位与单价。
        </template>
        <template v-else>
          仅展示 {{ currentModel.shortName }} 实际支持的清晰度档位（16 像素对齐，与计费档位一致）。
        </template>
      </div>
    </div>

    <!-- 弹层 4：动态模型参数台 -->
    <div
      v-if="openPopover === 'more'"
      ref="popoverRef"
      class="st-pop wide-pop"
      role="menu"
      :style="popoverStyle"
    >
      <div class="st-pop-head">
        <span>模型参数</span>
        <span class="st-pop-head-sub">{{ currentModel.shortName }}</span>
      </div>

      <div class="st-pop-scroll">
        <!-- 一次生成张数 -->
        <div v-if="currentModel.maxN > 1" class="st-ctrl-row">
          <div>
            <div class="st-ctrl-lbl">一次生成</div>
            <div class="st-ctrl-sub">多张按实际张数合计</div>
          </div>
          <div class="st-seg">
            <button
              v-for="nVal in availableCounts"
              :key="nVal"
              type="button"
              :aria-pressed="effectiveCount === nVal"
              @click="count = nVal"
            >
              {{ nVal }} 张
            </button>
          </div>
        </div>

        <!-- 生成质量 -->
        <div v-if="currentModel.qualityLevels.length > 0" class="st-ctrl-block">
          <div style="display: flex; justify-content: space-between; align-items: baseline">
            <span class="st-ctrl-lbl">生成质量（{{ currentModel.qualityLevels.length }} 档）</span>
            <span class="st-ctrl-val">{{ quality }}</span>
          </div>
          <div class="st-ctrl-sub">{{ quality ? QUALITY_META[quality]?.desc : '' }}</div>

          <div v-if="currentModel.qualityLevels.length <= 4" class="st-seg full">
            <button
              v-for="qVal in currentModel.qualityLevels"
              :key="qVal"
              type="button"
              :aria-pressed="quality === qVal"
              @click="quality = qVal"
            >
              {{ QUALITY_META[qVal]?.label || qVal }}
            </button>
          </div>
          <div
            v-else
            class="st-opt-grid"
            :class="currentModel.qualityLevels.length === 5 ? 'cols-5' : 'cols-3'"
          >
            <button
              v-for="qVal in currentModel.qualityLevels"
              :key="qVal"
              type="button"
              class="st-opt-pill"
              :aria-pressed="quality === qVal"
              @click="quality = qVal"
            >
              {{ QUALITY_META[qVal]?.label || qVal }}
            </button>
          </div>
        </div>
        <div v-else class="st-pop-footnote">
          {{ currentModel.name }} 原生协议固定单次 1 张、无独立画质档位，由「清晰度」直接决定输出精细度；支持最多 {{ currentModel.maxRefImages }} 张参考图编辑。
        </div>

        <!-- 构图规划 complexity -->
        <div v-if="currentModel.supportsComplexity" class="st-ctrl-row">
          <div>
            <div class="st-ctrl-lbl">
              构图规划<code class="st-ctrl-code">complexity</code>
            </div>
            <div class="st-ctrl-sub">{{ COMPLEXITY_META[complexity].desc }}</div>
          </div>
          <div class="st-seg">
            <button
              v-for="cVal in (['low', 'medium', 'high'] as const)"
              :key="cVal"
              type="button"
              :aria-pressed="complexity === cVal"
              @click="complexity = cVal"
            >
              {{ COMPLEXITY_META[cVal].label }}
            </button>
          </div>
        </div>

        <!-- 输出格式 -->
        <template v-if="currentModel.outputFormats && currentModel.outputFormats.length > 1">
          <div class="st-pop-sep"></div>
          <div class="st-ctrl-block">
            <div style="display: flex; justify-content: space-between; align-items: baseline">
              <span class="st-ctrl-lbl">输出格式</span>
              <span class="st-ctrl-sub" style="margin: 0">{{ FORMAT_META[outputFormat].desc }}</span>
            </div>
            <div class="st-opt-grid cols-3">
              <button
                v-for="fmt in currentModel.outputFormats"
                :key="fmt"
                type="button"
                class="st-opt-pill"
                :aria-pressed="outputFormat === fmt"
                @click="selectOutputFormat(fmt)"
              >
                {{ FORMAT_META[fmt].label }}
              </button>
            </div>
          </div>

          <div
            v-if="currentModel.supportsCompression && (outputFormat === 'jpeg' || outputFormat === 'webp')"
            class="st-sub-panel"
          >
            <div style="display: flex; justify-content: space-between">
              <span class="st-ctrl-lbl">压缩质量 (output_compression)</span>
              <span class="st-ctrl-val">{{ outputCompression }}%</span>
            </div>
            <input
              v-model.number="outputCompression"
              class="st-range-input"
              type="range"
              min="10"
              max="100"
              step="5"
            />
          </div>
        </template>

        <!-- 背景 -->
        <div v-if="currentModel.supportsBackground" class="st-ctrl-row">
          <div>
            <div class="st-ctrl-lbl">背景</div>
            <div class="st-ctrl-sub">
              {{ canTransparent ? '透明背景保留 Alpha 通道' : '透明需先选 PNG 或 WebP' }}
            </div>
          </div>
          <div class="st-seg">
            <button type="button" :aria-pressed="background === 'auto'" @click="background = 'auto'">自动</button>
            <button type="button" :aria-pressed="background === 'opaque'" @click="background = 'opaque'">不透明</button>
            <button
              type="button"
              :disabled="!canTransparent"
              :aria-pressed="background === 'transparent'"
              @click="background = 'transparent'"
            >
              透明
            </button>
          </div>
        </div>

        <!-- 参考图保真度 -->
        <div v-if="currentModel.supportsInputFidelity" class="st-ctrl-row">
          <div>
            <div class="st-ctrl-lbl">参考图保真</div>
            <div class="st-ctrl-sub">改图时保留主体纹理/人脸细节</div>
          </div>
          <div class="st-seg">
            <button type="button" :aria-pressed="inputFidelity === 'low'" @click="inputFidelity = 'low'">标准</button>
            <button type="button" :aria-pressed="inputFidelity === 'high'" @click="inputFidelity = 'high'">高保真</button>
          </div>
        </div>

        <!-- 安全过滤 -->
        <div v-if="currentModel.supportsModeration" class="st-ctrl-row">
          <div>
            <div class="st-ctrl-lbl">安全过滤</div>
            <div class="st-ctrl-sub">moderation 严格度</div>
          </div>
          <div class="st-seg">
            <button type="button" :aria-pressed="moderation === 'auto'" @click="moderation = 'auto'">标准</button>
            <button type="button" :aria-pressed="moderation === 'low'" @click="moderation = 'low'">宽松</button>
          </div>
        </div>

        <!-- GPT 风格透传 -->
        <div v-if="currentModel.supportsGptStylePassthrough" class="st-ctrl-block">
          <div style="display: flex; justify-content: space-between; align-items: baseline">
            <span class="st-ctrl-lbl">风格透传<code class="st-ctrl-code">style</code></span>
            <span class="st-ctrl-sub" style="margin: 0">留空不发送</span>
          </div>
          <div class="st-ctrl-sub" style="margin-top: -2px">
            主要通过描述词与构图规划控制风格；此处兼容透传 style 字段
          </div>
          <div class="st-seg full">
            <button type="button" :aria-pressed="gptStyle === ''" @click="gptStyle = ''">不传</button>
            <button type="button" :aria-pressed="gptStyle === 'natural'" @click="gptStyle = 'natural'">自然 natural</button>
            <button type="button" :aria-pressed="gptStyle === 'vivid'" @click="gptStyle = 'vivid'">鲜明 vivid</button>
          </div>
          <input
            v-model="gptStyle"
            type="text"
            class="st-mini-input"
            placeholder="也可输入自定义 style 值（默认留空）"
          />
        </div>
      </div>

      <div class="st-pop-footer">
        <div style="display: flex; align-items: center; gap: 8px">
          <span class="st-ctrl-lbl">本次合计</span>
          <button
            v-if="customOptionTags.length > 0"
            type="button"
            class="st-reset-link"
            @click="resetModelAdvancedOptions"
          >
            恢复默认
          </button>
        </div>
        <span class="st-row-total-val">
          {{ costEstimate.known ? `约 ${formatStudioUsd(costEstimate.totalPrice)}` : '按实际结算' }}
        </span>
      </div>
    </div>

    <!-- 放大灯箱 -->
    <div
      v-if="zoomState && zoomedImage && zoomedVersion"
      class="st-zoom-layer"
      role="dialog"
      aria-label="放大预览"
      @click.self="zoomState = null"
    >
      <div class="st-zoom-bar">
        <span class="st-zoom-meta">
          第 {{ zoomState.v + 1 }} 版
          <template v-if="zoomedVersion.images.length > 1">
            · 第 {{ zoomState.k + 1 }}/{{ zoomedVersion.images.length }} 张
          </template>
          · {{ getStudioModel(zoomedVersion.model).name }} · {{ zoomedVersion.tier }}
          <template v-if="zoomedImage.width && zoomedImage.height">
            ({{ zoomedImage.width }}×{{ zoomedImage.height }})
          </template>
        </span>
        <div class="st-zoom-controls">
          <button
            type="button"
            class="st-zoom-btn"
            :class="{ 'is-fav': zoomedVersion.favs[zoomState.k] }"
            :aria-pressed="zoomedVersion.favs[zoomState.k]"
            @click="toggleFav(zoomState.v, zoomState.k)"
          >
            {{ zoomedVersion.favs[zoomState.k] ? '★ 已收藏' : '☆ 收藏' }}
          </button>
          <button type="button" class="st-zoom-btn" @click="downloadVersionImage(zoomState.v, zoomState.k)">
            下载原图
          </button>
          <button type="button" class="st-zoom-btn" aria-label="关闭预览" @click="zoomState = null">
            关闭
          </button>
        </div>
      </div>

      <button
        v-if="zoomedVersion.images.length > 1"
        type="button"
        class="st-zoom-btn st-zoom-nav prev"
        aria-label="上一张"
        @click.stop="stepZoom(-1)"
      >
        ‹
      </button>

      <img
        class="st-zoom-img"
        :src="zoomedImage.objectUrl"
        :alt="`第 ${zoomState.v + 1} 版预览`"
        @click="zoomState = null"
      />

      <button
        v-if="zoomedVersion.images.length > 1"
        type="button"
        class="st-zoom-btn st-zoom-nav next"
        aria-label="下一张"
        @click.stop="stepZoom(1)"
      >
        ›
      </button>
    </div>

    <!-- 复制为 API 代码弹窗 -->
    <div
      v-if="codeModalVerIdx >= 0 && versions[codeModalVerIdx]"
      class="st-modal-backdrop"
      role="dialog"
      aria-labelledby="stCodeModalTitle"
      @click.self="codeModalVerIdx = -1"
    >
      <div class="st-modal-card">
        <div class="st-modal-head">
          <h2 id="stCodeModalTitle" class="st-modal-title">
            复制为 API 代码（第 {{ codeModalVerIdx + 1 }} 版）
          </h2>
          <button type="button" class="st-modal-close" aria-label="关闭" @click="codeModalVerIdx = -1">×</button>
        </div>
        <div class="st-modal-body">
          <div class="st-code-toolbar">
            <div class="st-seg">
              <button type="button" :aria-pressed="codeLang === 'curl'" @click="codeLang = 'curl'">cURL</button>
              <button type="button" :aria-pressed="codeLang === 'python'" @click="codeLang = 'python'">Python</button>
            </div>
            <button
              type="button"
              class="st-btn-sm"
              :class="{ copied: codeCopied }"
              @click="copyCodeSnippet"
            >
              {{ codeCopied ? '已复制 ✓' : '复制代码' }}
            </button>
          </div>
          <pre class="st-code-pre">{{ activeCodeSnippet }}</pre>
          <p class="st-code-note">
            复用本次模型与全部参数，不保证复现完全相同的图。请将 <code>YOUR_GABUGU_API_KEY</code> 替换为您的真实 API 密钥。
          </p>
        </div>
      </div>
    </div>

    <!-- 两版对比弹窗 -->
    <div
      v-if="compareOpen && compareLeftVersion && compareRightVersion"
      class="st-modal-backdrop"
      role="dialog"
      aria-labelledby="stCompareModalTitle"
      @click.self="compareOpen = false"
    >
      <div class="st-modal-card wide">
        <div class="st-modal-head">
          <h2 id="stCompareModalTitle" class="st-modal-title">版本对比</h2>
          <button type="button" class="st-modal-close" aria-label="关闭" @click="compareOpen = false">×</button>
        </div>
        <div class="st-modal-body">
          <div class="st-compare-controls">
            <div class="st-compare-pickers">
              <select v-model.number="compareL" class="st-compare-select" aria-label="左侧版本">
                <option v-for="(ver, idx) in versions" :key="ver.id" :value="idx">
                  第 {{ idx + 1 }} 版（{{ getStudioModel(ver.model).shortName }} · {{ ver.ratioCode }} · {{ ver.tier }}）
                </option>
              </select>
              <span class="st-compare-vs">对比</span>
              <select v-model.number="compareR" class="st-compare-select" aria-label="右侧版本">
                <option v-for="(ver, idx) in versions" :key="ver.id" :value="idx">
                  第 {{ idx + 1 }} 版（{{ getStudioModel(ver.model).shortName }} · {{ ver.ratioCode }} · {{ ver.tier }}）
                </option>
              </select>
            </div>

            <div class="st-seg">
              <button type="button" :aria-pressed="compareMode === 'side'" @click="compareMode = 'side'">并排查看</button>
              <button type="button" :aria-pressed="compareMode === 'curtain'" @click="compareMode = 'curtain'">滑动卷帘</button>
            </div>
          </div>

          <div v-if="compareMode === 'side'" class="st-compare-side">
            <div class="st-compare-pane">
              <div class="st-compare-media">
                <img :src="compareLeftImage?.objectUrl" :alt="`第 ${compareL + 1} 版`" />
              </div>
              <div class="st-compare-pane-meta">{{ compareMetaText(compareL, compareLeftVersion) }}</div>
            </div>
            <div class="st-compare-pane">
              <div class="st-compare-media">
                <img :src="compareRightImage?.objectUrl" :alt="`第 ${compareR + 1} 版`" />
              </div>
              <div class="st-compare-pane-meta">{{ compareMetaText(compareR, compareRightVersion) }}</div>
            </div>
          </div>

          <div
            v-else
            ref="curtainRef"
            class="st-compare-curtain"
            @pointerdown="onCurtainPointerDown"
            @pointermove="onCurtainPointerMove"
            @pointerup="draggingCurtain = false"
            @pointercancel="draggingCurtain = false"
          >
            <img :src="compareRightImage?.objectUrl" :alt="`第 ${compareR + 1} 版`" draggable="false" />
            <div
              class="st-curtain-top"
              :style="{ clipPath: `inset(0 ${100 - curtainPos}% 0 0)` }"
            >
              <img :src="compareLeftImage?.objectUrl" :alt="`第 ${compareL + 1} 版`" draggable="false" />
            </div>
            <div class="st-curtain-line" :style="{ left: `${curtainPos}%` }"></div>
            <div
              class="st-curtain-handle"
              role="slider"
              tabindex="0"
              aria-label="拖动对比滑块"
              aria-valuemin="5"
              aria-valuemax="95"
              :aria-valuenow="Math.round(curtainPos)"
              :style="{ left: `${curtainPos}%` }"
              @keydown="onCurtainKeydown"
            >
              ↔
            </div>
            <span class="st-curtain-tag left">第 {{ compareL + 1 }} 版 · {{ getStudioModel(compareLeftVersion.model).shortName }}</span>
            <span class="st-curtain-tag right">第 {{ compareR + 1 }} 版 · {{ getStudioModel(compareRightVersion.model).shortName }}</span>
          </div>

          <div class="st-compare-diffs">
            <span v-for="(d, idx) in compareDiffTags" :key="idx" class="st-diff-tag">{{ d }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 轻提示 Toast -->
    <div v-if="toastState" class="st-toast" :class="toastState.kind" role="status">
      <span class="st-toast-dot"></span>
      <span>{{ toastState.msg }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/studio.css'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { ComponentPublicInstance } from 'vue'
import { userAPI } from '@/api/user'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey, Group } from '@/types'
import StudioVendorIcon from '@/components/studio/StudioVendorIcon.vue'
import {
  BASE_RATIOS,
  COMPLEXITY_META,
  DEFAULT_STUDIO_MODEL_ID,
  FORMAT_META,
  QUALITY_META,
  STUDIO_MODEL_LIST,
  TIER_META,
  VENDOR_META,
  VENDOR_ORDER,
  buildApiSnippet,
  classifyImageBillingTier,
  effectiveDir,
  getRatioEntry,
  getStudioModel,
  numericAspect,
  ratioLabelForDir,
  resolveDimensions,
  resolveRatioCode,
  type ModelRatioEntry,
  type StudioBackground,
  type StudioComplexity,
  type StudioInputFidelity,
  type StudioModelSpec,
  type StudioModeration,
  type StudioOrientation,
  type StudioOutputFormat,
  type StudioTierId,
  type StudioVendorId,
} from '@/studio/models'
import {
  estimateModelMinPrice,
  estimateStudioCost,
  estimateStudioCostRange,
  formatStudioUsd,
  formatWalletUsd,
} from '@/studio/pricing'
import {
  StudioApiError,
  generateStudioImages,
  loadStudioContext,
  prepareUploadReference,
  resolveImageToBlob,
  resolveOrCreateStudioApiKey,
  type StudioGeneratedImage,
  type StudioRefPayload,
} from '@/studio/api'
import {
  SITE_THEME_STORAGE_KEY,
  applyStudioTheme,
  readInheritedStudioTheme,
  type StudioTheme,
} from '@/studio/theme'
import {
  attachStudioJobListener,
  clearStudioHistory,
  getActiveStudioJob,
  loadStudioHistory,
  revokeVersionObjectUrls,
  runManagedStudioJob,
  saveStudioDraft,
  saveStudioVersion,
  type StudioJobListener,
  type StudioPendingJob,
  type StudioVersion,
} from '@/studio/historyStore'

interface StudioRefItem {
  id: number
  type: 'upload' | 'version'
  tag: string
  tip: string
  thumbUrl: string
  filename: string
  mimeType: string
  blob: Blob | null
  versionIdx?: number
  imgIdx?: number
}

interface ConfirmAction {
  label: string
  primary?: boolean
  run: () => void
}

interface ConfirmState {
  title?: string
  body: string
  actions: ConfirmAction[]
}

// ---------- 主题（进入时继承全站当前浅色/深色主题，手动切换时同步回全站） ----------
const studioTheme = ref<StudioTheme>(readInheritedStudioTheme())
const themeSpinning = ref(false)

function toggleStudioTheme() {
  const next: StudioTheme = studioTheme.value === 'dark' ? 'light' : 'dark'
  studioTheme.value = next
  applyStudioTheme(next)
  themeSpinning.value = true
  window.setTimeout(() => {
    themeSpinning.value = false
  }, 280)
}

function onStorageThemeSync(e: StorageEvent) {
  if (e.key !== SITE_THEME_STORAGE_KEY) return
  const next: StudioTheme =
    e.newValue === 'dark'
      ? 'dark'
      : e.newValue === 'light'
        ? 'light'
        : window.matchMedia?.('(prefers-color-scheme: dark)').matches
          ? 'dark'
          : 'light'
  studioTheme.value = next
  document.documentElement.classList.toggle('dark', next === 'dark')
}

// ---------- 线上账户、余额、分组与仅驻留内存的生图凭证 ----------
const authStore = useAuthStore()
const contextLoaded = ref(false)
const balanceLoaded = ref(false)
const balance = ref(0)
const walletFlash = ref(false)
const imageGroup = ref<Group | null>(null)
const userGroupRates = ref<Record<number, number>>({})
const historyPersisted = ref(true)
let activeUserId: number | null = authStore.user?.id ?? null
let hydratedUserId: number | null = null
let cachedStudioKey: ApiKey | null = null
let isUnmounted = false
let delayedBalanceTimer: number | null = null
let draftSaveTimer: number | null = null
let draftDirty = false

// ---------- 模型与参数状态 ----------
const modelId = ref<string>(DEFAULT_STUDIO_MODEL_ID)
const baseRatio = ref<string>('1:1')
const orientation = ref<StudioOrientation>('landscape')
const tier = ref<StudioTierId>('2K')
const count = ref<number>(1)
const quality = ref<string | null>('medium')
const complexity = ref<StudioComplexity>('high')
const outputFormat = ref<StudioOutputFormat>('png')
const background = ref<StudioBackground>('auto')
const outputCompression = ref<number>(85)
const inputFidelity = ref<StudioInputFidelity>('low')
const moderation = ref<StudioModeration>('auto')
const gptStyle = ref<string>('')

// ---------- 输入与会话状态 ----------
const promptText = ref('')
const isComposing = ref(false)
const refs = ref<StudioRefItem[]>([])
let nextRefId = 1
let nextVersionId = 1

const versions = ref<StudioVersion[]>([])
const currentVersionIdx = ref(-1)
const favOnly = ref(false)
const generating = ref(false)
const pendingJob = ref<StudioPendingJob | null>(null)
const sessionSpent = ref(0)
const inlineError = ref('')
const confirmState = ref<ConfirmState | null>(null)

// ---------- 交互反馈 / 布局状态 ----------
const isScrolled = ref(false)
const showJumpLatest = ref(false)
const isMobile = ref(typeof window !== 'undefined' ? window.innerWidth <= 760 : false)
const dockPulse = ref(false)
const composerNudge = ref(false)
const dragOver = ref(false)
let dragDepth = 0
const flashCardIdx = ref(-1)
const poppingStarKey = ref('')
const pricePop = ref(false)

// ---------- 弹层状态 ----------
const openPopover = ref<'model' | 'ratio' | 'size' | 'more' | null>(null)
const popoverStyle = ref<Record<string, string>>({})
const modelVendorFilter = ref<'all' | StudioVendorId>('all')
const showAllModels = ref(false)
const hoveredRatio = ref<string | null>(null)
let activePopAnchor: HTMLElement | null = null

// ---------- 灯箱 / 代码 / 对比弹窗 ----------
const zoomState = ref<{ v: number; k: number } | null>(null)
const codeModalVerIdx = ref(-1)
const codeLang = ref<'curl' | 'python'>('curl')
const codeCopied = ref(false)
const compareOpen = ref(false)
const compareMode = ref<'side' | 'curtain'>('side')
const compareL = ref(0)
const compareR = ref(1)
const curtainPos = ref(50)
const draggingCurtain = ref(false)

// ---------- Toast ----------
const toastState = ref<{ msg: string; kind: '' | 'ok' | 'warn' } | null>(null)
let toastTimer: number | null = null

function toast(msg: string, kind: '' | 'ok' | 'warn' = '') {
  toastState.value = { msg, kind }
  if (toastTimer) window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => {
    toastState.value = null
  }, 2800)
}

// ---------- 模板引用 ----------
const promptRef = ref<HTMLTextAreaElement | null>(null)
const refFileInputRef = ref<HTMLInputElement | null>(null)
const btnModelRef = ref<HTMLButtonElement | null>(null)
const btnRatioRef = ref<HTMLButtonElement | null>(null)
const btnSizeRef = ref<HTMLButtonElement | null>(null)
const btnMoreRef = ref<HTMLButtonElement | null>(null)
const popoverRef = ref<HTMLElement | null>(null)
const composerDockRef = ref<HTMLElement | null>(null)
const feedStreamRef = ref<HTMLElement | null>(null)
const genCardRef = ref<HTMLElement | null>(null)
const curtainRef = ref<HTMLElement | null>(null)
const cardRefs = new Map<number, HTMLElement>()
let feedObserver: IntersectionObserver | null = null
const visibleRatioMap = new Map<number, number>()

// ---------- 计算属性 ----------
const currentModel = computed(() => getStudioModel(modelId.value))
const hasResult = computed(() => versions.value.length > 0 || pendingJob.value !== null)
const effectiveDirection = computed(() => effectiveDir(currentModel.value, baseRatio.value, orientation.value))
const currentRatioCode = computed(() => resolveRatioCode(baseRatio.value, effectiveDirection.value))
const currentDims = computed(() =>
  resolveDimensions(currentModel.value, tier.value, baseRatio.value, effectiveDirection.value)
)
const effectiveCount = computed(() => Math.min(count.value, currentModel.value.maxN))

const costEstimate = computed(() =>
  estimateStudioCost(imageGroup.value, userGroupRates.value, currentDims.value.apiSize, effectiveCount.value)
)

const costRange = computed(() =>
  estimateStudioCostRange(
    imageGroup.value,
    userGroupRates.value,
    currentModel.value.tiers,
    effectiveCount.value
  )
)

const isInsufficientBalance = computed(() => {
  if (!balanceLoaded.value) return false
  if (currentDims.value.isAuto) {
    if (!costRange.value.known) return false
    return balance.value + 1e-6 < costRange.value.minTotalPrice
  }
  if (!costEstimate.value.known) return false
  return balance.value + 1e-6 < costEstimate.value.totalPrice
})

const ratioDirLabel = computed(() => {
  const def = BASE_RATIOS[baseRatio.value]
  if (!def || !def.directional) return ''
  if (effectiveDirection.value === 'landscape') return '横'
  if (effectiveDirection.value === 'portrait') return '竖'
  return ''
})

const ratioSwatchBox = computed(() => {
  const r = currentRatioCode.value === 'auto' ? 1 : numericAspect(baseRatio.value, effectiveDirection.value)
  const box = 13
  let w = box
  let h = box
  if (r > 1) h = Math.max(5, Math.round(box / Math.min(r, 2.6)))
  else if (r < 1) w = Math.max(5, Math.round(box * Math.max(r, 1 / 2.6)))
  return { w, h }
})

const availableCounts = computed(() => (currentModel.value.maxN >= 4 ? [1, 2, 4] : [1, 2]))
const canTransparent = computed(() => outputFormat.value === 'png' || outputFormat.value === 'webp')

const customOptionTags = computed(() => {
  const m = currentModel.value
  const tags: string[] = []
  if (effectiveCount.value > 1) tags.push(`${effectiveCount.value}张`)
  if (m.qualityLevels.length && quality.value && quality.value !== m.defaultQuality) {
    tags.push(`${QUALITY_META[quality.value]?.label || quality.value}画质`)
  }
  if (m.supportsComplexity && complexity.value !== m.defaultComplexity) {
    tags.push(`规划·${COMPLEXITY_META[complexity.value].label}`)
  }
  if (m.outputFormats && outputFormat.value !== m.defaultFormat) {
    tags.push(outputFormat.value.toUpperCase())
  }
  if (m.supportsBackground && background.value !== 'auto') {
    tags.push(background.value === 'transparent' ? '透明' : '不透明')
  }
  if (
    m.supportsCompression &&
    (outputFormat.value === 'jpeg' || outputFormat.value === 'webp') &&
    outputCompression.value !== 85
  ) {
    tags.push(`压缩${outputCompression.value}%`)
  }
  if (m.supportsInputFidelity && inputFidelity.value !== 'low') tags.push('高保真')
  if (m.supportsModeration && moderation.value !== 'auto') tags.push('宽松审核')
  if (m.supportsGptStylePassthrough && gptStyle.value.trim()) tags.push(`style:${gptStyle.value.trim()}`)
  return tags
})

const promptPlaceholder = computed(() => {
  if (refs.value.length > 1) return `描述如何融合或参考这 ${refs.value.length} 张图生成新画面…`
  if (refs.value.length === 1) return '描述想对这张参考图做什么改动…'
  if (versions.value.length > 0) return '继续描述新画面，或把鼠标移到任意图上点「以这张改」…'
  return '描述你想画的画面…'
})

const pendingStatusText = computed(() => {
  const job = pendingJob.value
  if (!job) return '正在生成，请稍候…'
  if (job.refCount > 1) return `正在融合 ${job.refCount} 张参考图生成中…`
  if (job.refCount === 1) return '正在基于参考图生成改图中…'
  return '正在调用生图模型处理中…'
})

const pendingSkeletonSize = computed(() => {
  const job = pendingJob.value
  const r = job ? job.w / Math.max(1, job.h) : 1
  const winW = typeof window !== 'undefined' ? window.innerWidth : 1200
  const winH = typeof window !== 'undefined' ? window.innerHeight : 800
  const maxW = Math.min(winW - 88, 460)
  const maxH = Math.min(winH * 0.4, 340)
  let sw = maxW
  let sh = maxW / r
  if (sh > maxH) {
    sh = maxH
    sw = maxH * r
  }
  return { w: Math.round(Math.max(170, sw)), h: Math.round(Math.max(136, sh)) }
})

const featuredModels = computed(() => STUDIO_MODEL_LIST.filter((m) => m.featured))
const otherModels = computed(() => STUDIO_MODEL_LIST.filter((m) => !m.featured))
const filteredVendorModels = computed(() =>
  modelVendorFilter.value === 'all'
    ? STUDIO_MODEL_LIST
    : STUDIO_MODEL_LIST.filter((m) => m.protocol === modelVendorFilter.value)
)

const totalFavs = computed(() =>
  versions.value.reduce((sum, v) => sum + v.favs.filter(Boolean).length, 0)
)
const visibleVersionCount = computed(() =>
  favOnly.value ? versions.value.filter((v) => versionHasFav(v)).length : versions.value.length
)

// 比例弹层实时预览
const modelSupportsPortrait = computed(() =>
  currentModel.value.ratios.some((r) => r.dirs.includes('portrait'))
)
const previewBaseRatio = computed(() => hoveredRatio.value || baseRatio.value)
const previewEffDir = computed(() => {
  const def = BASE_RATIOS[previewBaseRatio.value]
  if (!def || !def.directional) return 'none' as const
  const entry = getRatioEntry(currentModel.value, previewBaseRatio.value)
  if (entry && entry.dirs.includes(orientation.value)) return orientation.value
  return entry ? entry.dirs[0] : 'landscape'
})
const ratioPreviewCode = computed(() => {
  const code = resolveRatioCode(previewBaseRatio.value, previewEffDir.value)
  return code === 'auto' ? 'auto' : code
})
const ratioPreviewBox = computed(() => {
  const r = previewBaseRatio.value === 'auto' ? 1 : numericAspect(previewBaseRatio.value, previewEffDir.value)
  const maxW = 92
  const maxH = 60
  let bw = maxW
  let bh = maxW / r
  if (bh > maxH) {
    bh = maxH
    bw = maxH * r
  }
  return { w: Math.max(18, Math.round(bw)), h: Math.max(16, Math.round(bh)) }
})
const ratioPreviewCaption = computed(() => {
  const b = previewBaseRatio.value
  const def = BASE_RATIOS[b]
  if (b === 'auto') return '自动匹配画面比例'
  if (b === '1:1') return '1:1 正方（无横竖区分）'
  const dirWord = previewEffDir.value === 'landscape' ? '横版' : '竖版'
  return `${ratioPreviewCode.value} · ${dirWord}（${ratioLabelForDir(def, previewEffDir.value)}）`
})
const ratioPreviewSub = computed(() => {
  if (previewBaseRatio.value === 'auto') return '改图时推荐，自动保持原图构图'
  const dims = resolveDimensions(currentModel.value, tier.value, previewBaseRatio.value, previewEffDir.value)
  return `${TIER_META[tier.value].label} 输出 ${dims.pixels}`
})

const zoomedVersion = computed(() =>
  zoomState.value ? versions.value[zoomState.value.v] || null : null
)
const zoomedImage = computed(() =>
  zoomState.value && zoomedVersion.value
    ? zoomedVersion.value.images[zoomState.value.k] || null
    : null
)

const activeCodeSnippet = computed(() => {
  const v = versions.value[codeModalVerIdx.value]
  if (!v) return ''
  const origin = typeof window !== 'undefined' && window.location.origin.startsWith('http')
    ? window.location.origin
    : 'https://gabugu.com'
  return buildApiSnippet(v, codeLang.value, origin)
})

const compareLeftVersion = computed(() => versions.value[compareL.value] || null)
const compareRightVersion = computed(() => versions.value[compareR.value] || null)
const compareLeftImage = computed(() => {
  const v = compareLeftVersion.value
  return v ? v.images[v.selectedIdx] || v.images[0] || null : null
})
const compareRightImage = computed(() => {
  const v = compareRightVersion.value
  return v ? v.images[v.selectedIdx] || v.images[0] || null : null
})
const compareDiffTags = computed(() => {
  const vL = compareLeftVersion.value
  const vR = compareRightVersion.value
  if (!vL || !vR) return []
  const diffs: string[] = []
  if (vL.model !== vR.model) diffs.push(`模型：${getStudioModel(vL.model).shortName} → ${getStudioModel(vR.model).shortName}`)
  if (vL.ratioCode !== vR.ratioCode) diffs.push(`画幅：${vL.ratioCode} → ${vR.ratioCode}`)
  if (vL.tier !== vR.tier) diffs.push(`清晰度：${vL.tier} → ${vR.tier}`)
  if (vL.quality !== vR.quality) diffs.push(`画质：${vL.quality || '无'} → ${vR.quality || '无'}`)
  if (vL.complexity !== vR.complexity) diffs.push(`规划：${vL.complexity || '无'} → ${vR.complexity || '无'}`)
  if (vL.outputFormat !== vR.outputFormat) diffs.push(`格式：${vL.outputFormat || 'png'} → ${vR.outputFormat || 'png'}`)
  if (vL.background !== vR.background) diffs.push(`背景：${vL.background || 'auto'} → ${vR.background || 'auto'}`)
  if (vL.gptStyle !== vR.gptStyle) diffs.push(`风格：${vL.gptStyle || '默认'} → ${vR.gptStyle || '默认'}`)
  if (vL.costKnown && vR.costKnown && vL.cost !== vR.cost) {
    diffs.push(`费用：${formatStudioUsd(vL.cost)} → ${formatStudioUsd(vR.cost)}`)
  }
  if (!diffs.length) diffs.push('两组参数一致，仅随机结果不同')
  return diffs
})

watch(
  () => `${costEstimate.value.totalPrice}:${effectiveCount.value}`,
  (_next, prev) => {
    if (!prev) return
    pricePop.value = false
    void nextTick(() => {
      pricePop.value = true
      window.setTimeout(() => {
        pricePop.value = false
      }, 300)
    })
  }
)

// ---------- 辅助函数 ----------
function versionHasFav(v: StudioVersion): boolean {
  return v.favs.some(Boolean)
}

function modelMinPriceLabel(m: StudioModelSpec): string {
  const minVal = estimateModelMinPrice(imageGroup.value, userGroupRates.value, m.tiers)
  return minVal != null ? `${formatStudioUsd(minVal)} 起` : ''
}

function ratioTileCode(entry: ModelRatioEntry): string {
  const def = BASE_RATIOS[entry.base]
  const dir = !def?.directional
    ? 'none'
    : entry.dirs.includes(orientation.value)
      ? orientation.value
      : entry.dirs[0]
  const code = resolveRatioCode(entry.base, dir)
  return code === 'auto' ? '自动' : code
}

function ratioTileName(entry: ModelRatioEntry): string {
  const def = BASE_RATIOS[entry.base]
  if (!def) return ''
  const canPortrait = entry.dirs.includes('portrait')
  if (def.directional && orientation.value === 'portrait' && !canPortrait) return '仅横版'
  const dir = !def.directional
    ? 'none'
    : entry.dirs.includes(orientation.value)
      ? orientation.value
      : entry.dirs[0]
  return ratioLabelForDir(def, dir)
}

function tierRightMeta(tId: StudioTierId): string {
  const dims = resolveDimensions(currentModel.value, tId, baseRatio.value, effectiveDirection.value)
  const est = estimateStudioCost(
    imageGroup.value,
    userGroupRates.value,
    dims.isAuto ? tId : dims.apiSize,
    1
  )
  if (est.known) {
    return dims.isAuto
      ? `出图落入 ${tId} · ${formatStudioUsd(est.unitPrice)}`
      : `${dims.pixels} · ${formatStudioUsd(est.unitPrice)}`
  }
  return dims.pixels
}

function buildVersionMetaTags(v: StudioVersion): string[] {
  const tags: string[] = [
    `${v.ratioCode} · ${v.tier}${v.dimensions !== 'auto' ? ` (${v.dimensions})` : ''}`,
  ]
  if (v.quality) tags.push(`画质 ${QUALITY_META[v.quality]?.label || v.quality}`)
  if (v.complexity) tags.push(`规划 ${COMPLEXITY_META[v.complexity].label}`)
  if (v.outputFormat && v.outputFormat !== 'png') tags.push(v.outputFormat.toUpperCase())
  if (v.background === 'transparent') tags.push('透明背景')
  if (v.gptStyle) tags.push(`style=${v.gptStyle}`)
  if (v.images.length > 1) tags.push(`${v.images.length} 张`)
  return tags
}

function compareMetaText(idx: number, v: StudioVersion): string {
  const q = v.quality ? ` · ${QUALITY_META[v.quality]?.label || v.quality}` : ''
  return `第 ${idx + 1} 版 · ${getStudioModel(v.model).shortName} · ${v.ratioCode} · ${v.tier}${q}`
}

function triggerDockPulse() {
  dockPulse.value = false
  void nextTick(() => {
    dockPulse.value = true
    window.setTimeout(() => {
      dockPulse.value = false
    }, 660)
  })
}

function triggerComposerNudge() {
  composerNudge.value = false
  void nextTick(() => {
    composerNudge.value = true
    window.setTimeout(() => {
      composerNudge.value = false
    }, 380)
  })
}

function autoGrowPrompt() {
  const el = promptRef.value
  if (!el) return
  el.style.height = 'auto'
  const maxH = hasResult.value ? 150 : 240
  el.style.height = `${Math.min(el.scrollHeight, maxH)}px`
}

// ---------- 弹层定位与开关 ----------
function positionPopover() {
  const pop = popoverRef.value
  const anchor = activePopAnchor
  if (!pop || !anchor) return
  pop.style.maxHeight = ''
  const rect = anchor.getBoundingClientRect()
  const pw = pop.offsetWidth
  let ph = pop.offsetHeight
  let left = Math.min(rect.left, window.innerWidth - pw - 12)
  if (left < 12) left = 12
  const availBelow = window.innerHeight - rect.bottom - 20
  const availAbove = rect.top - 20
  let openBelow = ph <= availBelow
  let maxHeight: string | undefined
  if (!openBelow && ph > availAbove) {
    openBelow = availBelow >= availAbove
    const cap = Math.max(240, openBelow ? availBelow : availAbove)
    maxHeight = `${cap}px`
    ph = Math.min(ph, cap)
  }
  const top = openBelow ? rect.bottom + 8 : Math.max(12, rect.top - ph - 8)
  popoverStyle.value = {
    left: `${left}px`,
    top: `${top}px`,
    ...(maxHeight ? { maxHeight } : {}),
  }
}

function togglePopover(kind: 'model' | 'ratio' | 'size' | 'more', anchor: HTMLElement | null) {
  if (openPopover.value === kind) {
    openPopover.value = null
    activePopAnchor = null
    return
  }
  if (kind === 'model') modelVendorFilter.value = 'all'
  hoveredRatio.value = null
  activePopAnchor = anchor
  openPopover.value = kind
  void nextTick(() => {
    positionPopover()
  })
}

watch([modelVendorFilter, showAllModels, outputFormat], () => {
  if (openPopover.value) {
    void nextTick(() => positionPopover())
  }
})

// ---------- 模型与参数切换 ----------
function selectModelFromMenu(nextId: string) {
  openPopover.value = null
  activePopAnchor = null
  if (nextId === modelId.value) return

  const next = getStudioModel(nextId)
  const curDir = effectiveDirection.value
  const curEntry = getRatioEntry(next, baseRatio.value)
  const ratioOk = !!curEntry && (curDir === 'none' || curEntry.dirs.includes(curDir))
  const tierOk = next.tiers.includes(tier.value)
  const refEditOk = refs.value.length === 0 || next.supportsEdit
  const refCountOk = refs.value.length <= next.maxRefImages

  if (ratioOk && tierOk && refEditOk && refCountOk) {
    applyModel(nextId, baseRatio.value, orientation.value, tier.value, 'keep')
    return
  }

  const targetBase = curEntry
    ? baseRatio.value
    : getRatioEntry(next, '1:1')
      ? '1:1'
      : next.ratios[0].base
  const targetEntry = getRatioEntry(next, targetBase)!
  const targetDir: StudioOrientation = targetEntry.dirs.includes(orientation.value)
    ? orientation.value
    : targetEntry.dirs[0] === 'portrait'
      ? 'portrait'
      : 'landscape'
  const targetTier = tierOk ? tier.value : next.defaultTier
  const refAction = !refEditOk ? 'clear' : !refCountOk ? 'trim' : 'keep'

  const issues: string[] = []
  if (!ratioOk) {
    const targetCode = resolveRatioCode(
      targetBase,
      targetEntry.dirs[0] === 'none' ? 'none' : targetDir
    )
    issues.push(`不支持当前画幅 ${currentRatioCode.value}（将切换为 ${targetCode}）`)
  }
  if (!tierOk) {
    issues.push(
      `不支持清晰度 ${TIER_META[tier.value].label}（支持 ${next.tiers.join(' / ')}，将改用 ${TIER_META[targetTier].label}）`
    )
  }
  if (!refEditOk) {
    issues.push('不支持参考图改图，切换后将移除当前参考图')
  } else if (!refCountOk) {
    issues.push(
      `最多支持 ${next.maxRefImages} 张参考图（当前已选 ${refs.value.length} 张，切换后将保留前 ${next.maxRefImages} 张）`
    )
  }

  confirmState.value = {
    title: `${next.name} `,
    body: `${issues.join('；')}。`,
    actions: [
      { label: '保持当前模型', run: () => {} },
      {
        label: '按兼容设置切换',
        primary: true,
        run: () => applyModel(nextId, targetBase, targetDir, targetTier, refAction),
      },
    ],
  }
}

function applyModel(
  nextId: string,
  nextBaseRatio: string,
  nextOrientation: StudioOrientation,
  nextTier: StudioTierId,
  refAction: 'keep' | 'trim' | 'clear'
) {
  const next = getStudioModel(nextId)
  modelId.value = nextId
  baseRatio.value = nextBaseRatio
  orientation.value = nextOrientation
  tier.value = nextTier
  if (refAction === 'clear') refs.value = []
  else if (refAction === 'trim') refs.value = refs.value.slice(0, next.maxRefImages)

  if (count.value > next.maxN) count.value = next.maxN
  if (!next.qualityLevels.length) {
    quality.value = null
  } else if (!quality.value || !next.qualityLevels.includes(quality.value)) {
    quality.value = next.defaultQuality
  }
  if (next.outputFormats) {
    if (!next.outputFormats.includes(outputFormat.value)) {
      outputFormat.value = next.defaultFormat || 'png'
    }
  } else {
    outputFormat.value = 'png'
  }
  if (!next.supportsBackground) background.value = 'auto'
  if (next.supportsComplexity && !COMPLEXITY_META[complexity.value]) {
    complexity.value = next.defaultComplexity || 'high'
  }
  confirmState.value = null
}

function setOrientation(nextDir: StudioOrientation) {
  orientation.value = nextDir
  const entry = getRatioEntry(currentModel.value, baseRatio.value)
  if (entry && BASE_RATIOS[baseRatio.value]?.directional && !entry.dirs.includes(nextDir)) {
    toast(`${baseRatio.value} 仅支持横版，请在下方选支持竖版的比例`)
  }
}

function selectRatioTile(baseId: string) {
  const entry = getRatioEntry(currentModel.value, baseId)
  const def = BASE_RATIOS[baseId]
  baseRatio.value = baseId
  if (def?.directional && entry) {
    if (!entry.dirs.includes(orientation.value)) {
      orientation.value = entry.dirs[0] === 'portrait' ? 'portrait' : 'landscape'
      toast(`${baseId} 该模型仅支持横版，已切为横版`)
    }
  }
  confirmState.value = null
  openPopover.value = null
  activePopAnchor = null
}

function selectTier(tId: StudioTierId) {
  tier.value = tId
  confirmState.value = null
  openPopover.value = null
  activePopAnchor = null
}

function selectOutputFormat(fmt: StudioOutputFormat) {
  outputFormat.value = fmt
  if (fmt === 'jpeg' && background.value === 'transparent') {
    background.value = 'auto'
  }
}

function resetModelAdvancedOptions() {
  const m = currentModel.value
  count.value = 1
  quality.value = m.defaultQuality
  complexity.value = m.defaultComplexity || 'high'
  outputFormat.value = m.defaultFormat || 'png'
  background.value = 'auto'
  outputCompression.value = 85
  inputFidelity.value = 'low'
  moderation.value = 'auto'
  gptStyle.value = ''
}

function runConfirmAction(act: ConfirmAction) {
  confirmState.value = null
  act.run()
}

// ---------- 参考图上传 / 粘贴 / 拖放 / 历史图挂入 ----------
function triggerRefFilePicker() {
  if (generating.value) return
  if (refFileInputRef.value) {
    refFileInputRef.value.value = ''
    refFileInputRef.value.click()
  }
}

async function addLocalReferenceFiles(fileList: FileList | File[]) {
  const m = currentModel.value
  if (!m.supportsEdit) {
    toast(`${m.name} 暂不支持参考图改图，请先切换模型`, 'warn')
    return
  }
  const files = Array.from(fileList || []).filter((f) => f && f.type && f.type.startsWith('image/'))
  if (!files.length) {
    toast('请提供图片文件（PNG / JPEG / WebP）', 'warn')
    return
  }
  const remaining = m.maxRefImages - refs.value.length
  if (remaining <= 0) {
    toast(`${m.shortName} 最多支持 ${m.maxRefImages} 张参考图，可先移除后再加`, 'warn')
    return
  }
  const slice = files.slice(0, remaining)
  const dropped = files.length - slice.length
  let added = 0

  for (const file of slice) {
    if (refs.value.length >= currentModel.value.maxRefImages) break
    try {
      const prepared = await prepareUploadReference(file)
      refs.value.push({
        id: nextRefId++,
        type: 'upload',
        tag: (prepared.filename || '参考图').replace(/\.[^.]+$/, '').slice(0, 6),
        tip: `本地参考图：${prepared.filename}`,
        thumbUrl: prepared.thumbUrl,
        filename: prepared.filename,
        mimeType: prepared.mimeType,
        blob: prepared.blob,
      })
      added += 1
    } catch (err) {
      toast(err instanceof Error ? err.message : '无法读取部分图片文件', 'warn')
    }
  }

  if (added > 0) {
    triggerDockPulse()
    promptRef.value?.focus({ preventScroll: true })
    const suffix = dropped > 0 ? `（已达上限，略过 ${dropped} 张）` : ''
    toast(
      added === 1
        ? `已添加参考图：${slice[0].name || '剪贴板图片'}${suffix}`
        : `已添加 ${added} 张参考图${suffix}`,
      'ok'
    )
  }
}

function onRefFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    void addLocalReferenceFiles(input.files)
  }
}

function onPromptPaste(e: ClipboardEvent) {
  const items = e.clipboardData?.items
  if (!items) return
  const imgFiles: File[] = []
  for (const item of Array.from(items)) {
    if (item.type && item.type.startsWith('image/')) {
      const f = item.getAsFile()
      if (f) imgFiles.push(f)
    }
  }
  if (imgFiles.length) {
    e.preventDefault()
    void addLocalReferenceFiles(imgFiles)
  }
}

function onDragEnter(e: DragEvent) {
  if (e.dataTransfer && Array.from(e.dataTransfer.types).includes('Files')) {
    e.preventDefault()
    dragDepth += 1
    dragOver.value = true
  }
}
function onDragOver(e: DragEvent) {
  if (dragOver.value) e.preventDefault()
}
function onDragLeave() {
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragOver.value = false
}
function onDrop(e: DragEvent) {
  e.preventDefault()
  dragDepth = 0
  dragOver.value = false
  if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
    void addLocalReferenceFiles(e.dataTransfer.files)
  }
}

function addVersionRef(verIdx: number, imgIdx: number) {
  const v = versions.value[verIdx]
  if (!v) return
  const img = v.images[imgIdx] || v.images[0]
  if (!img) return
  const m = currentModel.value
  if (!m.supportsEdit) {
    toast(`${m.name} 暂不支持参考图改图，请先切换模型`, 'warn')
    return
  }
  const exists = refs.value.some(
    (r) => r.type === 'version' && r.versionIdx === verIdx && r.imgIdx === imgIdx
  )
  if (exists) {
    triggerDockPulse()
    promptRef.value?.focus({ preventScroll: true })
    toast('这张图已在下方参考槽中', 'ok')
    return
  }
  if (refs.value.length >= m.maxRefImages) {
    triggerDockPulse()
    toast(`${m.shortName} 最多支持 ${m.maxRefImages} 张参考图，可先点 × 移除再加`, 'warn')
    return
  }

  const many = v.images.length > 1
  refs.value.push({
    id: nextRefId++,
    type: 'version',
    tag: `v${verIdx + 1}${many ? `·${imgIdx + 1}` : ''}`,
    tip: `第 ${verIdx + 1} 版${many ? ` · 第 ${imgIdx + 1} 张` : ''}`,
    thumbUrl: img.thumbUrl || img.objectUrl,
    filename: `studio-v${verIdx + 1}-${imgIdx + 1}.${img.ext}`,
    mimeType: img.mimeType,
    blob: img.blob,
    versionIdx: verIdx,
    imgIdx,
  })
  currentVersionIdx.value = verIdx
  v.selectedIdx = imgIdx
  triggerDockPulse()
  promptRef.value?.focus({ preventScroll: true })
  const label = `第 ${verIdx + 1} 版${many ? `第 ${imgIdx + 1} 张` : ''}`
  toast(
    refs.value.length > 1
      ? `已加入${label}（共 ${refs.value.length}/${m.maxRefImages} 张参考）`
      : `已挂入${label}作为参考图，请在下方输入改动说明`,
    'ok'
  )
}

function removeRef(id: number) {
  refs.value = refs.value.filter((r) => r.id !== id)
}

function clearAllRefs() {
  refs.value = []
}

function formatBranchLabel(refItems: StudioRefItem[]): string {
  if (!refItems.length) return ''
  if (refItems.length === 1) {
    const r = refItems[0]
    if (r.type === 'version' && r.versionIdx != null) {
      const v = versions.value[r.versionIdx]
      const many = v && v.images.length > 1
      return `基于第 ${r.versionIdx + 1} 版${many ? `·第 ${(r.imgIdx ?? 0) + 1} 张` : ''} 改图`
    }
    return `参考图：${r.filename || '本地图片'}`
  }
  const verTags: string[] = []
  let uploadCount = 0
  for (const r of refItems) {
    if (r.type === 'version' && r.versionIdx != null) {
      const v = versions.value[r.versionIdx]
      const tag = `第 ${r.versionIdx + 1} 版${v && v.images.length > 1 ? `·${(r.imgIdx ?? 0) + 1}` : ''}`
      if (!verTags.includes(tag)) verTags.push(tag)
    } else {
      uploadCount += 1
    }
  }
  const detail =
    verTags.slice(0, 2).join('、') +
    (verTags.length > 2 ? '等' : '') +
    (uploadCount ? `${verTags.length ? '+' : ''}${uploadCount}张本地图` : '')
  return `多图融合 · ${refItems.length} 张参考${detail ? `（${detail}）` : ''}`
}

// ---------- 提交生成 ----------
function onPromptKeydown(e: KeyboardEvent) {
  if (isComposing.value || e.isComposing || e.keyCode === 229) return
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    void submit()
  }
}

function handleGenButtonClick() {
  if (isInsufficientBalance.value) {
    window.open('/purchase', '_blank', 'noopener,noreferrer')
    toast('已在新标签页打开充值页，充值后点右上角余额即可刷新，草稿不会丢失', 'ok')
    return
  }
  void submit()
}

async function refreshBalance(showToast = false) {
  try {
    const profile = await userAPI.getProfile()
    balance.value = Number(profile.balance ?? 0)
    balanceLoaded.value = true
    walletFlash.value = true
    window.setTimeout(() => {
      walletFlash.value = false
    }, 500)
    if (showToast) toast('账户余额已刷新', 'ok')
  } catch {
    if (showToast) toast('刷新余额失败，请稍后重试', 'warn')
  }
}

function recalcSessionSpent() {
  let total = 0
  for (const v of versions.value) {
    if (v.costKnown && v.cost > 0) {
      total = Math.round((total + v.cost) * 10000) / 10000
    }
  }
  sessionSpent.value = total
}

function createJobListener(
  ownerUserId: number,
  hadRefs = false,
  submittedText = ''
): StudioJobListener {
  return {
    onSuccess: (version) => {
      if (isUnmounted || activeUserId !== ownerUserId) {
        revokeVersionObjectUrls(version)
        return
      }
      const existingIdx = versions.value.findIndex((item) => item.id === version.id)
      let targetIdx: number
      if (existingIdx >= 0) {
        revokeVersionObjectUrls(version)
        targetIdx = existingIdx
      } else {
        versions.value.push(version)
        targetIdx = versions.value.length - 1
      }
      nextVersionId = Math.max(nextVersionId, version.id + 1)
      currentVersionIdx.value = targetIdx
      recalcSessionSpent()

      // 成功后才清空输入框与参考槽，确保任何失败都不丢用户草稿
      if (submittedText && promptText.value.trim() === submittedText) {
        draftDirty = false
        if (draftSaveTimer) {
          window.clearTimeout(draftSaveTimer)
          draftSaveTimer = null
        }
        promptText.value = ''
        void nextTick(() => autoGrowPrompt())
      }
      if (submittedText) {
        refs.value = []
      }

      pendingJob.value = null
      generating.value = false

      // 刷新真实余额；由于后端 RecordUsage 异步入账，若首次刷新尚未扣减则延时补刷一次
      const balanceBeforeRefresh = balance.value
      void refreshBalance(false).then(() => {
        if (!isUnmounted && Math.abs(balance.value - balanceBeforeRefresh) < 1e-6) {
          if (delayedBalanceTimer) window.clearTimeout(delayedBalanceTimer)
          delayedBalanceTimer = window.setTimeout(() => {
            delayedBalanceTimer = null
            if (!isUnmounted) void refreshBalance(false)
          }, 1800)
        }
      })

      void nextTick(() => {
        scrollCardIntoView(targetIdx, false)
        updateJumpLatest()
      })
      if (hadRefs) {
        toast(`已生成第 ${targetIdx + 1} 版；滚轮向上可随时对比或继续改图`, 'ok')
      }
    },
    onError: (err) => {
      if (isUnmounted || activeUserId !== ownerUserId) return
      pendingJob.value = null
      generating.value = false
      if (err instanceof StudioApiError) {
        if (err.isAuthKeyError) {
          cachedStudioKey = null
        }
        if (err.isBalanceError) {
          void refreshBalance(false)
        }
        inlineError.value = err.message
        toast(err.message, 'warn')
      } else {
        const msg = err instanceof Error ? err.message : '生成失败，请稍后重试（您的草稿与参考图已保留）。'
        inlineError.value = msg
        toast(msg, 'warn')
      }
    },
  }
}

async function hydrateHistoryForUser(userId: number) {
  if (!userId || hydratedUserId === userId) return
  activeUserId = userId

  const loaded = await loadStudioHistory(userId)
  if (isUnmounted || activeUserId !== userId) {
    for (const v of loaded.versions) {
      revokeVersionObjectUrls(v)
    }
    return
  }

  hydratedUserId = userId
  historyPersisted.value = loaded.storageAvailable

  const existingById = new Set(versions.value.map((v) => v.id))
  const merged: StudioVersion[] = []
  for (const v of loaded.versions) {
    if (existingById.has(v.id)) {
      revokeVersionObjectUrls(v)
    } else {
      merged.push(v)
    }
  }
  if (merged.length > 0) {
    versions.value = [...merged, ...versions.value].sort((a, b) => {
      if (a.createdAt !== b.createdAt) return a.createdAt - b.createdAt
      return a.id - b.id
    })
  }
  nextVersionId = Math.max(nextVersionId, loaded.nextVersionId)
  recalcSessionSpent()

  if (!promptText.value.trim() && loaded.draftPrompt.trim()) {
    promptText.value = loaded.draftPrompt
    void nextTick(() => autoGrowPrompt())
  }

  // 检查是否仍有该用户的在途后台生图任务
  const activePending = getActiveStudioJob(userId)
  if (activePending) {
    generating.value = true
    pendingJob.value = activePending
    attachStudioJobListener(userId, createJobListener(userId, activePending.refCount > 0, ''))
  }

  if (versions.value.length > 0) {
    const lastIdx = versions.value.length - 1
    currentVersionIdx.value = lastIdx
    void nextTick(() => {
      if (pendingJob.value) {
        genCardRef.value?.scrollIntoView({ behavior: 'auto', block: 'center' })
      } else {
        scrollCardIntoView(lastIdx, false, true)
      }
      updateJumpLatest()
    })
  }
}

watch(promptText, (val) => {
  if (!activeUserId || !hydratedUserId) return
  draftDirty = true
  if (draftSaveTimer) window.clearTimeout(draftSaveTimer)
  const targetUserId = activeUserId
  draftSaveTimer = window.setTimeout(() => {
    draftSaveTimer = null
    draftDirty = false
    void saveStudioDraft(targetUserId, val)
  }, 300)
})

async function submit() {
  if (generating.value) return
  if (confirmState.value) {
    toast('请先处理上方提示', 'warn')
    return
  }
  if (!imageGroup.value) {
    toast('当前账号暂未开通图像生成分组权限', 'warn')
    return
  }

  const text = promptText.value.trim()
  if (!text) {
    promptRef.value?.focus()
    triggerComposerNudge()
    toast('先写一句话描述你想画的画面')
    return
  }
  if (isInsufficientBalance.value) {
    window.open('/purchase', '_blank', 'noopener,noreferrer')
    toast('账户余额不足，已在新标签页打开充值页（当前草稿已保留）', 'warn')
    return
  }

  inlineError.value = ''
  openPopover.value = null
  activePopAnchor = null
  if (favOnly.value) favOnly.value = false

  const m = currentModel.value
  const dims = currentDims.value
  const n = effectiveCount.value
  const est = costEstimate.value
  const refsSnapshot = [...refs.value]
  const branchLabel = formatBranchLabel(refsSnapshot)
  const groupSnapshot = imageGroup.value
  const ratesSnapshot = { ...userGroupRates.value }
  const modelSnapshot = modelId.value
  const baseRatioSnapshot = baseRatio.value
  const orientationSnapshot = orientation.value
  const ratioCodeSnapshot = currentRatioCode.value
  const tierSnapshot = tier.value
  const qualitySnapshot = m.qualityLevels.length ? quality.value : null
  const complexitySnapshot = m.supportsComplexity ? complexity.value : null
  const outputFormatSnapshot = m.outputFormats ? outputFormat.value : null
  const backgroundSnapshot = m.supportsBackground ? background.value : null
  const outputCompressionSnapshot =
    m.supportsCompression && (outputFormat.value === 'jpeg' || outputFormat.value === 'webp')
      ? outputCompression.value
      : null
  const inputFidelitySnapshot =
    m.supportsInputFidelity && refsSnapshot.length > 0 ? inputFidelity.value : null
  const moderationSnapshot = m.supportsModeration ? moderation.value : null
  const gptStyleSnapshot = m.supportsGptStylePassthrough ? gptStyle.value.trim() : ''
  const versionIdSnapshot = nextVersionId++
  const ownerUserId = activeUserId ?? authStore.user?.id ?? null

  const refDescriptors = refsSnapshot.map((r) => {
    const srcImg =
      !r.blob && r.type === 'version' && r.versionIdx != null
        ? versions.value[r.versionIdx]?.images[r.imgIdx ?? 0] ?? null
        : null
    return {
      blob: r.blob,
      srcImg,
      filename: r.filename,
      mimeType: r.mimeType,
    }
  })

  const jobSnapshot: StudioPendingJob = {
    prompt: text,
    model: modelSnapshot,
    ratioCode: ratioCodeSnapshot,
    tier: tierSnapshot,
    w: dims.w,
    h: dims.h,
    count: n,
    refCount: refsSnapshot.length,
    branchLabel,
    cost: est.totalPrice,
    costKnown: !dims.isAuto && est.known,
  }

  generating.value = true
  pendingJob.value = jobSnapshot

  void nextTick(() => {
    genCardRef.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  })

  const executeJob = async (): Promise<StudioVersion> => {
    // 1. 解析参考图高保真 Blob
    const refPayloads: StudioRefPayload[] = []
    for (const r of refDescriptors) {
      let blob = r.blob
      if (!blob && r.srcImg) {
        blob = await resolveImageToBlob(r.srcImg)
      }
      if (!blob) {
        throw new StudioApiError('部分参考图未能读取，请重新选择后再试。')
      }
      refPayloads.push({
        blob,
        filename: r.filename,
        mimeType: blob.type || r.mimeType || 'image/png',
      })
    }

    // 2. 自动解析或按需创建「GABUGU 画室」分组密钥（全程仅驻留内存）；
    //    若线上密钥在会话中被删除/禁用/达额，则自动清理内存缓存并静默换新重试一次。
    const runGenerate = async (apiKeyStr: string) =>
      generateStudioImages({
        apiKey: apiKeyStr,
        model: modelSnapshot,
        prompt: text,
        apiSize: dims.apiSize,
        count: n,
        quality: qualitySnapshot,
        complexity: complexitySnapshot,
        outputFormat: outputFormatSnapshot,
        background: backgroundSnapshot,
        outputCompression: outputCompressionSnapshot,
        inputFidelity: inputFidelitySnapshot,
        moderation: moderationSnapshot,
        gptStyle: gptStyleSnapshot,
        refs: refPayloads,
      })

    let images: StudioGeneratedImage[]
    let activeKey = await resolveOrCreateStudioApiKey(groupSnapshot.id, cachedStudioKey)
    if (!isUnmounted) cachedStudioKey = activeKey
    try {
      images = await runGenerate(activeKey.key)
    } catch (firstErr) {
      if (firstErr instanceof StudioApiError && firstErr.isAuthKeyError) {
        cachedStudioKey = null
        activeKey = await resolveOrCreateStudioApiKey(groupSnapshot.id, null)
        if (!isUnmounted) cachedStudioKey = activeKey
        images = await runGenerate(activeKey.key)
      } else {
        throw firstErr
      }
    }

    const firstW = images[0]?.width || dims.w
    const firstH = images[0]?.height || dims.h
    const resolvedBillingSize =
      dims.isAuto && images[0]?.width && images[0]?.height
        ? `${images[0].width}x${images[0].height}`
        : dims.apiSize
    const resolvedTier = dims.isAuto ? classifyImageBillingTier(resolvedBillingSize) : tierSnapshot
    const actualEst = estimateStudioCost(
      groupSnapshot,
      ratesSnapshot,
      resolvedBillingSize,
      images.length
    )

    return {
      id: versionIdSnapshot,
      createdAt: Date.now(),
      prompt: text,
      model: modelSnapshot,
      baseRatio: baseRatioSnapshot,
      orientation: orientationSnapshot,
      ratioCode: ratioCodeSnapshot,
      tier: resolvedTier,
      apiSize: dims.apiSize,
      dimensions: firstW && firstH ? `${firstW}×${firstH}` : dims.pixels,
      w: firstW,
      h: firstH,
      count: images.length,
      quality: qualitySnapshot,
      complexity: complexitySnapshot,
      outputFormat: outputFormatSnapshot,
      background: backgroundSnapshot,
      outputCompression: outputCompressionSnapshot,
      inputFidelity: inputFidelitySnapshot,
      moderation: moderationSnapshot,
      gptStyle: gptStyleSnapshot,
      refCount: refPayloads.length,
      branchLabel,
      cost: actualEst.totalPrice,
      costKnown: actualEst.known,
      images,
      favs: Array(images.length).fill(false),
      selectedIdx: 0,
    }
  }

  if (ownerUserId && ownerUserId > 0) {
    await runManagedStudioJob(
      ownerUserId,
      jobSnapshot,
      createJobListener(ownerUserId, refsSnapshot.length > 0, text),
      executeJob
    )
    return
  }

  const fallbackListener = createJobListener(0, refsSnapshot.length > 0, text)
  try {
    const version = await executeJob()
    if (isUnmounted) {
      revokeVersionObjectUrls(version)
      return
    }
    activeUserId = 0
    fallbackListener.onSuccess(version)
  } catch (err) {
    activeUserId = 0
    fallbackListener.onError(err)
  }
}

// ---------- 历史卡片交互：选图 / 收藏 / 下载 / 灯箱 / 复制代码 / 对比 / 复用 ----------
function handleSlotClick(verIdx: number, imgIdx: number) {
  const v = versions.value[verIdx]
  if (!v) return
  currentVersionIdx.value = verIdx
  if (v.images.length === 1) {
    openZoom(verIdx, 0)
    return
  }
  v.selectedIdx = imgIdx
  if (activeUserId) void saveStudioVersion(activeUserId, v)
}

function toggleFav(verIdx: number, imgIdx: number) {
  const v = versions.value[verIdx]
  if (!v) return
  const next = !v.favs[imgIdx]
  v.favs[imgIdx] = next
  if (activeUserId) void saveStudioVersion(activeUserId, v)
  if (next) {
    poppingStarKey.value = `${verIdx}:${imgIdx}`
    window.setTimeout(() => {
      if (poppingStarKey.value === `${verIdx}:${imgIdx}`) poppingStarKey.value = ''
    }, 340)
  }
}

function toggleFavFilter() {
  favOnly.value = !favOnly.value
  void nextTick(() => updateJumpLatest())
}

async function downloadVersionImage(verIdx: number, imgIdx: number) {
  const v = versions.value[verIdx]
  const img = v?.images[imgIdx]
  if (!v || !img) return

  const filename = `gabugu-studio-v${verIdx + 1}-${imgIdx + 1}-${img.width || v.w}x${img.height || v.h}.${img.ext}`
  if (img.blob) {
    const url = URL.createObjectURL(img.blob)
    const a = document.createElement('a')
    a.download = filename
    a.href = url
    a.click()
    window.setTimeout(() => URL.revokeObjectURL(url), 5000)
    toast(`已下载第 ${verIdx + 1} 版原图（${img.width || v.w}×${img.height || v.h}）`, 'ok')
    return
  }

  try {
    const blob = await resolveImageToBlob(img)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.download = filename
    a.href = url
    a.click()
    window.setTimeout(() => URL.revokeObjectURL(url), 5000)
    toast(`已下载第 ${verIdx + 1} 版原图`, 'ok')
  } catch {
    if (img.remoteUrl) {
      window.open(img.remoteUrl, '_blank', 'noopener,noreferrer')
      toast('已在新标签页打开原图，可右键另存为', 'ok')
    } else {
      toast('下载失败，请稍后重试', 'warn')
    }
  }
}

function openZoom(verIdx: number, imgIdx: number) {
  zoomState.value = { v: verIdx, k: imgIdx }
}

function stepZoom(delta: number) {
  if (!zoomState.value || !zoomedVersion.value) return
  const total = zoomedVersion.value.images.length
  if (total <= 1) return
  zoomState.value = {
    v: zoomState.value.v,
    k: (zoomState.value.k + delta + total) % total,
  }
}

function openCodeModal(verIdx: number) {
  codeCopied.value = false
  codeModalVerIdx.value = verIdx
}

async function copyText(str: string, okMsg: string) {
  let copied = false
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(str)
      copied = true
    }
  } catch {
    // 降级到 execCommand
  }
  if (!copied) {
    const ta = document.createElement('textarea')
    ta.value = str
    ta.style.cssText = 'position:fixed;opacity:0;pointer-events:none;'
    document.body.appendChild(ta)
    ta.select()
    try {
      copied = document.execCommand('copy')
    } catch {
      copied = false
    }
    ta.remove()
  }
  toast(copied ? okMsg : '复制受限，请手动选中后按 Ctrl+C', copied ? 'ok' : 'warn')
}

async function copyCodeSnippet() {
  await copyText(activeCodeSnippet.value, '已复制 API 调用代码')
  codeCopied.value = true
  window.setTimeout(() => {
    codeCopied.value = false
  }, 1800)
}

function openCompareModal(verIdx: number) {
  if (versions.value.length < 2) return
  compareR.value = verIdx
  compareL.value = verIdx > 0 ? verIdx - 1 : versions.value.length - 1
  compareOpen.value = true
}

function updateCurtainFromClientX(clientX: number) {
  const el = curtainRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  if (!rect.width) return
  const pct = ((clientX - rect.left) / rect.width) * 100
  curtainPos.value = Math.max(5, Math.min(95, pct))
}

function onCurtainPointerDown(e: PointerEvent) {
  draggingCurtain.value = true
  curtainRef.value?.setPointerCapture(e.pointerId)
  updateCurtainFromClientX(e.clientX)
}

function onCurtainPointerMove(e: PointerEvent) {
  if (draggingCurtain.value) updateCurtainFromClientX(e.clientX)
}

function onCurtainKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowLeft') {
    e.preventDefault()
    curtainPos.value = Math.max(5, curtainPos.value - 5)
  } else if (e.key === 'ArrowRight') {
    e.preventDefault()
    curtainPos.value = Math.min(95, curtainPos.value + 5)
  }
}

function reuseVersion(verIdx: number) {
  const v = versions.value[verIdx]
  if (!v) return
  const refill = () => {
    const targetModel = getStudioModel(v.model)
    refs.value = []
    modelId.value = v.model
    baseRatio.value = v.baseRatio
    orientation.value = v.orientation
    tier.value = v.tier
    count.value = v.count
    quality.value = v.quality ?? targetModel.defaultQuality
    complexity.value = v.complexity ?? targetModel.defaultComplexity ?? 'high'
    outputFormat.value = v.outputFormat ?? targetModel.defaultFormat ?? 'png'
    background.value = v.background ?? 'auto'
    outputCompression.value = v.outputCompression ?? 85
    inputFidelity.value = v.inputFidelity ?? 'low'
    moderation.value = v.moderation ?? 'auto'
    gptStyle.value = v.gptStyle || ''
    promptText.value = v.prompt
    confirmState.value = null
    void nextTick(() => {
      autoGrowPrompt()
      triggerDockPulse()
      promptRef.value?.focus({ preventScroll: true })
    })
    toast(`已回填第 ${verIdx + 1} 版的全部参数与描述，未自动生成`, 'ok')
  }
  if (promptText.value.trim()) {
    confirmState.value = {
      body: '输入框里还有未提交的文字，回填会覆盖当前内容。',
      actions: [
        { label: '取消', run: () => {} },
        { label: '覆盖并回填', primary: true, run: refill },
      ],
    }
    triggerDockPulse()
    return
  }
  refill()
}

function revokeAllVersionUrls() {
  for (const v of versions.value) {
    for (const img of v.images) {
      if (img.objectUrl && img.objectUrl.startsWith('blob:')) {
        URL.revokeObjectURL(img.objectUrl)
      }
    }
  }
}

function resetCanvas() {
  revokeAllVersionUrls()
  versions.value = []
  nextVersionId = 1
  currentVersionIdx.value = -1
  refs.value = []
  favOnly.value = false
  sessionSpent.value = 0
  inlineError.value = ''
  confirmState.value = null
  zoomState.value = null
  codeModalVerIdx.value = -1
  compareOpen.value = false
  draftDirty = false
  if (draftSaveTimer) {
    window.clearTimeout(draftSaveTimer)
    draftSaveTimer = null
  }
  promptText.value = ''
  visibleRatioMap.clear()
  if (feedObserver) {
    feedObserver.disconnect()
    feedObserver = null
  }
  cardRefs.clear()
  if (activeUserId) {
    void clearStudioHistory(activeUserId)
  }
  void nextTick(() => {
    autoGrowPrompt()
    window.scrollTo({ top: 0 })
    promptRef.value?.focus()
  })
}

function handleNewCanvasClick() {
  if (generating.value) {
    toast('正在处理中，请稍候', 'warn')
    return
  }
  const hasDraft = promptText.value.trim().length > 0 || refs.value.length > 0
  if (versions.value.length > 0 || hasDraft) {
    const what = versions.value.length
      ? `本机保存的 ${versions.value.length} 个历史版本${hasDraft ? '和当前输入内容' : ''}`
      : '当前尚未生成的内容'
    confirmState.value = {
      body: `开始新画布会清空${what}（请先下载需要的作品），账户余额不受影响。`,
      actions: [
        { label: '取消', run: () => {} },
        { label: '清空并开始', primary: true, run: resetCanvas },
      ],
    }
    triggerDockPulse()
    return
  }
  resetCanvas()
}

// ---------- 滚动定位与 IntersectionObserver ----------
function setCardRef(idx: number, el: Element | ComponentPublicInstance | null) {
  const htmlEl = el instanceof HTMLElement ? el : null
  const prev = cardRefs.get(idx)
  if (prev && feedObserver) {
    feedObserver.unobserve(prev)
  }
  if (htmlEl) {
    cardRefs.set(idx, htmlEl)
    ensureFeedObserver()
    feedObserver?.observe(htmlEl)
  } else {
    cardRefs.delete(idx)
    visibleRatioMap.delete(idx)
  }
}

function ensureFeedObserver() {
  if (feedObserver || typeof IntersectionObserver === 'undefined') return
  feedObserver = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        const idx = Number((entry.target as HTMLElement).dataset.ver)
        if (entry.isIntersecting) visibleRatioMap.set(idx, entry.intersectionRatio)
        else visibleRatioMap.delete(idx)
      }
      if (visibleRatioMap.size === 0) return
      let bestIdx = currentVersionIdx.value
      let bestRatio = -1
      visibleRatioMap.forEach((ratio, idx) => {
        if (ratio > bestRatio) {
          bestRatio = ratio
          bestIdx = idx
        }
      })
      if (bestIdx >= 0 && bestIdx !== currentVersionIdx.value) {
        currentVersionIdx.value = bestIdx
      }
    },
    { rootMargin: '-14% 0px -34% 0px', threshold: [0.15, 0.35, 0.65] }
  )
}

function scrollCardIntoView(idx: number, flash: boolean, instant = false) {
  const card = cardRefs.get(idx)
  if (!card) return
  const topSafe = (isMobile.value ? 56 : 64) + 14
  const dockH = (composerDockRef.value?.offsetHeight || 165) + 14
  const availH = Math.max(200, window.innerHeight - topSafe - dockH)
  const rect = card.getBoundingClientRect()
  const cardTopDoc = window.scrollY + rect.top
  const idealTop =
    rect.height <= availH
      ? cardTopDoc - topSafe - Math.round((availH - rect.height) / 2)
      : cardTopDoc - topSafe
  window.scrollTo({ top: Math.max(0, idealTop), behavior: instant ? 'auto' : 'smooth' })
  currentVersionIdx.value = idx
  if (flash) {
    flashCardIdx.value = -1
    void nextTick(() => {
      flashCardIdx.value = idx
      window.setTimeout(() => {
        if (flashCardIdx.value === idx) flashCardIdx.value = -1
      }, 760)
    })
  }
}

function updateJumpLatest() {
  const distToBottom = document.documentElement.scrollHeight - (window.scrollY + window.innerHeight)
  showJumpLatest.value = versions.value.length >= 2 && distToBottom > 220
}

function onWindowScroll() {
  isScrolled.value = window.scrollY > 8
  updateJumpLatest()
}

function onWindowResize() {
  isMobile.value = window.innerWidth <= 760
  if (openPopover.value) {
    openPopover.value = null
    activePopAnchor = null
  }
}

function onDocumentMouseDown(e: MouseEvent) {
  if (!openPopover.value) return
  const target = e.target as Node | null
  if (popoverRef.value && target && popoverRef.value.contains(target)) return
  if (activePopAnchor && target && activePopAnchor.contains(target)) return
  openPopover.value = null
  activePopAnchor = null
}

function onDocumentKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (zoomState.value) {
      zoomState.value = null
      return
    }
    if (codeModalVerIdx.value >= 0) {
      codeModalVerIdx.value = -1
      return
    }
    if (compareOpen.value) {
      compareOpen.value = false
      return
    }
    if (openPopover.value) {
      openPopover.value = null
      activePopAnchor = null
      return
    }
    if (confirmState.value) {
      confirmState.value = null
    }
    return
  }
  if (zoomState.value && zoomedVersion.value && zoomedVersion.value.images.length > 1) {
    if (e.key === 'ArrowLeft') {
      e.preventDefault()
      stepZoom(-1)
    } else if (e.key === 'ArrowRight') {
      e.preventDefault()
      stepZoom(1)
    }
  }
}

// ---------- 生命周期 ----------
onMounted(async () => {
  window.addEventListener('scroll', onWindowScroll, { passive: true })
  window.addEventListener('resize', onWindowResize)
  window.addEventListener('storage', onStorageThemeSync)
  document.addEventListener('mousedown', onDocumentMouseDown)
  document.addEventListener('keydown', onDocumentKeyDown)

  promptRef.value?.focus()

  // 若登录态已有用户 ID，立即并行从本机 IndexedDB 恢复历史记录，不等网络请求
  const initialUserId = authStore.user?.id ?? null
  if (initialUserId && initialUserId > 0) {
    void hydrateHistoryForUser(initialUserId)
  }

  try {
    const ctx = await loadStudioContext()
    if (isUnmounted) return
    balance.value = Number(ctx.user.balance ?? 0)
    balanceLoaded.value = true
    imageGroup.value = ctx.imageGroup
    userGroupRates.value = ctx.userGroupRates
    cachedStudioKey = ctx.existingKey
    if (ctx.user.id && ctx.user.id > 0) {
      await hydrateHistoryForUser(ctx.user.id)
    }
  } catch {
    if (!isUnmounted) {
      inlineError.value = '加载账户与生图分组信息失败，请刷新页面重试。'
    }
  } finally {
    if (!isUnmounted) {
      contextLoaded.value = true
    }
  }
})

onBeforeUnmount(() => {
  isUnmounted = true
  window.removeEventListener('scroll', onWindowScroll)
  window.removeEventListener('resize', onWindowResize)
  window.removeEventListener('storage', onStorageThemeSync)
  document.removeEventListener('mousedown', onDocumentMouseDown)
  document.removeEventListener('keydown', onDocumentKeyDown)
  if (toastTimer) window.clearTimeout(toastTimer)
  if (delayedBalanceTimer) {
    window.clearTimeout(delayedBalanceTimer)
    delayedBalanceTimer = null
  }
  if (draftSaveTimer) {
    window.clearTimeout(draftSaveTimer)
    draftSaveTimer = null
  }
  if (draftDirty && activeUserId && activeUserId > 0) {
    draftDirty = false
    void saveStudioDraft(activeUserId, promptText.value)
  }
  if (feedObserver) {
    feedObserver.disconnect()
    feedObserver = null
  }
  // 离开画室时解绑页面监听，但不中断在途生图请求，确保后台完成后自动写入本机历史
  if (activeUserId && activeUserId > 0) {
    attachStudioJobListener(activeUserId, null)
  }
  cachedStudioKey = null
  revokeAllVersionUrls()
})
</script>
