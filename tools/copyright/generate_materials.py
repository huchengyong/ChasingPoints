import os
import sys
from pathlib import Path

ROOT = Path("/Users/wisesearch/Projects/ChasingPoints")
OUTPUT_DIR = ROOT / "docs" / "copyright"
PY_DEPS = ROOT / ".tmp_pydeps"

if str(PY_DEPS) not in sys.path:
    sys.path.insert(0, str(PY_DEPS))

from reportlab.lib.colors import HexColor
from reportlab.lib.pagesizes import A4
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.cidfonts import UnicodeCIDFont
from reportlab.pdfgen import canvas

RIGHTS_HOLDER = "上海点早早互联网络科技有限公司"
SOFTWARE_NAME = "追分台球社交对局软件"
SOFTWARE_SHORT_NAME = "追分"
VERSION = "1.0.0"

PAGE_WIDTH, PAGE_HEIGHT = A4
PROGRAM_LINES_PER_PAGE = 50
PROGRAM_PAGE_COUNT = 60
PROGRAM_EXCERPT_LINES = PROGRAM_LINES_PER_PAGE * 30
DOC_LINES_PER_PAGE = 32

pdfmetrics.registerFont(UnicodeCIDFont("STSong-Light"))


def walk_program_files():
    files = []
    backend_root = ROOT / "backend"
    for dirpath, _, filenames in os.walk(backend_root):
        for filename in filenames:
            if not (filename.endswith(".go") or filename.endswith(".api")):
                continue
            if filename.endswith("_test.go"):
                continue
            files.append(Path(dirpath) / filename)
    return sorted(files, key=lambda item: str(item))


def build_program_lines():
    lines = []
    files = walk_program_files()
    for file_path in files:
        relative_path = file_path.relative_to(ROOT).as_posix()
        lines.append(f"// File: {relative_path}")
        content = file_path.read_text(encoding="utf-8").replace("\r\n", "\n").split("\n")
        lines.extend(line.replace("\t", "    ") for line in content)
    return files, lines


def fit_line(text, font_name, font_size, max_width):
    if pdfmetrics.stringWidth(text, font_name, font_size) <= max_width:
        return text
    suffix = " ..."
    trimmed = text
    while trimmed and pdfmetrics.stringWidth(trimmed + suffix, font_name, font_size) > max_width:
        trimmed = trimmed[:-1]
    return (trimmed + suffix) if trimmed else suffix.strip()


def program_pages():
    files, lines = build_program_lines()
    if len(lines) < PROGRAM_EXCERPT_LINES * 2:
        raise RuntimeError("源程序总行数不足，无法生成前后各30页摘录。")

    excerpt_lines = lines[:PROGRAM_EXCERPT_LINES] + lines[-PROGRAM_EXCERPT_LINES:]
    pages = []
    for index in range(0, len(excerpt_lines), PROGRAM_LINES_PER_PAGE):
        pages.append(excerpt_lines[index:index + PROGRAM_LINES_PER_PAGE])

    if len(pages) != PROGRAM_PAGE_COUNT:
        raise RuntimeError(f"程序鉴别材料页数异常：{len(pages)}")

    return files, excerpt_lines, pages


def chunk_text(text, width=28):
    if not text:
        return [""]
    parts = []
    remaining = text.strip()
    while len(remaining) > width:
        parts.append(remaining[:width])
        remaining = remaining[width:]
    parts.append(remaining)
    return parts


def prefixed_lines(prefix, text):
    return chunk_text(f"{prefix}{text}")


def pad_doc_lines(lines, page_index):
    supplements = [
        "本页内容与当前登记版本功能范围保持一致。",
        "本页说明遵循统一接口、统一鉴权和统一数据模型。",
        "系统在异常场景下提供提示、重试或状态恢复能力。",
        "相关功能的页面展示以配置和接口返回结果为准。",
        "用户端、后台端和官网端共享统一业务目标和命名口径。",
        "所有材料中的软件名称、权利人和版本号均保持一致。",
        "相关数据写入后可在对应页面、列表或统计模块中复用。",
        "配置变更可通过部署环境变量或配置文件进行控制。"
    ]
    page_lines = list(lines)
    fill_index = 0
    while len(page_lines) < DOC_LINES_PER_PAGE:
        page_lines.append(supplements[(page_index + fill_index) % len(supplements)])
        fill_index += 1
    return page_lines[:DOC_LINES_PER_PAGE]


