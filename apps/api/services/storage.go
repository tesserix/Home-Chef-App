package services

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/homechef/api/config"
)

var storageClient *storage.Client

func InitStorage() error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx) // Uses default credentials (WI on GKE)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}
	storageClient = client
	return nil
}

func CloseStorage() {
	if storageClient != nil {
		storageClient.Close()
	}
}

// UploadFile uploads a file to the specified bucket and returns the object path
func UploadFile(ctx context.Context, bucket, objectPath string, reader io.Reader, contentType string) (string, error) {
	obj := storageClient.Bucket(bucket).Object(objectPath)
	writer := obj.NewWriter(ctx)
	writer.ContentType = contentType
	writer.CacheControl = "public, max-age=31536000" // 1 year cache for immutable uploads

	if _, err := io.Copy(writer, reader); err != nil {
		return "", fmt.Errorf("failed to write to GCS: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close GCS writer: %w", err)
	}

	return objectPath, nil
}

// UploadPublicFile uploads to the public assets bucket.
// Returns the public URL: https://storage.googleapis.com/{bucket}/{path}
func UploadPublicFile(ctx context.Context, folder string, fileName string, reader io.Reader, contentType string) (string, error) {
	bucket := config.AppConfig.GCSPublicBucket
	ext := path.Ext(fileName)
	objectPath := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), ext)

	if _, err := UploadFile(ctx, bucket, objectPath, reader, contentType); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objectPath), nil
}

// UploadPrivateFile uploads to the private docs bucket.
// Returns the object path (not a public URL — use signed URLs for access).
func UploadPrivateFile(ctx context.Context, folder string, fileName string, reader io.Reader, contentType string) (string, error) {
	bucket := config.AppConfig.GCSPrivateBucket
	ext := path.Ext(fileName)
	objectPath := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), ext)

	if _, err := UploadFile(ctx, bucket, objectPath, reader, contentType); err != nil {
		return "", err
	}

	return objectPath, nil
}

// GenerateSignedURL generates a temporary signed URL for a private file
func GenerateSignedURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	bucket := config.AppConfig.GCSPrivateBucket
	url, err := storageClient.Bucket(bucket).SignedURL(objectPath, &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(expiry),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}
	return url, nil
}

// DeleteFile removes a file from a bucket
func DeleteFile(ctx context.Context, bucket, objectPath string) error {
	return storageClient.Bucket(bucket).Object(objectPath).Delete(ctx)
}

// IsImageContentType checks if a content type is an allowed image type
func IsImageContentType(ct string) bool {
	ct = strings.ToLower(ct)
	return ct == "image/jpeg" || ct == "image/png" || ct == "image/webp"
}

// IsDocContentType checks if a content type is an allowed document type
func IsDocContentType(ct string) bool {
	ct = strings.ToLower(ct)
	return ct == "image/jpeg" || ct == "image/png" || ct == "application/pdf"
}
