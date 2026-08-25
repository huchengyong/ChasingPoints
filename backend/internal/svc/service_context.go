package svc

import (
	"context"
	"strings"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/pkg/geocode"
	oauthverify "chasing_points/internal/pkg/oauth"
	"chasing_points/internal/pkg/push"
	qiniuupload "chasing_points/internal/pkg/qiniu"
	"chasing_points/internal/pkg/wechatmini"
	"chasing_points/internal/pkg/wsticket"
	"chasing_points/internal/sms"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	defaultDBMaxIdleConns = 10
	defaultDBMaxOpenConns = 100
	defaultQuotaPrefix    = "geo:quota"
	defaultQuotaTTL       = 70 * time.Second
	defaultGeocodeSource  = "apihz"
	defaultGeocodeLocker  = "chasing_points-api"
)

type ServiceContext struct {
	Config                          config.Config
	DB                              *gorm.DB
	Redis                           *redis.Client
	SmsClient                       *sms.AliSmsClient
	CodeManager                     *sms.CodeManager
	AreaModel                       *model.AreaModel
	UserModel                       *model.UserModel
	UserDataLifecycleModel          *model.UserDataLifecycleModel
	OauthModel                      *model.UserOauthModel
	MatchModel                      *model.MatchModel
	CompetitiveReadModel            *model.CompetitiveReadModel
	RankingModel                    *model.RankingModel
	AchievementModel                *model.AchievementModel
	UserAchievementModel            *model.UserAchievementModel
	UserTitleModel                  *model.UserTitleModel
	AchievementProgressEventModel   *model.AchievementProgressEventModel
	FriendModel                     *model.FriendModel
	FollowModel                     *model.FollowModel
	SocialPostModel                 *model.SocialPostModel
	NotificationModel               *model.NotificationModel
	UserNotificationPreferenceModel *model.UserNotificationPreferenceModel
	ChallengeModel                  *model.ChallengeModel
	TournamentModel                 *model.TournamentModel
	TournamentParticipantModel      *model.TournamentParticipantModel
	EventNewsModel                  *model.EventNewsModel
	PlayerModel                     *model.PlayerModel
	RulesContentModel               *model.RulesContentModel
	SeasonModel                     *model.SeasonModel
	SeasonRecordModel               *model.SeasonRecordModel
	SeasonChallengeSnapshotModel    *model.SeasonChallengeSnapshotModel
	SeasonSettlementModel           *model.SeasonSettlementModel
	VenueModel                      *model.VenueModel
	VenueCheckinModel               *model.VenueCheckinModel
	VenueGeocodeTaskModel           *model.VenueGeocodeTaskModel
	GeocodeAccountModel             *model.GeocodeAccountModel
	FavoriteVenueRewardConfigModel  *model.FavoriteVenueRewardConfigModel
	FavoriteVenueRewardRecordModel  *model.FavoriteVenueRewardRecordModel
	MemberSubscriptionOrderModel    *model.MemberSubscriptionOrderModel
	MemberGrowthProfileModel        *model.MemberGrowthProfileModel
	MemberGrowthLogModel            *model.MemberGrowthLogModel
	MemberRightsConfigModel         *model.MemberRightsConfigModel
	ReputationConfigModel           *model.ReputationConfigModel
	UserReputationProfileModel      *model.UserReputationProfileModel
	UserReputationLogModel          *model.UserReputationLogModel
	FeedbackTicketModel             *model.FeedbackTicketModel
	TournamentMatchModel            *model.TournamentMatchModel
	AdminModel                      *model.AdminModel
	AdminLoginLogModel              *model.AdminLoginLogModel
	PushService                     *push.PushService
	QiniuUploadService              *qiniuupload.UploadService
	Geocoder                        geocode.Geocoder
	GeocodeWorker                   *geocode.Worker
	WechatMiniClient                wechatmini.Client
	OAuthVerifier                   oauthverify.Verifier
	WSTicketStore                   wsticket.Store
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := mustOpenDB(c)
	rdb := newRedisClient(c)
	smsClient := mustNewSmsClient(c)
	codeManager := sms.NewCodeManager(rdb)

	models := newServiceModels(db)
	geocodeDeps := newGeocodeDependencies(c, rdb, models)
	wechatMiniClient := newWechatMiniClient(c)

	return &ServiceContext{
		Config:                          c,
		DB:                              db,
		Redis:                           rdb,
		SmsClient:                       smsClient,
		CodeManager:                     codeManager,
		AreaModel:                       models.AreaModel,
		UserModel:                       models.UserModel,
		UserDataLifecycleModel:          models.UserDataLifecycleModel,
		OauthModel:                      models.OauthModel,
		MatchModel:                      models.MatchModel,
		CompetitiveReadModel:            models.CompetitiveReadModel,
		RankingModel:                    models.RankingModel,
		AchievementModel:                models.AchievementModel,
		UserAchievementModel:            models.UserAchievementModel,
		UserTitleModel:                  models.UserTitleModel,
		AchievementProgressEventModel:   models.AchievementProgressEventModel,
		FriendModel:                     models.FriendModel,
		FollowModel:                     models.FollowModel,
		SocialPostModel:                 models.SocialPostModel,
		NotificationModel:               models.NotificationModel,
		UserNotificationPreferenceModel: models.UserNotificationPreferenceModel,
		ChallengeModel:                  models.ChallengeModel,
		TournamentModel:                 models.TournamentModel,
		TournamentParticipantModel:      models.TournamentParticipantModel,
		EventNewsModel:                  models.EventNewsModel,
		PlayerModel:                     models.PlayerModel,
		RulesContentModel:               models.RulesContentModel,
		SeasonModel:                     models.SeasonModel,
		SeasonRecordModel:               models.SeasonRecordModel,
		SeasonChallengeSnapshotModel:    models.SeasonChallengeSnapshotModel,
		SeasonSettlementModel:           models.SeasonSettlementModel,
		VenueModel:                      models.VenueModel,
		VenueCheckinModel:               models.VenueCheckinModel,
		VenueGeocodeTaskModel:           models.VenueGeocodeTaskModel,
		GeocodeAccountModel:             models.GeocodeAccountModel,
		FavoriteVenueRewardConfigModel:  models.FavoriteVenueRewardConfigModel,
		FavoriteVenueRewardRecordModel:  models.FavoriteVenueRewardRecordModel,
		MemberSubscriptionOrderModel:    models.MemberSubscriptionOrderModel,
		MemberGrowthProfileModel:        models.MemberGrowthProfileModel,
		MemberGrowthLogModel:            models.MemberGrowthLogModel,
		MemberRightsConfigModel:         models.MemberRightsConfigModel,
		ReputationConfigModel:           models.ReputationConfigModel,
		UserReputationProfileModel:      models.UserReputationProfileModel,
		UserReputationLogModel:          models.UserReputationLogModel,
		FeedbackTicketModel:             models.FeedbackTicketModel,
		TournamentMatchModel:            models.TournamentMatchModel,
		AdminModel:                      models.AdminModel,
		AdminLoginLogModel:              models.AdminLoginLogModel,
		PushService:                     newPushService(c),
		QiniuUploadService:              newQiniuUploadService(c),
		Geocoder:                        geocodeDeps.Client,
		GeocodeWorker:                   geocodeDeps.Worker,
		WechatMiniClient:                wechatMiniClient,
		OAuthVerifier:                   oauthverify.NewProviderVerifier(c.Security),
		WSTicketStore:                   wsticket.NewRedisStore(rdb),
	}
}

