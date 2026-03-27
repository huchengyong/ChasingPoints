export const pickFeaturedEventNewsPayload = (response = {}) => {
  return response && response.event ? response.event : null
}

export const pickEventNewsDetailPayload = (response = {}) => {
  return response && response.event ? response.event : null
}
