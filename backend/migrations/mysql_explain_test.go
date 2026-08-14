package migrations

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

func TestMySQLHotReadPlans(t *testing.T) {
	if os.Getenv("RUN_MYSQL_EXPLAIN_TESTS") != "1" {
		t.Skip("set RUN_MYSQL_EXPLAIN_TESTS=1 to run representative MySQL EXPLAIN checks")
	}
	adminDSN := mysqlTestDSN(t, "")
	admin, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Fatalf("open MySQL admin connection: %v", err)
	}
	defer admin.Close()
	if err := admin.Ping(); err != nil {
		t.Fatalf("ping MySQL: %v", err)
	}

	databaseName := fmt.Sprintf("chasing_points_explain_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + databaseName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create temporary explain database: %v", err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE IF EXISTS `" + databaseName + "`"); err != nil {
			t.Errorf("drop temporary explain database: %v", err)
		}
	}()

	dsn := mysqlTestDSN(t, databaseName)
	command := exec.Command("goose", "-dir", ".", "mysql", dsn, "up")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("apply migrations to temporary explain database: %v: %s", err, output)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open temporary explain database: %v", err)
	}
	defer db.Close()
	seedMySQLExplainData(t, db)

	tests := []struct {
		name           string
		table          string
		expectedKey    string
		rejectFilesort bool
		query          string
	}{
		{
			name: "completed matches", table: "matches", expectedKey: "idx_matches_status_mode_game_completed", rejectFilesort: true,
			query: "SELECT id FROM matches WHERE status=2 AND match_mode='ranked' AND game_type=3 AND completed_at>='2025-01-01' AND completed_at<'2026-01-01' ORDER BY completed_at,id LIMIT 100",
		},
		{
			name: "competitive profile creator score fallback", table: "matches", expectedKey: "idx_matches_user_game_status_mode_score", rejectFilesort: true,
			query: "SELECT id,my_score FROM matches WHERE user_id=1 AND game_type=3 AND status=2 AND match_mode='ranked' AND result IN (1,2) ORDER BY my_score DESC,completed_at DESC,id DESC LIMIT 1",
		},
		{
			name: "competitive profile opponent score fallback", table: "matches", expectedKey: "idx_matches_opponent_game_status_mode_score", rejectFilesort: true,
			query: "SELECT id,opponent_score FROM matches WHERE opponent_id=1 AND user_id<>1 AND game_type=3 AND status=2 AND match_mode='ranked' AND result IN (1,2) ORDER BY opponent_score DESC,completed_at DESC,id DESC LIMIT 1",
		},
		{
			name: "friend requests", table: "friend_requests", expectedKey: "idx_friend_requests_to_status_id", rejectFilesort: true,
			query: "SELECT id FROM friend_requests WHERE to_user_id=1 AND status=0 ORDER BY id DESC LIMIT 20",
		},
		{
			name: "challenge expiry", table: "challenges", expectedKey: "idx_challenges_status_expires_id", rejectFilesort: true,
			query: "SELECT id FROM challenges WHERE status=0 AND expires_at<='2026-01-01' ORDER BY expires_at,id LIMIT 100",
		},
		{
			name: "season leaderboard", table: "r", expectedKey: "idx_season_game_rank", rejectFilesort: true,
			query: "SELECT r.id,r.user_id FROM season_records r LEFT JOIN users u ON u.id=r.user_id WHERE r.season_id=1 AND r.game_type=3 ORDER BY r.end_rank_score DESC,r.wins DESC,r.id DESC LIMIT 20",
		},
		{
			name: "due seasons", table: "s", expectedKey: "idx_seasons_end_id_status_start", rejectFilesort: true,
			query: "SELECT s.id FROM seasons s WHERE s.status IN (0,1) AND s.start_date>='2020-01-01' AND s.end_date<'2060-01-01' AND NOT EXISTS (SELECT 1 FROM season_settlements ss WHERE ss.season_id=s.id AND ss.status='completed') ORDER BY s.end_date,s.id LIMIT 20",
		},
		{
			name: "season challenge archive", table: "achievement_progress_events", expectedKey: "idx_achievement_events_season_archive",
			query: "SELECT user_id,game_type FROM achievement_progress_events FORCE INDEX (idx_achievement_events_season_archive) WHERE metric_key IN ('matches_total','wins_total','tournament_finish_total') AND occurred_at>='2025-01-01' AND occurred_at<'2026-01-01' GROUP BY user_id,game_type ORDER BY user_id,game_type LIMIT 500",
		},
		{
			name: "nearby venues", table: "venues", expectedKey: "idx_venues_nearby_candidates",
			query: "SELECT id,(6371000*acos(cos(radians(31.2304))*cos(radians(latitude))*cos(radians(longitude)-radians(121.4737))+sin(radians(31.2304))*sin(radians(latitude)))) AS distance FROM venues FORCE INDEX (idx_venues_nearby_candidates) WHERE status=1 AND geo_status=1 AND duplicate_of_venue_id IS NULL AND latitude<>0 AND longitude<>0 AND latitude BETWEEN 31.18 AND 31.28 AND longitude BETWEEN 121.42 AND 121.53 HAVING distance<5000 ORDER BY distance LIMIT 20",
		},
		{
			name: "referee history", table: "m", expectedKey: "idx_matches_referee_history", rejectFilesort: true,
			query: "SELECT m.id FROM matches m WHERE m.referee_user_id=1 AND m.deleted_at IS NULL AND m.status IN (2,3) ORDER BY m.end_time DESC,m.id DESC LIMIT 20",
		},
		{
			name: "public matches", table: "m", expectedKey: "idx_matches_public_status_time", rejectFilesort: true,
			query: "SELECT m.id FROM matches m WHERE m.deleted_at IS NULL AND m.visibility='public' AND m.status=1 ORDER BY m.match_time DESC,m.id DESC LIMIT 20",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := explainMySQLQuery(t, db, tt.query, tt.table)
			if plan.Key != tt.expectedKey {
				t.Fatalf("expected key %s, got key=%s type=%s rows=%d extra=%s", tt.expectedKey, plan.Key, plan.AccessType, plan.Rows, plan.Extra)
			}
			if plan.AccessType == "ALL" {
				t.Fatalf("hot read plan must not scan the full table: %+v", plan)
			}
			if tt.rejectFilesort && strings.Contains(plan.Extra, "Using filesort") {
				t.Fatalf("hot read plan must preserve index ordering: %+v", plan)
			}
		})
	}
}

