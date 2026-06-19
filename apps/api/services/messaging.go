package services

// messaging.go — admin-mediated in-app messaging (#53, sub-issue #303).
//
// Topology (product decision): there is NO direct chef↔customer channel. A
// customer's message is addressed to the chef but routed to an ADMIN relay
// queue; an admin reviews and relays it (or blocks it). The same in reverse for
// the chef. Admins can also message either party directly. This keeps every
// chef↔customer exchange mediated + auditable, and (with PII masking) prevents
// off-platform contact exchange.
//
// Storage is MongoDB (per the #53 decision). The business logic here is written
// against a MessageStore interface so it is unit-tested with an in-memory fake;
// the Mongo implementation is in messaging_mongo.go.

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Roles + statuses.
const (
	MsgRoleCustomer = "customer"
	MsgRoleChef     = "chef"
	MsgRoleAdmin    = "admin"

	RelayPending = "pending" // awaiting admin review/relay
	RelayRelayed = "relayed" // delivered to the recipient
	RelayBlocked = "blocked" // admin withheld it

	ConversationOpen   = "open"
	ConversationClosed = "closed"
)

var (
	ErrMessagingUnavailable = errors.New("messaging is unavailable")
	ErrMessageNotFound      = errors.New("message not found")
	ErrMessageNotPending    = errors.New("message is not pending relay")
	ErrEmptyMessage         = errors.New("message content is empty")
)

// Conversation is the per-order mediated thread between a customer and a chef.
type Conversation struct {
	ID            string     `bson:"_id" json:"id"`
	OrderID       string     `bson:"orderId" json:"orderId"`
	CustomerID    string     `bson:"customerId" json:"customerId"`
	ChefID        string     `bson:"chefId" json:"chefId"`
	Status        string     `bson:"status" json:"status"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	LastMessageAt *time.Time `bson:"lastMessageAt,omitempty" json:"lastMessageAt,omitempty"`
}

// MediatedMessage is one message in a conversation. Content is always the
// PII-masked version. A customer→chef (or chef→customer) message starts
// RelayPending and is only visible to its recipient once an admin relays it.
type MediatedMessage struct {
	ID             string     `bson:"_id" json:"id"`
	ConversationID string     `bson:"conversationId" json:"conversationId"`
	OrderID        string     `bson:"orderId" json:"orderId"`
	SenderID       string     `bson:"senderId" json:"senderId"`
	SenderRole     string     `bson:"senderRole" json:"senderRole"`
	RecipientRole  string     `bson:"recipientRole" json:"recipientRole"`
	Content        string     `bson:"content" json:"content"`
	PIIDetected    bool       `bson:"piiDetected" json:"piiDetected"`
	RelayStatus    string     `bson:"relayStatus" json:"relayStatus"`
	RelayedByID    string     `bson:"relayedById,omitempty" json:"relayedById,omitempty"`
	RelayedAt      *time.Time `bson:"relayedAt,omitempty" json:"relayedAt,omitempty"`
	// Attachment (#304) — the file lives in GridFS; these describe it for the UI
	// + the authorized download.
	AttachmentID  string    `bson:"attachmentId,omitempty" json:"attachmentId,omitempty"`
	Filename      string    `bson:"filename,omitempty" json:"filename,omitempty"`
	ContentType   string    `bson:"contentType,omitempty" json:"contentType,omitempty"`
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`
}

// MessageStore is the persistence boundary (Mongo in prod, a fake in tests).
type MessageStore interface {
	GetOrCreateConversation(ctx context.Context, orderID, customerID, chefID string) (*Conversation, error)
	GetConversation(ctx context.Context, id string) (*Conversation, error)
	InsertMessage(ctx context.Context, m *MediatedMessage) error
	GetMessage(ctx context.Context, id string) (*MediatedMessage, error)
	ListConversationMessages(ctx context.Context, conversationID string) ([]MediatedMessage, error)
	SetRelayStatus(ctx context.Context, id, status, adminID string, at time.Time) error
	ListPendingRelay(ctx context.Context) ([]MediatedMessage, error)
	TouchConversation(ctx context.Context, conversationID string, at time.Time) error
	GetMessageByAttachment(ctx context.Context, attachmentID string) (*MediatedMessage, error)
}

// pushSender is the push hook (so tests can stub it); defaults to the real push.
var messagingPush = func(userID uuid.UUID, title, body string, data map[string]string) {
	_ = SendPushNotification(userID, title, body, data)
}

// MessagingService is the mediated-messaging business logic.
type MessagingService struct {
	store MessageStore
}

