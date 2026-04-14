package logic

import (
	"errors"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errReputationPenaltyDuplicate = errors.New("reputation penalty duplicate")

type ReputationPenaltyInput struct {
	UserID               int64
	MatchID              int64
	PenaltyScore         int
	ReasonCode           string
	ReasonDetail         string
	CountAsAbnormalMatch bool
}

type ReputationPenaltyResult struct {
	Applied   bool
	Duplicate bool
	Profile   *model.UserReputationProfile
	Log       *model.UserReputationLog
}

type ReputationService struct {
	svcCtx *svc.ServiceContext
	now    func() time.Time
}

func NewReputationService(svcCtx *svc.ServiceContext, now func() time.Time) *ReputationService {
	if now == nil {
		now = NowUTC8
	}
	return &ReputationService{
		svcCtx: svcCtx,
		now:    now,
	}
}

func (s *ReputationService) GetConfig() (ReputationRuntimeConfig, error) {
	return NewReputationConfigService(s.svcCtx).GetConfig()
}

func (s *ReputationService) GetOrCreateProfile(userID int64) (*model.UserReputationProfile, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.UserReputationProfileModel == nil {
		return nil, errors.New("reputation service profile model is nil")
	}

	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}

	profile, err := s.findOrCreateProfile(userID, normalizedInitialReputationScore(cfg.BaseRules))
	if err != nil {
		return nil, err
	}

	now := InUTC8(s.now())
	changed := clampProfileScore(profile, cfg.BaseRules)
	if profile.LastRecoveredAt == nil || profile.LastRecoveredAt.IsZero() {
		profile.LastRecoveredAt = timePointer(now)
		changed = true
	} else if applyReputationRecovery(profile, cfg.BaseRules, cfg.RecoveryRules, now) {
		changed = true
	}

	if changed {
		if err := s.svcCtx.UserReputationProfileModel.Save(profile); err != nil {
			return nil, err
		}
	}

	return profile, nil
}

func (s *ReputationService) ApplyPenalty(input ReputationPenaltyInput) (ReputationPenaltyResult, error) {
	result := ReputationPenaltyResult{}
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil {
		return result, errors.New("reputation service db is nil")
	}
	if s.svcCtx.UserReputationProfileModel == nil || s.svcCtx.UserReputationLogModel == nil {
		return result, errors.New("reputation service models are nil")
	}

	input.ReasonCode = strings.TrimSpace(input.ReasonCode)
	if input.UserID <= 0 {
		return result, errors.New("reputation penalty user_id is required")
	}
	if input.MatchID <= 0 {
		return result, errors.New("reputation penalty match_id is required")
	}
	if input.PenaltyScore <= 0 {
		return result, errors.New("reputation penalty score must be positive")
	}
	if input.ReasonCode == "" {
		return result, errors.New("reputation penalty reason_code is required")
	}

	existingLog, err := s.svcCtx.UserReputationLogModel.FindByUserMatchChangeTypeAndReasonCode(
		input.UserID,
		input.MatchID,
		model.ReputationChangeTypePenalty,
		input.ReasonCode,
	)
	if err != nil {
		return result, err
	}
	if existingLog != nil {
		profile, err := s.GetOrCreateProfile(input.UserID)
		if err != nil {
			return result, err
		}
		result.Duplicate = true
		result.Profile = profile
		result.Log = existingLog
		return result, nil
	}

	cfg, err := s.GetConfig()
	if err != nil {
		return result, err
	}

	now := InUTC8(s.now())
	var savedProfile *model.UserReputationProfile
	var createdLog *model.UserReputationLog

	err = s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		profile, err := s.findOrCreateProfileWithTx(tx, input.UserID, normalizedInitialReputationScore(cfg.BaseRules))
		if err != nil {
			return err
		}

		lockedProfile := &model.UserReputationProfile{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", input.UserID).
			First(lockedProfile).Error; err != nil {
			return err
		}
		profile = lockedProfile
		clampProfileScore(profile, cfg.BaseRules)

		if profile.LastRecoveredAt == nil || profile.LastRecoveredAt.IsZero() {
			profile.LastRecoveredAt = timePointer(now)
		} else {
			applyReputationRecovery(profile, cfg.BaseRules, cfg.RecoveryRules, now)
		}

		beforeScore := profile.ReputationScore
		afterScore := beforeScore - input.PenaltyScore
		if afterScore < cfg.BaseRules.MinScore {
			afterScore = cfg.BaseRules.MinScore
		}

		profile.ReputationScore = afterScore
		profile.TotalPenaltyCount++
		if input.CountAsAbnormalMatch {
			profile.TotalAbnormalMatchCount++
		}
		profile.LastPenalizedAt = timePointer(now)
		if afterScore < cfg.BaseRules.BanThreshold && cfg.BaseRules.BanDurationHours > 0 {
			banStart := now
			if profile.BanUntil != nil && profile.BanUntil.After(banStart) {
				banStart = *profile.BanUntil
			}
			banUntil := InUTC8(banStart).Add(time.Duration(cfg.BaseRules.BanDurationHours) * time.Hour)
			profile.BanUntil = timePointer(banUntil)
		}

		if err := s.svcCtx.UserReputationProfileModel.SaveWithTx(tx, profile); err != nil {
			return err
		}

		matchID := input.MatchID
		createdLog = &model.UserReputationLog{
			UserID:          input.UserID,
			MatchID:         &matchID,
			ChangeType:      model.ReputationChangeTypePenalty,
			ChangeScore:     -input.PenaltyScore,
			BeforeScore:     beforeScore,
			AfterScore:      afterScore,
			ReasonCode:      input.ReasonCode,
			ReasonDetail:    input.ReasonDetail,
			OperatorAdminID: 0,
		}
		if err := s.svcCtx.UserReputationLogModel.CreateWithTx(tx, createdLog); err != nil {
			if isDuplicateReputationLogError(err) {
				return errReputationPenaltyDuplicate
			}
			return err
		}

		savedProfile = profile
		return nil
	})
	if err != nil {
		if errors.Is(err, errReputationPenaltyDuplicate) {
			profile, getProfileErr := s.GetOrCreateProfile(input.UserID)
			if getProfileErr != nil {
				return result, getProfileErr
			}
			existingLog, getLogErr := s.svcCtx.UserReputationLogModel.FindByUserMatchChangeTypeAndReasonCode(
				input.UserID,
				input.MatchID,
				model.ReputationChangeTypePenalty,
				input.ReasonCode,
			)
			if getLogErr != nil {
				return result, getLogErr
			}
			result.Duplicate = true
			result.Profile = profile
			result.Log = existingLog
			return result, nil
		}
		return result, err
	}

	result.Applied = true
	result.Profile = savedProfile
	result.Log = createdLog
	return result, nil
}

