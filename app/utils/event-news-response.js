export const pickFeaturedEventNewsPayload = (response = {}) => {
  return response && response.event ? response.event : null
}

export const pickEventNewsViewPayload = (response = {}) => {
  if (!response || !response.event_news) return null

  return {
    eventNews: response.event_news,
    tournament: response.tournament || null,
    matches: Array.isArray(response.matches) ? response.matches : []
  }
}
