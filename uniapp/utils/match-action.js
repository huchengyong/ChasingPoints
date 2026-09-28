let actionSequence = 0

export const createMatchActionId = () => {
  actionSequence = (actionSequence + 1) % 1000000
  const timePart = Date.now().toString(36)
  const sequencePart = actionSequence.toString(36).padStart(4, '0')
  const randomPart = Math.random().toString(36).slice(2, 8)
  return `ma_${timePart}_${sequencePart}_${randomPart}`
}

export const buildMatchActionPayload = (payload = {}, baseRevision = 0, createActionId = createMatchActionId) => ({
  ...payload,
  client_action_id: createActionId(),
  base_revision: Number.isFinite(Number(baseRevision)) ? Number(baseRevision) : 0
})
