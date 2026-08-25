import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const manifestSource = readFileSync(new URL('../manifest.json', import.meta.url), 'utf8')
const manifest = JSON.parse(manifestSource.replace(/\/\*[\s\S]*?\*\//g, ''))

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
		const m = String(permissionEntry).match(/android\\.permission\\.([A-Za-z0-9_]+)/)
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
