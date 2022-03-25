package grpc

import (
	"time"

	"google.golang.org/grpc"
)

// Dial tries to dial to a grpc server and returns the client connection.
// If the server is not available, it will retry for the timeout given with a delay that is specified in backoff
func Dial(URI string, timeout time.Duration, backoff time.Duration) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error

	for i := 0; i < int(timeout/backoff); i++ {
		conn, err = grpc.Dial(URI, grpc.WithInsecure())
		if err == nil {
			return conn, nil
		}
		time.Sleep(backoff)
	}

	return nil, err
}