func applyReputationRecovery(profile *model.UserReputationProfile, baseRules model.ReputationBaseRules, rules model.ReputationRecoveryRules, now time.Time) bool {
	if profile == nil {
		return false
	}

	now = InUTC8(now)
	changed := clampProfileScore(profile, baseRules)
	if profile.LastRecoveredAt == nil || profile.LastRecoveredAt.IsZero() {
		profile.LastRecoveredAt = timePointer(now)
		return true
	}

	lastRecoveredAt := InUTC8(*profile.LastRecoveredAt)
	if !rules.Enabled || rules.RecoverPerHour <= 0 || !now.After(lastRecoveredAt) {
		return changed
	}

	elapsedHours := int(now.Sub(lastRecoveredAt) / time.Hour)
	if elapsedHours <= 0 {
		return changed
	}

	recoveredScore := elapsedHours * rules.RecoverPerHour
	recoveryCap := effectiveReputationRecoveryCap(baseRules, rules)
	if profile.ReputationScore < recoveryCap {
		profile.ReputationScore += recoveredScore
		if profile.ReputationScore > recoveryCap {
			profile.ReputationScore = recoveryCap
		}
	}

	advanced := lastRecoveredAt.Add(time.Duration(elapsedHours) * time.Hour)
	profile.LastRecoveredAt = timePointer(advanced)
	return true
}

func normalizedInitialReputationScore(baseRules model.ReputationBaseRules) int {
	return clampReputationScore(baseRules.InitialScore, baseRules)
}

func (s *ReputationService) findOrCreateProfile(userID int64, initialScore int) (*model.UserReputationProfile, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.UserReputationProfileModel == nil {
		return nil, errors.New("reputation service profile model is nil")
	}

	profile, err := s.svcCtx.UserReputationProfileModel.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	if profile != nil {
		return profile, nil
	}

	profile = &model.UserReputationProfile{
		UserID:                  userID,
		ReputationScore:         initialScore,
		TotalPenaltyCount:       0,
		TotalAbnormalMatchCount: 0,
	}
	if err := s.svcCtx.UserReputationProfileModel.Save(profile); err != nil {
		if !isDuplicateReputationProfileError(err) {
			return nil, err
		}
		return s.svcCtx.UserReputationProfileModel.FindByUserID(userID)
	}
	return profile, nil
}

func (s *ReputationService) findOrCreateProfileWithTx(tx *gorm.DB, userID int64, initialScore int) (*model.UserReputationProfile, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.UserReputationProfileModel == nil {
		return nil, errors.New("reputation service profile model is nil")
	}

	profile, err := s.svcCtx.UserReputationProfileModel.FindByUserIDWithTx(tx, userID)
	if err != nil {
		return nil, err
	}
	if profile != nil {
		return profile, nil
	}

	profile = &model.UserReputationProfile{
		UserID:                  userID,
		ReputationScore:         initialScore,
		TotalPenaltyCount:       0,
		TotalAbnormalMatchCount: 0,
	}
	if err := s.svcCtx.UserReputationProfileModel.SaveWithTx(tx, profile); err != nil {
		if !isDuplicateReputationProfileError(err) {
			return nil, err
		}
		return s.svcCtx.UserReputationProfileModel.FindByUserIDWithTx(tx, userID)
	}
	return profile, nil
}

func effectiveReputationRecoveryCap(baseRules model.ReputationBaseRules, rules model.ReputationRecoveryRules) int {
	recoveryCap := rules.RecoverMaxScore
	if recoveryCap <= 0 {
		recoveryCap = baseRules.MaxScore
	}
	if recoveryCap > baseRules.MaxScore {
		recoveryCap = baseRules.MaxScore
	}
	if recoveryCap < baseRules.MinScore {
		recoveryCap = baseRules.MinScore
	}
	return recoveryCap
}

func clampProfileScore(profile *model.UserReputationProfile, baseRules model.ReputationBaseRules) bool {
	if profile == nil {
		return false
	}
	clamped := clampReputationScore(profile.ReputationScore, baseRules)
	if clamped == profile.ReputationScore {
		return false
	}
	profile.ReputationScore = clamped
	return true
}

func clampReputationScore(score int, baseRules model.ReputationBaseRules) int {
	if score < baseRules.MinScore {
		return baseRules.MinScore
	}
	if baseRules.MaxScore > 0 && score > baseRules.MaxScore {
		return baseRules.MaxScore
	}
	return score
}

func isDuplicateReputationLogError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") || strings.Contains(msg, "duplicate entry")
}

func isDuplicateReputationProfileError(err error) bool {
	return isDuplicateReputationLogError(err)
}
