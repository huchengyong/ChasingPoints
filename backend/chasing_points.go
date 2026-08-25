package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"chasing_points/internal/config"
	"chasing_points/internal/handler"
	logicx "chasing_points/internal/logic"
	matchlogic "chasing_points/internal/logic/match"
	tournamentlogic "chasing_points/internal/logic/tournament"
	"chasing_points/internal/logic/wstsync"
	"chasing_points/internal/middleware"
	"chasing_points/internal/pkg/httperror"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/chasing_points-api.yaml", "the config file")

func main() {
	flag.Parse()

	// 加载 .env 文件中的环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	if err := c.ValidateProductionSecurity(); err != nil {
		log.Fatalf("生产安全配置校验失败: %v", err)
	}
	httperror.Configure()

	server := newAPIServer(c)
	defer server.Stop()

	svcCtx := svc.NewServiceContext(c)
	if err := logicx.EnsureDefaultRulesContent(svcCtx); err != nil {
		log.Fatalf("初始化规则内容失败: %v", err)
	}
	handler.RegisterHandlers(server, svcCtx)

	// 注册全局中间件
	server.Use(middleware.RequestContextMiddleware)
	server.Use(middleware.RequestObservabilityMiddleware)
	server.Use(middleware.NewOptionalPublicJWTMiddleware(c.Auth.AccessSecret).Handle)
	server.Use(middleware.NewActiveUserSessionMiddleware(svcCtx.UserModel).Handle)

	// 初始化 WebSocket Hub
	ws.InitGlobalHub()

	// 注册 WebSocket 路由
	server.AddRoute(rest.Route{
		Method:  "GET",
		Path:    "/api/match/ws",
		Handler: ws.MatchWSHandler(svcCtx),
	})

	// 注册用户级 WebSocket 路由（用于匹配通知）
	server.AddRoute(rest.Route{
		Method:  "GET",
		Path:    "/api/user/ws",
		Handler: ws.UserWSHandler(svcCtx),
	})

	if c.Geocode.WorkerEnabled && svcCtx.GeocodeWorker != nil {
		go svcCtx.GeocodeWorker.Start(context.Background())
	}
	if c.WSTSync.Enabled {
		go wstsync.NewAutoSyncWorker(svcCtx, c.WSTSync).Start(context.Background())
		if c.WSTSync.HotEnabled {
			go wstsync.NewHotAutoSyncWorker(svcCtx, c.WSTSync).Start(context.Background())
		}
	}
	go logicx.NewSeasonRolloverWorker(svcCtx).Start(context.Background())
	go logicx.NewChallengeExpiryWorker(svcCtx).Start(context.Background())
	go matchlogic.NewFinishRequestExpiryWorker(svcCtx).Start(context.Background())
	go matchlogic.NewAchievementSyncWorker(svcCtx).Start(context.Background())
	go tournamentlogic.NewTournamentBracketGenerationWorker(svcCtx).Start(context.Background())

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.StartWithOpts(withWechatMessagePushRoute(c))
}

func newAPIServer(c config.Config) *rest.Server {
	return rest.MustNewServer(c.RestConf, rest.WithCors(c.Security.HTTPOriginList()...))
}
