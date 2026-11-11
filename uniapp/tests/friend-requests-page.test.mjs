import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const friendRequestsSource = readFileSync(
  new URL('../subPages/social/friendRequests.vue', import.meta.url),
  'utf8'
)

test('friend requests page uses a cardified request layout with separated header and action rows', () => {
  assert.match(friendRequestsSource, /class="request-card-main"/)
  assert.match(friendRequestsSource, /class="request-head"/)
  assert.match(friendRequestsSource, /class="request-meta"/)
  assert.match(friendRequestsSource, /class="request-actions"/)
  assert.match(friendRequestsSource, /class="action-button action-accept"/)
  assert.match(friendRequestsSource, /class="action-button action-reject"/)
})

test('friend requests cards can reject and blacklist the applicant from a more action', () => {
  assert.match(friendRequestsSource, /blacklistFriend/)
  assert.match(friendRequestsSource, /class="request-more"/)
  assert.match(friendRequestsSource, /uni\.showActionSheet/)
  assert.match(friendRequestsSource, /itemList: \['拒绝并加入黑名单'\]/)
  assert.match(friendRequestsSource, /resolveFriendRequestUserId/)
  assert.match(friendRequestsSource, /已加入黑名单/)
})

test('friend requests actions are guarded per card while one is in flight', () => {
  assert.match(friendRequestsSource, /busyRequestIds/)
  assert.match(friendRequestsSource, /isItemBusy/)
  assert.match(friendRequestsSource, /setItemBusy\(item\.id, true\)/)
  assert.match(friendRequestsSource, /:disabled="isItemBusy\(item\)"/)
})

test('friend requests page keeps request time on one line and gives actions a polished button system', () => {
  assert.match(friendRequestsSource, /\.request-time\s*\{[\s\S]*white-space:\s*nowrap;/)
  assert.match(friendRequestsSource, /\.request-item\s*\{[\s\S]*border:\s*1rpx solid/)
  assert.match(friendRequestsSource, /\.request-item\s*\{[\s\S]*box-shadow:/)
  assert.match(friendRequestsSource, /\.action-button\s*\{[\s\S]*height:\s*64rpx;/)
  assert.match(friendRequestsSource, /\.action-button\s*\{[\s\S]*font-weight:\s*600;/)
  assert.match(friendRequestsSource, /\.action-accept\s*\{[\s\S]*color:\s*#ffffff;/)
})