func newWechatMiniClient(c config.Config) wechatmini.Client {
	return wechatmini.NewHTTPClient(
		c.WechatMiniProgram.AppId,
		c.WechatMiniProgram.AppSecret,
		time.Duration(c.WechatMiniProgram.RequestTimeoutMs)*time.Millisecond,
	)
}

type serviceModels struct {
	AreaModel                       *model.AreaModel
	UserModel                       *model.UserModel
	UserDataLifecycleModel          *model.UserDataLifecycleModel
	OauthModel                      *model.UserOauthModel
	MatchModel                      *model.MatchModel
	CompetitiveReadModel            *model.CompetitiveReadModel
	RankingModel                    *model.RankingModel
	AchievementModel                *model.AchievementModel
	UserAchievementModel            *model.UserAchievementModel
	UserTitleModel                  *model.UserTitleModel
	AchievementProgressEventModel   *model.AchievementProgressEventModel
	FriendModel                     *model.FriendModel
	FollowModel                     *model.FollowModel
	SocialPostModel                 *model.SocialPostModel
	NotificationModel               *model.NotificationModel
	UserNotificationPreferenceModel *model.UserNotificationPreferenceModel
	ChallengeModel                  *model.ChallengeModel
	TournamentModel                 *model.TournamentModel
	TournamentParticipantModel      *model.TournamentParticipantModel
	EventNewsModel                  *model.EventNewsModel
	PlayerModel                     *model.PlayerModel
	RulesContentModel               *model.RulesContentModel
	SeasonModel                     *model.SeasonModel
	SeasonRecordModel               *model.SeasonRecordModel
	SeasonChallengeSnapshotModel    *model.SeasonChallengeSnapshotModel
	SeasonSettlementModel           *model.SeasonSettlementModel
	VenueModel                      *model.VenueModel
	VenueCheckinModel               *model.VenueCheckinModel
	VenueGeocodeTaskModel           *model.VenueGeocodeTaskModel
	GeocodeAccountModel             *model.GeocodeAccountModel
	FavoriteVenueRewardConfigModel  *model.FavoriteVenueRewardConfigModel
	FavoriteVenueRewardRecordModel  *model.FavoriteVenueRewardRecordModel
	MemberSubscriptionOrderModel    *model.MemberSubscriptionOrderModel
	MemberGrowthProfileModel        *model.MemberGrowthProfileModel
	MemberGrowthLogModel            *model.MemberGrowthLogModel
	MemberRightsConfigModel         *model.MemberRightsConfigModel
	ReputationConfigModel           *model.ReputationConfigModel
	UserReputationProfileModel      *model.UserReputationProfileModel
	UserReputationLogModel          *model.UserReputationLogModel
	FeedbackTicketModel             *model.FeedbackTicketModel
	TournamentMatchModel            *model.TournamentMatchModel
	AdminModel                      *model.AdminModel
	AdminLoginLogModel              *model.AdminLoginLogModel
}

