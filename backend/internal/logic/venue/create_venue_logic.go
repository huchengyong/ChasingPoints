package venue

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CreateVenueLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建球馆
func NewCreateVenueLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateVenueLogic {
	return &CreateVenueLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CreateVenueLogic) CreateVenue(req *types.CreateVenueReq) (resp *types.CreateVenueResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CreateVenueResp{Success: false, Message: "登录状态已失效"}, nil
	}

	if err := validateCreateVenueReq(req); err != nil {
		return &types.CreateVenueResp{Success: false, Message: err.Error()}, nil
	}

	maxAttempts := l.svcCtx.Config.Geocode.MaxAttempts
	venue, task, err := buildVenueAndTaskFromReq(userIdInt, req, maxAttempts, l.svcCtx.Config.Geocode.AutoPublish, time.Now())
	if err != nil {
		return &types.CreateVenueResp{Success: false, Message: err.Error()}, nil
	}

	existingVenue, err := l.svcCtx.VenueModel.FindByFullAddress(venue.FullAddress)
	if err != nil {
		l.Logger.Errorf("查询重复球馆失败: fullAddress=%s err=%v", venue.FullAddress, err)
		return &types.CreateVenueResp{Success: false, Message: "提交失败，请稍后再试"}, nil
	}
	if existingVenue != nil {
		return &types.CreateVenueResp{Success: false, Message: "该地址已存在，请勿重复上传"}, nil
	}

	if err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := l.svcCtx.VenueModel.CreateWithTx(tx, venue); err != nil {
			return err
		}
		if task == nil {
			return nil
		}
		task.VenueId = venue.Id
		if err := l.svcCtx.VenueGeocodeTaskModel.CreateWithTx(tx, task); err != nil {
			return err
		}
		return nil
	}); err != nil {
		if model.IsDuplicateVenueFullAddressError(err) {
			return &types.CreateVenueResp{Success: false, Message: "该地址已存在，请勿重复上传"}, nil
		}
		l.Logger.Errorf("创建球馆及地理解析任务失败: userId=%d err=%v", userIdInt, err)
		return &types.CreateVenueResp{Success: false, Message: "提交失败，请稍后再试"}, nil
	}
	if err := BumpVenueCacheVersion(l.ctx, l.svcCtx); err != nil {
		l.Logger.Errorf("失效球馆缓存失败: venueId=%d err=%v", venue.Id, err)
	}

	return buildCreateVenueResp(venue.Id), nil
}

func validateCreateVenueReq(req *types.CreateVenueReq) error {
	if req == nil {
		return errors.New("请求不能为空")
	}
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("请输入球馆名称")
	}
	if strings.TrimSpace(req.City) == "" {
		return errors.New("请输入城市")
	}
	if strings.TrimSpace(req.Address) == "" {
		return errors.New("请输入详细地址")
	}
	if _, err := validateCreateVenueLocation(req.Latitude, req.Longitude); err != nil {
		return err
	}
	return nil
}

// validateCreateVenueLocation 判断小程序是否地图选点以及坐标是否可用。
func validateCreateVenueLocation(latitude, longitude float64) (bool, error) {
	if latitude == 0 && longitude == 0 {
		return false, nil
	}
	if latitude == 0 || longitude == 0 {
		return false, errors.New("位置信息不完整，请重新选点")
	}
	if math.IsNaN(latitude) || math.IsNaN(longitude) || math.IsInf(latitude, 0) || math.IsInf(longitude, 0) ||
		latitude < 3.8 || latitude > 53.6 || longitude < 73.4 || longitude > 135.1 {
		return false, errors.New("选点不在服务范围内，请重新选点")
	}
	return true, nil
}

func buildVenueAndTaskFromReq(userId int64, req *types.CreateVenueReq, maxAttempts int, autoPublish bool, now time.Time) (*model.Venue, *model.VenueGeocodeTask, error) {
	if err := validateCreateVenueReq(req); err != nil {
		return nil, nil, err
	}

	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	venue := &model.Venue{
		Name:          strings.TrimSpace(req.Name),
		Address:       strings.TrimSpace(req.Address),
		City:          strings.TrimSpace(req.City),
		District:      strings.TrimSpace(req.District),
		FullAddress:   model.BuildVenueFullAddress(req.City, req.District, req.Address),
		Latitude:      0,
		Longitude:     0,
		Phone:         "",
		Images:        "[]",
		BusinessHours: "",
		TableCount:    0,
		PriceRange:    "",
		Description:   "",
		OwnerUserId:   userId,
		Status:        model.VenueStatusPending, // 用户提交的球馆默认为待审核状态
		GeoStatus:     model.VenueGeoStatusPending,
	}

	hasLocation, err := validateCreateVenueLocation(req.Latitude, req.Longitude)
	if err != nil {
		return nil, nil, err
	}
	if hasLocation {
		// 用户在地图上亲手选的点按最高精度处理，不再走后台地址解析。
		venue.Latitude = req.Latitude
		venue.Longitude = req.Longitude
		venue.GeoStatus = model.VenueGeoStatusSuccess
		venue.GeoSource = "user_pick"
		venue.GeoLevel = "用户选点"
		venue.GeoUpdatedAt = &now
		if autoPublish {
			venue.Status = model.VenueStatusPublished
		}
		return venue, nil, nil
	}

	task := &model.VenueGeocodeTask{
		Status:      model.VenueGeocodeTaskStatusPending,
		Attempts:    0,
		MaxAttempts: maxAttempts,
	}

	return venue, task, nil
}

func buildCreateVenueResp(venueId int64) *types.CreateVenueResp {
	return &types.CreateVenueResp{
		Success: true,
		VenueId: venueId,
		Message: "已提交，审核通过后会展示给附近球友",
	}
}
