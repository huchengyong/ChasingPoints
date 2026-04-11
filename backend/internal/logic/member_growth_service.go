package logic

import (
	"errors"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

const MemberGrowthDailyCap = 5

type MemberGrowthLevelRule struct {
	Level          int
	RequiredPoints int
}

type MemberGrowthAwardResult struct {
	Granted          bool
	Reason           string
	GrowthPoints     int
	GrowthLevel      int
	TodayGrowthCount int
	NextLevel        int
	NextLevelPoints  int
	RemainingPoints  int
	Frozen           bool
}

type MemberGrowthSnapshot struct {
	GrowthPoints     int
	GrowthLevel      int
	TodayGrowthCount int
	DailyCap         int
	NextLevel        int
	NextLevelPoints  int
	RemainingPoints  int
	Frozen           bool
}

type MemberGrowthService struct {
	svcCtx *svc.ServiceContext
	now    func() time.Time
}

func NewMemberGrowthService(svcCtx *svc.ServiceContext, now func() time.Time) *MemberGrowthService {
	if now == nil {
		now = NowUTC8
	}
	return &MemberGrowthService{
		svcCtx: svcCtx,
		now:    now,
	}
}

func MemberGrowthLevelRules() []MemberGrowthLevelRule {
	return []MemberGrowthLevelRule{
		{Level: 1, RequiredPoints: 0},
		{Level: 2, RequiredPoints: 10},
		{Level: 3, RequiredPoints: 60},
		{Level: 4, RequiredPoints: 260},
		{Level: 5, RequiredPoints: 760},
	}
}

func ResolveMemberGrowthLevel(points int) int {
	level := 1
	for _, rule := range MemberGrowthLevelRules() {
		if points >= rule.RequiredPoints {
			level = rule.Level
		}
	}
	return level
}

func ResolveNextMemberGrowthRule(points int) (level int, requiredPoints int, remainingPoints int) {
	for _, rule := range MemberGrowthLevelRules() {
		if rule.RequiredPoints > points {
			return rule.Level, rule.RequiredPoints, rule.RequiredPoints - points
		}
	}
	return 0, 0, 0
}

func (s *MemberGrowthService) BuildSnapshot(user *model.User, profile *model.MemberGrowthProfile) MemberGrowthSnapshot {
	now := s.now()
	profileCopy := profile
	if profileCopy == nil {
		profileCopy = &model.MemberGrowthProfile{
			GrowthPoints: 0,
			GrowthLevel:  1,
		}
	}
	todayGrowthCount := profileCopy.TodayGrowthCount
	if !sameGrowthDay(profileCopy.TodayGrowthDate, now) {
		todayGrowthCount = 0
	}
	nextLevel, nextPoints, remaining := ResolveNextMemberGrowthRule(profileCopy.GrowthPoints)
	return MemberGrowthSnapshot{
		GrowthPoints:     profileCopy.GrowthPoints,
		GrowthLevel:      ResolveMemberGrowthLevel(profileCopy.GrowthPoints),
		TodayGrowthCount: todayGrowthCount,
		DailyCap:         MemberGrowthDailyCap,
		NextLevel:        nextLevel,
		NextLevelPoints:  nextPoints,
		RemainingPoints:  remaining,
		Frozen:           !memberGrowthMembershipActive(user, now),
	}
}

func (s *MemberGrowthService) GetSnapshotForUser(userId int64) (MemberGrowthSnapshot, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.UserModel == nil || s.svcCtx.MemberGrowthProfileModel == nil {
		return MemberGrowthSnapshot{}, errors.New("member growth service not ready")
	}
	user, err := s.svcCtx.UserModel.FindById(userId)
	if err != nil {
		return MemberGrowthSnapshot{}, err
	}
	profile, err := s.svcCtx.MemberGrowthProfileModel.FindByUserId(userId)
	if err != nil {
		return MemberGrowthSnapshot{}, err
	}
	return s.BuildSnapshot(user, profile), nil
}

func (s *MemberGrowthService) AwardCompletedMatch(userId, matchId int64) (MemberGrowthAwardResult, error) {
	result := MemberGrowthAwardResult{
		GrowthLevel: 1,
	}
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return result, errors.New("member growth service db is nil")
	}
	if s.svcCtx.UserModel == nil || s.svcCtx.MemberGrowthProfileModel == nil || s.svcCtx.MemberGrowthLogModel == nil {
		return result, errors.New("member growth service models are nil")
	}

	now := InUTC8(s.now())
	var snapshot MemberGrowthSnapshot
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		user, err := s.svcCtx.UserModel.FindById(userId)
		if err != nil {
			return err
		}
		if user == nil {
			result.Reason = "user_not_found"
			return nil
		}

		profile, err := s.svcCtx.MemberGrowthProfileModel.FindByUserIdWithTx(tx, userId)
		if err != nil {
			return err
		}

		if grantedLog, err := s.svcCtx.MemberGrowthLogModel.FindByUserMatchAndSource(userId, matchId, model.MemberGrowthSourceRealMatchCompleted); err != nil {
			return err
		} else if grantedLog != nil {
			result.Reason = "duplicate_match"
			snapshot = s.BuildSnapshot(user, profile)
			return nil
		}

		if !memberGrowthMembershipActive(user, now) {
			result.Reason = "membership_inactive"
			snapshot = s.BuildSnapshot(user, profile)
			return nil
		}

		if profile == nil {
			profile, err = s.svcCtx.MemberGrowthProfileModel.FindOrCreateWithTx(tx, userId)
			if err != nil {
				return err
			}
		}

		if !sameGrowthDay(profile.TodayGrowthDate, now) {
			profile.TodayGrowthDate = timePointer(startOfGrowthDay(now))
			profile.TodayGrowthCount = 0
		}

		if profile.TodayGrowthCount >= MemberGrowthDailyCap {
			result.Reason = "daily_cap_reached"
			snapshot = s.BuildSnapshot(user, profile)
			return nil
		}

		profile.GrowthPoints += 1
		profile.GrowthLevel = ResolveMemberGrowthLevel(profile.GrowthPoints)
		profile.TodayGrowthCount += 1
		profile.TodayGrowthDate = timePointer(startOfGrowthDay(now))
		profile.LastGrowthAt = timePointer(now)
		if err := s.svcCtx.MemberGrowthProfileModel.UpdateWithTx(tx, profile); err != nil {
			return err
		}

		if err := s.svcCtx.MemberGrowthLogModel.CreateWithTx(tx, &model.MemberGrowthLog{
			UserId:       userId,
			MatchId:      matchId,
			GrowthPoints: 1,
			Source:       model.MemberGrowthSourceRealMatchCompleted,
		}); err != nil {
			return err
		}

		result.Granted = true
		result.Reason = "granted"
		snapshot = s.BuildSnapshot(user, profile)
		return nil
	})
	if err != nil {
		return result, err
	}

	result.GrowthPoints = snapshot.GrowthPoints
	result.GrowthLevel = snapshot.GrowthLevel
	result.TodayGrowthCount = snapshot.TodayGrowthCount
	result.NextLevel = snapshot.NextLevel
	result.NextLevelPoints = snapshot.NextLevelPoints
	result.RemainingPoints = snapshot.RemainingPoints
	result.Frozen = snapshot.Frozen
	return result, nil
}

func memberGrowthMembershipActive(user *model.User, now time.Time) bool {
	if user == nil || user.MemberExpiresAt == nil {
		return false
	}
	return InUTC8(*user.MemberExpiresAt).After(InUTC8(now))
}

func sameGrowthDay(stored *time.Time, now time.Time) bool {
	if stored == nil || stored.IsZero() {
		return false
	}
	left := InUTC8(*stored)
	right := InUTC8(now)
	return left.Year() == right.Year() && left.Month() == right.Month() && left.Day() == right.Day()
}

func startOfGrowthDay(now time.Time) time.Time {
	now = InUTC8(now)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, UTC8Location)
}

func timePointer(t time.Time) *time.Time {
	return &t
}
