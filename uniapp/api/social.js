/**
 * 动态广场 API 接口
 */
import { get, post } from '@/utils/request.js'

/**
 * 发布动态
 * @param {Object} data { content, images, post_type, match_id }
 * @returns {Promise}
 */
export const createPost = (data) => {
  return post('/api/social/post', data)
}

/**
 * 获取关注的人的动态
 * @param {Object} params { page, page_size }
 * @returns {Promise}
 */
export const getPostList = (params = {}) => {
  return get('/api/social/feed', params)
}

/**
 * 获取广场公开动态
 * @param {Object} params { page, page_size }
 * @returns {Promise}
 */
export const getPublicPosts = (params = {}) => {
  return get('/api/social/public', params)
}

/**
 * 点赞动态
 * @param {Object} data { post_id }
 * @returns {Promise}
 */
export const likePost = (data) => {
  return post('/api/social/like', data)
}

/**
 * 取消点赞
 * @param {Object} data { post_id }
 * @returns {Promise}
 */
export const unlikePost = (data) => {
  return post('/api/social/unlike', data)
}

/**
 * 发表评论
 * @param {Object} data { post_id, content }
 * @returns {Promise}
 */
export const commentPost = (data) => {
  return post('/api/social/comment', data)
}

/**
 * 删除动态
 * @param {Object} data { post_id }
 * @returns {Promise}
 */
export const deletePost = (data) => {
  return post('/api/social/delete', data)
}

/**
 * 获取动态评论列表
 * @param {Object} params { post_id, page, page_size }
 * @returns {Promise}
 */
export const getPostComments = (params = {}) => {
  return get('/api/social/comments', params)
}

/**
 * 获取我的动态
 * @param {Object} params { page, page_size }
 * @returns {Promise}
 */
export const getMyPosts = (params = {}) => {
  return get('/api/social/mine', params)
}
