package services

// messaging_mongo.go — the MongoDB implementation of MessageStore (#53/#303).
// Two collections: conversations (one per order) and mediated_messages. The
// business logic + invariants live in messaging.go and are unit-tested against a
// fake; this is the thin persistence adapter.

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	collConversations = "conversations"
	collMessages      = "mediated_messages"
)

type mongoMessageStore struct {
	convs *mongo.Collection
	msgs  *mongo.Collection
}

// NewMongoMessageStore returns a Mongo-backed store, or nil when Mongo is not
// connected (callers treat nil as "messaging unavailable").
func NewMongoMessageStore() MessageStore {
	mc := GetMongoClient()
	if !mc.IsConnected() {
		return nil
	}
	return &mongoMessageStore{
		convs: mc.Collection(collConversations),
		msgs:  mc.Collection(collMessages),
	}
}

// MessagingFromMongo builds the MessagingService over the live Mongo store, or
// nil when Mongo is unavailable.
func MessagingFromMongo() *MessagingService {
	store := NewMongoMessageStore()
	if store == nil {
		return nil
	}
	return NewMessagingService(store)
}

func sortByCreated() *options.FindOptionsBuilder {
	return options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})
}

func (s *mongoMessageStore) GetOrCreateConversation(ctx context.Context, orderID, customerID, chefID string) (*Conversation, error) {
	var c Conversation
	err := s.convs.FindOne(ctx, bson.M{"orderId": orderID}).Decode(&c)
	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}
	c = Conversation{
		ID: uuid.NewString(), OrderID: orderID, CustomerID: customerID, ChefID: chefID,
		Status: ConversationOpen, CreatedAt: time.Now(),
	}
	if _, err := s.convs.InsertOne(ctx, c); err != nil {
		// Lost a create race — re-read the winner.
		if e := s.convs.FindOne(ctx, bson.M{"orderId": orderID}).Decode(&c); e == nil {
			return &c, nil
		}
		return nil, err
	}
	return &c, nil
}

func (s *mongoMessageStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	var c Conversation
	err := s.convs.FindOne(ctx, bson.M{"_id": id}).Decode(&c)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrMessageNotFound
	}
	return &c, err
}

func (s *mongoMessageStore) InsertMessage(ctx context.Context, m *MediatedMessage) error {
	_, err := s.msgs.InsertOne(ctx, m)
	return err
}

func (s *mongoMessageStore) GetMessage(ctx context.Context, id string) (*MediatedMessage, error) {
	var m MediatedMessage
	err := s.msgs.FindOne(ctx, bson.M{"_id": id}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrMessageNotFound
	}
	return &m, err
}

func (s *mongoMessageStore) ListConversationMessages(ctx context.Context, conversationID string) ([]MediatedMessage, error) {
	cur, err := s.msgs.Find(ctx, bson.M{"conversationId": conversationID}, sortByCreated())
	if err != nil {
		return nil, err
	}
	var out []MediatedMessage
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *mongoMessageStore) SetRelayStatus(ctx context.Context, id, status, adminID string, at time.Time) error {
	_, err := s.msgs.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"relayStatus": status, "relayedById": adminID, "relayedAt": at,
	}})
	return err
}

func (s *mongoMessageStore) ListPendingRelay(ctx context.Context) ([]MediatedMessage, error) {
	cur, err := s.msgs.Find(ctx, bson.M{"relayStatus": RelayPending}, sortByCreated())
	if err != nil {
		return nil, err
	}
	var out []MediatedMessage
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *mongoMessageStore) GetMessageByAttachment(ctx context.Context, attachmentID string) (*MediatedMessage, error) {
	var m MediatedMessage
	err := s.msgs.FindOne(ctx, bson.M{"attachmentId": attachmentID}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrMessageNotFound
	}
	return &m, err
}

func (s *mongoMessageStore) TouchConversation(ctx context.Context, conversationID string, at time.Time) error {
	_, err := s.convs.UpdateOne(ctx, bson.M{"_id": conversationID}, bson.M{"$set": bson.M{"lastMessageAt": at}})
	return err
}

func (s *mongoMessageStore) ListConversations(ctx context.Context, filter ConversationFilter, limit, offset int) ([]Conversation, int, error) {
	q := bson.M{}
	if filter.OrderID != "" {
		q["orderId"] = filter.OrderID
	}
	if filter.CustomerID != "" {
		q["customerId"] = filter.CustomerID
	}
	if filter.ChefID != "" {
		q["chefId"] = filter.ChefID
	}
	if filter.Status != "" {
		q["status"] = filter.Status
	}
	if filter.From != nil || filter.To != nil {
		rng := bson.M{}
		if filter.From != nil {
			rng["$gte"] = *filter.From
		}
		if filter.To != nil {
			rng["$lte"] = *filter.To
		}
		q["createdAt"] = rng
	}
	total, err := s.convs.CountDocuments(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "lastMessageAt", Value: -1}, {Key: "createdAt", Value: -1}}).
		SetSkip(int64(offset)).SetLimit(int64(limit))
	cur, err := s.convs.Find(ctx, q, opts)
	if err != nil {
		return nil, 0, err
	}
	var out []Conversation
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, int(total), nil
}

// EnsureMessagingIndexes creates the supporting indexes (called at startup when
// Mongo is connected). Best-effort: index creation is idempotent.
func EnsureMessagingIndexes(ctx context.Context) error {
	mc := GetMongoClient()
	if !mc.IsConnected() {
		return nil
	}
	if _, err := mc.Collection(collConversations).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "orderId", Value: 1}}, Options: options.Index().SetUnique(true)},
		// Admin audit listing — sort by recency + filter by participant/status.
		{Keys: bson.D{{Key: "lastMessageAt", Value: -1}}},
		{Keys: bson.D{{Key: "customerId", Value: 1}}},
		{Keys: bson.D{{Key: "chefId", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	}); err != nil {
		return err
	}
	_, err := mc.Collection(collMessages).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "conversationId", Value: 1}, {Key: "createdAt", Value: 1}}},
		{Keys: bson.D{{Key: "relayStatus", Value: 1}}},
	})
	return err
}
