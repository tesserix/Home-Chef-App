package services

// messaging_test.go — admin-mediated messaging logic (#303), tested against an
// in-memory MessageStore (no live Mongo). Pins the mediation rules: a
// customer/chef message is held PENDING (never visible to the other party until
// an admin relays it), PII is masked, the admin inbox surfaces pending messages,
// relay delivers + pushes the recipient, block withholds, and per-role thread
// views only expose what each party is allowed to see.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	convs map[string]*Conversation
	msgs  map[string]*MediatedMessage
	order []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{convs: map[string]*Conversation{}, msgs: map[string]*MediatedMessage{}}
}

func (f *fakeStore) GetOrCreateConversation(_ context.Context, orderID, customerID, chefID string) (*Conversation, error) {
	for _, c := range f.convs {
		if c.OrderID == orderID {
			return c, nil
		}
	}
	c := &Conversation{ID: uuid.NewString(), OrderID: orderID, CustomerID: customerID, ChefID: chefID, Status: ConversationOpen, CreatedAt: time.Now()}
	f.convs[c.ID] = c
	return c, nil
}
func (f *fakeStore) GetConversation(_ context.Context, id string) (*Conversation, error) {
	if c, ok := f.convs[id]; ok {
		return c, nil
	}
	return nil, ErrMessageNotFound
}
func (f *fakeStore) InsertMessage(_ context.Context, m *MediatedMessage) error {
	f.msgs[m.ID] = m
	f.order = append(f.order, m.ID)
	return nil
}
func (f *fakeStore) GetMessage(_ context.Context, id string) (*MediatedMessage, error) {
	if m, ok := f.msgs[id]; ok {
		return m, nil
	}
	return nil, ErrMessageNotFound
}
func (f *fakeStore) ListConversationMessages(_ context.Context, conversationID string) ([]MediatedMessage, error) {
	var out []MediatedMessage
	for _, id := range f.order {
		if m := f.msgs[id]; m.ConversationID == conversationID {
			out = append(out, *m)
		}
	}
	return out, nil
}
func (f *fakeStore) SetRelayStatus(_ context.Context, id, status, adminID string, at time.Time) error {
	m, ok := f.msgs[id]
	if !ok {
		return ErrMessageNotFound
	}
	m.RelayStatus, m.RelayedByID, m.RelayedAt = status, adminID, &at
	return nil
}
func (f *fakeStore) ListPendingRelay(_ context.Context) ([]MediatedMessage, error) {
	var out []MediatedMessage
	for _, id := range f.order {
		if m := f.msgs[id]; m.RelayStatus == RelayPending {
			out = append(out, *m)
		}
	}
	return out, nil
}
func (f *fakeStore) TouchConversation(_ context.Context, conversationID string, at time.Time) error {
	if c, ok := f.convs[conversationID]; ok {
		c.LastMessageAt = &at
	}
	return nil
}

// stubPush captures pushes for assertions.
func stubMessagingPush(t *testing.T) *int {
	t.Helper()
	n := 0
	orig := messagingPush
	t.Cleanup(func() { messagingPush = orig })
	messagingPush = func(uuid.UUID, string, string, map[string]string) { n++ }
	return &n
}

func ids() (order, cust, chef, admin string) {
	return uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
}

func TestCustomerSend_HeldPendingAndMaskedPII(t *testing.T) {
	pushes := stubMessagingPush(t)
	svc := NewMessagingService(newFakeStore())
	order, cust, chef, _ := ids()

	m, err := svc.CustomerSend(context.Background(), order, cust, chef, "call me on 9876543210 please")
	require.NoError(t, err)
	require.Equal(t, RelayPending, m.RelayStatus)   // not delivered to the chef yet
	require.True(t, m.PIIDetected)                  // phone masked
	require.NotContains(t, m.Content, "9876543210") // PII redacted
	require.Equal(t, 0, *pushes)                    // no push until an admin relays

	// The chef must NOT see a pending message.
	chefThread, _ := svc.ThreadFor(context.Background(), m.ConversationID, MsgRoleChef)
	require.Empty(t, chefThread)
	// The admin inbox surfaces it.
	inbox, _ := svc.AdminInbox(context.Background())
	require.Len(t, inbox, 1)
}

