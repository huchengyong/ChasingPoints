import test from 'node:test'
import assert from 'node:assert/strict'

import {
  getFavoriteVenueRewardPopupStorageKey,
  shouldShowFavoriteVenueRewardPopup,
  resolveFavoriteVenueRewardTaskCard,
  resolveFavoriteVenueRewardPopupCopy,
  resolveFavoriteVenueMemberCard
} from '../utils/favorite-venue-reward.js'

test('getFavoriteVenueRewardPopupStorageKey scopes dismissal by user id', () => {
  assert.equal(getFavoriteVenueRewardPopupStorageKey(12), 'favorite-venue-reward-popup-dismissed:12')
})

test('shouldShowFavoriteVenueRewardPopup only returns true for enabled not-started tasks', () => {
  assert.equal(shouldShowFavoriteVenueRewardPopup({
    userId: 18,
    popupDismissed: false,
    rewardStatus: {
      enabled: true,
      popup_enabled: true,
      status: 'not_started'
    }
  }), true)

  assert.equal(shouldShowFavoriteVenueRewardPopup({
    userId: 18,
    popupDismissed: true,
    rewardStatus: {
      enabled: true,
      popup_enabled: true,
      status: 'not_started'
    }
  }), false)

  assert.equal(shouldShowFavoriteVenueRewardPopup({
    userId: 18,
    popupDismissed: false,
    rewardStatus: {
      enabled: true,
      popup_enabled: true,
      status: 'pending_review'
    }
  }), false)
})

test('resolveFavoriteVenueRewardTaskCard returns entry copy for not started users', () => {
  const card = resolveFavoriteVenueRewardTaskCard({
    enabled: true,
    reward_days: 30,
    status: 'not_started'
  })

  assert.equal(card.visible, true)
  assert.equal(card.title, '补充常玩球馆，送 1 个月会员')
  assert.equal(card.actionText, '去领取')
})

test('resolveFavoriteVenueRewardTaskCard returns waiting copy for pending review users', () => {
  const card = resolveFavoriteVenueRewardTaskCard({
    enabled: true,
    reward_days: 30,
    status: 'pending_review',
    submitted_venue_name: '南山台球会'
  })

  assert.equal(card.visible, true)
  assert.equal(card.title, '常玩球馆审核中')
  assert.match(card.description, /南山台球会/)
})

test('resolveFavoriteVenueRewardTaskCard returns granted copy for rewarded users', () => {
  const card = resolveFavoriteVenueRewardTaskCard({
    enabled: true,
    reward_days: 30,
    status: 'reward_granted',
    member_expires_at: '2026-04-26 12:00:00'
  })

  assert.equal(card.visible, true)
  assert.equal(card.title, '已获赠 1 个月会员')
  assert.match(card.description, /2026-04-26 12:00:00/)
  assert.equal(card.actionText, '')
})

test('resolveFavoriteVenueRewardTaskCard returns retry copy for rejected users', () => {
  const card = resolveFavoriteVenueRewardTaskCard({
    enabled: true,
    reward_days: 30,
    status: 'rejected',
    reject_reason: '地址信息不完整'
  })

  assert.equal(card.visible, true)
  assert.equal(card.title, '补充信息后可重新领取会员')
  assert.match(card.description, /地址信息不完整/)
  assert.equal(card.actionText, '重新提交')
})

test('resolveFavoriteVenueRewardTaskCard hides untouched tasks when activity is disabled', () => {
  const card = resolveFavoriteVenueRewardTaskCard({
    enabled: false,
    reward_days: 30,
    status: 'not_started'
  })

  assert.equal(card.visible, false)
})

test('resolveFavoriteVenueRewardPopupCopy uses reward duration in the title', () => {
  const copy = resolveFavoriteVenueRewardPopupCopy({
    reward_days: 30
  })

  assert.equal(copy.title, '新用户福利：送 1 个月会员')
  assert.match(copy.description, /审核通过后自动到账/)
  assert.equal(copy.primaryText, '去领取')
})

test('resolveFavoriteVenueMemberCard returns active member summary for rewarded users', () => {
  const card = resolveFavoriteVenueMemberCard({
    reward_days: 30,
    member_expires_at: '2026-04-26 12:00:00'
  }, new Date('2026-03-26T12:00:00+08:00'))

  assert.equal(card.visible, true)
  assert.equal(card.statusText, '会员中')
  assert.equal(card.title, '1 个月订阅会员进行中')
  assert.match(card.description, /2026-04-26 12:00:00/)
  assert.deepEqual(card.benefits.map(item => item.title), [
    '有效期清晰可见',
    '会员时长自动累计',
    '权益会继续在这里更新'
  ])
})

test('resolveFavoriteVenueMemberCard returns expired summary after member expires', () => {
  const card = resolveFavoriteVenueMemberCard({
    reward_days: 30,
    member_expires_at: '2026-04-01 12:00:00'
  }, new Date('2026-04-02T12:00:00+08:00'))

  assert.equal(card.visible, true)
  assert.equal(card.statusText, '已到期')
  assert.equal(card.title, '订阅会员已到期')
  assert.match(card.description, /2026-04-01 12:00:00/)
})

test('resolveFavoriteVenueMemberCard does not depend on host string date parsing', () => {
  const RealDate = Date

  global.Date = class extends RealDate {
    constructor(value) {
      if (arguments.length === 1 && typeof value === 'string') {
        super('invalid')
        return
      }
      super(...arguments)
    }

    static now() {
      return RealDate.now()
    }

    static parse(value) {
      return RealDate.parse(value)
    }

    static UTC(...args) {
      return RealDate.UTC(...args)
    }
  }

  try {
    const card = resolveFavoriteVenueMemberCard({
      reward_days: 30,
      member_expires_at: '2026-04-01 12:00:00'
    }, new RealDate('2026-04-02T12:00:00+08:00'))

    assert.equal(card.statusText, '已到期')
  } finally {
    global.Date = RealDate
  }
})
