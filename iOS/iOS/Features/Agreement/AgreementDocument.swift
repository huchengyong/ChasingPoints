import Foundation

/// 应用内静态协议正文，逐字迁移自 `uniapp/subPages/agreement/*.vue`。
struct AgreementSection: Equatable {
    let title: String
    let paragraphs: [String]
}

struct AgreementDocument: Equatable {
    let navigationTitle: String
    let title: String
    let updatedAt: String
    let sections: [AgreementSection]
}

enum AgreementContent {
    static let userAgreement = AgreementDocument(
        navigationTitle: "用户协议",
        title: "追分用户服务协议",
        updatedAt: "更新日期：2024年12月1日",
        sections: [
            AgreementSection(title: "一、总则", paragraphs: [
                "欢迎您使用追分！本协议是您与追分之间关于您使用追分服务所订立的协议。请您仔细阅读本协议，您访问或使用追分，即表示您已阅读并同意受本协议的约束。",
            ]),
            AgreementSection(title: "二、服务内容", paragraphs: [
                "1. 追分为用户提供台球比赛记录、数据分析、智能匹配对手等服务。",
                "2. 用户可以通过本应用记录个人比赛数据，查看排行榜，与其他用户进行互动。",
                "3. 我们会不断完善和优化服务内容，为用户提供更好的体验。",
            ]),
            AgreementSection(title: "三、用户账号", paragraphs: [
                "1. 用户在使用本服务前需要注册账号。注册时，用户应当提供真实、准确、完整的信息。",
                "2. 用户应当妥善保管自己的账号和密码，对账号下的所有活动负责。",
                "3. 如发现账号被盗用或其他安全问题，用户应立即通知我们。",
            ]),
            AgreementSection(title: "四、用户行为规范", paragraphs: [
                "1. 用户在使用本服务时，应遵守国家法律法规和社会公德。",
                "2. 用户不得利用本服务从事任何违法违规活动。",
                "3. 用户不得发布虚假信息、恶意攻击其他用户或干扰服务正常运行。",
            ]),
            AgreementSection(title: "五、知识产权", paragraphs: [
                "1. 追分的所有内容，包括但不限于文字、图片、软件、音频、视频等，均受著作权法保护。",
                "2. 未经我们书面许可，用户不得复制、修改、传播或以其他方式使用上述内容。",
            ]),
            AgreementSection(title: "六、免责声明", paragraphs: [
                "1. 对于因不可抗力或非我们过错导致的服务中断，我们不承担责任。",
                "2. 用户因自身原因造成的损失，我们不承担责任。",
            ]),
            AgreementSection(title: "七、协议修改", paragraphs: [
                "我们保留随时修改本协议的权利。修改后的协议将在应用内公布，用户继续使用本服务即表示同意修改后的协议。",
            ]),
            AgreementSection(title: "八、联系我们", paragraphs: [
                "如您对本协议有任何疑问，请通过应用内的反馈渠道联系我们。",
            ]),
        ]
    )

    static let privacyPolicy = AgreementDocument(
        navigationTitle: "隐私政策",
        title: "追分隐私政策",
        updatedAt: "更新日期：2024年12月1日",
        sections: [
            AgreementSection(title: "引言", paragraphs: [
                "追分非常重视用户的隐私保护。本隐私政策旨在向您说明我们如何收集、使用、存储和保护您的个人信息。请您仔细阅读本政策，以便更好地了解我们的隐私保护措施。",
            ]),
            AgreementSection(title: "一、我们收集的信息", paragraphs: [
                "1. 账号信息：当您注册账号时，我们会收集您的手机号码、昵称、头像等基本信息。",
                "2. 比赛数据：您在使用应用时记录的比赛信息，包括比分、对手、时间等。",
                "3. 设备信息：为了提供更好的服务，我们可能会收集您的设备型号、操作系统版本等信息。",
                "4. 位置信息：如您授权，我们可能会收集您的位置信息用于附近球友匹配功能。",
            ]),
            AgreementSection(title: "二、信息的使用", paragraphs: [
                "1. 提供服务：我们使用收集的信息为您提供核心功能服务，如比赛记录、数据分析、对手匹配等。",
                "2. 改进服务：通过分析用户行为数据，我们不断优化产品功能和用户体验。",
                "3. 安全保障：我们使用这些信息来识别和防范安全风险，保护您的账号安全。",
                "4. 通知推送：在您授权的情况下，我们会向您发送服务通知和活动信息。",
            ]),
            AgreementSection(title: "三、信息的存储与保护", paragraphs: [
                "1. 我们采用业界标准的安全技术措施来保护您的个人信息。",
                "2. 您的数据存储在安全的服务器上，并进行加密处理。",
                "3. 我们会定期进行安全审计，确保信息安全。",
                "4. 我们只在必要的期限内保留您的个人信息。",
            ]),
            AgreementSection(title: "四、信息的共享", paragraphs: [
                "1. 未经您的同意，我们不会向第三方共享您的个人信息。",
                "2. 以下情况除外：\n- 获得您的明确授权\n- 法律法规要求\n- 保护公共利益\n- 与关联公司共享（需遵守相同隐私政策）",
            ]),
            AgreementSection(title: "五、您的权利", paragraphs: [
                "1. 您可以随时查看和修改您的个人信息。",
                "2. 您可以选择关闭某些数据收集功能。",
                "3. 您可以申请删除您的账号和相关数据。",
                "4. 您可以撤回之前给予的授权同意。",
            ]),
            AgreementSection(title: "六、未成年人保护", paragraphs: [
                "我们非常重视对未成年人信息的保护。如果您是未成年人，请在监护人的陪同下阅读本政策，并在监护人同意后使用我们的服务。",
            ]),
            AgreementSection(title: "七、政策更新", paragraphs: [
                "我们可能会不时更新本隐私政策。更新后的政策将在应用内公布，并注明更新日期。重大变更时，我们会向您发送通知。",
            ]),
            AgreementSection(title: "八、联系我们", paragraphs: [
                "如您对本隐私政策有任何疑问或建议，请通过应用内的反馈渠道与我们联系。我们将在15个工作日内回复您的请求。",
            ]),
        ]
    )
}
