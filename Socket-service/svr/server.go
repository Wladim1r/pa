package svr

import (
	"context"
	"log/slog"
	"net"
	"sync"

	"github.com/Wladim1r/proto-crypto/gen/socket"
	"google.golang.org/grpc"
)

const (
	port = "0.0.0.0:50051"
)

type server struct {
	socket.UnimplementedSocketServiceServer
	outputChan <-chan []byte
	mainCtx    context.Context
}

func register(gRPC *grpc.Server, outChan chan []byte, ctx context.Context) {
	socket.RegisterSocketServiceServer(gRPC, &server{
		outputChan: outChan,
		mainCtx:    ctx,
	})
}

func (s *server) ReceiveRawMsg(
	req *socket.RawMsgRequest,
	stream socket.SocketService_ReceiveRawMsgServer,
) error {
	slog.Info("Client connected to ReceiveRawMsg stream")

	for {
		select {
		case <-stream.Context().Done():
			slog.Warn("Got Interruption signal from streaming server from stream context")
			return stream.Context().Err()
		case <-s.mainCtx.Done():
			slog.Info("Got Interruption signal from streaming server from main context")
			return stream.Context().Err()
		case msg, ok := <-s.outputChan:
			if !ok {
				slog.Warn("Output chan closed")
				return nil
			}
			if err := stream.Send(&socket.RawMsgResponse{Data: msg}); err != nil {
				slog.Error("Could not send raw message to client", "error", err)
				return err
			}
		}
	}
}

func StartServer(wg *sync.WaitGroup, outputChan chan []byte, ctx context.Context) {
	defer wg.Done()

	listen, err := net.Listen("tcp", port)
	if err != nil {
		slog.Error("Could not listening connection", "error", err)
		return
	}

	svr := grpc.NewServer()

	register(svr, outputChan, ctx)

	slog.Info("👂 Server listening", "port", port)

	go func() {
		if err := svr.Serve(listen); err != nil {
			slog.Error("Failed to listening server")
		}
	}()

	<-ctx.Done()
	slog.Info("Got interruption signal, stopping gRPC server")
	svr.GracefulStop()
}
