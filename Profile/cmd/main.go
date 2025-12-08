package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"

	hand "github.com/Wladim1r/profile/internal/api/auth/handlers"
	handler "github.com/Wladim1r/profile/internal/api/profile/handlers"
	"github.com/Wladim1r/profile/internal/api/profile/repository"
	"github.com/Wladim1r/profile/internal/api/profile/service"
	"github.com/Wladim1r/profile/lib/getenv"
	"github.com/Wladim1r/profile/lib/midware"
	"github.com/Wladim1r/profile/periferia/db"
	"github.com/Wladim1r/proto-crypto/gen/protos/auth-portfile"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	db := db.MustLoad()
	uRepo, cRepo := repository.NewProfileRepository(db)
	uRepo.CreateTables()
	uServ, cServ := service.NewProfileService(uRepo, cRepo)
	handler := handler.NewHandler(uServ, cServ)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	wg := new(sync.WaitGroup)

	conn, err := grpc.NewClient(
		getenv.GetString("GRPC_ADDR", "localhost:50051"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}

	authConn := auth.NewAuthClient(conn)

	hand := hand.NewClient(authConn, uServ)

	r := gin.Default()

	v1 := r.Group("/v1")
	{
		v1.POST("/register", hand.Registration)
		v1.POST("/login", hand.Login)

		v1.POST("/refresh", midware.CheckAuth(true), hand.Refresh)

		auth := v1.Group("/auth")
		auth.Use(midware.CheckAuth(false))
		{
			auth.POST("/test", hand.Test)
			auth.POST("/logout", hand.Logout)
		}
	}

	v2 := r.Group("/v2")
	v2.Use(midware.CheckAuth(false))
	{
		coins := v2.Group("/coin")
		{
			coins.GET("/symbols", handler.GetCoins)
			coins.POST("/symbol", handler.AddCoin)
			coins.PATCH("/symbol", handler.UpdateCoin)
			coins.DELETE("/symbol", handler.DeleteCoin)
		}

		user := v2.Group("/user")
		{
			user.GET("/profile", handler.GetUserProfile)
			user.DELETE("/profile", handler.DeleteUserProfile)
		}
	}

	server := http.Server{
		Addr:    getenv.GetString("SERVER_ADDR", ":8080"),
		Handler: r,
	}

	wg.Add(1)
	go server.ListenAndServe()

	<-c
	wg.Wait()
}
