package wstsync

import (
	"context"
	"fmt"
	"strings"

	"chasing_points/internal/model"
	qiniuupload "chasing_points/internal/pkg/qiniu"

	"github.com/zeromicro/go-zero/core/logx"
)

type WSTImageMirror interface {
	MirrorWSTImage(ctx context.Context, category, sourceURL string) (string, error)
	IsManagedPublicURL(rawURL string) bool
}

func (s *Service) mirrorPreparedImages(
	ctx context.Context,
	players []PlayerUpsertRecord,
	tournaments []TournamentUpsertRecord,
) error {
	if err := s.mirrorPlayerImages(ctx, players); err != nil {
		return err
	}
	return s.mirrorTournamentImages(ctx, tournaments)
}

func (s *Service) mirrorPlayerImages(ctx context.Context, records []PlayerUpsertRecord) error {
	if len(records) == 0 {
		return nil
	}

	sourceIDs := make([]string, 0, len(records))
	for _, record := range records {
		sourceIDs = append(sourceIDs, record.SourcePlayerId)
	}
	existing, err := model.NewPlayerModel(s.svcCtx.DB).FindBySourcePlayerIds(wstSourceType, sourceIDs)
	if err != nil {
		return fmt.Errorf("load existing WST players for image mirror: %w", err)
	}

	for index := range records {
		currentURL := existing[records[index].SourcePlayerId].Avatar
		records[index].Avatar = s.mirrorImageOrFallback(
			ctx,
			qiniuupload.WSTImageCategoryPlayer,
			records[index].Avatar,
			currentURL,
			"player",
			records[index].SourcePlayerId,
		)
	}
	return nil
}

func (s *Service) mirrorTournamentImages(ctx context.Context, records []TournamentUpsertRecord) error {
	if len(records) == 0 {
		return nil
	}

	sourceIDs := make([]string, 0, len(records))
	for _, record := range records {
		sourceIDs = append(sourceIDs, record.SourceTournamentId)
	}
	existing, err := model.NewTournamentModel(s.svcCtx.DB).FindBySourceTournamentIds(wstSourceType, sourceIDs)
	if err != nil {
		return fmt.Errorf("load existing WST tournaments for image mirror: %w", err)
	}

	for index := range records {
		currentURL := existing[records[index].SourceTournamentId].CoverImage
		records[index].CoverImage = s.mirrorImageOrFallback(
			ctx,
			qiniuupload.WSTImageCategoryTournament,
			records[index].CoverImage,
			currentURL,
			"tournament",
			records[index].SourceTournamentId,
		)
	}
	return nil
}

func (s *Service) mirrorImageOrFallback(
	ctx context.Context,
	category string,
	sourceURL string,
	currentURL string,
	resourceType string,
	sourceID string,
) string {
	fallback := ""
	if s.imageMirror != nil && s.imageMirror.IsManagedPublicURL(currentURL) {
		fallback = strings.TrimSpace(currentURL)
	}

	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return fallback
	}
	if s.imageMirror == nil {
		logx.WithContext(ctx).Errorf("WST %s image mirror unavailable: source_id=%s", resourceType, sourceID)
		return fallback
	}

	mirroredURL, err := s.imageMirror.MirrorWSTImage(ctx, category, sourceURL)
	if err != nil {
		logx.WithContext(ctx).Errorf("WST %s image mirror failed: source_id=%s err=%v", resourceType, sourceID, err)
		return fallback
	}
	if !s.imageMirror.IsManagedPublicURL(mirroredURL) {
		logx.WithContext(ctx).Errorf("WST %s image mirror returned unmanaged URL: source_id=%s", resourceType, sourceID)
		return fallback
	}
	return strings.TrimSpace(mirroredURL)
}