type geocodeDependencies struct {
	Client geocode.Geocoder
	Worker *geocode.Worker
}

func (s *ServiceContext) DBWithContext(ctx context.Context) *gorm.DB {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.WithContext(ctx)
}

// WithContext returns a shallow, request-scoped service context whose Gorm
// models carry the HTTP context. It leaves worker and test contexts untouched.
func (s *ServiceContext) WithContext(ctx context.Context) *ServiceContext {
	if s == nil || s.DB == nil || observability.RequestMetricsFromContext(ctx) == nil {
		return s
	}
	if s.DB.Statement != nil && observability.RequestMetricsFromContext(s.DB.Statement.Context) != nil {
		return s
	}
	scoped := *s
	scoped.DB = s.DB.WithContext(ctx)
	models := newServiceModels(scoped.DB)
	scoped.AreaModel = models.AreaModel
	scoped.UserModel = models.UserModel
	scoped.UserDataLifecycleModel = models.UserDataLifecycleModel
	scoped.OauthModel = models.OauthModel
	scoped.MatchModel = models.MatchModel
	scoped.CompetitiveReadModel = models.CompetitiveReadModel
	scoped.RankingModel = s.RankingModel.WithDB(scoped.DB)
	scoped.AchievementModel = models.AchievementModel
	scoped.UserAchievementModel = models.UserAchievementModel
	scoped.UserTitleModel = models.UserTitleModel
	scoped.AchievementProgressEventModel = models.AchievementProgressEventModel
	scoped.FriendModel = models.FriendModel
	scoped.FollowModel = models.FollowModel
	scoped.SocialPostModel = models.SocialPostModel
	scoped.NotificationModel = models.NotificationModel
	scoped.UserNotificationPreferenceModel = models.UserNotificationPreferenceModel
	scoped.ChallengeModel = models.ChallengeModel
	scoped.TournamentModel = models.TournamentModel
	scoped.TournamentParticipantModel = models.TournamentParticipantModel
	scoped.EventNewsModel = models.EventNewsModel
	scoped.PlayerModel = models.PlayerModel
	scoped.RulesContentModel = models.RulesContentModel
	scoped.SeasonModel = models.SeasonModel
	scoped.SeasonRecordModel = models.SeasonRecordModel
	scoped.SeasonChallengeSnapshotModel = models.SeasonChallengeSnapshotModel
	scoped.SeasonSettlementModel = models.SeasonSettlementModel
	scoped.VenueModel = models.VenueModel
	scoped.VenueCheckinModel = models.VenueCheckinModel
	scoped.VenueGeocodeTaskModel = models.VenueGeocodeTaskModel
	scoped.GeocodeAccountModel = models.GeocodeAccountModel
	scoped.FavoriteVenueRewardConfigModel = models.FavoriteVenueRewardConfigModel
	scoped.FavoriteVenueRewardRecordModel = models.FavoriteVenueRewardRecordModel
	scoped.MemberSubscriptionOrderModel = models.MemberSubscriptionOrderModel
	scoped.MemberGrowthProfileModel = models.MemberGrowthProfileModel
	scoped.MemberGrowthLogModel = models.MemberGrowthLogModel
	scoped.MemberRightsConfigModel = models.MemberRightsConfigModel
	scoped.ReputationConfigModel = models.ReputationConfigModel
	scoped.UserReputationProfileModel = models.UserReputationProfileModel
	scoped.UserReputationLogModel = models.UserReputationLogModel
	scoped.FeedbackTicketModel = models.FeedbackTicketModel
	scoped.TournamentMatchModel = models.TournamentMatchModel
	scoped.AdminModel = models.AdminModel
	scoped.AdminLoginLogModel = models.AdminLoginLogModel
	return &scoped
}

