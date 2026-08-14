package main

import (
	"flag"
	"fmt"
	"log"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

var (
	configFile = flag.String("f", "etc/chasing_points-api.yaml", "the config file")
	batchSize  = flag.Int("batch-size", 200, "users per batch")
	dryRun     = flag.Bool("dry-run", false, "only count users missing ranking profiles")
)

func main() {
	flag.Parse()
	if *batchSize <= 0 || *batchSize > 1000 {
		*batchSize = 200
	}
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	svcCtx := svc.NewServiceContext(c)

	cursor, users, repaired := int64(0), 0, 0
	for {
		ids, err := svcCtx.UserModel.ListIDsAfter(cursor, *batchSize)
		if err != nil {
			log.Fatalf("读取用户批次失败: %v", err)
		}
		if len(ids) == 0 {
			break
		}
		for _, userID := range ids {
			cursor = userID
			users++
			rankings, err := svcCtx.RankingModel.ListByUserId(userID)
			if err != nil {
				log.Fatalf("读取用户段位失败: user=%d err=%v", userID, err)
			}
			if len(rankings) == len(model.SupportedRankingGameTypes()) {
				continue
			}
			repaired++
			if !*dryRun {
				if _, err := svcCtx.RankingModel.FindOrCreateByGameTypes(userID); err != nil {
					log.Fatalf("补齐用户段位失败: user=%d err=%v", userID, err)
				}
			}
		}
		if len(ids) < *batchSize {
			break
		}
	}
	fmt.Printf("rank profile repair completed: dry_run=%t users=%d repaired=%d\n", *dryRun, users, repaired)
}
