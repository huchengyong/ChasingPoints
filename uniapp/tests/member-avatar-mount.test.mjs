import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { compileScript, parse } from '@vue/compiler-sfc'
import { JSDOM } from 'jsdom'

const appDir = dirname(dirname(fileURLToPath(import.meta.url)))
const componentPath = join(appDir, 'components/memberAvatar.vue')
const realUtil = (name) => pathToFileURL(join(appDir, 'utils', name)).href

// 组件只依赖两个纯函数模块，直接用真实实现，避免复写业务规则。
const moduleImports = {
  '@/utils/member-identity.js': realUtil('member-identity.js'),
  '@/utils/user-profile.js': realUtil('user-profile.js')
}

async function mountComponent(props) {
  const dom = new JSDOM('<!doctype html><html><body></body></html>', { url: 'http://localhost/' })
  const previous = installDOMGlobals(dom.window)
  const tempDir = await mkdtemp(join(appDir, '.member-avatar-mount-'))
  let wrapper
  try {
    const source = await readFile(componentPath, 'utf8')
    const { descriptor, errors } = parse(source, { filename: componentPath })
    assert.deepEqual(errors, [])
    const compiled = compileScript(descriptor, {
      id: 'member-avatar-mount',
      inlineTemplate: true,
      templateOptions: {
        compilerOptions: { isCustomElement: (tag) => ['view', 'text', 'image'].includes(tag) }
      }
    })
    let moduleSource = compiled.content
    for (const [specifier, target] of Object.entries(moduleImports)) {
      moduleSource = moduleSource.replaceAll(`from '${specifier}'`, `from '${target}'`)
    }
    await writeFile(join(tempDir, 'MemberAvatar.mjs'), moduleSource)

    const [{ mount }, module] = await Promise.all([
      import('@vue/test-utils'),
      import(`${pathToFileURL(join(tempDir, 'MemberAvatar.mjs')).href}?v=${Date.now()}`)
    ])
    wrapper = mount(module.default, { props, attachTo: document.body })
    const root = wrapper.element
    return {
      root,
      core: root.querySelector('.member-avatar-core'),
      image: root.querySelector('.member-avatar-image'),
      frame: root.querySelector('.member-avatar-frame'),
      badge: root.querySelector('.member-avatar-badge')
    }
  } finally {
    wrapper?.unmount()
    await rm(tempDir, { recursive: true, force: true })
    restoreDOMGlobals(previous)
    dom.window.close()
  }
}

test('non-member renders only the avatar, even when a rank is known', async () => {
  const { image, frame, badge } = await mountComponent({ avatar: 'https://img.example/a.png', userId: 7, vipLevel: 0, rankLevel: 5 })
  assert.equal(image.getAttribute('src'), 'https://img.example/a.png')
  assert.equal(frame, null)
  assert.equal(badge, null)
})

test('missing member fields are treated as non-member and fall back to the default avatar', async () => {
  const { image, frame, badge } = await mountComponent({ avatar: '', userId: 8 })
  assert.equal(image.getAttribute('src'), '/static/images/default-avatars/ball-08.png')
  assert.equal(frame, null)
  assert.equal(badge, null)
})

test('member shows the rank frame and the tier badge', async () => {
  const { frame, badge } = await mountComponent({ avatar: 'a.png', userId: 7, vipLevel: 3, rankLevel: 4 })
  assert.equal(frame.getAttribute('src'), 'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_platinum.png')
  assert.equal(badge.textContent.trim(), 'VIP3')
  assert.ok(badge.classList.contains('member-avatar-badge--advanced'))
})

test('each growth level uses the badge style of its tier', async () => {
  const expected = { 1: 'standard', 2: 'standard', 3: 'advanced', 4: 'advanced', 5: 'premium' }
  for (const [level, tier] of Object.entries(expected)) {
    const { badge } = await mountComponent({ avatar: 'a.png', vipLevel: Number(level) })
    assert.ok(badge.classList.contains(`member-avatar-badge--${tier}`), `VIP${level} -> ${tier}`)
    assert.equal(badge.textContent.trim(), `VIP${level}`)
  }
})

test('a member without a known rank keeps the badge but has no frame', async () => {
  const { frame, badge } = await mountComponent({ avatar: 'a.png', vipLevel: 2, rankLevel: 0 })
  assert.equal(frame, null)
  assert.equal(badge.textContent.trim(), 'VIP2')
})

test('showBadge=false hides only the badge and badgeCorner moves it to the corner', async () => {
  const hidden = await mountComponent({ avatar: 'a.png', vipLevel: 3, rankLevel: 4, showBadge: false })
  assert.equal(hidden.badge, null)
  assert.notEqual(hidden.frame, null)

  const corner = await mountComponent({ avatar: 'a.png', vipLevel: 3, badgeCorner: true })
  assert.ok(corner.badge.classList.contains('member-avatar-badge--corner'))
})

test('ring color only applies to avatars without a frame', async () => {
  const plain = await mountComponent({ avatar: 'a.png', vipLevel: 0, rankLevel: 4, ringColor: '#fbbf24' })
  assert.match(plain.image.getAttribute('style') || '', /border/)
  const free = await mountComponent({ avatar: 'a.png', vipLevel: 0, ringColor: '' })
  assert.doesNotMatch(free.image.getAttribute('style') || '', /border/)
  // 会员的头像框本身就是环，不再叠加色环。
  const member = await mountComponent({ avatar: 'a.png', vipLevel: 3, rankLevel: 4, ringColor: '#fbbf24' })
  assert.doesNotMatch(member.image.getAttribute('style') || '', /border/)
})

test('frame and badge are drawn over the avatar inside the avatar-sized core, not in the layout box', async () => {
  // jsdom 不认 rpx，尺寸换算由 resolveMemberAvatarMetrics 的单测覆盖；这里只确认层级结构。
  const { root, core, image, frame, badge } = await mountComponent({ avatar: 'a.png', vipLevel: 3, rankLevel: 4, size: 96, reserveFrame: true })
  assert.deepEqual([...root.children], [core])
  assert.deepEqual([...core.children], [image, frame, badge])
})

function installDOMGlobals(window) {
  const keys = ['window', 'document', 'navigator', 'Node', 'Element', 'HTMLElement', 'SVGElement', 'Event']
  const previous = new Map()
  for (const key of keys) {
    previous.set(key, Object.getOwnPropertyDescriptor(globalThis, key))
    Object.defineProperty(globalThis, key, { configurable: true, writable: true, value: window[key] })
  }
  return previous
}

function restoreDOMGlobals(previous) {
  for (const [key, descriptor] of previous) {
    if (descriptor) {
      Object.defineProperty(globalThis, key, descriptor)
    } else {
      delete globalThis[key]
    }
  }
}
