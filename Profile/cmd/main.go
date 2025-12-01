package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"

	hand "github.com/Wladim1r/profile/internal/api/handlers"
	"github.com/Wladim1r/profile/lib/getenv"
	"github.com/Wladim1r/proto-crypto/gen/protos/auth-portfile"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// db := db.MustLoad()

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

	hand := hand.NewClient(authConn)

	r := gin.Default()

	r.POST("/register", hand.Registration)
	r.POST("/login", hand.Login)

	auth := r.Group("/auth")
	{
		auth.POST("/test", hand.Test)
		auth.POST("/refresh", hand.Refresh)
		auth.POST("/logout", hand.Logout)
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
