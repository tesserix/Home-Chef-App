package services

// messaging_attachments.go — chat document/attachment upload on MongoDB GridFS
// (#53, sub-issue #304). The file bytes live in a GridFS bucket; the message
// (messaging.go) carries the attachment id + filename + content type and the
// admin-mediation + per-role-visibility rules gate who can download it.

import (
	"context"
	"io"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	chatAttachmentBucket   = "chat_attachments"
	maxChatAttachmentBytes = 10 << 20 // 10 MiB
)

// AllowedChatAttachmentType reports whether a content type may be attached.
func AllowedChatAttachmentType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/webp", "application/pdf":
		return true
	default:
		return false
	}
}

// MaxChatAttachmentBytes is the upload size cap.
func MaxChatAttachmentBytes() int64 { return maxChatAttachmentBytes }

// UploadChatAttachment stores a file in GridFS and returns its id (hex). The
// content type rides along in the GridFS metadata too, for completeness.
func UploadChatAttachment(ctx context.Context, filename, contentType string, r io.Reader) (string, error) {
	mc := GetMongoClient()
	if !mc.IsConnected() {
		return "", ErrMessagingUnavailable
	}
	bucket := mc.DB().GridFSBucket(options.GridFSBucket().SetName(chatAttachmentBucket))
	id, err := bucket.UploadFromStream(ctx, filename, r,
		options.GridFSUpload().SetMetadata(bson.M{"contentType": contentType}))
	if err != nil {
		return "", err
	}
	return id.Hex(), nil
}

// DownloadChatAttachment streams a stored attachment to w. Authorization is the
// caller's responsibility (see MessagingService.AuthorizeAttachmentDownload).
func DownloadChatAttachment(ctx context.Context, attachmentID string, w io.Writer) error {
	mc := GetMongoClient()
	if !mc.IsConnected() {
		return ErrMessagingUnavailable
	}
	oid, err := bson.ObjectIDFromHex(attachmentID)
	if err != nil {
		return ErrMessageNotFound
	}
	bucket := mc.DB().GridFSBucket(options.GridFSBucket().SetName(chatAttachmentBucket))
	if _, err := bucket.DownloadToStream(ctx, oid, w); err != nil {
		return err
	}
	return nil
}