func (s *ServiceContext) CompetitiveReadModelsEnabled() bool {
	if s == nil || s.CompetitiveReadModel == nil {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(s.Config.CompetitiveReadModel.ReadMode))
	return mode == "enabled"
}

func mustOpenDB(c config.Config) *gorm.DB {
	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{
		Logger: observability.NewGormLogger(time.Duration(c.Observability.SlowSQLThresholdMs) * time.Millisecond),
	})
	if err != nil {
		panic("连接数据库失败: " + err.Error())
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic("获取数据库连接失败: " + err.Error())
	}

	sqlDB.SetMaxIdleConns(defaultDBMaxIdleConns)
	sqlDB.SetMaxOpenConns(defaultDBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return db
}

func newRedisClient(c config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     c.Redis.Host,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
	})
}

func mustNewSmsClient(c config.Config) *sms.AliSmsClient {
	smsClient, err := sms.NewAliSmsClient(&sms.AliSmsConfig{
		AccessKeyId:     c.AliSms.AccessKeyId,
		AccessKeySecret: c.AliSms.AccessKeySecret,
		SignName:        c.AliSms.SignName,
		TemplateCode:    c.AliSms.TemplateCode,
		Endpoint:        c.AliSms.Endpoint,
	})
	if err != nil {
		panic("初始化SMS客户端失败: " + err.Error())
	}
	return smsClient
}

