package svc

import (
	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/geocode"
	"chasing_points/internal/pkg/push"
	qiniuupload "chasing_points/internal/pkg/qiniu"
	"chasing_points/internal/sms"
	"time"

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
	OauthModel                      *model.UserOauthModel
	MatchModel                      *model.MatchModel
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
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := mustOpenDB(c)
	rdb := newRedisClient(c)
	smsClient := mustNewSmsClient(c)
	codeManager := sms.NewCodeManager(rdb)

	models := newServiceModels(db)
	geocodeDeps := newGeocodeDependencies(c, rdb, models)

	return &ServiceContext{
		Config:                          c,
		DB:                              db,
		Redis:                           rdb,
		SmsClient:                       smsClient,
		CodeManager:                     codeManager,
		AreaModel:                       models.AreaModel,
		UserModel:                       models.UserModel,
		OauthModel:                      models.OauthModel,
		MatchModel:                      models.MatchModel,
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
	}
}

type serviceModels struct {
	AreaModel                       *model.AreaModel
	UserModel                       *model.UserModel
	OauthModel                      *model.UserOauthModel
	MatchModel                      *model.MatchModel
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

func mustOpenDB(c config.Config) *gorm.DB {
	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{})
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
		OauthModel:                      model.NewUserOauthModel(db),
		MatchModel:                      model.NewMatchModel(db),
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
