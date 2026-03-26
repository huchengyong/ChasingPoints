package logic

import (
	"context"
	"errors"
	"strings"

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
		svcCtx: svcCtx,
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
	venue, task, err := buildVenueAndTaskFromReq(userIdInt, req, maxAttempts)
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
	return nil
}

func buildVenueAndTaskFromReq(userId int64, req *types.CreateVenueReq, maxAttempts int) (*model.Venue, *model.VenueGeocodeTask, error) {
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
		Message: "已提交，审核通过后会员将自动到账",
	}
}
