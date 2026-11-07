package match

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/types"
	"gorm.io/gorm"
)

func TestDegradedFinishSnapshotKeepsViewerScores(t *testing.T) {
	for _, userID := range []int64{1001, 2002} {
		name := "player1"
		if userID == 2002 {
			name = "player2"
		}
		t.Run(name, func(t *testing.T) {
			svcCtx, m, _ := reviewV9PostCommitFixture(t, false)
			failed := false
			if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("review_v10_snapshot", func(tx *gorm.DB) {
				if !failed && tx.Statement.Table == "match_rounds" {
					failed = true
					tx.AddError(errors.New("injected snapshot round read failure"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			response, err := NewFinishMatchLogic(startReputationCtx(userID), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: m.Id, ClientActionId: "v10-degraded", BaseRevision: 0})
			if err != nil || response == nil || !response.Success || !failed {
				t.Fatalf("degraded finish not reached: response=%+v err=%v failed=%v", response, err, failed)
			}
			stored, err := svcCtx.MatchModel.FindById(m.Id)
			if err != nil || stored == nil || stored.Status != 2 {
				t.Fatalf("not committed: %+v err=%v", stored, err)
			}
			snapshot := response.Snapshot
			t.Logf("viewer=%s success=%v revision=%d top-level score=%d:%d snapshot score=%d:%d role=%q completed_by=%d source=%q", name, response.Success, response.ServerRevision, response.MyScore, response.OpponentScore, snapshot.MyScore, snapshot.OpponentScore, snapshot.ViewerRole, snapshot.CompletedByUserId, snapshot.CompletionSource)
			if userID == 2002 {
				raw, err := json.Marshal(response)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile("/tmp/quick-match-review-v10-degraded-response.json", raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if snapshot.MyScore != response.MyScore || snapshot.OpponentScore != response.OpponentScore {
				t.Error("successful authoritative snapshot must preserve viewer-perspective scores; App prefers snapshot over top-level scores")
			}
			if snapshot.ViewerRole != name || snapshot.CompletedByUserId != userID || snapshot.CompletionSource != model.CompletionSourcePlayerDirect {
				t.Error("known match identity and completion metadata must not be zeroed in degraded snapshot")
			}
		})
	}
}

func TestSnookerFinishReloadFailureStillNotifies(t *testing.T) {
	for _, inject := range []bool{false, true} {
		name := "healthy"
		if inject {
			name = "reload_failure"
		}
		t.Run(name, func(t *testing.T) {
			svcCtx := newSnookerActionTestSvc(t)
			if err := svcCtx.DB.AutoMigrate(&model.Challenge{}, &model.Notification{}); err != nil {
				t.Fatal(err)
			}
			svcCtx.ChallengeModel = model.NewChallengeModel(svcCtx.DB)
			svcCtx.NotificationModel = model.NewNotificationModel(svcCtx.DB)
			m := seedSnookerActionMatch(t, svcCtx, 88, 1, 303)
			challengeID := int64(77)
			if err := svcCtx.ChallengeModel.Create(&model.Challenge{Id: challengeID, FromUserId: 101, ToUserId: 202, Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
			if err := svcCtx.DB.Model(&model.Match{}).Where("id = ?", m.Id).Update("challenge_id", challengeID).Error; err != nil {
				t.Fatal(err)
			}
			old := ws.GlobalHub
			hub := ws.NewHub()
			ws.GlobalHub = hub
			t.Cleanup(func() { ws.GlobalHub = old })
			readCount := 0
			failed := false
			if inject {
				if err := svcCtx.DB.Callback().Query().Before("gorm:query").Register("review_v10_snooker_reload", func(tx *gorm.DB) {
					_, inTx := tx.Statement.ConnPool.(*sql.Tx)
					if !inTx && tx.Statement.Table == "matches" {
						readCount++
						if readCount == 2 {
							failed = true
							tx.AddError(errors.New("injected post-commit snooker match read failure"))
						}
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			response, err := NewSnookerFrameActionLogic(matchLogicCtx(303), svcCtx).SnookerFrameAction(&types.SnookerFrameActionReq{MatchId: m.Id, Actor: 1, Action: model.SnookerFrameActionAwardFrame, Winner: 1, Reason: "裁判判局", ClientActionId: "v10-award-match", BaseRevision: 0})
			stored, readErr := svcCtx.MatchModel.FindById(m.Id)
			linked, linkErr := svcCtx.ChallengeModel.FindById(challengeID)
			if err != nil || readErr != nil || linkErr != nil || stored == nil || stored.Status != 2 || linked == nil || linked.Status != model.ChallengeStatusCompleted || (inject && !failed) {
				t.Fatalf("fixture not committed: response=%+v match=%+v challenge=%+v errors=%v/%v/%v failed=%v", response, stored, linked, err, readErr, linkErr, failed)
			}
			invalidations := drainChallengeInvalidations(hub)
			ends := reviewV9DrainMatchEnd(hub)
			var notifications int64
			if err := svcCtx.DB.Model(&model.Notification{}).Where("type = ?", "match_result").Count(&notifications).Error; err != nil {
				t.Fatal(err)
			}
			t.Logf("response.success=%v db.status=%d challenge.status=%d challenge events=%d match_end=%d notifications=%d", response.Success, stored.Status, linked.Status, invalidations, len(ends), notifications)
			if invalidations != 2 || len(ends) != 1 || notifications != 2 {
				t.Error("committed snooker finish must retain notifications even if match reload fails, as pool auto-finish already does")
			}
		})
	}
}

func TestDegradedSnapshotPreservesKnownSnookerSettings(t *testing.T) {
	for _, inject := range []bool{false, true} {
		name := "healthy"
		if inject {
			name = "degraded"
		}
		t.Run(name, func(t *testing.T) {
			svc := newSnookerActionTestSvc(t)
			if err := svc.DB.AutoMigrate(&model.Challenge{}, &model.Notification{}); err != nil {
				t.Fatal(err)
			}
			svc.ChallengeModel = model.NewChallengeModel(svc.DB)
			svc.NotificationModel = model.NewNotificationModel(svc.DB)
			m := seedSnookerActionMatch(t, svc, 88, 0, 303)
			challengeID := int64(77)
			if err := svc.ChallengeModel.Create(&model.Challenge{Id: challengeID, FromUserId: 101, ToUserId: 202, Status: model.ChallengeStatusStarted, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
			if err := svc.DB.Model(&model.Match{}).Where("id = ?", m.Id).Updates(map[string]interface{}{"challenge_id": challengeID, "snooker_format": model.SnookerFormatFree, "starting_actor": 2}).Error; err != nil {
				t.Fatal(err)
			}
			frame, err := NewSnookerFrameActionLogic(matchLogicCtx(303), svc).SnookerFrameAction(&types.SnookerFrameActionReq{MatchId: m.Id, Actor: 1, Action: model.SnookerFrameActionAwardFrame, Winner: 1, Reason: "裁判判局", ClientActionId: "v11-frame", BaseRevision: 0})
			if err != nil || frame == nil || !frame.Success || frame.Snapshot.Status != 1 || frame.Snapshot.CurrentFrameStarted {
				t.Fatalf("failed to record completed frame in free match: %+v err=%v", frame, err)
			}
			failed := false
			if inject {
				if err := svc.DB.Callback().Query().Before("gorm:query").Register("review_v11_degraded_snooker", func(tx *gorm.DB) {
					_, inTx := tx.Statement.ConnPool.(*sql.Tx)
					if !inTx && !failed && tx.Statement.Table == "match_rounds" {
						failed = true
						tx.AddError(errors.New("injected post-commit snapshot read failure"))
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			response, err := NewFinishMatchLogic(matchLogicCtx(101), svc).FinishMatch(&types.FinishMatchReq{MatchId: m.Id, ClientActionId: "v11-finish", BaseRevision: frame.ServerRevision})
			if err != nil || response == nil || !response.Success || (inject && !failed) {
				t.Fatalf("expected committed finish: response=%+v err=%v injected=%v", response, err, failed)
			}
			stored, err := svc.MatchModel.FindById(m.Id)
			if err != nil || stored == nil || stored.Status != 2 {
				t.Fatalf("expected terminal DB match: %+v err=%v", stored, err)
			}
			snapshot := response.Snapshot
			t.Logf("success=%v DB rule_version=%d format=%q starting_actor=%d; snapshot rule_version=%d format=%q starting_actor=%d", response.Success, stored.SnookerRulesVersion, stored.SnookerFormat, stored.StartingActor, snapshot.SnookerRulesVersion, snapshot.SnookerFormat, snapshot.StartingActor)
			if inject {
				raw, err := json.Marshal(response)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile("/tmp/quick-match-review-v11-snooker-response.json", raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if snapshot.SnookerRulesVersion != stored.SnookerRulesVersion || snapshot.SnookerFormat != stored.SnookerFormat || snapshot.StartingActor != stored.StartingActor {
				t.Error("settings already available on the committed match must not be replaced by authoritative zero values in a successful snapshot")
			}
		})
	}
}
