export interface ActionLink {
  label: string
  to: string
}

export interface FeatureItem {
  title: string
  description: string
}

export interface FaqItem {
  question: string
  answer: string
}

export const homePageContent = {
  hero: {
    eyebrow: '追分竞技记录',
    title: '每一杆，都值得被记录',
    description:
      '追分是一款面向台球爱好者的竞技记录 App，帮你记录战绩、生成战报、找到旗鼓相当的对手。',
    primaryAction: {
      label: '立即下载',
      to: '/download'
    } satisfies ActionLink,
    secondaryAction: {
      label: '查看功能亮点',
      to: '#features'
    } satisfies ActionLink
  },
  values: [
    {
      title: '记录每一场对局',
      description: '从日常约球到认真较量，把比分、结果和对手信息稳定沉淀下来。'
    },
    {
      title: '生成属于你的战报',
      description: '一场精彩对局结束后，快速整理成适合分享和回看的战报内容。'
    },
    {
      title: '找到更合适的球局',
      description: '把记录和竞技体验串起来，让你更容易找到真正旗鼓相当的对手。'
    }
  ] satisfies FeatureItem[],
  highlights: [
    {
      title: '对局记录',
      description: '比分、对手和结果留得住。'
    },
    {
      title: '战报生成',
      description: '精彩对局结束后快速沉淀分享内容。'
    },
    {
      title: '排名沉淀',
      description: '持续看到自己的竞技状态和战绩变化。'
    },
    {
      title: '球友连接',
      description: '让约球、对局和记录形成完整体验。'
    }
  ] satisfies FeatureItem[],
  scenarios: [
    {
      title: '朋友约球之后，不想让比分只停留在聊天记录里',
      description: '把一次普通对局认真记录下来，日后也能回看。'
    },
    {
      title: '打出一场精彩比赛，想立刻做成能分享的战报',
      description: '追分把记录和表达连在一起，不用再手动整理。'
    },
    {
      title: '想知道自己最近是不是越来越稳',
      description: '把零散胜负汇总成持续可见的战绩变化。'
    }
  ] satisfies FeatureItem[],
  downloadCallout: {
    title: '现在开始，把每一次上桌都变成可被记录的竞技时刻。',
    primaryAction: {
      label: '前往下载页',
      to: '/download'
    } satisfies ActionLink
  },
  faq: [
    {
      question: '追分是什么？',
      answer: '追分是一款面向台球爱好者的竞技记录 App，帮助你记录对局、查看战绩并生成战报。'
    },
    {
      question: '追分适合哪些用户？',
      answer: '如果你平时会和朋友约球、在意对局胜负、想留下自己的竞技成长，追分就很适合你。'
    },
    {
      question: 'iPhone 和 Android 都能下载吗？',
      answer: '可以，官网下载页会分别提供 iOS 与 Android 的正式入口。'
    },
    {
      question: '需要注册才能使用吗？',
      answer: '首期官网会明确说明下载方式和基础使用路径，App 内仍以手机号验证为主。'
    },
    {
      question: '对局记录和战报有什么价值？',
      answer: '它们能帮你回看自己的对局表现，也能把值得纪念的一场球更完整地留下来。'
    }
  ] satisfies FaqItem[]
}