// NewMessagingService builds the service over a store.
func NewMessagingService(store MessageStore) *MessagingService {
	return &MessagingService{store: store}
}

// send is the shared path: PII-mask, persist a pending message, bump the
// conversation. notifyRecipientUser is pushed only when the message is delivered
// immediately (admin direct sends); customer/chef sends sit pending for review.
func (s *MessagingService) send(ctx context.Context, conv *Conversation, senderID, senderRole, recipientRole, content string, relay string) (*MediatedMessage, error) {
	if content == "" {
		return nil, ErrEmptyMessage
	}
	sanitized, hasPII, _ := FilterChatMessage(content)
	now := time.Now()
	msg := &MediatedMessage{
		ID:             uuid.NewString(),
		ConversationID: conv.ID,
		OrderID:        conv.OrderID,
		SenderID:       senderID,
		SenderRole:     senderRole,
		RecipientRole:  recipientRole,
		Content:        sanitized,
		PIIDetected:    hasPII,
		RelayStatus:    relay,
		CreatedAt:      now,
	}
	if relay == RelayRelayed {
		msg.RelayedAt = &now
	}
	if err := s.store.InsertMessage(ctx, msg); err != nil {
		return nil, err
	}
	_ = s.store.TouchConversation(ctx, conv.ID, now)
	return msg, nil
}

// CustomerSend records a customer's message to the chef — held pending an admin
// relay (mediated). The admin inbox surfaces it.
func (s *MessagingService) CustomerSend(ctx context.Context, orderID, customerID, chefID, content string) (*MediatedMessage, error) {
	conv, err := s.store.GetOrCreateConversation(ctx, orderID, customerID, chefID)
	if err != nil {
		return nil, err
	}
	return s.send(ctx, conv, customerID, MsgRoleCustomer, MsgRoleChef, content, RelayPending)
}

// ChefSend records a chef's message to the customer — held pending an admin relay.
func (s *MessagingService) ChefSend(ctx context.Context, orderID, customerID, chefID, content string) (*MediatedMessage, error) {
	conv, err := s.store.GetOrCreateConversation(ctx, orderID, customerID, chefID)
	if err != nil {
		return nil, err
	}
	return s.send(ctx, conv, chefID, MsgRoleChef, MsgRoleCustomer, content, RelayPending)
}

// AdminSend lets an admin message a party directly (delivered immediately) and
// pushes that recipient.
func (s *MessagingService) AdminSend(ctx context.Context, conversationID, adminID, recipientRole, content string) (*MediatedMessage, error) {
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	msg, err := s.send(ctx, conv, adminID, MsgRoleAdmin, recipientRole, content, RelayRelayed)
	if err != nil {
		return nil, err
	}
	s.pushRecipient(conv, recipientRole)
	return msg, nil
}

// AdminRelay delivers a pending customer/chef message to its recipient and
// pushes them. Idempotent-ish: a non-pending message returns ErrMessageNotPending.
func (s *MessagingService) AdminRelay(ctx context.Context, messageID, adminID string) (*MediatedMessage, error) {
	msg, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg.RelayStatus != RelayPending {
		return nil, ErrMessageNotPending
	}
	now := time.Now()
	if err := s.store.SetRelayStatus(ctx, messageID, RelayRelayed, adminID, now); err != nil {
		return nil, err
	}
	msg.RelayStatus, msg.RelayedByID, msg.RelayedAt = RelayRelayed, adminID, &now
	if conv, err := s.store.GetConversation(ctx, msg.ConversationID); err == nil {
		s.pushRecipient(conv, msg.RecipientRole)
	}
	return msg, nil
}

// AdminBlock withholds a pending message (e.g. abusive / contact-sharing).
func (s *MessagingService) AdminBlock(ctx context.Context, messageID, adminID string) error {
	msg, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}
	if msg.RelayStatus != RelayPending {
		return ErrMessageNotPending
	}
	return s.store.SetRelayStatus(ctx, messageID, RelayBlocked, adminID, time.Now())
}

// pushRecipient sends a "new message" push to the party a message was delivered to.
func (s *MessagingService) pushRecipient(conv *Conversation, recipientRole string) {
	var uid string
	switch recipientRole {
	case MsgRoleCustomer:
		uid = conv.CustomerID
	case MsgRoleChef:
		uid = conv.ChefID
	default:
		return
	}
	parsed, err := uuid.Parse(uid)
	if err != nil {
		return
	}
	messagingPush(parsed, "New message", "You have a new message about your order.", map[string]string{
		"type": "chat", "conversationId": conv.ID, "orderId": conv.OrderID,
	})
}

