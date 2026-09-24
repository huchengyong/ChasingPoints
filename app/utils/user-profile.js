const DEFAULT_AVATAR_DIRECTORY = '/static/images/default-avatars'

export const DEFAULT_USER_AVATAR = `${DEFAULT_AVATAR_DIRECTORY}/ball-01.png`

export function getDefaultAvatarPath(userId) {
	const normalizedUserId = Number(userId)
	if (!Number.isInteger(normalizedUserId) || normalizedUserId <= 0) {
		return DEFAULT_USER_AVATAR
	}

	const ballNumber = ((normalizedUserId - 1) % 15) + 1
	return `${DEFAULT_AVATAR_DIRECTORY}/ball-${String(ballNumber).padStart(2, '0')}.png`
}

export function resolveAvatarUrl(avatar, userId) {
	const normalizedAvatar = String(avatar || '').trim()
	return normalizedAvatar || getDefaultAvatarPath(userId)
}
