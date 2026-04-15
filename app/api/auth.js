/**
 * 认证相关 API
 */

import { post } from '@/utils/request.js'

/**
 * 发送短信验证码
 * @param {String} phone 手机号
 * @param {String} scene 场景 login/bind
 * @returns {Promise}
 */
export const sendSms = (phone, scene = 'login') => {
	return post('/api/auth/send-sms', { phone, scene })
}

/**
 * 用户登录（短信验证码登录）
 * @param {Object} data 登录数据
 * @param {String} data.phone 手机号
 * @param {String} data.sms_code 短信验证码
 * @returns {Promise}
 */
export const login = (data) => {
	return post('/api/auth/login', data)
}

/**
 * OAuth 登录
 * @param {Object} data OAuth登录数据
 * @param {String} data.provider 提供商 (weixin|huawei)
 * @param {String} data.nick_name 昵称
 * @param {String} data.avatar_url 头像URL
 * @param {String} data.open_id OpenID
 * @param {String} data.platform 平台 (app-plus)
 * @param {String} data.union_id UnionID (可选)
 * @returns {Promise}
 */
export const loginByOauth = (data) => {
	return post('/api/auth/login-by-oauth', data)
}

/**
 * 刷新登录态
 * @param {String} refreshToken 刷新令牌
 * @returns {Promise}
 */
export const refreshToken = (refreshToken) => {
	return post('/api/auth/refresh-token', { refresh_token: refreshToken })
}

/**
 * 绑定手机号
 * @param {String} phone 手机号
 * @param {String} smsCode 短信验证码
 * @returns {Promise}
 */
export const bindPhone = (phone, smsCode) => {
	return post('/api/auth/bind-phone', { phone, sms_code: smsCode })
}

export default {
	sendSms,
	login,
	loginByOauth,
	refreshToken,
	bindPhone
}
