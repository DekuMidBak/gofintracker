package app

import (
	"context"
	"errors"
	"net"

	"google.golang.org/grpc"
)

func ServeGRPC(ctx context.Context, listener net.Listener, server *grpc.Server) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	case <-ctx.Done():
		server.GracefulStop()

		if err := <-errCh; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return err
		}

		return nil
	}
}
