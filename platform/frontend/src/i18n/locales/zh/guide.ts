const guide = {
  "nav": "新手引导",
  "title": "新手引导",
  "description": "认识原理，连接桌面工具，完成第一次对话。",
  "eyebrow": "新手引导",
  "headline": "从零开始使用 Gabugu",
  "intro": "先看懂 AI 怎样工作，再把 Codex 桌面端连接到 Gabugu。安装、配置和第一次对话，跟着这份指南完成。",
  "startLearning": "开始配置",
  "jumpToSetup": "先了解原理",
  "readGuide": "阅读指南",
  "toc": "本页目录",
  "backToTop": "回到顶部",
  "chapters": {
    "basics": "认识原理",
    "setup": "开始使用",
    "resources": "工具资源",
    "faq": "常见问题"
  },
  "journey": {
    "understand": "了解原理",
    "prepare": "准备工具",
    "configure": "配置密钥",
    "verify": "开始对话"
  },
  "basics": {
    "eyebrow": "核心原理",
    "title": "一次对话，经过三个角色",
    "intro": "你在工具里提问，Gabugu 负责连接，AI 模型生成回答。",
    "request": "问题",
    "reply": "回答",
    "yourTool": "你用的工具",
    "clientRole": "在 Codex 等应用中提问、阅读回答，或处理文件。",
    "gateway": "Gabugu",
    "gatewayRole": "验证密钥、转发请求，并记录本次用量。",
    "models": "AI 模型服务",
    "modelRole": "理解问题、生成回答，再经 Gabugu 返回你的工具。",
    "flowCaption": "Gabugu 负责连接，模型负责生成回答。可用模型与价格以当前分组为准。",
    "configTitle": "配置里的三个字段，分别是什么？",
    "configIntro": "CC Switch 可以帮助管理配置。需要手动填写时，以「使用密钥」生成的说明为准。",
    "keyLabel": "API 密钥",
    "addressLabel": "地址（Base URL）",
    "modelLabel": "模型名",
    "ccSwitchRole": "CC Switch 能存好几套配置，你选哪套，它就把哪套写进工具。",
    "clientConfig": "写进工具",
    "configCaption": "API 密钥用于访问服务，接口地址决定连接位置，模型名指定要使用的模型。",
    "termsTitle": "不熟悉这些词？展开看看",
    "gatewayKind": "API 网关"
  },
  "configDiagram": {
    "managerTitle": "配置管理",
    "connection": "连接信息",
    "gabuguProfile": "Gabugu 配置",
    "source": "在「使用密钥」中获取",
    "manage": "集中管理配置",
    "active": "启用中",
    "destination": "写入所选工具",
    "keyHelp": "访问服务的凭证",
    "addressHelp": "请求发送到哪里",
    "modelHelp": "使用哪一个模型",
    "import": "导入",
    "apply": "启用",
    "profile": "密钥 · 地址 · 模型",
    "managerHelp": "管理配置，按需切换。",
    "desktop": "桌面应用",
    "codingTool": "编程工具",
    "clientHelp": "配置会写入对应工具的本地文件。",
    "caption": "配置关系示意。真实密钥、接口地址和模型名，请以「使用密钥」生成的说明为准。"
  },
  "terms": {
    "account": {
      "title": "账号",
      "body": "账号是你登录网站用的。它和 API 密钥不是一回事，别混着理解。"
    },
    "key": {
      "title": "API 密钥",
      "body": "给程序用的通行证。当密码一样收好，别发到群里，也别截图外传。"
    },
    "address": {
      "title": "地址（Base URL）",
      "body": "工具往哪儿发请求，就靠这个地址。用 Gabugu，就填 Gabugu 给你的那个。"
    },
    "model": {
      "title": "模型",
      "body": "负责理解输入和生成回答的 AI。模型名需要与密钥分组支持的名称一致。"
    },
    "token": {
      "title": "Token",
      "body": "模型用来计量内容的单位，不等于中文字数。你的输入、上下文和模型输出都可能计入用量，具体以使用记录为准。"
    },
    "credit": {
      "title": "余额与订阅额度",
      "body": "你在 Gabugu 里还能花多少（站内按美元 USD 计量）。ChatGPT、Claude 这些官网订阅是分开算的，不会自动变成这里的余额或额度。"
    }
  },
  "setup": {
    "eyebrow": "开始配置",
    "title": "选好工具，照着步骤开始",
    "intro": "默认按 Codex 桌面端讲解。选择你的系统，页面会显示对应的安装和操作步骤。",
    "osLabel": "你的电脑",
    "clientLabel": "使用工具",
    "windows": "Windows",
    "macos": "macOS",
    "claude": "Claude Code",
    "codex": "Codex 桌面端",
    "claudeDescription": "在终端中使用的编程工具；这里保留它独立的安装与启动步骤。",
    "codexDescription": "在桌面应用中打开文件夹、选择模型并开始对话。",
    "terminalWindows": "在开始菜单里搜 PowerShell，打开它。",
    "terminalMac": "按 Command + 空格，搜 Terminal，打开它。",
    "completeLabel": "确认结果",
    "actionLabel": "操作",
    "resultLabel": "结果",
    "manualNotice": "要是点了没弹出窗口，就在 Gabugu API 密钥那一行点「使用密钥」，照它给出的配置填。",
    "linuxNotice": "Linux 可以看官方文档；这页的安装步骤以 Windows 和 macOS 为主。",
    "copyCommand": "复制命令",
    "copyPrompt": "复制示例问题"
  },
  "steps": {
    "prepare": {
      "title": "先看余额与可用模型",
      "body": "打开主页和「模型广场」看一眼：还剩多少余额或订阅额度，准备使用的分组支持哪些模型与价格。够用就直接往下走，不用重复购买。",
      "result": "你知道现在能用哪些模型与分组，以及是否需要先充值。",
      "recharge": "余额不够就在这里充值",
      "subscription": "需要就购买 Gabugu 订阅套餐",
      "both": "按自己的用法挑，充值余额或者购买订阅套餐都行。够用就直接下一步。",
      "unavailable": "到主页看当前账户状态，可用入口会跟着账户模式变化。"
    },
    "choose": {
      "title": "挑一个客户端",
      "body": "两个都能帮你读项目、讲代码、干活。先挑一个就行。建密钥时记得选对应的分组，不然用不了；「使用密钥」里会列出支持哪些客户端。",
      "result": "你已经定了一个，也知道建密钥时该选哪个分组。",
      "claude": "选择 Claude Code",
      "codex": "选择 Codex 桌面端"
    },
    "installSwitch": {
      "title": "装 CC Switch",
      "body": "它专门管配置。先装它，再装你挑的那个客户端。",
      "result": "CC Switch 能打开，看得见「应用」页。",
      "windows": "打开官方发布页，下 Windows.msi，双击装。绝大多数电脑选 x64；如果「设置 → 系统 → 系统信息」里写着 ARM，就下 Windows-arm64.msi。装完在开始菜单里打开 CC Switch。",
      "macos": "在官方发布页下 macOS.dmg，双击，把 CC Switch 拖进「应用程序」，再从那里打开。这个包苹果芯和 Intel 都能装。"
    },
    "installClient": {
      "title": "装你挑的那个客户端",
      "body": "在 CC Switch 的「应用」页里装。要是那儿点不动，就去它的官方文档装。",
      "result": "「应用」列表里，你挑的那个已经显示版本号了。",
      "instruction": "左边栏点「应用」，找到你刚才挑的那个，点安装。装完看到版本号就算成功；没装上，先看它给的提示。"
    },
    "connect": {
      "title": "建密钥，把配置接上",
      "body": "密钥就是让工具能替你用额度的东西。先建一个跟你客户端对得上的，再按下面任一种方式接上。",
      "result": "配置写进去了。到底能不能用，还得下一步真发一次才知道。",
      "import": "打开「API 密钥」→ 建密钥，起个名字，分组选跟你客户端对得上的那个。存好后，在这条密钥那一行点「导入到 CCS」，浏览器要是问你，就允许它打开 CC Switch，再核对导进来的内容。",
      "manual": "打开「API 密钥」→ 建密钥，起个名字，分组选对。存好后点这条密钥的「使用密钥」，选你用的客户端和系统，照着弹窗里的配置一项项填。",
      "importResult": "CC Switch 里，你挑的那个客户端下面多了 Gabugu 这一条。",
      "manualResult": "你照着「使用密钥」把配置存进客户端了。"
    },
    "launch": {
      "title": "打开直连，把工具跑起来",
      "body": "用 CC Switch 导入的，在左边栏选 Claude Code 或 Codex，进「直连」，找到 Gabugu 那条点「启用」。手动配好的，直接开个新终端往下走。",
      "result": "工具起来了，屏幕上有地方让你输入问题。",
      "enabled": "已启用",
      "direct": "在你那个工具的「直连」页把 Gabugu 这条启用。这张图只说明状态在哪儿看。",
      "windows": "把下面的命令粘到 PowerShell 里回车。它会在你的用户目录里建一个 gabugu-start 练习文件夹，进去，然后启动工具。",
      "macos": "把下面的命令粘到终端里回车。它会在你的个人目录里建一个 gabugu-start 练习文件夹，进去，然后启动工具。",
      "claude": "运行 claude",
      "codex": "运行 codex"
    },
    "verify": {
      "title": "发一句话，看它答不答",
      "body": "在刚跑起来的工具里输入下面这句话。等它完整答完，再回 Gabugu 对一下这次调用。不同工具答得不一样，很正常。",
      "result": "工具完整答完了，并且能在「使用记录」里找到这一次和花了多少。",
      "simpleResult": "工具答上来了。",
      "simpleBody": "把下面的问题贴到工具的输入框并发送，等待完整回答。当前模式的额度可以在主页查看。",
      "prompt": "用三句话说明你能怎么帮我学编程，不要改文件。",
      "costNote": "花多少钱按页面上显示的价格算，以实际记录为准。",
      "usageUnavailable": "当前模式看不到使用记录。在工具里看它答没答上，额度去主页看。"
    }
  },
  "actions": {
    "keys": "API 密钥",
    "modelPlaza": "模型广场",
    "channels": "可用渠道",
    "purchaseRecharge": "充值余额",
    "purchaseSubscription": "购买订阅",
    "purchaseBoth": "充值 / 订阅",
    "subscriptions": "我的订阅",
    "usage": "使用记录",
    "monitor": "渠道状态",
    "dashboard": "主页",
    "download": "下载",
    "officialDocs": "官方文档",
    "manual": "手动配置",
    "openGuide": "回到对应步骤"
  },
  "resources": {
    "eyebrow": "工具资源",
    "title": "工具和资料，都在这里",
    "intro": "下载走官方入口，配置以站内「使用密钥」为准。",
    "externalTitle": "官方资源",
    "sdkTitle": "其他客户端与代码接入速查",
    "sdkIntro": "如需配置 Gemini CLI、Grok CLI 或 OpenCode，请在「API 密钥」创建对应分组密钥并点击「使用密钥」直接复制配置；在自己的代码或兼容应用中接入时，按下方协议地址配置即可（每条密钥仅可访问所属分组支持的模型）：",
    "copyEndpoint": "复制地址",
    "protocols": {
      "openai": {
        "title": "OpenAI 兼容协议",
        "models": "GPT / Grok / DeepSeek 等",
        "auth": "使用 Authorization: Bearer 密钥头；支持 Chat Completions 与 Responses。"
      },
      "anthropic": {
        "title": "Anthropic 原生协议",
        "models": "Claude 系列",
        "auth": "SDK 会自动拼接 /v1/messages；支持 x-api-key 或 Authorization: Bearer。"
      },
      "gemini": {
        "title": "Gemini 原生协议",
        "models": "Gemini 系列",
        "auth": "官方 GenAI SDK / Gemini CLI 填此根地址并指定 api_version=v1beta（自动拼接 /v1beta/...）；直调 REST 或 OpenCode / Vercel AI SDK 需手动加 /v1beta。支持 x-goog-api-key、Authorization: Bearer 或 ?key=。"
      }
    },
    "internalTitle": "Gabugu 里的页面",
    "ccswitchTitle": "CC Switch",
    "ccswitchDescription": "配置管理器：管客户端，也管连接配置。",
    "ccswitchManualTitle": "CC Switch 中文手册",
    "ccswitchManualDescription": "安装、界面、切换配置、出问题怎么办，都写在里面。",
    "claudeTitle": "Claude Code",
    "claudeDescription": "Claude Code 的安装与使用文档。",
    "codexTitle": "Codex 桌面端",
    "codexDescription": "下载官方桌面应用，在应用中进入 Codex。",
    "keysDescription": "创建、复制和管理你的 API 密钥。",
    "modelPlazaDescription": "查看各分组可用模型与实时价格。",
    "channelsDescription": "查看可用模型与价格。",
    "usageDescription": "核对成功调用的请求、用量和费用。",
    "monitorDescription": "查看当前渠道是否可用。",
    "purchaseDescription": "在线充值余额或查看订阅套餐。",
    "subscriptionsDescription": "查看当前订阅套餐、配额和到期时间。",
    "dashboardDescription": "进入 Gabugu 主页。",
    "external": "外部链接以页面上显示的官方入口为准。",
    "linux": "Linux 用户先看对应客户端的官方安装文档。"
  },
  "faq": {
    "eyebrow": "常见问题",
    "title": "常见问题",
    "intro": "出错了先保留客户端里的报错原文，再结合时间去核对模型、密钥、接口地址和「渠道状态」。",
    "key": {
      "question": "API 密钥就是账号密码吗？",
      "answer": "不是。账号用来登录网页，密钥给工具调接口用。两个都得收好。"
    },
    "tools": {
      "question": "装了 CC Switch 就能对话了吗？",
      "answer": "还不够。还得装一个客户端、配好 Gabugu 密钥、账户里有额度。CC Switch 只管配置，不替客户端干活，也不给额度。"
    },
    "command": {
      "question": "终端提示 command not found 怎么办？",
      "answer": "说明还没装好，或者没进 PATH。回 CC Switch 的「应用」页装，或者照官方文档装完，再重开终端。"
    },
    "import": {
      "question": "点了「导入到 CCS」没反应？",
      "answer": "先允许浏览器打开 CC Switch。还不行，就在密钥那一行点「使用密钥」，照着生成的配置手动填。"
    },
    "login": {
      "question": "配好了为什么还让我登录？",
      "answer": "先看终端里的原话，确认 CC Switch 的「直连」已经启用，再开个新终端重进一次。有些客户端有自己的登录流程，绕不开。"
    },
    "model": {
      "question": "模型或地址填错了会怎样？",
      "answer": "可能直接失败，也可能提示不支持这个模型。核对地址是不是 Gabugu 的，模型名是不是当前开通里有的。"
    },
    "credit": {
      "question": "为什么没额度或者没记录？",
      "answer": "先确认客户端是否真的成功返回回答、密钥与所选分组是否正确。未成功完成的请求不会进入「使用记录」，请核对账户余额或订阅配额、日期筛选范围及「渠道状态」。"
    },
    "errors": {
      "question": "哪里看报错和历史？",
      "answer": "报错请先保留客户端返回的原始信息（如 401/403/404/429/502），再去「渠道状态」确认线路是否正常；已成功计费的历史调用可在「使用记录」按时间、模型和密钥核对。"
    }
  },
  "media": {
    "expand": "放大图片",
    "close": "关闭图片",
    "unavailable": "图片暂不可用",
    "demoLabel": "演示截图",
    "officialLabel": "官方截图",
    "schematicLabel": "示意图",
    "keysCreateTitle": "建一个 API 密钥",
    "keysCreateAlt": "Gabugu 创建 API 密钥的演示截图",
    "keysListTitle": "密钥列表",
    "keysListAlt": "Gabugu 密钥列表的演示截图",
    "ccsAppsTitle": "CC Switch 应用",
    "ccsAppsAlt": "CC Switch 4.0.0 官方应用管理截图，展示各客户端的安装入口和版本号",
    "ccsProviderTitle": "CC Switch 连接配置",
    "ccsProviderAlt": "示意图：在你选的那个工具的直连页启用 Gabugu 配置",
    "terminalTitle": "终端启动",
    "terminalAlt": "终端里运行客户端的演示截图",
    "usageTitle": "查看使用记录",
    "usageAlt": "Gabugu 使用记录页面的演示截图",
    "officialSource": "官方来源",
    "demonstrationOnly": "只用来指位置，界面可能随版本变。",
    "sourceVersion": "来源版本：{version}",
    "keysCodexTitle": "Codex：选择 OpenAI 分组",
    "keysCodexAlt": "Codex 演示密钥表单，已选择 OpenAI 厂商及兼容分组",
    "keysActionsTitle": "保存后，使用这两个入口",
    "keysActionsAlt": "API 密钥操作区的使用密钥和导入到 CCS 两个按钮"
  },
  "footer": {
    "title": "准备好，开始第一段对话",
    "body": "创建一条密钥，按教程连接工具。需要时，随时回来查看。",
    "action": "去建密钥",
    "maintenance": "客户端和渠道会更新，这页也跟着改。"
  },
  "desktop": {
    "badge": "桌面端入门",
    "stepCount": "六步完成首次连接",
    "download": "下载桌面应用",
    "installDocs": "官方安装说明",
    "gatewayDocs": "桌面端配置说明",
    "sourceNotice": "官方当前将 Codex 入口放在 ChatGPT 桌面应用中；已安装独立 Codex 的用户可直接配置并重新打开。",
    "steps": {
      "prepare": {
        "title": "确认账户余额与可用模型",
        "body": "打开主页确认账户有可用余额或订阅额度，并在「模型广场」查看准备使用分组的模型与价格。已有余额或套餐就可以继续。",
        "result": "你知道账户是否可用，以及所选分组能用哪些模型。"
      },
      "installClient": {
        "title": "安装 Codex 桌面端",
        "body": "从官方入口下载与你的系统对应的桌面应用。完成安装后先打开一次；已安装的用户可以继续下一步。",
        "result": "应用可以正常打开，并能找到 Codex 入口。"
      },
      "installSwitch": {
        "title": "准备配置工具 CC Switch",
        "body": "CC Switch 帮你保存和切换 Gabugu 配置。推荐使用它导入密钥；熟悉手动配置的用户也可以按下一步的「使用密钥」说明设置。",
        "manualBody": "CC Switch 可以管理多套配置。本站当前使用手动配置：按下一步的「使用密钥」说明操作；如果只需要这一套配置，可以跳过安装 CC Switch。",
        "result": "CC Switch 已打开，或你已决定使用手动配置。"
      },
      "connect": {
        "title": "创建密钥，连接 Gabugu",
        "body": "在「API 密钥」创建一条新密钥，选择支持 Codex 的 OpenAI 分组。然后按下方入口导入或手动配置。",
        "manualBody": "在「API 密钥」创建一条新密钥，选择支持 Codex 的 OpenAI 分组，再按「使用密钥」的说明保存配置。",
        "result": "Gabugu 配置已经保存到当前电脑的 Codex 配置目录。"
      },
      "launch": {
        "title": "重新打开，选择工作文件夹",
        "body": "配置已保存后，在桌面应用里完成下面三步。",
        "result": "聊天窗口已打开，当前工作文件夹和所选模型都已确认。"
      },
      "verify": {
        "title": "发出第一条消息",
        "body": "把下面的问题贴到 Codex 输入框并发送。收到完整回答后，回到 Gabugu 的「使用记录」核对这次请求。",
        "result": "Codex 收到完整回答，使用记录中也出现了对应的请求。"
      }
    },
    "install": {
      "windows": "在官方下载页选择 Windows，按页面提示完成安装。安装后从开始菜单打开桌面应用，再进入 Codex。",
      "macos": "在官方下载页选择 macOS，打开下载的安装文件并按提示安装。安装后从「应用程序」打开，再进入 Codex。"
    },
    "connect": {
      "import": "保存密钥后，在这条密钥的操作区点击「导入到 CCS」，允许浏览器打开 CC Switch。进入「Codex → 直连」，核对导入的 Gabugu 配置并启用。",
      "manual": "保存密钥后点击「使用密钥」，选择 Codex 和你的系统。按弹窗生成的文件说明保存配置；已有配置时只合并需要的字段，保留其他设置。",
      "importResult": "CC Switch 的 Codex 页面显示 Gabugu 配置已启用。",
      "manualResult": "你已根据「使用密钥」保存所需的 Codex 配置文件。",
      "pathTitle": "手动配置时，文件在哪里？",
      "pathWindows": "按 Win + R，输入下面的路径并打开。按「使用密钥」给出的文件名保存；目录不存在时先创建。",
      "pathMac": "在 Finder 按 Command + Shift + G，输入下面的路径并打开。按「使用密钥」给出的文件名保存；目录不存在时先创建。",
      "copyPath": "复制文件夹路径",
      "restart": "配置保存后，需要完全退出并重新打开桌面应用。只关闭当前聊天不会重新加载配置。"
    },
    "launch": {
      "restart": "启用配置并重新打开",
      "restartBody": "在 CC Switch 的 Codex 页面启用 Gabugu；手动配置的用户确认文件已保存。Windows 从托盘退出应用，macOS 使用 Command + Q，再重新打开。",
      "folder": "选择一个练习文件夹",
      "folderBody": "在文件管理器里新建 gabugu-start 空文件夹，然后在应用中打开它。",
      "chat": "进入 Codex，新建聊天",
      "chatBody": "进入 Codex 模式，新建聊天或任务。确认工作文件夹和模型，再输入问题。",
      "restartBodyWindows": "启用 Gabugu 配置后，从应用菜单或系统托盘选择退出，再从开始菜单重新打开。",
      "restartBodyMac": "启用 Gabugu 配置后，按 Command + Q 完全退出，再从「应用程序」重新打开。"
    },
    "preview": {
      "project": "项目",
      "newChat": "新建聊天",
      "model": "选择可用模型",
      "local": "本地工作文件夹",
      "heading": "从一个小问题开始",
      "description": "先在练习文件夹里熟悉工具。",
      "prompt": "请介绍你可以怎样帮助我，不要修改文件。",
      "caption": "桌面操作示意：新建聊天，确认工作文件夹和可用模型，再输入问题。界面会随版本更新。"
    },
    "faq": {
      "commandQuestion": "已装好桌面端，还要运行命令吗？",
      "commandAnswer": "使用 Codex 桌面端时，直接从开始菜单或应用程序打开即可。配置生效后，在窗口里选择文件夹、新建聊天并发送消息。",
      "loginAnswer": "先查看应用显示的具体提示。应用登录和 Gabugu API 认证是不同的环节；确认配置已保存并完全重启。若仍失败，保留错误原文并按「使用密钥」说明核对。"
    },
    "resourcesNote": "Codex 的下载安装、桌面配置与官方说明都在这里。"
  }
}

export default { guide }
