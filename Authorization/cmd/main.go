package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"

	hand "github.com/Wladim1r/auth/internal/api/handlers"
	repo "github.com/Wladim1r/auth/internal/api/repository"
	serv "github.com/Wladim1r/auth/internal/api/service"
	"github.com/Wladim1r/auth/periferia/db"
	"google.golang.org/grpc"
)

func main() {
	db := db.MustLoad()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	wg := new(sync.WaitGroup)

	uRepo, tRepo := repo.NewRepositories(db)
	uRepo.CreateTable()
	uServ, tServ := serv.NewServices(uRepo, tRepo)

	listen, err := net.Listen("tcp", ":50051")
	if err != nil {
		panic(err)
	}

	svr := grpc.NewServer()

	hand.RegisterServer(svr, uServ, tServ, uRepo)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := svr.Serve(listen); err != nil {
			slog.Error("Failed to listening server")
		}
	}()

	<-c
	wg.Wait()
}
