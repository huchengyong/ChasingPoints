import fs from 'node:fs'
import path from 'node:path'
import { execFileSync } from 'node:child_process'

const ROOT = '/Users/wisesearch/Projects/ChasingPoints'
const OUTPUT_DIR = path.join(ROOT, 'docs', 'copyright')

const RIGHTS_HOLDER = '上海点早早互联网络科技有限公司'
const SOFTWARE_NAME = '追分台球社交对局软件'
const SOFTWARE_SHORT_NAME = '追分'
const VERSION = '1.0.0'

const PROGRAM_LINES_PER_PAGE = 50
const PROGRAM_PAGE_COUNT = 60
const PROGRAM_EXCERPT_LINES = PROGRAM_LINES_PER_PAGE * 30
const DOC_LINES_PER_PAGE = 32

const CHROME_CANDIDATES = [
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Chromium.app/Contents/MacOS/Chromium',
  '/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge'
]

const PROGRAM_EXTENSIONS = new Set(['.go', '.api'])

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true })
}

function escapeHtml(value) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function findChromeBinary() {
  const envBinary = process.env.CHROME_BIN
  if (envBinary && fs.existsSync(envBinary)) {
    return envBinary
  }

  return CHROME_CANDIDATES.find((candidate) => fs.existsSync(candidate))
}

function walkFiles(dirPath, collector = []) {
  for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
    const absolutePath = path.join(dirPath, entry.name)
    if (entry.isDirectory()) {
      walkFiles(absolutePath, collector)
      continue
    }

    collector.push(absolutePath)
  }

  return collector
}

function getProgramSourceFiles() {
  const backendRoot = path.join(ROOT, 'backend')
  const files = walkFiles(backendRoot)
    .filter((filePath) => PROGRAM_EXTENSIONS.has(path.extname(filePath)))
    .filter((filePath) => !filePath.includes(`${path.sep}vendor${path.sep}`))
    .filter((filePath) => !filePath.endsWith('_test.go'))
    .sort((a, b) => a.localeCompare(b, 'zh-Hans-CN'))

  return files
}

function normalizeSourceLine(line) {
  return line.replace(/\t/g, '    ')
}

function buildProgramLinePool() {
  const files = getProgramSourceFiles()
  const lines = []

  for (const filePath of files) {
    const relativePath = path.relative(ROOT, filePath).replaceAll(path.sep, '/')
    lines.push(`// 文件：${relativePath}`)

    const content = fs.readFileSync(filePath, 'utf8').replace(/\r\n/g, '\n').split('\n')
    for (const line of content) {
      lines.push(normalizeSourceLine(line))
    }
  }

  return { files, lines }
}

function paginateLines(lines, linesPerPage, emptyLineFactory = () => '') {
  const pages = []
  for (let i = 0; i < lines.length; i += linesPerPage) {
    const pageLines = lines.slice(i, i + linesPerPage)
    while (pageLines.length < linesPerPage) {
      pageLines.push(emptyLineFactory())
    }
    pages.push(pageLines)
  }
  return pages
}