func TestAdminRelay_DeliversAndPushes(t *testing.T) {
	pushes := stubMessagingPush(t)
	svc := NewMessagingService(newFakeStore())
	order, cust, chef, admin := ids()

	m, _ := svc.CustomerSend(context.Background(), order, cust, chef, "I have a nut allergy")
	relayed, err := svc.AdminRelay(context.Background(), m.ID, admin)
	require.NoError(t, err)
	require.Equal(t, RelayRelayed, relayed.RelayStatus)
	require.Equal(t, admin, relayed.RelayedByID)
	require.Equal(t, 1, *pushes) // the chef (recipient) is pushed

	// Now the chef sees it; the inbox is empty.
	chefThread, _ := svc.ThreadFor(context.Background(), m.ConversationID, MsgRoleChef)
	require.Len(t, chefThread, 1)
	inbox, _ := svc.AdminInbox(context.Background())
	require.Empty(t, inbox)

	// Relaying again fails (not pending).
	_, err = svc.AdminRelay(context.Background(), m.ID, admin)
	require.ErrorIs(t, err, ErrMessageNotPending)
}

func TestThreadFor_PerRoleVisibility(t *testing.T) {
	stubMessagingPush(t)
	svc := NewMessagingService(newFakeStore())
	order, cust, chef, admin := ids()

	cm, _ := svc.CustomerSend(context.Background(), order, cust, chef, "where is my order")
	_, _ = svc.ChefSend(context.Background(), order, cust, chef, "running 10 min late")
	_, _ = svc.AdminRelay(context.Background(), cm.ID, admin) // relay only the customer msg

	conv := cm.ConversationID
	customerThread, _ := svc.ThreadFor(context.Background(), conv, MsgRoleCustomer)
	chefThread, _ := svc.ThreadFor(context.Background(), conv, MsgRoleChef)
	adminThread, _ := svc.ThreadFor(context.Background(), conv, MsgRoleAdmin)

	require.Len(t, customerThread, 1) // own message only (chef's not relayed to them)
	require.Len(t, chefThread, 2)     // own message + the relayed customer message
	require.Len(t, adminThread, 2)    // admin sees everything
}

func TestAdminBlock_WithholdsMessage(t *testing.T) {
	svc := NewMessagingService(newFakeStore())
	order, cust, chef, admin := ids()
	m, _ := svc.CustomerSend(context.Background(), order, cust, chef, "here's my number 9998887776")

	require.NoError(t, svc.AdminBlock(context.Background(), m.ID, admin))
	chefThread, _ := svc.ThreadFor(context.Background(), m.ConversationID, MsgRoleChef)
	require.Empty(t, chefThread) // blocked → never delivered
	inbox, _ := svc.AdminInbox(context.Background())
	require.Empty(t, inbox) // no longer pending
}

func TestAdminSend_DeliveredImmediately(t *testing.T) {
	pushes := stubMessagingPush(t)
	store := newFakeStore()
	svc := NewMessagingService(store)
	order, cust, chef, admin := ids()
	conv, _ := store.GetOrCreateConversation(context.Background(), order, cust, chef)

	m, err := svc.AdminSend(context.Background(), conv.ID, admin, MsgRoleCustomer, "We're looking into your order.")
	require.NoError(t, err)
	require.Equal(t, RelayRelayed, m.RelayStatus)
	require.Equal(t, 1, *pushes) // customer pushed immediately

	customerThread, _ := svc.ThreadFor(context.Background(), conv.ID, MsgRoleCustomer)
	require.Len(t, customerThread, 1)
}

func TestCustomerSend_EmptyRejected(t *testing.T) {
	svc := NewMessagingService(newFakeStore())
	order, cust, chef, _ := ids()
	_, err := svc.CustomerSend(context.Background(), order, cust, chef, "")
	require.ErrorIs(t, err, ErrEmptyMessage)
}
