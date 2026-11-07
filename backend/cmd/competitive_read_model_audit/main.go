package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"chasing_points/internal/config"
	"chasing_points/internal/logic"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
)

var (
	configFile = flag.String("f", "etc/chasing_points-api.yaml", "the config file")
	sampleSize = flag.Int("sample-size", 100, "number of completed matches to audit, maximum 1000")
)

func main() {
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	summary, err := logic.NewCompetitiveReadAuditService(svc.NewServiceContext(c)).Audit(context.Background(), *sampleSize)
	if err != nil {
		log.Fatalf("竞技读模型审计失败: %v", err)
	}
	fmt.Printf("competitive read model audit: matches=%d users=%d differences=%d\n", summary.MatchesSampled, summary.UsersSampled, len(summary.Differences))
	for _, difference := range summary.Differences {
		fmt.Printf("difference kind=%s match_id=%d user_id=%d detail=%s\n", difference.Kind, difference.MatchId, difference.UserId, difference.Detail)
	}
}
