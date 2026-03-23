package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"billiard_master/internal/config"
	"billiard_master/internal/handler"
	"billiard_master/internal/middleware"
	"billiard_master/internal/pkg/ws"
	"billiard_master/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/billiard_master-api.yaml", "the config file")

func main() {
	flag.Parse()

	// 加载 .env 文件中的环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	server := rest.MustNewServer(c.RestConf, rest.WithCors("*", "http://localhost:3000", "https://admin-bm.dianzaozao.com"))
	defer server.Stop()

	svcCtx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, svcCtx)

	// 注册全局中间件
	server.Use(middleware.RequestContextMiddleware)

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

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
