import test from 'node:test'
import assert from 'node:assert/strict'

import {
  getFavoriteVenueRewardPopupStorageKey,
  getFavoriteVenueRewardFloatSnoozeKey,
  calcFavoriteVenueRewardFloatSnoozeUntil,
  isFavoriteVenueRewardFloatSnoozed,
  resolveFavoriteVenueRewardFloatSnoozeMigration,
  shouldShowFavoriteVenueRewardPopup,
  resolveFavoriteVenueRewardTaskCard,
  resolveFavoriteVenueRewardPopupCopy,
  resolveFavoriteVenueMemberCard
} from '../utils/favorite-venue-reward.js'
import { resolveCompactRewardEntry } from '../utils/user-homepage.js'

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

  assert.equal(card.visible, false)
  assert.equal(card.title, '')
  assert.equal(card.description, '')
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

  assert.equal(copy.title, '补充常玩球馆，送 1 个月会员')
  assert.match(copy.description, /审核通过后自动到账/)
  assert.equal(copy.primaryText, '去领取')
})

test('resolveFavoriteVenueRewardTaskCard does not use new-user wording for the default entry card', () => {
  const card = resolveFavoriteVenueRewardTaskCard({
    enabled: true,
    reward_days: 30,
    status: 'not_started'
  })

  assert.equal(card.statusText, '常玩球馆奖励')
  assert.doesNotMatch(card.title, /新用户福利/)
})

