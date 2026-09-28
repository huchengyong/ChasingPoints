import { chmodSync, existsSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDirectory = dirname(fileURLToPath(import.meta.url))
const appDirectory = resolve(scriptDirectory, '..')

export const REQUIRED_SIGNING_FIELDS = [
	'certpath',
	'keyAlias',
	'keyPassword',
	'profile',
	'signAlg',
	'storeFile',
	'storePassword'
]

const requiredSigningNames = ['default', 'release']

const readJSON = (path) => JSON.parse(readFileSync(path, 'utf8'))

const isPlainObject = (value) => value !== null && typeof value === 'object' && !Array.isArray(value)

export const validateSigningConfigs = (signingConfigs) => {
	if (!isPlainObject(signingConfigs)) {
		throw new Error('本地签名配置必须是 JSON 对象')
	}

	for (const name of requiredSigningNames) {
		const material = signingConfigs[name]
		if (!isPlainObject(material)) {
			throw new Error(`缺少 ${name} 签名配置`)
		}

		for (const field of REQUIRED_SIGNING_FIELDS) {
			if (typeof material[field] !== 'string' || material[field].trim() === '') {
				throw new Error(`${name}.${field} 必须配置`)
			}
		}
	}
}

export const buildHarmonyProfile = (template, signingConfigs) => {
	if (!isPlainObject(template?.app) || !Array.isArray(template.app.products)) {
		throw new Error('Harmony build profile 模板无效')
	}

	validateSigningConfigs(signingConfigs)
	const profile = JSON.parse(JSON.stringify(template))
	profile.app.signingConfigs = requiredSigningNames.map((name) => ({
		name,
		type: 'HarmonyOS',
		material: signingConfigs[name]
	}))
	return profile
}

export const prepareHarmonySigning = ({
	templatePath = resolve(scriptDirectory, 'harmony-build-profile.template.json'),
	signingPath = resolve(appDirectory, '.harmony-signing.local.json'),
	outputPath = resolve(appDirectory, 'harmony-configs/build-profile.json5')
} = {}) => {
	if (!existsSync(signingPath)) {
		throw new Error(`未找到本地签名配置：${signingPath}。请从 scripts/harmony-signing.local.example.json 复制创建。`)
	}

	const profile = buildHarmonyProfile(readJSON(templatePath), readJSON(signingPath))
	writeFileSync(outputPath, `${JSON.stringify(profile, null, 2)}\n`, { mode: 0o600 })
	chmodSync(outputPath, 0o600)
	return outputPath
}

const isDirectExecution = process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isDirectExecution) {
	try {
		const outputPath = prepareHarmonySigning()
		console.log(`已生成 HBuilderX 本地签名配置：${outputPath}`)
	} catch (error) {
		console.error(`生成 HBuilderX 本地签名配置失败：${error.message}`)
		process.exitCode = 1
	}
}
