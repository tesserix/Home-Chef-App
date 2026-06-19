package services

// mongo.go — the optional MongoDB connection backing HomeChef in-app chat
// (admin-mediated messaging) + document uploads (#53). Like Redis, it is
// best-effort: when MONGODB_URI is empty or Mongo is unreachable the API keeps
// running and the Mongo-backed features no-op (IsConnected() == false), so the
// rest of the platform is unaffected.

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/homechef/api/config"
)

// MongoClient wraps a connected *mongo.Database. Acquire the process-wide
// instance with GetMongoClient().
type MongoClient struct {
	mu     sync.RWMutex
	client *mongo.Client
	db     *mongo.Database
}

var (
	mongoOnce     sync.Once
	mongoInstance *MongoClient
)

// GetMongoClient returns the process-wide Mongo client singleton.
func GetMongoClient() *MongoClient {
	mongoOnce.Do(func() { mongoInstance = &MongoClient{} })
	return mongoInstance
}

// Connect dials MongoDB using config (MONGODB_URI / MONGODB_DB). It is a no-op
// (returns nil) when unconfigured, so callers can always call it. A dial/ping
// failure is returned so main() can log-and-continue.
func (m *MongoClient) Connect() error {
	uri := config.AppConfig.MongoURI
	if uri == "" {
		log.Println("MongoDB not configured (MONGODB_URI empty) — chat/upload-on-Mongo disabled")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return err
	}

	m.mu.Lock()
	m.client = client
	m.db = client.Database(config.AppConfig.MongoDBName)
	m.mu.Unlock()
	log.Printf("MongoDB connection established (db=%s)", config.AppConfig.MongoDBName)
	return nil
}

// IsConnected reports whether a live Mongo database is available.
func (m *MongoClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client != nil && m.db != nil
}

// DB returns the connected database, or nil when not connected.
func (m *MongoClient) DB() *mongo.Database {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.db
}

// Collection returns a handle to a collection, or nil when not connected.
func (m *MongoClient) Collection(name string) *mongo.Collection {
	db := m.DB()
	if db == nil {
		return nil
	}
	return db.Collection(name)
}

// Close disconnects the client (safe when never connected).
func (m *MongoClient) Close(ctx context.Context) {
	m.mu.RLock()
	c := m.client
	m.mu.RUnlock()
	if c != nil {
		_ = c.Disconnect(ctx)
	}
}
