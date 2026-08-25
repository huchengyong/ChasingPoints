import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { basename } from 'node:path'

const manifestSource = readFileSync(new URL('../manifest.json', import.meta.url), 'utf8')
const manifest = JSON.parse(manifestSource.replace(/\/\*[\s\S]*?\*\//g, ''))

const readPngDimensions = (relativePath) => {
	const file = new URL(`../${relativePath}`, import.meta.url)
	const source = readFileSync(file)
	assert.deepEqual([...source.subarray(0, 8)], [137, 80, 78, 71, 13, 10, 26, 10], `${relativePath} 不是 PNG`)
	return {
		width: source.readUInt32BE(16),
		height: source.readUInt32BE(20)
	}
}

const collectStringValues = (value) => {
	if (typeof value === 'string') return [value]
	if (!value || typeof value !== 'object') return []
	return Object.values(value).flatMap(collectStringValues)
}

test('manifest should not contain user-home absolute paths', () => {
	assert.doesNotMatch(manifestSource, /\/Users\//, '发现 /Users/ 绝对路径')
})

test('manifest should not expose signing secrets or certificate material inline', () => {
	assert.doesNotMatch(manifestSource, /\"(?:cert|profile|store|key)Path\"\s*:\s*\"/, '发现 signing 绝对路径')
	assert.doesNotMatch(manifestSource, /\"(?:keyPassword|storePassword)\"\s*:\s*\"/, '发现签名密码')
	assert.doesNotMatch(manifestSource, /\"keyAlias\"\s*:\s*\"/, '发现 signing Key Alias')
})

test('manifest should not contain forbidden Android permissions', () => {
	const permissionsSource = manifest['app-plus']?.distribute?.android?.permissions || []
	const permissionNames = permissionsSource.map((permissionEntry) => {
		const m = String(permissionEntry).match(/android\.permission\.([A-Za-z0-9_]+)/)
		return m?.[1] ? `android.permission.${m[1]}` : String(permissionEntry)
	})
	const forbidden = [
		'android.permission.READ_LOGS',
		'android.permission.GET_ACCOUNTS',
		'android.permission.READ_PHONE_STATE',
		'android.permission.WRITE_SETTINGS',
		'android.permission.MOUNT_UNMOUNT_FILESYSTEMS',
		'android.permission.WRITE_SETTINGS'
	]

	for (const deniedPermission of forbidden) {
		assert.equal(permissionNames.includes(deniedPermission), false, `检测到未授权高风险权限：${deniedPermission}`)
	}
})

test('manifest module declarations should stay within approved set', () => {
	const appPlusModules = Object.keys(manifest['app-plus']?.modules || {})
	const approvedAppPlusModules = ['OAuth', 'Geolocation', 'Barcode', 'Push', 'Share', 'Payment']
	for (const moduleName of appPlusModules) {
		assert.ok(
			approvedAppPlusModules.includes(moduleName),
			`未在白名单中的 app-plus 模块：${moduleName}`
		)
	}

	const harmonyModules = Object.keys(manifest['app-harmony']?.distribute?.modules || {})
	const approvedHarmonyModules = ['uni-oauth', 'uni-payment', 'uni-verify']
	for (const moduleName of harmonyModules) {
		assert.ok(
			approvedHarmonyModules.includes(moduleName),
			`未在白名单中的 harmony 模块：${moduleName}`
		)
	}
})

test('manifest should keep Harmony resources relative', () => {
	const { icons, splashScreens } = manifest['app-harmony']?.distribute || {}
	assert.doesNotMatch(icons.foreground || '', /^\/Users\//, 'harmony icon foreground 为绝对路径')
	assert.doesNotMatch(icons.background || '', /^\/Users\//, 'harmony icon background 为绝对路径')
	assert.doesNotMatch(splashScreens?.startWindowIcon || '', /^\/Users\//, 'harmony 启动图标为绝对路径')
})

test('manifest app icons should be versioned PNG build inputs', () => {
	const iconPaths = [...new Set(collectStringValues(manifest['app-plus']?.distribute?.icons))]
	assert.ok(iconPaths.length > 0, '未配置 App 图标')

	for (const relativePath of iconPaths) {
		assert.equal(existsSync(new URL(`../${relativePath}`, import.meta.url)), true, `缺少图标：${relativePath}`)
		const expectedSize = Number(basename(relativePath).split('x')[0])
		const { width, height } = readPngDimensions(relativePath)
		assert.equal(width, expectedSize, `${relativePath} 宽度不匹配`)
		assert.equal(height, expectedSize, `${relativePath} 高度不匹配`)
	}
})

test('Harmony native icon and splash resources should be complete', () => {
	const resourceRoot = 'harmony-configs/entry/src/main/resources/base'
	const layeredImage = JSON.parse(readFileSync(new URL(`../${resourceRoot}/media/layered_image.json`, import.meta.url), 'utf8'))
	const colors = JSON.parse(readFileSync(new URL(`../${resourceRoot}/element/color.json`, import.meta.url), 'utf8'))

	assert.deepEqual(layeredImage['layered-image'], {
		background: '$media:icon_background',
		foreground: '$media:icon_foreground'
	})
	assert.ok(colors.color.some((color) => color.name === 'start_window_background'), '缺少 Harmony 启动页背景色')

	for (const relativePath of [
		`${resourceRoot}/media/icon_background.png`,
		`${resourceRoot}/media/icon_foreground.png`,
		`${resourceRoot}/media/startIcon.png`
	]) {
		assert.equal(existsSync(new URL(`../${relativePath}`, import.meta.url)), true, `缺少 Harmony 资源：${relativePath}`)
		const { width, height } = readPngDimensions(relativePath)
		assert.equal(width, 192, `${relativePath} 宽度不匹配`)
		assert.equal(height, 192, `${relativePath} 高度不匹配`)
	}
})
