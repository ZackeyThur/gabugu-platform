export default {
  arena: {
    pageTitle: 'Model Rankings',
    eyebrow: 'LMArena Leaderboard · Synced daily',
    title: 'Model Rankings',
    subtitle: 'Rankings from anonymous head-to-head votes by people worldwide, synced daily from the official LMArena data.',
    nav: {
      rankings: 'Rankings',
      modelPlaza: 'Model Plaza',
      login: 'Sign in',
      backToDashboard: 'Console'
    },
    categories: {
      text: { label: 'Text', desc: 'Conversation, reasoning and instruction following' },
      code: { label: 'Code', desc: 'Real coding and web app generation' },
      vision: { label: 'Vision', desc: 'Images, charts and visual Q&A' },
      image: { label: 'Image', desc: 'Text-to-image quality and prompt fidelity' }
    },
    status: {
      published: 'Published {date}',
      synced: 'Synced {time}',
      comparedTo: 'vs {date}',
      styleControl: 'Style control',
      modelCount: '{count} models',
      share: 'Share image',
      refresh: 'Refresh',
      votesUnit: '{votes} votes'
    },
    podium: {
      leadBy: '+{gap} over #2',
      behindLeader: '-{gap} behind #1',
      tiedLeader: 'Tied with #1',
      inspectHint: 'Click to inspect'
    },
    insights: {
      riser: 'Biggest riser',
      newEntries: 'New this release',
      topOpen: 'Top open model',
      newCount: '{count} models',
      rankAt: '#{rank}',
      none: 'None',
      noBaseline: 'Shown after the next release',
      locate: 'Jump to row'
    },
    filters: {
      search: 'Search model or organization',
      all: 'All',
      proprietary: 'Proprietary',
      open: 'Open',
      allOrgs: 'All labs',
      results: '{count} results',
      clear: 'Clear'
    },
    table: {
      rank: 'Rank',
      model: 'Model',
      distribution: 'Score & 95% CI',
      score: 'Score',
      delta: 'Change',
      votes: 'Votes',
      newBadge: 'NEW',
      openBadge: 'Open',
      showAll: 'Show all {count}',
      collapse: 'Collapse',
      empty: 'No models match',
      backToTop: 'Back to top'
    },
    inspector: {
      exactScore: 'Exact score',
      confidenceInterval: '95% confidence interval',
      gapToLeader: 'Gap to #1',
      leaderLabel: 'Track leader',
      votes: 'Blind-test votes',
      license: 'License',
      proprietary: 'Proprietary',
      movement: 'Release change',
      unchanged: 'Unchanged',
      crossTrack: 'Across other tracks',
      singleTrack: 'Ranked in this track only',
      copyModel: 'Copy name',
      copied: 'Copied',
      filterOrg: 'Only {org}',
      close: 'Close details'
    },
    states: {
      initializingTitle: 'First sync in progress',
      initializingDesc: 'The server is fetching the official LMArena data. This usually takes a few seconds.',
      loadFailedTitle: 'Rankings unavailable',
      loadFailedDesc: 'Check your connection and try again.',
      retry: 'Retry'
    },
    footer: {
      title: 'About the data',
      desc: 'Synced daily from the official LMArena (formerly Chatbot Arena) public dataset. Scores come from anonymous pairwise votes, fitted with a Bradley-Terry model. They reflect broad human preference; weigh them against your own use case.',
      arena: 'LMArena',
      dataset: 'Official dataset'
    },
    poster: {
      modalTitle: 'Share image',
      modalSubtitle: 'Tall, high-resolution image with a QR code, ready for group chats',
      modeLabel: 'Content',
      modeTop10: 'Top 10 of this track',
      modeOverview: 'Leaders of 4 tracks',
      themeLabel: 'Colors',
      themeDark: 'Dark',
      themeLight: 'Light',
      specHint: '1080 × 1760 HD PNG · Crisp in group chats',
      downloadPng: 'Download',
      downloaded: 'Download started',
      copyImage: 'Copy image',
      copyLink: 'Copy link',
      copiedImage: 'Image copied',
      copyImageUnsupported: 'This browser cannot copy images. Use Download instead.',
      copiedLink: 'Link copied',
      rendering: 'Rendering…',
      close: 'Close',
      publishDateLabel: 'Published',
      baselineDateLabel: 'vs',
      modelCount: '{count} models',
      lastChecked: 'Synced',
      footerScanHint: 'Scan for the full rankings',
      footerSource: 'Source: official LMArena dataset, synced daily',
      overviewTitle: 'Track leaders',
      overviewSubtitle: 'Text · Code · Vision · Image',
      top10Suffix: 'Top 10'
    }
  }
}
