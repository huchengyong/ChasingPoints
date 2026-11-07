import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')

// 取出页面里全部 <memberAvatar ... /> 标签，按标签断言绑定的会员字段。
const memberAvatarTags = (source) => source.match(/<memberAvatar[\s\S]*?\/>/g) || []

const assertTags = (path, expected) => {
  const source = read(path)
  const tags = memberAvatarTags(source)
  assert.equal(tags.length, expected.length, `${path} should render ${expected.length} memberAvatar tags`)
  expected.forEach((rule, index) => {
    for (const pattern of rule) {
      assert.match(tags[index], pattern, `${path} tag #${index + 1} must match ${pattern}`)
    }
  })
  assert.match(source, /import memberAvatar from '@\/components\/memberAvatar\.vue'/, `${path} must import the component`)
}

test('ranking page shows member identity on the podium, the list rows and my ranking card', () => {
  const source = read('pages/ranking/index.vue')
  assertTags('pages/ranking/index.vue', [
    [/podiumSlots\[0\]\.user\.vip_level/, /podiumSlots\[0\]\.user\.rank_level/, /fit-frame/, /ring-color="#9A8C67"/, /badge-corner/],
    [/podiumSlots\[1\]\.user\.vip_level/, /podiumSlots\[1\]\.user\.rank_level/, /fit-frame/, /ring-color="#fbbf24"/, /badge-corner/],
    [/podiumSlots\[2\]\.user\.vip_level/, /podiumSlots\[2\]\.user\.rank_level/, /fit-frame/, /ring-color="#a16207"/, /badge-corner/],
    [/item\.vip_level/, /item\.rank_level/, /reserve-frame/],
    [/myRanking\.vip_level/, /myRanking\.rank_level/, /reserve-frame/]
  ])
  // 已被组件取代的直接头像渲染不能残留。
  assert.doesNotMatch(source, /<image class="(avatar|item-avatar|my-avatar)"/)
})

test('home leaderboard summary shows member identity for the top three', () => {
  assertTags('pages/index/index.vue', [
    [/item\.vip_level/, /item\.rank_level/, /reserve-frame/]
  ])
})

test('friend list and friend homepage show member identity', () => {
  assertTags('subPages/social/friendList.vue', [
    [/item\.vip_level/, /item\.rank_level/, /reserve-frame/]
  ])
  const homepage = read('subPages/social/friendHomepage.vue')
  assertTags('subPages/social/friendHomepage.vue', [
    [/friendProfile\.vipLevel/, /friendProfile\.rankLevel/, /reserve-frame/]
  ])
  // 主页的会员字段来自好友列表透传的 query，而不是再请求一次。
  assert.match(homepage, /friendProfile\.vipLevel = normalized\.vipLevel/)
  assert.match(homepage, /friendProfile\.rankLevel = normalized\.rankLevel/)
})

test('match detail, result and playing pages show both sides with their own identity', () => {
  assertTags('subPages/match/matchDetail.vue', [
    [/player1Info\.vipLevel/, /player1Info\.rankLevel/],
    [/player2Info\.vipLevel/, /player2Info\.rankLevel/]
  ])
  assertTags('subPages/match/matchResult.vue', [
    [/matchData\.my_vip_level/, /matchData\.my_rank_level/],
    [/matchData\.opponent_vip_level/, /matchData\.opponent_rank_level/]
  ])
  assertTags('subPages/match/playing.vue', [
    [/myInfo\.vipLevel/, /myInfo\.rankLevel/, /reserve-frame/],
    [/opponentInfo\.vipLevel/, /opponentInfo\.rankLevel/, /reserve-frame/]
  ])
  const playing = read('subPages/match/playing.vue')
  assert.match(playing, /vipLevel: res\.match\.my_vip_level \|\| 0/)
  assert.match(playing, /rankLevel: res\.match\.my_rank_level \|\| 0/)
  assert.match(playing, /vipLevel: res\.match\.opponent_vip_level \|\| 0/)
  assert.match(playing, /rankLevel: res\.match\.opponent_rank_level \|\| 0/)
})

test('challenge cards and the waiting room show the sender and receiver VIP badge without a rank frame', () => {
  assertTags('subPages/social/challenges.vue', [
    [/opponentOf\(item\)\.vipLevel/]
  ])
  const challenges = read('subPages/social/challenges.vue')
  assert.match(challenges, /vipLevel: item\.from_user_id === myUserId\.value \? item\.to_vip_level : item\.from_vip_level/)
  assertTags('subPages/match/challengeWaiting.vue', [
    [/challenge\.from_vip_level/],
    [/challenge\.to_vip_level/]
  ])
  // 约球数据没有段位，不能绑定 rank-level 误选头像框。
  for (const path of ['subPages/social/challenges.vue', 'subPages/match/challengeWaiting.vue']) {
    for (const tag of memberAvatarTags(read(path))) assert.doesNotMatch(tag, /rank-level/)
  }
})

test('the profile page uses the shared identity function for its own frame and badge', () => {
  const source = read('pages/user/index.vue')
  assert.match(source, /resolveSelfMemberIdentity\(\{/)
  assert.match(source, /memberIdentity\.value\.frameUrl/)
  assert.doesNotMatch(source, /resolveMemberRankAvatarFrame/)
})