func newServiceModels(db *gorm.DB) serviceModels {
	areaModel := model.NewAreaModel(db)
	venueModel := model.NewVenueModel(db)
	venueCheckinModel := model.NewVenueCheckinModel(db)
	venueGeocodeTaskModel := model.NewVenueGeocodeTaskModel(db)
	geocodeAccountModel := model.NewGeocodeAccountModel(db)
	favoriteVenueRewardConfigModel := model.NewFavoriteVenueRewardConfigModel(db)
	favoriteVenueRewardRecordModel := model.NewFavoriteVenueRewardRecordModel(db)
	memberSubscriptionOrderModel := model.NewMemberSubscriptionOrderModel(db)
	memberGrowthProfileModel := model.NewMemberGrowthProfileModel(db)
	memberGrowthLogModel := model.NewMemberGrowthLogModel(db)
	memberRightsConfigModel := model.NewMemberRightsConfigModel(db)
	reputationConfigModel := model.NewReputationConfigModel(db)
	userReputationProfileModel := model.NewUserReputationProfileModel(db)
	userReputationLogModel := model.NewUserReputationLogModel(db)

	return serviceModels{
		AreaModel:                       areaModel,
		UserModel:                       model.NewUserModel(db),
		UserDataLifecycleModel:          model.NewUserDataLifecycleModel(db),
		OauthModel:                      model.NewUserOauthModel(db),
		MatchModel:                      model.NewMatchModel(db),
		CompetitiveReadModel:            model.NewCompetitiveReadModel(db),
		RankingModel:                    model.NewRankingModel(db),
		AchievementModel:                model.NewAchievementModel(db),
		UserAchievementModel:            model.NewUserAchievementModel(db),
		UserTitleModel:                  model.NewUserTitleModel(db),
		AchievementProgressEventModel:   model.NewAchievementProgressEventModel(db),
		FriendModel:                     model.NewFriendModel(db),
		FollowModel:                     model.NewFollowModel(db),
		SocialPostModel:                 model.NewSocialPostModel(db),
		NotificationModel:               model.NewNotificationModel(db),
		UserNotificationPreferenceModel: model.NewUserNotificationPreferenceModel(db),
		ChallengeModel:                  model.NewChallengeModel(db),
		TournamentModel:                 model.NewTournamentModel(db),
		TournamentParticipantModel:      model.NewTournamentParticipantModel(db),
		EventNewsModel:                  model.NewEventNewsModel(db),
		PlayerModel:                     model.NewPlayerModel(db),
		RulesContentModel:               model.NewRulesContentModel(db),
		SeasonModel:                     model.NewSeasonModel(db),
		SeasonRecordModel:               model.NewSeasonRecordModel(db),
		SeasonChallengeSnapshotModel:    model.NewSeasonChallengeSnapshotModel(db),
		SeasonSettlementModel:           model.NewSeasonSettlementModel(db),
		VenueModel:                      venueModel,
		VenueCheckinModel:               venueCheckinModel,
		VenueGeocodeTaskModel:           venueGeocodeTaskModel,
		GeocodeAccountModel:             geocodeAccountModel,
		FavoriteVenueRewardConfigModel:  favoriteVenueRewardConfigModel,
		FavoriteVenueRewardRecordModel:  favoriteVenueRewardRecordModel,
		MemberSubscriptionOrderModel:    memberSubscriptionOrderModel,
		MemberGrowthProfileModel:        memberGrowthProfileModel,
		MemberGrowthLogModel:            memberGrowthLogModel,
		MemberRightsConfigModel:         memberRightsConfigModel,
		ReputationConfigModel:           reputationConfigModel,
		UserReputationProfileModel:      userReputationProfileModel,
		UserReputationLogModel:          userReputationLogModel,
		FeedbackTicketModel:             model.NewFeedbackTicketModel(db),
		TournamentMatchModel:            model.NewTournamentMatchModel(db),
		AdminModel:                      model.NewAdminModel(db),
		AdminLoginLogModel:              model.NewAdminLoginLogModel(db),
	}
}

func newGeocodeDependencies(c config.Config, rdb *redis.Client, models serviceModels) geocodeDependencies {
	client := geocode.NewApihzClient("", time.Duration(c.Geocode.RequestTimeoutMs)*time.Millisecond)
	quotaLimiter := geocode.NewRedisQuotaLimiter(rdb, defaultQuotaPrefix, defaultQuotaTTL)
	accountSelector := geocode.NewAccountSelector(quotaLimiter)
	worker := geocode.NewWorker(
		geocode.NewVenueModelRepo(models.VenueModel),
		geocode.NewTaskQueueRepo(models.VenueGeocodeTaskModel),
		geocode.NewTaskStateRepo(models.VenueGeocodeTaskModel),
		geocode.NewAccountRepo(models.GeocodeAccountModel),
		accountSelector,
		client,
		defaultGeocodeSource,
		defaultGeocodeLocker,
		c.Geocode.BatchSize,
		time.Duration(c.Geocode.PollIntervalMs)*time.Millisecond,
		c.Geocode.AutoPublish,
	)

	return geocodeDependencies{
		Client: client,
		Worker: worker,
	}
}

func newQiniuUploadService(c config.Config) *qiniuupload.UploadService {
	service := qiniuupload.NewUploadService(qiniuupload.Config{
		AccessKey:    c.Qiniu.AccessKey,
		SecretKey:    c.Qiniu.SecretKey,
		Bucket:       c.Qiniu.Bucket,
		UploadURL:    c.Qiniu.UploadUrl,
		PublicDomain: c.Qiniu.PublicDomain,
	})
	if !service.Enabled() {
		return nil
	}
	return service
}

func newPushService(c config.Config) *push.PushService {
	return push.NewPushService(push.Config{
		AppId:        c.UniPush.AppId,
		AppKey:       c.UniPush.AppKey,
		MasterSecret: c.UniPush.MasterSecret,
		Enabled:      c.UniPush.Enabled,
	})
}
