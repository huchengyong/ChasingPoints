package challenge

import (
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
)

func beijingTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, logicx.UTC8Location)
	if err != nil {
		t.Fatalf("parse beijing time: %v", err)
	}
	return parsed
}

func TestValidateChallengeScheduleAcceptsValidSlots(t *testing.T) {
	now := beijingTime(t, "2026-08-25 15:20")
	scheduled, expiresAt, err := ValidateChallengeSchedule(ChallengeScheduleInput{ScheduledDate: "2026-08-25", StartHour: 16, EndHour: 17}, now)
	if err != nil {
		t.Fatalf("valid slot rejected: %v", err)
	}
	if want := beijingTime(t, "2026-08-25 00:00"); !scheduled.Equal(want) {
		t.Fatalf("unexpected scheduled date: %v", scheduled)
	}
	if want := beijingTime(t, "2026-08-26 07:00"); !expiresAt.Equal(want) {
		t.Fatalf("unexpected expires: %v", expiresAt)
	}
}

func TestValidateChallengeScheduleLateNight24NotExtended(t *testing.T) {
	now := beijingTime(t, "2026-08-25 22:00")
	scheduled, expiresAt, err := ValidateChallengeSchedule(ChallengeScheduleInput{ScheduledDate: "2026-08-25", StartHour: 23, EndHour: 24}, now)
	if err != nil {
		t.Fatalf("23-24 slot rejected: %v", err)
	}
	if want := beijingTime(t, "2026-08-26 07:00"); !expiresAt.Equal(want) {
		t.Fatalf("23-24 must expire next 07:00 not day after: %v", expiresAt)
	}
	_ = scheduled
}

func TestValidateChallengeScheduleRejectsInvalidInputs(t *testing.T) {
	now := beijingTime(t, "2026-08-25 15:20")
	cases := []ChallengeScheduleInput{
		{ScheduledDate: "2026-08-28", StartHour: 16, EndHour: 17},
		{ScheduledDate: "2026-08-25", StartHour: 17, EndHour: 17},
		{ScheduledDate: "2026-08-25", StartHour: 17, EndHour: 16},
		{ScheduledDate: "2026-08-25", StartHour: -1, EndHour: 2},
		{ScheduledDate: "2026-08-25", StartHour: 16, EndHour: 25},
		{ScheduledDate: "", StartHour: 16, EndHour: 17},
		{ScheduledDate: "2026-08-24", StartHour: 16, EndHour: 17},
	}
	for index, input := range cases {
		if _, _, err := ValidateChallengeSchedule(input, now); err == nil {
			t.Fatalf("case %d must be rejected: %+v", index, input)
		}
	}
}

func TestValidateChallengeScheduleRejectsFullyPastInterval(t *testing.T) {
	now := beijingTime(t, "2026-08-25 17:30")
	if _, _, err := ValidateChallengeSchedule(ChallengeScheduleInput{ScheduledDate: "2026-08-25", StartHour: 16, EndHour: 17}, now); err == nil {
		t.Fatal("fully past interval must be rejected")
	}
	// 仍在进行中的区间（17点开始，18点结束）允许。
	if _, _, err := ValidateChallengeSchedule(ChallengeScheduleInput{ScheduledDate: "2026-08-25", StartHour: 17, EndHour: 18}, now); err != nil {
		t.Fatalf("ongoing interval must be allowed: %v", err)
	}
}

func TestValidateChallengeScheduleKeepsActualDateAcrossMidnight(t *testing.T) {
	// 跨午夜后 App 仍提交旧的「今天」日期：已成昨天，拒绝，不静默滚到新的今天。
	now := beijingTime(t, "2026-08-26 00:30")
	if _, _, err := ValidateChallengeSchedule(ChallengeScheduleInput{ScheduledDate: "2026-08-25", StartHour: 16, EndHour: 17}, now); err == nil {
		t.Fatal("stale date across midnight must be rejected")
	}
	// 跨午夜前选的「明天」按实际日期保存，不改成另一天。
	scheduled, expiresAt, err := ValidateChallengeSchedule(ChallengeScheduleInput{ScheduledDate: "2026-08-26", StartHour: 16, EndHour: 17}, now)
	if err != nil {
		t.Fatalf("actual date still in range must be accepted: %v", err)
	}
	if want := beijingTime(t, "2026-08-26 00:00"); !scheduled.Equal(want) {
		t.Fatalf("scheduled date must stay as submitted: %v", scheduled)
	}
	if want := beijingTime(t, "2026-08-27 07:00"); !expiresAt.Equal(want) {
		t.Fatalf("unexpected expires: %v", expiresAt)
	}
}

func TestValidateChallengeScheduleAllowsDayAfterTomorrowOnly(t *testing.T) {
	now := beijingTime(t, "2026-08-25 15:20")
	if _, _, err := ValidateChallengeSchedule(ChallengeScheduleInput{ScheduledDate: "2026-08-27", StartHour: 0, EndHour: 1}, now); err != nil {
		t.Fatalf("day after tomorrow must be accepted: %v", err)
	}
	if _, _, err := ValidateChallengeSchedule(ChallengeScheduleInput{ScheduledDate: "2026-08-28", StartHour: 0, EndHour: 1}, now); err == nil {
		t.Fatal("three days ahead must be rejected")
	}
}

func TestChallengeScheduledStartTime(t *testing.T) {
	scheduled := beijingTime(t, "2026-08-26 00:00")
	start := ChallengeScheduledStartTime(scheduled, 16)
	if want := beijingTime(t, "2026-08-26 16:00"); !start.Equal(want) {
		t.Fatalf("unexpected start time: %v", start)
	}
}

func TestChallengeScheduleExpiredBoundary(t *testing.T) {
	expires := beijingTime(t, "2026-08-26 07:00")
	if ChallengeScheduleExpired(expires, beijingTime(t, "2026-08-26 06:59")) {
		t.Fatal("06:59 must not be expired")
	}
	if !ChallengeScheduleExpired(expires, beijingTime(t, "2026-08-26 07:00")) {
		t.Fatal("07:00 must be expired")
	}
}
