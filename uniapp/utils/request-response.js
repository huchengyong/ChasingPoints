export const shouldResolveBusinessResponse = (data = {}) => {
	if (!data || typeof data !== 'object') {
		return false
	}
	return data.code === 0 || !!data.success || typeof data.accepted === 'boolean'
}

export const unwrapBusinessResponse = (data = {}) => {
	if (!data || typeof data !== 'object') {
		return data
	}

	return Object.prototype.hasOwnProperty.call(data, 'data') ? data.data : data
}