// ThreadFor returns the messages visible to a role in a conversation: a party
// sees its own messages plus what an admin relayed to it; an admin sees all.
func (s *MessagingService) ThreadFor(ctx context.Context, conversationID, role string) ([]MediatedMessage, error) {
	all, err := s.store.ListConversationMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if role == MsgRoleAdmin {
		return all, nil
	}
	out := make([]MediatedMessage, 0, len(all))
	for _, m := range all {
		if m.SenderRole == role || (m.RecipientRole == role && m.RelayStatus == RelayRelayed) {
			out = append(out, m)
		}
	}
	return out, nil
}

// AdminInbox returns all messages awaiting relay (the mediation queue).
func (s *MessagingService) AdminInbox(ctx context.Context) ([]MediatedMessage, error) {
	return s.store.ListPendingRelay(ctx)
}

// sendAttachment persists a pending attachment message (the file is already in
// GridFS). Caption is optional; the content falls back to the filename.
func (s *MessagingService) sendAttachment(ctx context.Context, conv *Conversation, senderID, senderRole, recipientRole, attachmentID, filename, contentType, caption string) (*MediatedMessage, error) {
	content := caption
	if content == "" {
		content = "📎 " + filename
	}
	sanitized, hasPII, _ := FilterChatMessage(content)
	now := time.Now()
	msg := &MediatedMessage{
		ID: uuid.NewString(), ConversationID: conv.ID, OrderID: conv.OrderID,
		SenderID: senderID, SenderRole: senderRole, RecipientRole: recipientRole,
		Content: sanitized, PIIDetected: hasPII, RelayStatus: RelayPending,
		AttachmentID: attachmentID, Filename: filename, ContentType: contentType, CreatedAt: now,
	}
	if err := s.store.InsertMessage(ctx, msg); err != nil {
		return nil, err
	}
	_ = s.store.TouchConversation(ctx, conv.ID, now)
	return msg, nil
}

// CustomerSendAttachment records a customer's attachment to the chef (pending relay).
func (s *MessagingService) CustomerSendAttachment(ctx context.Context, orderID, customerID, chefID, attachmentID, filename, contentType, caption string) (*MediatedMessage, error) {
	conv, err := s.store.GetOrCreateConversation(ctx, orderID, customerID, chefID)
	if err != nil {
		return nil, err
	}
	return s.sendAttachment(ctx, conv, customerID, MsgRoleCustomer, MsgRoleChef, attachmentID, filename, contentType, caption)
}

// ChefSendAttachment records a chef's attachment to the customer (pending relay).
func (s *MessagingService) ChefSendAttachment(ctx context.Context, orderID, customerID, chefID, attachmentID, filename, contentType, caption string) (*MediatedMessage, error) {
	conv, err := s.store.GetOrCreateConversation(ctx, orderID, customerID, chefID)
	if err != nil {
		return nil, err
	}
	return s.sendAttachment(ctx, conv, chefID, MsgRoleChef, MsgRoleCustomer, attachmentID, filename, contentType, caption)
}

// AuthorizeAttachmentDownload checks that requesterUserID may download the
// attachment, and returns the message (for its content type). Admins always may;
// a participant may download the sender's own message, or a relayed one addressed
// to them. Returns the message + ok.
func (s *MessagingService) AuthorizeAttachmentDownload(ctx context.Context, attachmentID, requesterUserID, requesterRole string) (*MediatedMessage, bool, error) {
	msg, err := s.store.GetMessageByAttachment(ctx, attachmentID)
	if err != nil {
		return nil, false, err
	}
	if requesterRole == MsgRoleAdmin {
		return msg, true, nil
	}
	conv, err := s.store.GetConversation(ctx, msg.ConversationID)
	if err != nil {
		return nil, false, err
	}
	isParticipant := requesterUserID == conv.CustomerID || requesterUserID == conv.ChefID
	if !isParticipant {
		return msg, false, nil
	}
	// The sender can always re-download; the recipient only once relayed.
	if msg.SenderID == requesterUserID || msg.RelayStatus == RelayRelayed {
		return msg, true, nil
	}
	return msg, false, nil
}

// OrderThread resolves (or creates) the order's conversation and returns the
// messages visible to the given role plus the conversation. Used by the
// customer/chef list endpoints.
func (s *MessagingService) OrderThread(ctx context.Context, orderID, customerID, chefID, role string) ([]MediatedMessage, *Conversation, error) {
	conv, err := s.store.GetOrCreateConversation(ctx, orderID, customerID, chefID)
	if err != nil {
		return nil, nil, err
	}
	msgs, err := s.ThreadFor(ctx, conv.ID, role)
	return msgs, conv, err
}