test('resolveFavoriteVenueMemberCard returns active member summary for rewarded users', () => {
  const card = resolveFavoriteVenueMemberCard({
    reward_days: 30,
    member_expires_at: '2026-04-26 12:00:00'
  }, new Date('2026-03-26T12:00:00+08:00'))

  assert.equal(card.visible, true)
  assert.equal(card.statusText, '会员中')
  assert.equal(card.title, '会员生效中')
  assert.equal(card.description, '有效期至 2026-04-26 12:00:00')
  assert.deepEqual(card.benefits, [])
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

test('getFavoriteVenueRewardFloatSnoozeKey scopes snooze by user id and status', () => {
  assert.equal(getFavoriteVenueRewardFloatSnoozeKey(12, 'not_started'), 'favorite-venue-reward-float-snoozed-until:12:not_started')
  assert.equal(getFavoriteVenueRewardFloatSnoozeKey(42, 'rejected'), 'favorite-venue-reward-float-snoozed-until:42:rejected')
  assert.notEqual(
    getFavoriteVenueRewardFloatSnoozeKey(12, 'not_started'),
    getFavoriteVenueRewardFloatSnoozeKey(12, 'rejected')
  )
  assert.notEqual(
    getFavoriteVenueRewardFloatSnoozeKey(12, 'not_started'),
    getFavoriteVenueRewardFloatSnoozeKey(18, 'not_started')
  )
})

test('calcFavoriteVenueRewardFloatSnoozeUntil returns a date three days after the given now', () => {
  const now = new Date('2026-07-21T12:00:00+08:00')
  const deadline = calcFavoriteVenueRewardFloatSnoozeUntil(now)
  const expected = new Date('2026-07-24T12:00:00+08:00')
  assert.equal(deadline.toISOString(), expected.toISOString())
})

test('isFavoriteVenueRewardFloatSnoozed returns true only when the deadline is in the future', () => {
  const now = new Date('2026-07-22T00:00:00+08:00')
  const futureDeadline = new Date('2026-07-25T00:00:00+08:00')
  const pastDeadline = new Date('2026-07-21T00:00:00+08:00')

  assert.equal(isFavoriteVenueRewardFloatSnoozed(futureDeadline, now), true)
  assert.equal(isFavoriteVenueRewardFloatSnoozed(pastDeadline, now), false)
  assert.equal(isFavoriteVenueRewardFloatSnoozed(null, now), false)
  assert.equal(isFavoriteVenueRewardFloatSnoozed('invalid', now), false)
})

test('isFavoriteVenueRewardFloatSnoozed returns false when the deadline equals now', () => {
  const now = new Date('2026-07-24T12:00:00+08:00')
  const deadline = new Date('2026-07-24T12:00:00+08:00')
  assert.equal(isFavoriteVenueRewardFloatSnoozed(deadline, now), false)
})

test('resolveFavoriteVenueRewardFloatSnoozeMigration only produces migration for dismissed users without an existing snooze', () => {
  const noOld = resolveFavoriteVenueRewardFloatSnoozeMigration({
    userId: 10,
    rewardStatus: { enabled: true, status: 'not_started' },
    oldPopupDismissed: false,
    getFloatSnoozeValue: () => null
  })
  assert.equal(noOld, null)

  const alreadySnoozed = resolveFavoriteVenueRewardFloatSnoozeMigration({
    userId: 10,
    rewardStatus: { enabled: true, status: 'not_started' },
    oldPopupDismissed: true,
    getFloatSnoozeValue: () => new Date()
  })
  assert.equal(alreadySnoozed, null)

  const wrongStatus = resolveFavoriteVenueRewardFloatSnoozeMigration({
    userId: 10,
    rewardStatus: { enabled: true, status: 'pending_review' },
    oldPopupDismissed: true,
    getFloatSnoozeValue: () => null
  })
  assert.equal(wrongStatus, null)

  const migrates = resolveFavoriteVenueRewardFloatSnoozeMigration({
    userId: 10,
    rewardStatus: { enabled: true, status: 'not_started' },
    oldPopupDismissed: true,
    getFloatSnoozeValue: () => null
  })
  assert.notEqual(migrates, null)
  assert.equal(migrates.key, 'favorite-venue-reward-float-snoozed-until:10:not_started')
  assert.ok(migrates.snoozeUntil instanceof Date)
})

test('resolveFavoriteVenueRewardFloatSnoozeMigration does not migrate for rejected status', () => {
  const migrates = resolveFavoriteVenueRewardFloatSnoozeMigration({
    userId: 5,
    rewardStatus: { enabled: true, status: 'rejected' },
    oldPopupDismissed: true,
    getFloatSnoozeValue: () => null
  })
  assert.equal(migrates, null)
})

test('resolveCompactRewardEntry marks not_started and rejected as floatable with action', () => {
  const notStarted = resolveCompactRewardEntry({ enabled: true, status: 'not_started', reward_days: 30 })
  assert.equal(notStarted.visible, true)
  assert.equal(notStarted.floatable, true)
  assert.equal(notStarted.inline, false)
  assert.equal(notStarted.action, 'favorite-venue')
  assert.ok(notStarted.actionText)

  const rejected = resolveCompactRewardEntry({ enabled: true, status: 'rejected', reward_days: 30, reject_reason: '地址不完整' })
  assert.equal(rejected.visible, true)
  assert.equal(rejected.floatable, true)
  assert.equal(rejected.inline, false)
  assert.equal(rejected.action, 'favorite-venue')
  assert.ok(rejected.actionText)
})

test('resolveCompactRewardEntry marks pending_review as inline read-only without action', () => {
  const pending = resolveCompactRewardEntry({ enabled: true, status: 'pending_review', reward_days: 30, submitted_venue_name: '测试球馆' })
  assert.equal(pending.visible, true)
  assert.equal(pending.floatable, false)
  assert.equal(pending.inline, true)
  assert.equal(pending.action, '')
  assert.equal(pending.actionText, '')
})

test('resolveCompactRewardEntry hides reward_granted and disabled states', () => {
  const granted = resolveCompactRewardEntry({ enabled: true, status: 'reward_granted', reward_days: 30 })
  assert.equal(granted.visible, false)
  assert.equal(granted.floatable, false)
  assert.equal(granted.inline, false)

  const disabled = resolveCompactRewardEntry({ enabled: false, status: 'not_started', reward_days: 30 })
  assert.equal(disabled.visible, false)
  assert.equal(disabled.floatable, false)
  assert.equal(disabled.inline, false)
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