function renderProgramHtml(programPages, excerptedLines, sourceFiles) {
  const pageHtml = programPages
    .map((pageLines, pageIndex) => {
      const globalStart = pageIndex * PROGRAM_LINES_PER_PAGE
      const lineRows = pageLines
        .map((line, lineIndex) => {
          const lineNumber = globalStart + lineIndex + 1
          return `
            <div class="code-line">
              <span class="line-number">${lineNumber}</span>
              <span class="line-content">${escapeHtml(line || ' ')}</span>
            </div>
          `
        })
        .join('')

      return `
        <section class="page">
          <header class="page-header">
            <div>${escapeHtml(SOFTWARE_NAME)} 程序鉴别材料</div>
            <div>权利人：${escapeHtml(RIGHTS_HOLDER)}</div>
            <div>简称：${escapeHtml(SOFTWARE_SHORT_NAME)}　版本号：${escapeHtml(VERSION)}</div>
          </header>
          <main class="page-body">
            ${lineRows}
          </main>
          <footer class="page-footer">
            <span>前连续30页与后连续30页合并编制</span>
            <span>第 ${pageIndex + 1} 页 / 共 ${programPages.length} 页</span>
          </footer>
        </section>
      `
    })
    .join('')

  const fileList = sourceFiles
    .map((filePath) => `<li>${escapeHtml(path.relative(ROOT, filePath).replaceAll(path.sep, '/'))}</li>`)
    .join('')

  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <title>${escapeHtml(SOFTWARE_NAME)} 程序鉴别材料</title>
  <style>
    @page {
      size: A4 portrait;
      margin: 0;
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      color: #111827;
      background: #ffffff;
      font-family: "Courier New", "SFMono-Regular", monospace;
    }

    .page {
      width: 210mm;
      height: 297mm;
      page-break-after: always;
      padding: 11mm 9mm 10mm 9mm;
    }

    .page:last-child {
      page-break-after: auto;
    }

    .page-header,
    .page-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-size: 10px;
      line-height: 1.2;
      color: #374151;
      height: 10mm;
      border-bottom: 1px solid #d1d5db;
      padding-bottom: 2mm;
    }

    .page-footer {
      border-bottom: none;
      border-top: 1px solid #d1d5db;
      padding-top: 2mm;
      padding-bottom: 0;
      margin-top: 2mm;
      height: 8mm;
    }

    .page-body {
      height: 264mm;
      padding-top: 2mm;
      overflow: hidden;
    }

    .code-line {
      display: grid;
      grid-template-columns: 16mm 1fr;
      gap: 3mm;
      font-size: 8.9px;
      line-height: 1.38;
      white-space: pre;
    }

    .line-number {
      color: #6b7280;
      text-align: right;
      user-select: none;
    }

    .line-content {
      overflow: hidden;
      text-overflow: clip;
    }

    .appendix {
      display: none;
    }
  </style>
</head>
<body>
  ${pageHtml}
  <aside class="appendix">
    <h1>编制说明</h1>
    <p>本材料依据仓库主程序生成，摘取源程序前后各连续30页，每页50行。</p>
    <p>本次摘录源程序总行数：${excerptedLines}</p>
    <ul>${fileList}</ul>
  </aside>
