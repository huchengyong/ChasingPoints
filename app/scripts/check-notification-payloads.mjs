import fs from 'node:fs'
import path from 'node:path'

const pagePath = path.resolve('subPages/notification/index.vue')
const apiPath = path.resolve('api/notification.js')
const socialPagePath = path.resolve('pages/social/index.vue')
const sendFriendRequestLogicPath = path.resolve('backend/internal/logic/send_friend_request_logic.go')
const finishMatchLogicPath = path.resolve('backend/internal/logic/finish_match_logic.go')

const pageSource = fs.readFileSync(pagePath, 'utf8')
const apiSource = fs.readFileSync(apiPath, 'utf8')
const socialPageSource = fs.readFileSync(socialPagePath, 'utf8')
const sendFriendRequestLogicSource = fs.readFileSync(sendFriendRequestLogicPath, 'utf8')
const finishMatchLogicSource = fs.readFileSync(finishMatchLogicPath, 'utf8')

const assertions = [
	{
		name: '单条已读请求使用 notification_id',
		ok: pageSource.includes('await markAsRead({ notification_id: item.id })')
	},
	{
		name: '删除通知请求使用 notification_id',
		ok: pageSource.includes('await deleteNotification({ notification_id: id })')
	},
	{
		name: '通知 API 注释声明 notification_id',
		ok:
			apiSource.includes('@param {Object} data { notification_id }') &&
			apiSource.includes('@param {Object} data { notification_id }')
	},
	{
		name: '消息中心支持按通知数据跳转对局总结页',
		ok: pageSource.includes("target === '/subPages/match/matchResult' && data.match_id")
	},
	{
		name: '社交页好友请求入口显示角标',
		ok: socialPageSource.includes("v-if=\"friendRequestStore.pendingCount > 0\" class=\"tool-badge\"")
	},
	{
		name: '发送好友申请时创建通知消息',
		ok:
			sendFriendRequestLogicSource.includes('Title:   "收到好友申请"') &&
			sendFriendRequestLogicSource.includes('buildNotificationPayload("/subPages/social/friendRequests", 0, 0)')
	},
	{
		name: '对局结束通知写入对局总结页跳转数据',
		ok: finishMatchLogicSource.includes('buildNotificationPayload("/subPages/match/matchResult", match.Id, 0)')
	}
]

const failed = assertions.filter(item => !item.ok)

if (failed.length > 0) {
	console.error('notification payload checks failed:')
	failed.forEach(item => {
		console.error(`- ${item.name}`)
	})
	process.exit(1)
}

console.log('notification payload checks passed')
