package user

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/exportcursor"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const personalDataExportFormatVersion = "personal-data-export/v1"

type ExportPersonalDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 导出个人数据
func NewExportPersonalDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExportPersonalDataLogic {
	return &ExportPersonalDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *ExportPersonalDataLogic) ExportPersonalData(req *types.PersonalDataExportReq) (resp *types.PersonalDataExportResp, err error) {
	if req == nil {
		req = &types.PersonalDataExportReq{}
	}
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.PersonalDataExportResp{Success: false, Message: "获取用户信息失败"}, nil
	}
	if l.svcCtx == nil || l.svcCtx.UserDataLifecycleModel == nil || l.svcCtx.UserModel == nil {
		return &types.PersonalDataExportResp{Success: false, Message: "导出服务暂不可用"}, nil
	}
	user, err := l.svcCtx.UserModel.FindByIdWithContext(l.ctx, userID)
	if err != nil {
		l.Logger.Errorf("读取导出用户失败: userId=%d err=%v", userID, err)
		return &types.PersonalDataExportResp{Success: false, Message: "导出服务暂不可用"}, nil
	}
	if user == nil || user.Status != 1 {
		return &types.PersonalDataExportResp{Success: false, Message: "登录状态已失效，请重新登录"}, nil
	}

	signer, err := newPersonalDataExportCursorSigner(l.svcCtx)
	if err != nil {
		l.Logger.Errorf("初始化导出游标签名器失败: %v", err)
		return &types.PersonalDataExportResp{Success: false, Message: "导出服务暂不可用"}, nil
	}

	categories := model.PersonalDataExportCategories()
	categoryIndex := 0
	afterID := int64(0)
	snapshotAt := time.Now()
	if cursor := strings.TrimSpace(req.Cursor); cursor != "" {
		claims, verifyErr := signer.Verify(cursor)
		if verifyErr != nil || claims.UserID != userID || claims.FormatVersion != personalDataExportFormatVersion {
			return &types.PersonalDataExportResp{Success: false, Message: "导出请求已失效，请重新开始"}, nil
		}
		categoryIndex = personalDataExportCategoryIndex(categories, claims.Category)
		if categoryIndex < 0 || claims.SnapshotUnixNano > time.Now().Add(time.Minute).UnixNano() {
			return &types.PersonalDataExportResp{Success: false, Message: "导出请求已失效，请重新开始"}, nil
		}
		snapshotAt = time.Unix(0, claims.SnapshotUnixNano)
		afterID = claims.LastID
	}

	items := make([]types.PersonalDataExportItem, 0, model.PersonalDataExportMaxBatchSize)
	for categoryIndex < len(categories) && len(items) < model.PersonalDataExportMaxBatchSize {
		category := categories[categoryIndex]
		remaining := model.PersonalDataExportMaxBatchSize - len(items)
		records, hasMore, listErr := l.svcCtx.UserDataLifecycleModel.ListExportRecords(userID, category, afterID, snapshotAt, remaining)
		if listErr != nil {
			l.Logger.Errorf("读取个人数据导出分段失败: userId=%d category=%s err=%v", userID, category, listErr)
			return &types.PersonalDataExportResp{Success: false, Message: "导出服务暂不可用"}, nil
		}
		for _, record := range records {
			dataJSON, marshalErr := json.Marshal(record.Data)
			if marshalErr != nil {
				l.Logger.Errorf("序列化个人数据导出分段失败: userId=%d category=%s id=%d err=%v", userID, category, record.ID, marshalErr)
				return &types.PersonalDataExportResp{Success: false, Message: "导出服务暂不可用"}, nil
			}
			items = append(items, types.PersonalDataExportItem{
				Category: category,
				Id:       formatPersonalDataExportID(record.ID),
				DataJson: string(dataJSON),
			})
			afterID = record.ID
		}
		if hasMore {
			return l.personalDataExportPage(userID, signer, snapshotAt, category, afterID, items)
		}
		categoryIndex++
		afterID = 0
		if len(items) >= model.PersonalDataExportMaxBatchSize && categoryIndex < len(categories) {
			return l.personalDataExportPage(userID, signer, snapshotAt, categories[categoryIndex], 0, items)
		}
	}

	return &types.PersonalDataExportResp{
		Success:       true,
		FormatVersion: personalDataExportFormatVersion,
		SnapshotAt:    snapshotAt.Format(time.RFC3339),
		Items:         items,
		Complete:      true,
		ItemCount:     len(items),
	}, nil
}

func newPersonalDataExportCursorSigner(svcCtx *svc.ServiceContext) (*exportcursor.Signer, error) {
	if svcCtx == nil {
		return nil, exportcursor.ErrSignerConfig
	}
	return exportcursor.NewSigner(svcCtx.Config.Security.MatchInvite.SigningSecret)
}

func (l *ExportPersonalDataLogic) personalDataExportPage(userID int64, signer *exportcursor.Signer, snapshotAt time.Time, category string, afterID int64, items []types.PersonalDataExportItem) (*types.PersonalDataExportResp, error) {
	cursor, err := signer.Issue(exportcursor.Claims{
		FormatVersion:    personalDataExportFormatVersion,
		UserID:           userID,
		SnapshotUnixNano: snapshotAt.UnixNano(),
		Category:         category,
		LastID:           afterID,
	})
	if err != nil {
		l.Logger.Errorf("签发个人数据导出游标失败: userId=%d category=%s err=%v", userID, category, err)
		return &types.PersonalDataExportResp{Success: false, Message: "导出服务暂不可用"}, nil
	}
	return &types.PersonalDataExportResp{
		Success:       true,
		FormatVersion: personalDataExportFormatVersion,
		SnapshotAt:    snapshotAt.Format(time.RFC3339),
		Items:         items,
		NextCursor:    cursor,
		Complete:      false,
		ItemCount:     len(items),
	}, nil
}

func personalDataExportCategoryIndex(categories []string, category string) int {
	for index, candidate := range categories {
		if candidate == category {
			return index
		}
	}
	return -1
}

func formatPersonalDataExportID(id int64) string {
	return strconv.FormatInt(id, 10)
}
