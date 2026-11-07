function trim(value) {
	return typeof value === 'string' ? value.trim() : ''
}

export function resolveHuaweiCredential(loginResult = {}) {
	const authResult = loginResult?.authResult || {}
	const candidates = [
		{ credential: trim(loginResult.code), credentialType: 'authorization_code' },
		{ credential: trim(loginResult.authCode), credentialType: 'authorization_code' },
		{ credential: trim(authResult.authorizationCode), credentialType: 'authorization_code' },
		{ credential: trim(authResult.authorization_code), credentialType: 'authorization_code' },
		{ credential: trim(authResult.code), credentialType: 'authorization_code' },
		{ credential: trim(authResult.idToken), credentialType: 'id_token' },
		{ credential: trim(authResult.id_token), credentialType: 'id_token' },
		{ credential: trim(authResult.accessToken), credentialType: 'access_token' },
		{ credential: trim(authResult.access_token), credentialType: 'access_token' }
	]
	return candidates.find((item) => item.credential) || { credential: '', credentialType: '' }
}

export function buildHuaweiOAuthLoginPayload({ loginResult, userInfo, platform }) {
	const { credential, credentialType } = resolveHuaweiCredential(loginResult)
	if (!credential) {
		throw new Error('华为授权凭据缺失，请重试')
	}
	return {
		provider: 'huawei',
		credential,
		credential_type: credentialType,
		platform: trim(platform) || 'app-plus',
		nick_name: userInfo?.userInfo?.nickName || '',
		avatar_url: userInfo?.userInfo?.avatarUrl || ''
	}
}

export function buildHuaweiOAuthReauthPayload({ loginResult, platform }) {
	const { credential, credentialType } = resolveHuaweiCredential(loginResult)
	if (!credential) {
		throw new Error('华为授权凭据缺失，请重试')
	}
	return {
		provider: 'huawei',
		credential,
		credential_type: credentialType,
		platform: trim(platform) || 'app-plus'
	}
}
