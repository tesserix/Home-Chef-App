package handlers

import (
	"context"
	"net"
	"sync"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/homechef/api/services"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type payoutSecretServer struct {
	secretmanagerpb.UnimplementedSecretManagerServiceServer
	mu     sync.Mutex
	values map[string]string
}

func (s *payoutSecretServer) CreateSecret(_ context.Context, req *secretmanagerpb.CreateSecretRequest) (*secretmanagerpb.Secret, error) {
	return &secretmanagerpb.Secret{Name: req.Parent + "/secrets/" + req.SecretId}, nil
}

func (s *payoutSecretServer) AddSecretVersion(_ context.Context, req *secretmanagerpb.AddSecretVersionRequest) (*secretmanagerpb.SecretVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[req.Parent] = string(req.Payload.Data)
	return &secretmanagerpb.SecretVersion{Name: req.Parent + "/versions/1"}, nil
}

func setupPayoutSecretStore(t *testing.T) *payoutSecretServer {
	t.Helper()
	t.Setenv("APP_SECRET_STORE", "gcp")
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	backend := &payoutSecretServer{values: map[string]string{}}
	secretmanagerpb.RegisterSecretManagerServiceServer(server, backend)
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Errorf("serve secret manager: %v", err)
		}
	}()
	conn, err := grpc.NewClient("passthrough:///secret-manager", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	require.NoError(t, services.InitSecretManager(option.WithGRPCConn(conn)))
	t.Cleanup(func() { services.CloseSecretManager(); _ = conn.Close(); server.Stop(); _ = listener.Close() })
	return backend
}