</body>
</html>`
}

function padDocLines(lines, pageIndex) {
  const padded = [...lines]
  const supplements = [
    '本页内容与系统当前版本功能范围保持一致。',
    '相关数据状态以统一后端服务返回结果为准。',
    '移动端、后台与官网共享统一业务域模型。',
    '本节描述适用于正式发布版本 1.0.0。',
    '业务异常通过界面提示与日志记录协同处理。',
    '权限校验、参数校验与状态校验按接口约束执行。',
    '页面展示与实际配置可根据运营需求进行调整。',
    '系统在网络异常时保留重试与恢复处理能力。'
  ]

  let supplementIndex = 0
  while (padded.length < DOC_LINES_PER_PAGE) {
    const fallback = supplements[(pageIndex + supplementIndex) % supplements.length]
    padded.push(fallback)
    supplementIndex += 1
  }

  return padded.slice(0, DOC_LINES_PER_PAGE)
}

function chunkText(text, width = 28) {
  if (!text) {
    return ['']
  }

  const chunks = []
  let remaining = text.trim()
  while (remaining.length > width) {
    chunks.push(remaining.slice(0, width))
    remaining = remaining.slice(width)
  }
  chunks.push(remaining)
  return chunks
}

function withWrappedPrefix(prefix, text) {
  const segments = chunkText(`${prefix}${text}`)
  return segments
}

function buildDocPages() {
  const topics = [
    {
      title: '软件概述与编制说明',
      audience: '面向台球爱好者、赛事运营方、球房管理方和平台管理员。',
      modules: ['移动端用户应用', '后台管理端', '官网展示端', '统一后端服务'],
      functions: ['用户登录与身份识别', '对局记录与公开观战', '赛事资讯与赛季榜单', '球房签到与附近检索'],
      flows: ['统一接口契约由 backend/chasing_points.api 定义', '业务请求统一经过后端鉴权、逻辑层和数据层', '移动端与后台通过 HTTP 及 WebSocket 获取实时状态', '官网用于品牌展示、下载引导和协议页面公开访问'],
      notes: ['本文档用于说明软件主要用途、结构和操作流程。', '文档内容依据当前仓库代码与已实现能力编制。']
    },
    {
      title: '目标用户与业务场景',
      audience: '适用于个人台球用户、俱乐部、球房经营者和运营人员。',
      modules: ['约战对局', '战绩沉淀', '社交互动', '运营审核'],
      functions: ['支持新用户快速登录与绑定手机号', '支持熟人对局与陌生人约战', '支持查看排行榜、战绩、赛季数据', '支持运营人员维护资讯和审核内容'],
      flows: ['系统围绕台球运动的赛前、赛中、赛后流程设计。', '通过统一账号体系串联用户、赛事、球房和内容数据。', '通过消息通知和推送维持用户活跃与互动。'],
      notes: ['业务目标是提升对局记录效率和平台运营效率。', '软件强调多端协同与实时性。']
    },
    {
      title: '系统总体架构',
      audience: '供产品、实施、测试和运维人员理解整体结构。',
      modules: ['UniApp Vue3 移动端', 'Vue3 Vite 管理后台', 'Nuxt3 官网', 'go-zero 后端服务'],
      functions: ['前端承担页面展示与交互', '后端提供统一数据接口和实时能力', '后台支持审核与运营', '官网提供下载与品牌内容'],
      flows: ['后端采用 handler、logic、model 分层结构。', '共享依赖统一经 ServiceContext 注入。', '重要业务状态存储于 MySQL，短期缓存和验证码依赖 Redis。', '部分消息通过 WebSocket 和推送服务分发。'],
      notes: ['系统设计遵循前后端分离与接口统一原则。', '架构支持后续功能扩展和模块拆分。']
    },
    {
      title: '运行环境与部署模式',
      audience: '供实施、部署和运维阶段参考。',
      modules: ['Linux 服务器', 'MySQL 数据库', 'Redis 缓存', 'Nginx 反向代理'],
      functions: ['后端服务提供 REST 接口与 WebSocket 路由', '前端静态资源可部署于 Web 环境', '移动端构建后安装在用户设备上', '官网面向公开访问场景提供 SEO 页面'],
      flows: ['Nginx 转发请求至 go-zero 服务。', '数据库保存用户、对局、赛事、球房等核心数据。', 'Redis 保存验证码、配额和部分实时状态。', '推送和短信能力由配置控制启用。'],
      notes: ['部署时需配置域名、数据库连接和密钥。', '系统支持开发与生产环境切换。']
    },
    {
      title: '账号登录与注册',
      audience: '面向移动端普通用户和管理员用户。',
      modules: ['短信验证码登录', '第三方授权登录', '手机号绑定', '管理员登录'],
      functions: ['用户可通过短信验证码完成登录或注册', '支持第三方 OAuth 登录后补绑手机号', '后台管理员通过专用入口完成登录', '系统返回访问令牌与刷新令牌'],
      flows: ['登录接口由 auth 分组提供。', '成功登录后前端保存 token 并更新用户状态。', '需要补全手机号时展示绑定流程。', '鉴权失败时统一跳转登录页或提示重新认证。'],
      notes: ['登录流程强调易用性与安全性平衡。', '短信验证码场景区分登录和绑定。']
    },
    {
      title: '个人资料与隐私设置',
      audience: '面向需要维护个人资料的终端用户。',
      modules: ['昵称维护', '头像展示', '隐私设置', '会员状态展示'],
      functions: ['用户可修改昵称并查看基础资料', '支持查看个人统计与会员中心信息', '支持读取和修改隐私设置', '支持显示账户当前权限与状态'],
      flows: ['个人资料由 user 分组接口返回。', '隐私设置变更后即时写入数据库。', '前端 store 负责持久化用户状态。', '页面根据接口返回字段控制展示范围。'],
      notes: ['个人信息展示遵循最小必要原则。', '用户可按产品设定管理资料公开范围。']
    },
    {
      title: '首页与信息聚合',
      audience: '面向移动端首页浏览场景。',
      modules: ['欢迎页', '首页总览', '入口漏斗', '推荐信息'],
      functions: ['首页提供快速进入对局、排行和动态的入口', '展示用户段位、近期数据和重点提示', '欢迎页与登录页承接新用户漏斗', '通过统一格式工具处理时间和展示文案'],
      flows: ['首页逻辑依赖统一请求层与纯函数工具。', '用户未登录时引导进入登录流程。', '已登录用户可直接查看核心模块入口。', '业务提示与状态根据接口结果实时更新。'],
      notes: ['首页是各业务能力的聚合入口。', '页面结构兼顾首屏效率与后续扩展。']
    },
    {
      title: '对局大厅与公开观战',
      audience: '面向正在浏览对局或观战的用户。',
      modules: ['对局大厅', '正在进行列表', '公开对局详情', '观战模式'],
      functions: ['展示平台进行中的对局列表', '支持按页查询公开对局信息', '可查看双方选手、比分、局记录和时长', '支持第三方视角进入观战模式'],
      flows: ['公开接口无需登录即可访问部分数据。', '对局大厅页面支持下拉刷新与状态更新。', '对局详情页根据对局状态渲染不同信息。', '服务端计算持续时长并返回当前局面。'],
      notes: ['公开观战提升平台内容活跃度。', '数据展示兼顾实时性与只读安全性。']
    },
    {
      title: '对局创建与对手选择',
      audience: '面向发起新对局的终端用户。',
      modules: ['创建对局', '选择对手', '设定局制', '确认开局'],
      functions: ['支持选择台球玩法和局数', '支持选择注册用户或填写非注册对手', '支持对手信息预览和校验', '支持初始化比分与回合数据'],
      flows: ['创建请求经 match 分组接口处理。', '必要参数在前端和后端均执行校验。', '创建成功后返回对局标识和初始化状态。', '页面根据结果跳转到进行中页面。'],
      notes: ['创建流程强调简洁和低输入成本。', '系统保留后续扩展更多规则选项的能力。']
    },
    {
      title: '对局进行与实时同步',
      audience: '面向正在记录比赛过程的用户。',
      modules: ['比分录入', '局次推进', '实时修订号', 'WebSocket 同步'],
      functions: ['支持更新每局得分和胜负结果', '支持当前局分数与整场比分同步', '支持使用服务端修订号控制状态一致性', '支持用户级和对局级实时通知'],
      flows: ['前端通过 websocket.js 维护长连接。', '后端提供 match/ws 与 user/ws 两条路由。', '收到快照或增量后页面即时刷新。', '网络异常时按策略重连并恢复状态。'],
      notes: ['实时同步是对局模块的关键能力。', '系统优先保证比分与状态的一致性。']
    },
    {
      title: '对局结果与历史记录',
      audience: '面向赛后复盘和记录查询场景。',
      modules: ['对局结算', '历史列表', '战报分享', '交锋记录'],
      functions: ['比赛结束后生成最终比分与胜负结果', '用户可查看个人历史对局记录', '支持生成战绩海报和分享数据', '支持查看对手维度与双人交锋维度的历史汇总'],
      flows: ['结果页面根据服务端返回数据展示摘要。', '分享页使用统一数据整形逻辑生成海报内容。', '历史记录支持分页查询和按对象跳转。', '统计摘要依赖服务端聚合查询。'],
      notes: ['赛后内容帮助用户沉淀长期数据价值。', '分享能力可增强传播与留存。']
    },
    {
      title: '排行榜与统计分析',
      audience: '面向关注竞技表现和成长轨迹的用户。',
      modules: ['段位排行榜', '趋势统计', '单杆高分', '时长与强度分析'],
      functions: ['展示按排位分排序的榜单', '支持查看近期表现趋势和分数变化', '支持统计单杆高分与对手强度', '支持展示比赛时长和玩法维度统计'],
      flows: ['相关接口由 public、rank、stats 等分组提供。', '服务端根据历史对局数据进行聚合计算。', '前端按卡片和图表化方式展示结果。', '统计页根据玩法和维度切换展示内容。'],
      notes: ['统计分析提升软件的专业度与数据价值。', '展示结果强调易理解和可比较。']
    },
    {
      title: '好友关系与关注体系',
      audience: '面向社交关系维护场景。',
      modules: ['好友申请', '好友列表', '黑名单', '关注与粉丝'],
      functions: ['支持检索用户并发送好友申请', '支持接受、拒绝和删除好友关系', '支持好友黑名单管理', '支持关注、取消关注及粉丝列表展示'],
      flows: ['好友和关注分属独立接口分组。', '关系变更后同步更新通知与页面状态。', '相关入口在个人主页、动态和好友列表中互通。', '部分纯逻辑规则通过 tests 进行验证。'],
      notes: ['社交关系为约战和动态传播提供基础。', '系统对重复申请和非法状态做约束处理。']
    },
    {
      title: '动态广场与社交互动',
      audience: '面向内容发布和互动场景。',
      modules: ['动态广场', '发布动态', '我的动态', '内容审核'],
      functions: ['用户可查看动态列表和广场信息', '支持发布与管理个人动态', '支持展示战报类内容和互动入口', '后台可对动态进行审核处理'],
      flows: ['social 分组接口负责动态数据读写。', '前端页面支持分页加载与刷新。', '后台审核结果回写内容状态并通知用户。', '社交内容可与对局结果和好友关系联动。'],
      notes: ['动态模块增强平台社区属性。', '审核能力保障内容质量和合规性。']
    },
    {
      title: '成就、称号与挑战',
      audience: '面向成长激励和互动邀约场景。',
      modules: ['成就列表', '称号管理', '挑战发起', '待处理挑战'],
      functions: ['系统根据业务规则展示用户成就', '支持查看成就详情与称号管理', '支持向目标用户发送挑战邀约', '支持接受或拒绝待处理挑战'],
      flows: ['achievement 与 challenge 分组各自处理核心逻辑。', '前端通过列表和详情页展示成就与挑战状态。', '挑战状态变化会写入通知系统。', '称号类信息可在个人资料中展示。'],
      notes: ['激励与挑战机制可提升用户活跃度。', '所有状态切换均受服务端校验保护。']
    },
    {
      title: '赛事、赛季与报名签到',
      audience: '面向赛事参与用户和运营人员。',
      modules: ['赛事情报', '赛事报名', '签到检录', '赛季报告'],
      functions: ['展示赛事列表、阶段和详情', '支持加入、离开、取消与完赛处理', '支持比赛签到、赛程查看和战绩更新', '支持当前赛季、赛季榜单和赛季报告'],
      flows: ['赛事接口覆盖创建、查询、报名和对阵更新。', '赛事情报采用事件与阶段双表结构。', '赛季模块聚合用户在一个赛季内的数据表现。', '后台可维护资讯、阶段与发布状态。'],
      notes: ['赛事与赛季构成平台的组织化竞技能力。', '流程设计覆盖赛前、赛中和赛后。']
    },
    {
      title: '球房管理与签到定位',
      audience: '面向球房经营场景和用户到店场景。',
      modules: ['球房创建', '附近检索', '球房详情', '签到记录'],
      functions: ['用户可提交球房基础信息', '支持查询附近球房与球房详情', '支持在球房内完成签到记录', '后台可对球房进行审核与奖励配置管理'],
      flows: ['venue 分组接口负责球房业务。', '后端地理编码任务用于地址信息处理。', '签到记录用于沉淀用户到店行为数据。', '后台审核后球房状态同步至前端。'],
      notes: ['球房模块连接线上社区与线下门店。', '地址类能力依赖后端任务与配置。']
    },
    {
      title: '通知推送与消息提醒',
      audience: '面向需要及时获知状态变化的用户。',
      modules: ['系统通知', '未读计数', '全部已读', '推送令牌上报'],
      functions: ['支持查看通知列表与未读数量', '支持单条已读、全部已读和删除通知', '移动端启动时上报推送标识', '支持对局、挑战和审核类消息提醒'],
      flows: ['通知接口由 notification 分组提供。', 'App 级生命周期负责推送标识注册。', '服务端集成 UniPush 作为推送支撑。', '用户操作后未读数即时同步。'],
      notes: ['通知系统承担跨业务模块的消息触达职责。', '消息提醒同时结合站内与推送能力。']
    },
    {
      title: '管理后台与运营审核',
      audience: '面向管理员、审核员和运营人员。',
      modules: ['管理员登录', '首页统计', '用户管理', '内容与球房审核'],
      functions: ['后台提供登录与初始化管理员能力', '支持查看仪表盘统计、用户列表和对局列表', '支持维护赛事情报、球房审核和动态审核', '支持奖励配置及相关记录查询'],
      flows: ['后台统一经 axios 请求层访问后端。', '未授权访问由路由守卫拦截并跳转登录页。', '业务成功判定按 code 或 success 字段统一处理。', '后台模块复用同一后端服务中的 admin 路由。'],
      notes: ['后台是平台运营和治理的重要入口。', '管理功能注重数据可视和审核闭环。']
    },
    {
      title: '官网、运维与数据安全',
      audience: '面向访客、运维人员和合规场景。',
      modules: ['品牌首页', '下载页', '协议页', '运行维护'],
      functions: ['官网展示品牌信息、下载入口和联系方式', '提供隐私政策、用户协议和联系页', '支持 SEO 基础配置与静态页面发布', '支持数据库迁移、配置加载和日志排查'],
      flows: ['官网基于 Nuxt3 页面路由实现。', '正式域名与下载地址通过环境变量注入。', '后端通过 .env 与配置文件完成参数装配。', '数据库结构由 migrations 统一管理并配合模型使用。'],
      notes: ['系统运行关注账号安全、数据一致性和配置安全。', '敏感配置通过环境变量或部署配置管理。']
    }
  ]

  const pages = []

  topics.forEach((topic, topicIndex) => {
    const pageOneLines = [
      `章节：${topic.title}`,
      `软件名称：${SOFTWARE_NAME}`,
      `软件简称：${SOFTWARE_SHORT_NAME}`,
      `版本号：${VERSION}`,
      `权利人：${RIGHTS_HOLDER}`,
      `适用对象：${topic.audience}`,
      '',
      ...withWrappedPrefix('组成模块：', topic.modules.join('、')),
      '',
      ...withWrappedPrefix('主要功能：', topic.functions.join('；')),
      '',
      ...topic.notes.flatMap((note) => withWrappedPrefix('说明：', note))
    ]

    const pageTwoLines = [
      `章节：${topic.title} 功能流程`,
      '一、功能入口说明',
      ...topic.modules.flatMap((moduleName, index) => withWrappedPrefix(`${index + 1}. `, `${moduleName} 对应界面或服务节点负责承接本模块操作。`)),
      '',
      '二、典型业务动作',
      ...topic.functions.flatMap((feature, index) => withWrappedPrefix(`${index + 1}. `, feature)),
      '',
      '三、处理流程摘要',
      ...topic.flows.flatMap((flow, index) => withWrappedPrefix(`${index + 1}. `, flow))
    ]

    const pageThreeLines = [
      `章节：${topic.title} 运维与注意事项`,
      '一、数据与状态说明',
      ...topic.flows.flatMap((flow) => withWrappedPrefix('状态：', flow)),
      '',
      '二、操作提示',
      ...topic.notes.flatMap((note) => withWrappedPrefix('提示：', note)),
      '',
      '三、质量保障',
      '提示：界面输入、接口参数与权限均进行校验。',
      '提示：异常结果会通过提示语、日志或状态回写进行反馈。',
      '提示：前后端交互遵循统一接口契约，减少数据歧义。',
      '提示：配置项可按开发、测试、生产环境分别管理。'
    ]

    pages.push(padDocLines(pageOneLines, topicIndex * 3))
    pages.push(padDocLines(pageTwoLines, topicIndex * 3 + 1))
    pages.push(padDocLines(pageThreeLines, topicIndex * 3 + 2))
  })

  return pages
}

function renderDocumentHtml(docPages) {
  const pageHtml = docPages
    .map((pageLines, pageIndex) => {
      const lineHtml = pageLines
        .map((line) => `<div class="doc-line">${escapeHtml(line || ' ')}</div>`)
        .join('')

      return `
        <section class="page">
          <header class="page-header">
            <div>${escapeHtml(SOFTWARE_NAME)} 使用说明书</div>
            <div>权利人：${escapeHtml(RIGHTS_HOLDER)}</div>
            <div>版本号：${escapeHtml(VERSION)}</div>
          </header>
          <main class="page-body">
            ${lineHtml}
          </main>
          <footer class="page-footer">
            <span>文档鉴别材料</span>
            <span>第 ${pageIndex + 1} 页 / 共 ${docPages.length} 页</span>
          </footer>
        </section>
      `
    })
    .join('')

  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <title>${escapeHtml(SOFTWARE_NAME)} 文档鉴别材料</title>
  <style>
    @page {
      size: A4 portrait;
      margin: 0;
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      color: #111827;
      background: #ffffff;
      font-family: "PingFang SC", "Microsoft YaHei", sans-serif;
    }

    .page {
      width: 210mm;
      height: 297mm;
      page-break-after: always;
      padding: 15mm 16mm 12mm 16mm;
    }

    .page:last-child {
      page-break-after: auto;
    }

    .page-header,
    .page-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      height: 10mm;
      font-size: 10px;
      color: #374151;
      border-bottom: 1px solid #d1d5db;
      padding-bottom: 2mm;
    }

    .page-footer {
      border-bottom: none;
      border-top: 1px solid #d1d5db;
      padding-top: 2mm;
      padding-bottom: 0;
      margin-top: 2mm;
      height: 8mm;
    }

    .page-body {
      height: 252mm;
      padding-top: 4mm;
      overflow: hidden;
    }

    .doc-line {
      font-size: 12px;
      line-height: 1.62;
      min-height: 6.9mm;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: clip;
    }
  </style>
</head>
<body>
  ${pageHtml}
</body>
</html>`
}

