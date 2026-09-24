export const SNOOKER_BEST_OF_OPTIONS = [1, 3, 5, 7, 9]

export const buildSnookerBestOfLabels = () => SNOOKER_BEST_OF_OPTIONS.map(value => (
  `${value} 局 ${value === 1 ? '单局决胜' : `（抢 ${Math.floor(value / 2) + 1}）`}`
))

const selectActionSheetIndex = (uniApi, itemList) => new Promise((resolve) => {
  uniApi.showActionSheet({
    itemList,
    success: ({ tapIndex }) => resolve(tapIndex),
    fail: () => resolve(-1)
  })
})

export const chooseSnookerStartFormat = async (uniApi) => {
  if (!uniApi?.showActionSheet) return null
  const bestOfIndex = await selectActionSheetIndex(uniApi, buildSnookerBestOfLabels())
  if (bestOfIndex < 0) return null
  const startingActorIndex = await selectActionSheetIndex(uniApi, ['选手1（发起方）开球', '选手2（对手）开球'])
  if (startingActorIndex < 0) return null
  return {
    best_of_frames: SNOOKER_BEST_OF_OPTIONS[bestOfIndex],
    starting_actor: startingActorIndex === 1 ? 2 : 1
  }
}