def build_doc_pages():
    topics = [
        {
            "title": "软件概述与编制说明",
            "audience": "面向台球爱好者、赛事运营方、球房管理方和平台管理员。",
            "modules": ["移动端用户应用", "后台管理端", "官网展示端", "统一后端服务"],
            "functions": ["用户登录与身份识别", "对局记录与公开观战", "赛事资讯与赛季榜单", "球房签到与附近检索"],
            "flows": ["统一接口契约由后端 API 文件定义。", "前后端交互通过 HTTP 与 WebSocket 共同完成。", "业务逻辑统一落在后端分层结构中。", "文档内容依据当前代码实现能力编制。"],
            "notes": ["本文档用于软件著作权登记的文档鉴别材料。", "文档按照实际功能、模块和运行方式展开说明。"]
        },
        {
            "title": "目标用户与业务场景",
            "audience": "适用于个人台球用户、俱乐部、球房经营者和运营人员。",
            "modules": ["约战对局", "战绩沉淀", "社交互动", "运营审核"],
            "functions": ["支持用户快速登录和绑定手机号", "支持发起对局、观战和赛后复盘", "支持查看排行榜、赛季数据和挑战记录", "支持后台人员维护资讯和审核内容"],
            "flows": ["软件围绕赛前、赛中、赛后三个阶段设计。", "系统将用户、赛事、球房和内容数据统一管理。", "通过通知与推送增强用户触达和活跃。", "通过运营后台提高平台治理效率。"],
            "notes": ["业务目标是提升台球服务场景的信息化程度。", "软件兼顾普通用户体验和运营管理效率。"]
        },
        {
            "title": "系统总体架构",
            "audience": "供产品、实施、测试和运维人员理解整体结构。",
            "modules": ["UniApp 用户端", "Vue3 后台端", "Nuxt3 官网", "go-zero 后端"],
            "functions": ["前端承担页面展示与交互", "后端提供统一数据接口和实时能力", "后台承担审核与运营", "官网承担品牌展示和下载引导"],
            "flows": ["后端采用 handler、logic、model 分层结构。", "共享依赖通过 ServiceContext 统一注入。", "数据库保存核心业务数据，缓存承担验证码与状态辅助。", "多端共同服务于同一套业务模型。"],
            "notes": ["系统具备良好的模块化与扩展性。", "统一接口与统一命名有利于长期维护。"]
        },
        {
            "title": "运行环境与部署模式",
            "audience": "供实施、部署和维护阶段参考。",
            "modules": ["Linux 服务器", "MySQL", "Redis", "Nginx"],
            "functions": ["后端服务提供 REST 接口和 WebSocket 路由", "移动端安装在终端设备上运行", "后台和官网部署为 Web 站点", "短信和推送通过配置控制启停"],
            "flows": ["Nginx 可作为请求入口和反向代理。", "数据库负责持久化用户、对局和赛事等数据。", "Redis 用于验证码、限额和部分实时辅助状态。", "环境变量和配置文件共同参与部署装配。"],
            "notes": ["系统支持开发环境和生产环境切换。", "部署时需配置域名、数据库和第三方密钥。"]
        },
        {
            "title": "账号登录与注册",
            "audience": "面向移动端普通用户和管理员用户。",
            "modules": ["短信验证码登录", "第三方授权登录", "手机号绑定", "管理员登录"],
            "functions": ["支持验证码登录或注册", "支持 OAuth 登录后补绑手机号", "支持令牌式鉴权", "支持后台管理员独立登录入口"],
            "flows": ["登录接口由认证分组统一提供。", "成功登录后前端保存访问令牌并刷新状态。", "需要补绑手机号时进入补全流程。", "鉴权失败时统一提示并要求重新认证。"],
            "notes": ["登录流程强调便捷性和安全性平衡。", "认证结果会影响各页面访问权限。"]
        },
        {
            "title": "个人资料与隐私设置",
            "audience": "面向需要维护个人资料的终端用户。",
            "modules": ["昵称维护", "头像展示", "隐私设置", "会员状态"],
            "functions": ["支持查看并维护昵称等资料", "支持读取和修改隐私设置", "支持显示会员中心状态", "支持展示个人统计摘要"],
            "flows": ["个人信息由用户模块接口统一返回。", "隐私设置变更后即时写入数据库。", "前端状态存储模块负责本地持久化。", "页面根据后端字段控制展示范围。"],
            "notes": ["个人信息处理遵循最小必要原则。", "用户可根据产品规则控制资料展示范围。"]
        },
        {
            "title": "首页与信息聚合",
            "audience": "面向移动端首页浏览场景。",
            "modules": ["欢迎页", "首页总览", "入口漏斗", "推荐信息"],
            "functions": ["提供核心模块快捷入口", "展示个人段位和近期提示", "引导未登录用户进入登录流程", "聚合展示对局、排行和动态入口"],
            "flows": ["首页逻辑通过统一请求层获取数据。", "未登录用户优先进入欢迎页和登录页。", "已登录用户可直接查看核心数据入口。", "时间与文案格式由工具层统一处理。"],
            "notes": ["首页承担多业务入口聚合作用。", "页面兼顾首屏效率和后续扩展性。"]
        },
        {
            "title": "对局大厅与公开观战",
            "audience": "面向浏览和观战场景的用户。",
            "modules": ["对局大厅", "进行中列表", "公开详情", "观战模式"],
            "functions": ["展示平台正在进行的对局", "支持分页查询公开对局", "支持查看双方信息和比分", "支持第三方视角进入观战模式"],
            "flows": ["公开接口允许未登录访问部分信息。", "对局大厅支持刷新与状态更新。", "对局详情页根据状态渲染局记录和时长。", "服务端统一计算持续时间等派生数据。"],
            "notes": ["公开观战功能可提升平台内容活跃度。", "展示逻辑兼顾实时性和只读安全性。"]
        },
        {
            "title": "对局创建与对手选择",
            "audience": "面向发起新对局的终端用户。",
            "modules": ["创建对局", "选择对手", "设定局制", "确认开局"],
            "functions": ["支持选择玩法和局数", "支持选择注册用户或录入非注册对手", "支持初始化比分和回合状态", "支持创建后进入进行中页面"],
            "flows": ["创建请求经后端对局模块处理。", "前端和后端同时执行必要参数校验。", "创建成功后返回对局标识和初始状态。", "页面根据结果跳转到记录页面。"],
            "notes": ["创建流程强调低输入成本和操作直观。", "系统保留后续扩展更多规则配置的能力。"]
        },
        {
            "title": "对局进行与实时同步",
            "audience": "面向正在记录比分的用户。",
            "modules": ["比分录入", "局次推进", "修订号控制", "长连接同步"],
            "functions": ["支持更新每局得分和胜负", "支持更新整场比分和当前局状态", "支持服务端修订号控制一致性", "支持用户级和对局级通知"],
            "flows": ["前端通过统一 WebSocket 模块维护连接。", "后端提供 match/ws 与 user/ws 两条实时路由。", "收到快照或增量消息后页面即时刷新。", "网络异常时按策略重连并恢复状态。"],
            "notes": ["实时同步是对局模块的关键能力。", "系统优先保证比分和状态一致性。"]
        },
        {
            "title": "对局结果与历史记录",
            "audience": "面向赛后复盘和记录查询场景。",
            "modules": ["结算摘要", "历史列表", "战报分享", "交锋记录"],
            "functions": ["生成最终比分和胜负摘要", "支持查看个人历史对局记录", "支持生成战绩海报和分享数据", "支持按对手和双人维度复盘历史"],
            "flows": ["结果页面根据服务端结算数据展示摘要。", "分享页通过统一数据整形逻辑生成文案和海报内容。", "历史记录支持分页查询和目标对象跳转。", "统计摘要由后端聚合计算产生。"],
            "notes": ["赛后复盘帮助用户沉淀长期数据价值。", "分享能力可增强传播和留存。"]
        },
        {
            "title": "排行榜与统计分析",
            "audience": "面向关注竞技表现和成长轨迹的用户。",
            "modules": ["段位榜单", "趋势统计", "高分记录", "时长分析"],
            "functions": ["展示按排位分排序的榜单", "支持近期表现和分数变化分析", "支持单杆高分和对手强度统计", "支持比赛时长和玩法维度分析"],
            "flows": ["接口由公开、排行和统计模块共同提供。", "服务端根据历史对局进行聚合运算。", "前端按卡片和图表式视图展示结果。", "统计页面支持不同玩法和维度切换。"],
            "notes": ["统计分析提升软件专业度和数据价值。", "展示结果强调易读性和可比较性。"]
        },
        {
            "title": "好友关系与关注体系",
            "audience": "面向社交关系维护场景。",
            "modules": ["好友申请", "好友列表", "黑名单", "关注与粉丝"],
            "functions": ["支持搜索用户并发起好友申请", "支持接受、拒绝和删除好友关系", "支持好友黑名单管理", "支持关注、取关和粉丝列表展示"],
            "flows": ["好友和关注由独立业务分组提供接口。", "关系变更后同步更新通知和页面状态。", "相关入口在主页、动态和好友列表中互通。", "纯逻辑规则可通过测试验证。"],
            "notes": ["社交关系为约战和动态传播提供基础。", "系统对重复申请和非法状态进行约束。"]
        },
        {
            "title": "动态广场与社交互动",
            "audience": "面向内容发布和互动场景。",
            "modules": ["动态广场", "发布动态", "我的动态", "内容审核"],
            "functions": ["支持查看动态列表和广场内容", "支持发布和管理个人动态", "支持展示战报类内容", "支持后台对动态进行审核处理"],
            "flows": ["动态接口负责内容的查询与写入。", "前端页面支持分页加载和刷新。", "后台审核结果会回写内容状态并同步通知用户。", "社交内容与对局结果和好友关系可联动展示。"],
            "notes": ["动态模块强化平台社区属性。", "审核能力保障内容质量和合规性。"]
        },
        {
            "title": "成就、称号与挑战",
            "audience": "面向成长激励和互动邀约场景。",
            "modules": ["成就列表", "称号管理", "挑战发起", "待处理挑战"],
            "functions": ["展示用户成就和详情", "支持称号查看与管理", "支持发送挑战邀约", "支持接受或拒绝待处理挑战"],
            "flows": ["成就模块和挑战模块分别承担不同职责。", "前端通过列表和详情页展示状态变化。", "挑战状态变化后同步写入通知体系。", "称号信息可用于个人资料展示。"],
            "notes": ["激励机制有利于提升长期活跃度。", "所有状态切换都经过服务端校验。"]
        },
        {
            "title": "赛事、赛季与报名签到",
            "audience": "面向赛事参与用户和运营人员。",
            "modules": ["赛事情报", "赛事报名", "签到检录", "赛季报告"],
            "functions": ["展示赛事列表和阶段详情", "支持加入、离开、取消和完赛处理", "支持签到、赛程查看和对阵更新", "支持当前赛季和赛季榜单报告"],
            "flows": ["赛事接口覆盖创建、查询、报名和对阵维护。", "赛事情报采用事件与阶段结构组织。", "赛季模块聚合用户一个周期内的竞技表现。", "后台支持维护资讯、阶段和发布状态。"],
            "notes": ["赛事与赛季模块构成平台组织化竞技能力。", "流程覆盖赛前、赛中与赛后关键环节。"]
        },
        {
            "title": "球房管理与签到定位",
            "audience": "面向球房经营场景和用户到店场景。",
            "modules": ["球房创建", "附近检索", "球房详情", "签到记录"],
            "functions": ["支持提交球房基础信息", "支持查询附近球房与详情", "支持在球房场景中完成签到", "支持后台球房审核和奖励配置"],
            "flows": ["球房业务通过 venue 分组接口处理。", "后端地理编码任务用于辅助地址处理。", "签到记录用于沉淀用户线下行为数据。", "后台审核结果同步影响前端展示状态。"],
            "notes": ["球房模块连接线上社区与线下门店。", "地址处理能力依赖后端任务和配置。"]
        },
        {
            "title": "通知推送与消息提醒",
            "audience": "面向需要及时获知状态变化的用户。",
            "modules": ["通知列表", "未读计数", "全部已读", "推送令牌上报"],
            "functions": ["支持查看通知列表和未读数量", "支持单条已读、全部已读和删除通知", "支持 App 启动时上报推送标识", "支持对局、挑战和审核类提醒"],
            "flows": ["通知接口由通知模块统一提供。", "App 级生命周期负责推送标识注册。", "后端集成推送服务用于消息触达。", "用户操作完成后未读数即时刷新。"],
            "notes": ["通知系统承担跨模块消息触达职责。", "提醒能力同时覆盖站内和推送场景。"]
        },
        {
            "title": "管理后台与运营审核",
            "audience": "面向管理员、审核员和运营人员。",
            "modules": ["管理员登录", "首页统计", "用户管理", "内容审核"],
            "functions": ["后台提供登录和初始化管理员能力", "支持查看仪表盘统计和用户列表", "支持查看对局列表和赛事资讯管理", "支持球房审核、动态审核与奖励配置"],
            "flows": ["后台统一通过请求工具访问后端管理路由。", "未授权访问由路由守卫拦截并跳转登录页。", "业务成功按 code 或 success 字段统一判断。", "后台与移动端共享同一套后端服务。"],
            "notes": ["后台是平台运营和治理的重要入口。", "管理功能强调数据可视化与审核闭环。"]
        },
        {
            "title": "官网、运维与数据安全",
            "audience": "面向访客、运维人员和合规场景。",
            "modules": ["品牌首页", "下载页面", "协议页面", "运行维护"],
            "functions": ["官网展示品牌信息和下载入口", "提供隐私政策、用户协议和联系页", "支持 SEO 基础配置和静态发布", "支持数据库迁移、配置加载和日志排查"],
            "flows": ["官网通过页面路由公开展示内容。", "正式域名和下载地址可通过环境变量注入。", "后端通过配置文件和环境变量完成装配。", "数据库结构由迁移脚本统一维护。"],
            "notes": ["系统运行关注账号安全、数据一致性和配置安全。", "敏感配置通过部署环境进行管理。"]
        }
    ]

    pages = []
    for topic_index, topic in enumerate(topics):
        page_one = [
            f"章节：{topic['title']}",
            f"软件名称：{SOFTWARE_NAME}",
            f"软件简称：{SOFTWARE_SHORT_NAME}",
            f"版本号：{VERSION}",
            f"权利人：{RIGHTS_HOLDER}",
            f"适用对象：{topic['audience']}",
            "",
            *prefixed_lines("组成模块：", "、".join(topic["modules"])),
            "",
            *prefixed_lines("主要功能：", "；".join(topic["functions"])),
            "",
        ]
        for note in topic["notes"]:
            page_one.extend(prefixed_lines("说明：", note))

        page_two = [f"章节：{topic['title']} 功能流程", "一、功能入口说明"]
        for index, module_name in enumerate(topic["modules"], start=1):
            page_two.extend(prefixed_lines(f"{index}. ", f"{module_name} 对应界面或服务节点负责承接本模块操作。"))
        page_two.extend(["", "二、典型业务动作"])
        for index, feature in enumerate(topic["functions"], start=1):
            page_two.extend(prefixed_lines(f"{index}. ", feature))
        page_two.extend(["", "三、处理流程摘要"])
        for index, flow in enumerate(topic["flows"], start=1):
            page_two.extend(prefixed_lines(f"{index}. ", flow))

        page_three = [f"章节：{topic['title']} 运维与注意事项", "一、数据与状态说明"]
        for flow in topic["flows"]:
            page_three.extend(prefixed_lines("状态：", flow))
        page_three.extend(["", "二、操作提示"])
        for note in topic["notes"]:
            page_three.extend(prefixed_lines("提示：", note))
        page_three.extend([
            "",
            "三、质量保障",
            "提示：界面输入、接口参数与权限均进行校验。",
            "提示：异常结果通过提示语、日志或状态回写进行反馈。",
            "提示：前后端交互遵循统一接口契约，减少数据歧义。",
            "提示：配置项可按开发、测试和生产环境分别管理。"
        ])

        pages.append(pad_doc_lines(page_one, topic_index * 3))
        pages.append(pad_doc_lines(page_two, topic_index * 3 + 1))
        pages.append(pad_doc_lines(page_three, topic_index * 3 + 2))

    return pages