function printHtmlToPdf(htmlPath, pdfPath) {
  const chromeBinary = findChromeBinary()
  if (!chromeBinary) {
    throw new Error('未找到可用的 Chrome/Chromium/Edge，可通过 CHROME_BIN 指定浏览器路径。')
  }

  execFileSync(
    chromeBinary,
    [
      '--headless=new',
      '--disable-gpu',
      '--no-first-run',
      '--no-default-browser-check',
      '--allow-file-access-from-files',
      '--enable-local-file-accesses',
      '--disable-dev-shm-usage',
      '--no-pdf-header-footer',
      `--print-to-pdf=${pdfPath}`,
      `file://${htmlPath}`
    ],
    { stdio: 'inherit' }
  )
}

function writeFile(filePath, content) {
  fs.writeFileSync(filePath, content, 'utf8')
}

function generateProgramMaterial() {
  const { files, lines } = buildProgramLinePool()
  if (lines.length < PROGRAM_EXCERPT_LINES * 2) {
    throw new Error(`源程序总行数不足 ${PROGRAM_EXCERPT_LINES * 2} 行，无法截取前后各 30 页。`)
  }

  const excerptLines = [
    ...lines.slice(0, PROGRAM_EXCERPT_LINES),
    ...lines.slice(-PROGRAM_EXCERPT_LINES)
  ]
  const pages = paginateLines(excerptLines, PROGRAM_LINES_PER_PAGE)
  if (pages.length !== PROGRAM_PAGE_COUNT) {
    throw new Error(`程序鉴别材料页数异常，期望 ${PROGRAM_PAGE_COUNT} 页，实际 ${pages.length} 页。`)
  }

  const html = renderProgramHtml(pages, excerptLines.length, files)
  const htmlPath = path.join(OUTPUT_DIR, 'program-identification-material.html')
  const pdfPath = path.join(OUTPUT_DIR, '程序鉴别材料.pdf')
  writeFile(htmlPath, html)
  printHtmlToPdf(htmlPath, pdfPath)

  return { htmlPath, pdfPath, sourceFileCount: files.length, excerptLineCount: excerptLines.length }
}

