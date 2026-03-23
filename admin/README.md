# 球艺堂管理后台 (mini 版)

基于 Vue3 + Vite + TypeScript + Element Plus 的轻量级管理后台模板。

## 特性

- ⚡️ [Vue 3](https://vuejs.org/) - 渐进式 JavaScript 框架
- 🚀 [Vite](https://vitejs.dev/) - 下一代前端工具链
- 🔧 [TypeScript](https://www.typescriptlang.org/) - 类型安全
- 🎨 [Element Plus](https://element-plus.org/) - 桌面端组件库
- 📦 [Pinia](https://pinia.vuejs.org/) - 状态管理
- 🛣️ [Vue Router](https://router.vuejs.org/) - 路由管理
- 🔌 自动导入 - unplugin-auto-import / unplugin-vue-components

## 目录结构

```
admin/
├── public/                 # 静态资源
├── src/
│   ├── api/               # API 接口
│   ├── assets/            # 静态资源
│   ├── components/        # 公共组件
│   ├── hooks/             # 组合式函数
│   ├── layout/            # 布局组件
│   ├── router/            # 路由配置
│   ├── store/             # 状态管理
│   ├── styles/            # 全局样式
│   ├── utils/             # 工具函数
│   ├── views/             # 页面视图
│   ├── App.vue            # 根组件
│   └── main.ts            # 入口文件
├── types/                 # 类型定义
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## 快速开始

```bash
# 进入目录
cd admin

# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产环境
npm run build
```

## 首次初始化管理员

首次部署后，请在后端环境变量中配置 `ADMIN_SETUP_TOKEN`，然后在登录页点击“首次初始化管理员”，手动输入：

- 管理员邮箱
- 管理员密码
- `ADMIN_SETUP_TOKEN`

初始化成功后，再使用刚创建的管理员账号登录。源码中不再内置默认管理员账号密码。

## 功能模块

- ✅ 登录页
- ✅ 首页统计
- ✅ 用户管理
- ✅ 对局管理
- ✅ 球房管理

## 与后端对接

接口基地址通过环境变量控制：

- 开发环境：`admin/.env.development`
- 生产环境：`admin/.env.production`

如需修改后端地址，直接更新对应环境文件里的 `VITE_API_BASE_URL`。
