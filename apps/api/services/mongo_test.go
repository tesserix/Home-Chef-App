package services

// mongo_test.go — the Mongo client is optional/best-effort (#53). With no
// MONGODB_URI configured, Connect() is a clean no-op and the client reports
// not-connected, so the Mongo-backed chat/upload features disable gracefully
// without affecting the rest of the API.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
)

func TestMongoClient_NoOpWhenUnconfigured(t *testing.T) {
	orig := config.AppConfig
	t.Cleanup(func() { config.AppConfig = orig })
	config.AppConfig = &config.Config{MongoURI: "", MongoDBName: "homechef_chat"}

	m := &MongoClient{}
	require.NoError(t, m.Connect()) // no URI → no-op, no error
	require.False(t, m.IsConnected())
	require.Nil(t, m.DB())
	require.Nil(t, m.Collection("messages"))
	m.Close(context.Background()) // safe when never connected
}