function generateDocumentMaterial() {
  const pages = buildDocPages()
  const html = renderDocumentHtml(pages)
  const htmlPath = path.join(OUTPUT_DIR, 'document-identification-material.html')
  const pdfPath = path.join(OUTPUT_DIR, '文档鉴别材料.pdf')
  writeFile(htmlPath, html)
  printHtmlToPdf(htmlPath, pdfPath)

  return { htmlPath, pdfPath, pageCount: pages.length }
}

function writeManifest(programResult, documentResult) {
  const manifest = [
    `权利人：${RIGHTS_HOLDER}`,
    `软件全称：${SOFTWARE_NAME}`,
    `软件简称：${SOFTWARE_SHORT_NAME}`,
    `版本号：${VERSION}`,
    '',
    `程序鉴别材料 HTML：${programResult.htmlPath}`,
    `程序鉴别材料 PDF：${programResult.pdfPath}`,
    `程序摘录行数：${programResult.excerptLineCount}`,
    `程序源文件数量：${programResult.sourceFileCount}`,
    '',
    `文档鉴别材料 HTML：${documentResult.htmlPath}`,
    `文档鉴别材料 PDF：${documentResult.pdfPath}`,
    `文档页数：${documentResult.pageCount}`
  ].join('\n')

  writeFile(path.join(OUTPUT_DIR, 'README.txt'), manifest)
}

function main() {
  ensureDir(OUTPUT_DIR)
  const programResult = generateProgramMaterial()
  const documentResult = generateDocumentMaterial()
  writeManifest(programResult, documentResult)

  console.log(`程序鉴别材料：${programResult.pdfPath}`)
  console.log(`文档鉴别材料：${documentResult.pdfPath}`)
}

main()