def draw_header_footer(pdf, title, page_no, total_pages):
    pdf.setStrokeColor(HexColor("#d1d5db"))
    pdf.setFillColor(HexColor("#374151"))
    pdf.setFont("STSong-Light", 10)
    pdf.drawString(42, PAGE_HEIGHT - 32, title)
    pdf.drawString(210, PAGE_HEIGHT - 32, f"权利人：{RIGHTS_HOLDER}")
    pdf.drawRightString(PAGE_WIDTH - 42, PAGE_HEIGHT - 32, f"版本号：{VERSION}")
    pdf.line(42, PAGE_HEIGHT - 38, PAGE_WIDTH - 42, PAGE_HEIGHT - 38)
    pdf.line(42, 34, PAGE_WIDTH - 42, 34)
    pdf.drawString(42, 20, "软件著作权登记鉴别材料")
    pdf.drawRightString(PAGE_WIDTH - 42, 20, f"第 {page_no} 页 / 共 {total_pages} 页")


def generate_program_pdf():
    _, excerpt_lines, pages = program_pages()
    output_path = OUTPUT_DIR / "程序鉴别材料.pdf"
    pdf = canvas.Canvas(str(output_path), pagesize=A4)
    max_width = PAGE_WIDTH - 84 - 28
    line_height = 14.1

    for page_index, page_lines in enumerate(pages, start=1):
        draw_header_footer(pdf, f"{SOFTWARE_NAME} 程序鉴别材料", page_index, len(pages))
        pdf.setFillColor(HexColor("#6b7280"))
        pdf.setFont("Courier", 8)
        start_y = PAGE_HEIGHT - 58
        base_line_number = (page_index - 1) * PROGRAM_LINES_PER_PAGE

        for offset, line in enumerate(page_lines):
            y = start_y - offset * line_height
            pdf.drawRightString(72, y, str(base_line_number + offset + 1))
            pdf.setFillColor(HexColor("#111827"))
            pdf.setFont("STSong-Light", 8)
            pdf.drawString(82, y, fit_line(line or " ", "STSong-Light", 8, max_width))
            pdf.setFillColor(HexColor("#6b7280"))
            pdf.setFont("Courier", 8)

        pdf.showPage()

    pdf.save()
    return output_path, len(excerpt_lines)


