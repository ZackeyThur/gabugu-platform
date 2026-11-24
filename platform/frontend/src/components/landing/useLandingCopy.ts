import {
  computed,
  type Ref
} from 'vue'

// Every statement here was checked against the product before it was written; the list is in
// components/landing/README.md. Change a claim only together with that check.
const english = {
  skip: 'Skip to content',
  navigation: 'Main navigation',
  screenNav: 'Screens of this page',
  screenNames: {
    hero: 'Top',
    models: 'Models',
    connect: 'Integration',
    tools: 'Tools',
    pricing: 'Pricing',
    control: 'Control',
    faq: 'FAQ',
    start: 'Get started'
  },
  docs: 'Documentation',
  arenaRankings: 'Arena Rankings',
  login: 'Sign in',
  start: 'Get started',
  console: 'Open console',
  readDocs: 'Read the docs',
  seeHow: 'See how to connect',
  explore: 'Explore more',
  heroLineOne: 'Ideas move.',
  heroLineTwo: 'Possibilities follow.',
  heroDescription: 'Native OpenAI, Anthropic and Gemini protocols. Create an API key and call models from your code or your tools.',
  tagline: 'Unified AI API gateway',

  modelTitle: 'Many models. One entry.',
  modelDescription: 'Pick a model for the job and keep the protocol it speaks.',
  protocolOf: '{name} protocol',
  viewExample: 'See how to connect',
  modelNote: 'Which models are available and what they cost is settled by the site, not by this page.',
  exploreModels: 'Open the model catalog',
  signUpForCatalog: 'Sign up to see models and prices',
  signInForCatalog: 'Sign in to see models and prices',

  connectTitle: 'Same SDK. Point it here.',
  connectDescription: 'Point the SDK at {site} and use your key there. The rest of your code stays.',
  protocol: 'API protocol',
  copy: 'Copy',
  copied: 'Copied',
  copyFailed: 'Copy is unavailable. Select the code to copy it.',
  codeNote: 'Replace YOUR_API_KEY with your key. Which models and prices you see is settled by the site.',
  createKey: 'Create an API key',

  toolsTitle: 'Works with your tools.',
  toolsDescription: 'No code needed: copy a configuration into the tool you use.',
  toolSteps: ['Create an API key and pick a group', 'Choose "Use Key" in your key list', 'Copy the configuration for that group'],
  toolImport: ', or import it into CC Switch',
  toolNamesNote: 'depending on the group',
  toolGuide: 'New to this? After you sign in, a beginner guide covers the Codex desktop app and Claude Code.',

  pricingTitle: 'How a call is priced',
  formulaCost: 'Cost',
  formulaTerms: [{
    name: 'Base price',
    note: 'Each model has its own price, for input and for output.'
  }, {
    name: 'Group multiplier',
    note: 'The group you pick for a key sets the multiplier; an account\'s own rate takes its place.'
  }, {
    name: 'Usage',
    note: 'What you actually call, as your usage records show.'
  }],
  payBalance: 'Balance: top up, pay as you go',
  paySubscription: 'Subscription: buy a plan, use its quota while it lasts',
  pricingNote: 'Models billed per request, such as images, and peak hours follow their own rules. The model catalog and your usage records are what counts.',
  viewPricing: 'View models and pricing',

  controlTitle: 'Set limits. Check usage.',
  controlDescription: 'Every key takes its own limits, and every call is recorded.',
  sampleNote: 'Sample data. Not a real call and not a quote.',
  sampleUsed: 'Used $12.00',
  sampleCap: 'Cap $50.00, then requests stop',
  sampleLimits: ['$5 a day at most', 'Expires in 30 days', 'Allowed IPs only'],
  sampleRecord: [{
    label: 'Model',
    value: 'Sample model'
  }, {
    label: 'Tokens',
    value: '1,200'
  }, {
    label: 'Cost',
    value: '$0.01'
  }, {
    label: 'Time',
    value: '1.2 s'
  }],
  usageLine: 'A key\'s quota and usage summary can be checked without signing in.',
  checkUsage: 'Check usage by key',

  faqTitle: 'Before you start',
  faq: {
    pricing: {
      question: 'How am I charged?',
      answer: 'By usage: base price × group multiplier × usage; an account\'s own rate takes the place of the group\'s. The site has the price of every model.'
    },
    plans: {
      question: 'Balance or subscription?',
      answer: 'A balance is topped up and spent as you go. A subscription is a plan whose quota you use while it lasts.'
    },
    keys: {
      question: 'Does one key reach every model?',
      answer: 'A key belongs to one group, and the group decides which models it reaches. For another group, create another key.'
    },
    nocode: {
      question: 'Can I use it without writing code?',
      answer: 'Yes. After you sign in, a beginner guide connects the Codex desktop app or Claude Code, on Windows and on macOS.'
    }
  },

  closingLineOne: 'Make room for',
  closingLineTwo: 'your next idea.',
  closingSteps: ['Sign up', 'Create an API key', 'Make your first call'],

  contact: 'Contact',
  disclaimer: '{site} is an independent service. It is not affiliated with or endorsed by OpenAI, Anthropic, Google, xAI or DeepSeek; their names belong to their owners.',
  earthCredit: 'Earth imagery: NASA Blue Marble and Black Marble',
  cloudCredit: 'Clouds: ',
  cloudLicence: ', ',
  cloudChanged: ', adapted'
}

