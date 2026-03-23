export const shouldResolveBusinessResponse = (data = {}) => {
	if (!data || typeof data !== 'object') {
		return false
	}
	return data.code === 0 || !!data.success || typeof data.accepted === 'boolean'
}