def generate_document_pdf():
    pages = build_doc_pages()
    output_path = OUTPUT_DIR / "文档鉴别材料.pdf"
    pdf = canvas.Canvas(str(output_path), pagesize=A4)
    max_width = PAGE_WIDTH - 84
    line_height = 18.5

    for page_index, page_lines in enumerate(pages, start=1):
        draw_header_footer(pdf, f"{SOFTWARE_NAME} 使用说明书", page_index, len(pages))
        pdf.setFont("STSong-Light", 11)
        pdf.setFillColor(HexColor("#111827"))
        start_y = PAGE_HEIGHT - 62

        for offset, line in enumerate(page_lines):
            y = start_y - offset * line_height
            pdf.drawString(42, y, fit_line(line or " ", "STSong-Light", 11, max_width))

        pdf.showPage()

    pdf.save()
    return output_path, len(pages)


def write_manifest(program_pdf, program_lines, document_pdf, document_pages):
    manifest = OUTPUT_DIR / "README.txt"
    manifest.write_text(
        "\n".join([
            f"权利人：{RIGHTS_HOLDER}",
            f"软件全称：{SOFTWARE_NAME}",
            f"软件简称：{SOFTWARE_SHORT_NAME}",
            f"版本号：{VERSION}",
            "",
            f"程序鉴别材料：{program_pdf}",
            f"程序摘录总行数：{program_lines}",
            f"文档鉴别材料：{document_pdf}",
            f"文档总页数：{document_pages}"
        ]),
        encoding="utf-8",
    )


def main():
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    program_pdf, program_line_count = generate_program_pdf()
    document_pdf, document_page_count = generate_document_pdf()
    write_manifest(program_pdf, program_line_count, document_pdf, document_page_count)
    print(program_pdf)
    print(document_pdf)


if __name__ == "__main__":
    main()
