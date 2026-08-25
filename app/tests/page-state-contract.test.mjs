import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'
import { compileScript, compileTemplate, parse } from '@vue/compiler-sfc'
import {
  ASYNC_PAGE_STATUS,
  beginAsyncPageLoad,
  createAsyncPageState,
  getAsyncPageRequest,
  rejectAsyncPageLoad,
  resolveAsyncPageLoad
} from '../utils/async-page-state.js'
import { createRequestError } from '../utils/request-errors.js'

const appRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')

const CORE_DATA_PAGES = [
  'pages/social/index.vue',
  'subPages/notification/index.vue',
  'subPages/social/feed.vue',
  'subPages/social/myPosts.vue',
  'subPages/social/friendList.vue',
  'subPages/social/friendRequests.vue',
  'subPages/social/challenges.vue',
  'subPages/social/addFriend.vue',
  'subPages/social/friendHomepage.vue',
  'subPages/social/pkReport.vue',
  'subPages/user/notification.vue',
  'subPages/tournament/index.vue',
  'subPages/tournament/detail.vue',
  'subPages/tournament/bracket.vue',
  'subPages/season/index.vue',
  'subPages/season/report.vue',
  'subPages/venue/index.vue',
  'subPages/venue/detail.vue',
  'subPages/rules/index.vue',
  'subPages/rules/detail.vue',
  'subPages/rules/glossary.vue'
]

const PAGE_LOADERS = {
  'pages/social/index.vue': 'fetchData',
  'subPages/notification/index.vue': 'loadNotifications',
  'subPages/social/feed.vue': 'loadData',
  'subPages/social/myPosts.vue': 'loadPosts',
  'subPages/social/friendList.vue': 'loadData',
  'subPages/social/friendRequests.vue': 'loadData',
  'subPages/social/challenges.vue': 'fetchList',
  'subPages/social/addFriend.vue': 'doSearch',
  'subPages/social/friendHomepage.vue': 'loadData',
  'subPages/social/pkReport.vue': 'loadData',
  'subPages/user/notification.vue': 'loadNotificationSettings',
  'subPages/tournament/index.vue': 'fetchList',
  'subPages/tournament/detail.vue': 'fetchDetail',
  'subPages/tournament/bracket.vue': 'fetchBracket',
  'subPages/season/index.vue': 'loadSeasonOverview',
  'subPages/season/report.vue': 'fetchReport',
  'subPages/venue/index.vue': 'fetchList',
  'subPages/venue/detail.vue': 'fetchDetail',
  'subPages/rules/index.vue': 'handleSearch',
  'subPages/rules/detail.vue': 'loadContent',
  'subPages/rules/glossary.vue': 'loadGlossary'
}

const startLoad = (state = createAsyncPageState({ data: [] })) => {
  const nextState = beginAsyncPageLoad(state, { emptyData: [] })
  return { state: nextState, request: getAsyncPageRequest(nextState) }
}

test('核心数据页共用的状态契约区分成功空数据与各类失败', () => {
  for (const page of CORE_DATA_PAGES) {
    const emptyLoad = startLoad()
    const emptyState = resolveAsyncPageLoad(emptyLoad.state, emptyLoad.request, { data: [] })
    assert.equal(emptyState.status, ASYNC_PAGE_STATUS.EMPTY, `${page} 的成功空数据必须进入 empty`)

    const readyLoad = startLoad()
    const readyState = resolveAsyncPageLoad(readyLoad.state, readyLoad.request, { data: [{ id: 1 }] })
    assert.equal(readyState.status, ASYNC_PAGE_STATUS.READY, `${page} 的成功数据必须进入 ready`)

    for (const category of ['network', 'server', 'permission', 'not-found', 'rate-limit', 'business']) {
      const failedLoad = startLoad()
      const failedState = rejectAsyncPageLoad(
        failedLoad.state,
        failedLoad.request,
        createRequestError({ message: `${category} failure`, category })
      )
      assert.equal(failedState.status, ASYNC_PAGE_STATUS.ERROR, `${page} 的 ${category} 首次失败不能进入 empty`)
      assert.equal(failedState.error.category, category)
    }

    const refreshLoad = startLoad(readyState)
    const refreshFailedState = rejectAsyncPageLoad(
      refreshLoad.state,
      refreshLoad.request,
      createRequestError({ message: 'server failure', category: 'server' })
    )
    assert.equal(refreshFailedState.status, ASYNC_PAGE_STATUS.READY, `${page} 刷新失败必须保留旧内容`)
    assert.equal(refreshFailedState.refreshError.category, 'server')

    const sessionLoad = startLoad()
    const sessionState = rejectAsyncPageLoad(
      sessionLoad.state,
      sessionLoad.request,
      createRequestError({
        message: 'session invalid',
        category: 'session',
        handled: true
      })
    )
    assert.equal(sessionState.status, ASYNC_PAGE_STATUS.LOADING, `${page} 的全局会话失效不得落入本地 empty`)
    assert.equal(sessionState.error, null)
  }
})

test('核心数据页保留状态机接线并拒绝已知伪空模式', async () => {
  for (const page of CORE_DATA_PAGES) {
    const filename = resolve(appRoot, page)
    const source = await readFile(filename, 'utf8')
    const parsed = parse(source, { filename })

    assert.equal(parsed.errors.length, 0, `${page} 必须是有效 SFC`)
    assert.doesNotThrow(() => compileScript(parsed.descriptor, { id: 'page-state-contract' }), `${page} 的脚本必须可编译`)
    const template = compileTemplate({
      source: parsed.descriptor.template?.content || '',
      filename,
      id: 'page-state-contract'
    })
    assert.equal(template.errors.length, 0, `${page} 的模板必须可编译`)

    assert.match(source, /async-page-state\.js/, `${page} 必须接入异步页面状态 helper`)
    assert.match(source, /rejectAsyncPageLoad/, `${page} 首次读取失败必须进入 error 状态`)
    assert.match(source, /resolveAsyncPageErrorFeedback/, `${page} 必须保留错误类别到页面反馈层`)

    const loader = PAGE_LOADERS[page]
    const loaderStart = source.indexOf(`const ${loader} = async`)
    assert.notEqual(loaderStart, -1, `${page} 必须登记首屏读取函数`)
    const nextTopLevelConst = source.indexOf('\nconst ', loaderStart + 1)
    const loaderSource = source.slice(loaderStart, nextTopLevelConst === -1 ? source.length : nextTopLevelConst)
    assert.match(
      loaderSource,
      /catch\s*\([^)]*\)\s*\{[\s\S]*?rejectAsyncPageLoad/,
      `${page} 的 ${loader} 捕获失败时必须进入状态机，不能只记录日志后渲染空状态`
    )
    assert.doesNotMatch(
      source,
      /\.catch\s*\(\s*\(\s*\)\s*=>\s*\(\s*\{\s*list\s*:\s*\[\s*\]/,
      `${page} 不得把请求失败替换为空列表`
    )
  }
})
