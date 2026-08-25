import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { tmpdir } from 'node:os'
import {
	buildHarmonyProfile,
	prepareHarmonySigning,
	REQUIRED_SIGNING_FIELDS,
	validateSigningConfigs
} from '../scripts/prepare-harmony-signing.mjs'

const template = JSON.parse(readFileSync(new URL('../scripts/harmony-build-profile.template.json', import.meta.url), 'utf8'))
const example = JSON.parse(readFileSync(new URL('../scripts/harmony-signing.local.example.json', import.meta.url), 'utf8'))
const gitignore = readFileSync(new URL('../../.gitignore', import.meta.url), 'utf8')

const completeMaterial = (name) => ({
	certpath: `/private/${name}.cer`,
	keyAlias: `${name}-alias`,
	keyPassword: `${name}-key-password`,
	profile: `/private/${name}.p7b`,
	signAlg: 'SHA256withECDSA',
	storeFile: `/private/${name}.p12`,
	storePassword: `${name}-store-password`
})

test('HBuilderX local signing inputs and generated build profile stay outside Git', () => {
	assert.match(gitignore, /^app\/\.harmony-signing\.local\.json$/m)
	assert.match(gitignore, /^app\/harmony-configs\/build-profile\.json5$/m)
	assert.ok(Array.isArray(template.app.products))
	assert.deepEqual(template.app.signingConfigs, [])
	assert.equal(template.app.products.some(({ name, signingConfig }) => name === 'default' && signingConfig === 'default'), true)
	assert.equal(template.app.products.some(({ name, signingConfig }) => name === 'release' && signingConfig === 'release'), true)
})

test('local signing example contains no signing material', () => {
	for (const name of ['default', 'release']) {
		for (const field of REQUIRED_SIGNING_FIELDS) {
			if (field === 'signAlg') continue
			assert.equal(example[name][field], '')
		}
	}
})

test('build profile carries separate debug and release signing material', () => {
	const signingConfigs = {
		default: completeMaterial('debug'),
		release: completeMaterial('release')
	}
	const profile = buildHarmonyProfile(template, signingConfigs)

	assert.deepEqual(profile.app.signingConfigs, [
		{ name: 'default', type: 'HarmonyOS', material: signingConfigs.default },
		{ name: 'release', type: 'HarmonyOS', material: signingConfigs.release }
	])
	assert.deepEqual(template.app.signingConfigs, [])
})

test('incomplete local signing config fails before HBuilderX runs', () => {
	const signingConfigs = {
		default: completeMaterial('debug'),
		release: { ...completeMaterial('release'), storePassword: '' }
	}

	assert.throws(() => validateSigningConfigs(signingConfigs), /release\.storePassword 必须配置/)
})

test('generator writes the ignored HBuilderX build profile with owner-only permissions', () => {
	const directory = mkdtempSync(join(tmpdir(), 'chasing-points-harmony-signing-'))
	const templatePath = join(directory, 'template.json')
	const signingPath = join(directory, 'signing.json')
	const outputPath = join(directory, 'build-profile.json5')
	const signingConfigs = {
		default: completeMaterial('debug'),
		release: completeMaterial('release')
	}

	try {
		writeFileSync(templatePath, JSON.stringify(template))
		writeFileSync(signingPath, JSON.stringify(signingConfigs))
		assert.equal(prepareHarmonySigning({ templatePath, signingPath, outputPath }), outputPath)
		assert.deepEqual(JSON.parse(readFileSync(outputPath, 'utf8')).app.signingConfigs, [
			{ name: 'default', type: 'HarmonyOS', material: signingConfigs.default },
			{ name: 'release', type: 'HarmonyOS', material: signingConfigs.release }
		])
		assert.equal(statSync(outputPath).mode & 0o777, 0o600)
	} finally {
		rmSync(directory, { recursive: true, force: true })
	}
})