func mysqlTestDSN(t *testing.T, databaseName string) string {
	t.Helper()
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	user := os.Getenv("MYSQL_USER")
	if host == "" || port == "" || user == "" {
		t.Fatal("MYSQL_HOST, MYSQL_PORT and MYSQL_USER are required for MySQL EXPLAIN tests")
	}
	config := mysqlDriver.NewConfig()
	config.User = user
	config.Passwd = os.Getenv("MYSQL_PASSWORD")
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(host, port)
	config.DBName = databaseName
	config.ParseTime = true
	return config.FormatDSN()
}

func seedMySQLExplainData(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		"SET SESSION cte_max_recursion_depth = 25000",
		"CREATE TABLE verify_numbers (n INT NOT NULL PRIMARY KEY)",
		"INSERT INTO verify_numbers WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM seq WHERE n<20000) SELECT n FROM seq",
		"INSERT INTO users (id,nickname,avatar,status,push_token) SELECT n,CONCAT('user-',n),'',1,'' FROM verify_numbers WHERE n<=10000",
		"INSERT INTO matches (id,user_id,opponent_id,opponent_name,game_type,match_mode,visibility,finish_state,referee_user_id,status,result,my_score,opponent_score,match_time,end_time,completed_at,achievement_synced_at) SELECT n,IF(MOD(n,10)=0,1,MOD(n,10000)+1),IF(MOD(n,11)=0,1,MOD(n+1,10000)+1),CONCAT('opponent-',n),3,'ranked',IF(MOD(n,5)=0,'private','public'),'none',IF(MOD(n,10)=0,1,NULL),IF(MOD(n,7)=0,1,2),1,MOD(n,200),MOD(n+37,200),DATE_ADD('2025-01-01',INTERVAL n MINUTE),DATE_ADD('2025-01-01',INTERVAL n+10 MINUTE),DATE_ADD('2025-01-01',INTERVAL n+10 MINUTE),IF(MOD(n,11)=0,NULL,DATE_ADD('2025-01-01',INTERVAL n+11 MINUTE)) FROM verify_numbers",
		"INSERT INTO friend_requests (id,from_user_id,to_user_id,status,message) SELECT n,MOD(n,9999)+1,IF(MOD(n,10)=0,1,MOD(n+1,10000)+1),MOD(n,2),'' FROM verify_numbers",
		"INSERT INTO challenges (id,from_user_id,to_user_id,game_type,message,status,expires_at) SELECT n,MOD(n,9999)+1,IF(MOD(n,10)=0,1,MOD(n+1,10000)+1),3,'',MOD(n,3),DATE_ADD('2025-01-01',INTERVAL n MINUTE) FROM verify_numbers",
		"INSERT INTO seasons (id,name,start_date,end_date,status) SELECT n,CONCAT('S',n),DATE_ADD('2020-01-01',INTERVAL (n-1)*30 DAY),DATE_ADD('2020-01-01',INTERVAL n*30-1 DAY),IF(MOD(n,100)=0,1,0) FROM verify_numbers WHERE n<=500",
		"INSERT INTO season_settlements (season_id,status,completed_at,last_error) SELECT n,IF(MOD(n,2)=0,'completed','running'),IF(MOD(n,2)=0,DATE_ADD('2020-01-01',INTERVAL n*30 DAY),NULL),'' FROM verify_numbers WHERE n<=400",
		"INSERT INTO season_records (season_id,user_id,game_type,start_rank_score,end_rank_score,peak_rank_score,matches_played,wins,final_rank) SELECT 1,n,3,1000,1000+MOD(n,3000),1100+MOD(n,3000),20,MOD(n,20),0 FROM verify_numbers WHERE n<=10000",
		"INSERT INTO achievement_progress_events (id,user_id,source_type,source_id,game_type,metric_key,metric_value,occurred_at) SELECT n,MOD(n,10000)+1,'match',n,3,CASE MOD(n,3) WHEN 0 THEN 'matches_total' WHEN 1 THEN 'wins_total' ELSE 'tournament_finish_total' END,1,DATE_ADD('2025-01-01',INTERVAL n MINUTE) FROM verify_numbers",
		"INSERT INTO venues (id,name,address,city,district,full_address,latitude,longitude,phone,business_hours,price_range,status,geo_status,geo_source,geo_level,geo_error,reject_reason) SELECT n,CONCAT('venue-',n),'address','上海','浦东',CONCAT('上海浦东address-',n),31.2304+(MOD(n,200)-100)/1000.0,121.4737+(MOD(n,200)-100)/1000.0,'','','',IF(MOD(n,5)=0,0,1),1,'seed','','','' FROM verify_numbers",
		"ANALYZE TABLE matches,friend_requests,challenges,seasons,season_settlements,season_records,achievement_progress_events,venues",
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed representative MySQL EXPLAIN data: %v", err)
		}
	}
}

type mysqlExplainPlan struct {
	Table      string
	AccessType string
	Key        string
	Rows       int64
	Extra      string
}

func explainMySQLQuery(t *testing.T, db *sql.DB, query, tableName string) mysqlExplainPlan {
	t.Helper()
	rows, err := db.Query("EXPLAIN " + query)
	if err != nil {
		t.Fatalf("explain hot read query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id sql.NullInt64
		var selectType, table, accessType string
		var partitions, possibleKeys, key, keyLen, ref, extra sql.NullString
		var estimatedRows sql.NullInt64
		var filtered sql.NullFloat64
		if err := rows.Scan(&id, &selectType, &table, &partitions, &accessType, &possibleKeys, &key, &keyLen, &ref, &estimatedRows, &filtered, &extra); err != nil {
			t.Fatalf("scan MySQL EXPLAIN row: %v", err)
		}
		if table == tableName {
			return mysqlExplainPlan{Table: table, AccessType: accessType, Key: key.String, Rows: estimatedRows.Int64, Extra: extra.String}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate MySQL EXPLAIN rows: %v", err)
	}
	t.Fatalf("MySQL EXPLAIN did not include table %s", tableName)
	return mysqlExplainPlan{}
}
