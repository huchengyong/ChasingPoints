package model

import (
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

func TestBuildVenueFullAddress(t *testing.T) {
	got := BuildVenueFullAddress("上海市", "浦东新区", "东明路街道UK台球")
	want := "上海市浦东新区东明路街道UK台球"

	if got != want {
		t.Fatalf("expected full address %q, got %q", want, got)
	}
}

func TestBuildVenueFullAddressSkipsEmptyParts(t *testing.T) {
	got := BuildVenueFullAddress("深圳市", "", "南山区科技园")
	want := "深圳市南山区科技园"

	if got != want {
		t.Fatalf("expected full address %q, got %q", want, got)
	}
}

func TestVenueGeocodeTaskTableName(t *testing.T) {
	var task VenueGeocodeTask
	if got := task.TableName(); got != "venue_geocode_tasks" {
		t.Fatalf("expected table name venue_geocode_tasks, got %s", got)
	}
}

func TestGeocodeAccountTableName(t *testing.T) {
	var account GeocodeAccount
	if got := account.TableName(); got != "geocode_accounts" {
		t.Fatalf("expected table name geocode_accounts, got %s", got)
	}
}

func TestIsDuplicateVenueFullAddressError(t *testing.T) {
	err := &mysqlDriver.MySQLError{
		Number:  1062,
		Message: "Duplicate entry '上海市浦东新区东明路' for key 'uniq_full_address'",
	}

	if !IsDuplicateVenueFullAddressError(err) {
		t.Fatal("expected duplicate full address error to be recognized")
	}
}

func TestIsDuplicateVenueFullAddressErrorRejectsOtherErrors(t *testing.T) {
	err := &mysqlDriver.MySQLError{
		Number:  1062,
		Message: "Duplicate entry '13800138000' for key 'uniq_phone'",
	}

	if IsDuplicateVenueFullAddressError(err) {
		t.Fatal("expected non-full-address duplicate to be ignored")
	}
}
