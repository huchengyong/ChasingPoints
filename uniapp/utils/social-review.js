export const resolveMyPostReviewMeta = (status, rejectReason = '') => {
  if (status === 2) {
    return {
      tagText: '审核中，仅自己可见',
      tagType: 'pending',
      reasonVisible: false,
      reasonText: ''
    }
  }

  if (status === 3) {
    return {
      tagText: '审核未通过',
      tagType: 'rejected',
      reasonVisible: Boolean(rejectReason),
      reasonText: rejectReason || ''
    }
  }

  return {
    tagText: '已发布',
    tagType: 'published',
    reasonVisible: false,
    reasonText: ''
  }
}

export const buildPostReviewSuccessCopy = () => ({
  toast: '已提交审核',
  hint: '可在“我的动态”查看进度'
})
