export default {
  arena: {
    pageTitle: '模型榜单',
    eyebrow: 'LMArena 排行榜 · 每日同步',
    title: '模型榜单',
    subtitle: '全球用户匿名盲测投票排出的大模型名次，每天同步 LMArena 官方数据。',
    nav: {
      rankings: '模型榜单',
      modelPlaza: '模型广场',
      login: '登录',
      backToDashboard: '控制台'
    },
    categories: {
      text: { label: '综合对话', desc: '日常对话、推理与指令遵循' },
      code: { label: '编程代码', desc: '真实编程与网页应用生成' },
      vision: { label: '视觉理解', desc: '看图、读图表与图文问答' },
      image: { label: '图像生成', desc: '文生图的画面质量与提示词还原' }
    },
    status: {
      published: '官方 {date} 发布',
      synced: '{time} 同步',
      comparedTo: '对比 {date}',
      styleControl: '风格控制口径',
      modelCount: '{count} 个模型',
      share: '生成分享图',
      refresh: '刷新',
      votesUnit: '{votes} 票'
    },
    podium: {
      leadBy: '领先第二 +{gap}',
      behindLeader: '距榜首 -{gap}',
      tiedLeader: '与榜首同分',
      inspectHint: '点击查看详情'
    },
    insights: {
      riser: '本期最大上升',
      newEntries: '本期新上榜',
      topOpen: '开源最高',
      newCount: '{count} 个模型',
      rankAt: '第 {rank} 名',
      none: '暂无',
      noBaseline: '下次官方更新后显示',
      locate: '定位到榜单'
    },
    filters: {
      search: '搜索模型或机构',
      all: '全部',
      proprietary: '闭源',
      open: '开源',
      allOrgs: '全部机构',
      results: '{count} 个结果',
      clear: '清除'
    },
    table: {
      rank: '排名',
      model: '模型',
      distribution: '分数区间 (95% CI)',
      score: '分数',
      delta: '较上期',
      votes: '票数',
      newBadge: 'NEW',
      openBadge: '开源',
      showAll: '查看全部 {count} 名',
      collapse: '收起',
      empty: '没有符合条件的模型',
      backToTop: '回到顶部'
    },
    inspector: {
      exactScore: '精确分数',
      confidenceInterval: '95% 置信区间',
      gapToLeader: '与榜首分差',
      leaderLabel: '当前赛道榜首',
      votes: '盲测票数',
      license: '开源协议',
      proprietary: '闭源商业',
      movement: '较上期变化',
      unchanged: '持平',
      crossTrack: '其他赛道表现',
      singleTrack: '仅在当前赛道入榜',
      copyModel: '复制模型名',
      copied: '已复制',
      filterOrg: '只看 {org}',
      close: '收起详情'
    },
    states: {
      initializingTitle: '首次榜单同步进行中',
      initializingDesc: '服务器正在拉取 LMArena 官方数据，通常十几秒就好，请稍后刷新。',
      loadFailedTitle: '榜单暂时打不开',
      loadFailedDesc: '检查一下网络，再试一次。',
      retry: '重新加载'
    },
    footer: {
      title: '数据说明',
      desc: '数据每天同步自 LMArena（原 Chatbot Arena）官方公开数据集。分数由全球用户的匿名两两对战投票，按 Bradley-Terry 模型统计得出，反映大众偏好，选型时请结合自己的场景判断。',
      arena: 'LMArena 官网',
      dataset: '官方数据集'
    },
    poster: {
      modalTitle: '生成分享图',
      modalSubtitle: '竖版高清图，自带二维码，可直接发到微信、QQ 群',
      modeLabel: '内容',
      modeTop10: '当前赛道前十',
      modeOverview: '四赛道榜首',
      themeLabel: '配色',
      themeDark: '深色',
      themeLight: '浅色',
      specHint: '1080 × 1760 高清 PNG · 适配微信与 QQ 群聊预览',
      downloadPng: '下载图片',
      downloaded: '已开始下载图片',
      copyImage: '复制图片',
      copyLink: '复制链接',
      copiedImage: '图片已复制，可直接粘贴发群',
      copyImageUnsupported: '当前浏览器不支持复制图片，请点“下载图片”',
      copiedLink: '链接已复制',
      rendering: '正在生成…',
      close: '关闭',
      publishDateLabel: '官方发布',
      baselineDateLabel: '对比',
      modelCount: '共 {count} 个模型',
      lastChecked: '同步时间',
      footerScanHint: '扫码查看完整榜单',
      footerSource: '数据来源：LMArena 官方公开数据集，每日同步',
      overviewTitle: '四赛道榜首',
      overviewSubtitle: '综合对话 · 编程代码 · 视觉理解 · 图像生成',
      top10Suffix: '前十'
    }
  }
}