const chinese: typeof english = {
  skip: '跳到主要内容',
  navigation: '主导航',
  screenNav: '页面分屏导航',
  screenNames: {
    hero: '首屏',
    models: '模型',
    connect: '接入',
    tools: '工具',
    pricing: '计费',
    control: '掌控',
    faq: '常见问题',
    start: '开始使用'
  },
  docs: '开发文档',
  arenaRankings: '模型榜单',
  login: '登录',
  start: '开始使用',
  console: '进入控制台',
  readDocs: '查看接入文档',
  seeHow: '看看怎么接入',
  explore: '探索更多',
  heroLineOne: '让想法',
  heroLineTwo: '走得更远',
  heroDescription: '支持 OpenAI、Anthropic、Gemini 三种原生协议。创建一个 API Key，在代码或工具里调用模型。',
  tagline: '统一 AI API 网关',

  modelTitle: '多种模型，一个入口',
  modelDescription: '按任务挑模型，沿用它本来的协议。',
  protocolOf: '{name} 协议',
  viewExample: '看接入方式',
  modelNote: '可用模型与价格以站内为准，本页不列。',
  exploreModels: '进入模型目录',
  signUpForCatalog: '注册后查看模型与价格',
  signInForCatalog: '登录后查看模型与价格',

  connectTitle: 'SDK 照旧，地址换一下',
  connectDescription: '把 SDK 的接入地址指向 {site}，密钥用这里的，其余代码不动。',
  protocol: '接口协议',
  copy: '复制',
  copied: '已复制',
  copyFailed: '暂时无法复制，请选中代码手动复制。',
  codeNote: '把 YOUR_API_KEY 换成你的密钥；能用哪些模型、什么价格，以站内为准。',
  createKey: '创建 API Key',

  toolsTitle: '常用工具，照着配就行',
  toolsDescription: '不写代码也能用：把配置复制进你用的工具。',
  toolSteps: ['创建 API Key，选一个分组', '在密钥列表点「使用密钥」', '复制这个分组的工具配置'],
  toolImport: '，或导入 CC Switch',
  toolNamesNote: '按分组提供',
  toolGuide: '第一次用？登录后有新手指南，讲 Codex 桌面端和 Claude Code。',

  pricingTitle: '费用怎么算',
  formulaCost: '费用',
  formulaTerms: [{
    name: '基础单价',
    note: '每个模型各有单价，输入、输出分开算。'
  }, {
    name: '分组倍率',
    note: '创建 Key 时选的分组决定倍率；账号另有专属倍率时以专属为准。'
  }, {
    name: '用量',
    note: '按实际调用计，用量记录里能核对。'
  }],
  payBalance: '余额：充值后按量扣',
  paySubscription: '订阅：买套餐，有效期内按套餐额度用',
  pricingNote: '图片等按次计费的模型、高峰时段另有规则，以模型目录和用量记录为准。',
  viewPricing: '查看模型与定价',

  controlTitle: '管得住，查得到',
  controlDescription: '每个 Key 可以单独设限，每次调用都有记录。',
  sampleNote: '示例数据，非真实调用，不代表报价',
  sampleUsed: '已用 $12.00',
  sampleCap: '上限 $50.00，用完即停',
  sampleLimits: ['每天最多 $5', '30 天后到期', '限定 IP'],
  sampleRecord: [{
    label: '模型',
    value: '示例模型'
  }, {
    label: 'Token',
    value: '1,200'
  }, {
    label: '费用',
    value: '$0.01'
  }, {
    label: '耗时',
    value: '1.2 秒'
  }],
  usageLine: '不登录也能用 Key 查额度和用量汇总。',
  checkUsage: '按 Key 查用量',

  faqTitle: '开始之前',
  faq: {
    pricing: {
      question: '怎么收费？',
      answer: '按用量算：基础单价 × 分组倍率 × 用量；账号有专属倍率时按专属算。具体价格以站内为准。'
    },
    plans: {
      question: '余额和订阅有什么不同？',
      answer: '余额是充值后按量扣。订阅是买一个套餐，在有效期内按套餐的额度用。'
    },
    keys: {
      question: '一个 Key 能用所有模型吗？',
      answer: '一个 Key 属于一个分组，能用哪些模型看分组。想用别的分组，再建一个 Key。'
    },
    nocode: {
      question: '不会写代码能用吗？',
      answer: '能。登录后有新手指南，带你把 Codex 桌面端或 Claude Code 连上，Windows 和 macOS 都有。'
    }
  },

  closingLineOne: '为下一个想法',
  closingLineTwo: '打开更多可能',
  closingSteps: ['注册', '创建 API Key', '发起第一次调用'],

  contact: '联系方式',
  disclaimer: '{site} 是独立的第三方服务，与 OpenAI、Anthropic、Google、xAI、DeepSeek 没有隶属或授权关系，各名称归其权利人所有。',
  earthCredit: '地球影像：NASA Blue Marble 与 Black Marble',
  cloudCredit: '云层：',
  cloudLicence: '，',
  cloudChanged: '，有改动'
}

export type LandingCopy = typeof english

export function useLandingCopy(isEnglish: Ref<boolean>) {
  return computed(() => isEnglish.value ? english : chinese)
}
