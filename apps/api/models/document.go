package models

import (
	"time"

	"github.com/google/uuid"
)

type DocumentType string

const (
	DocPanCard         DocumentType = "pan_card"
	DocAadhaarCard     DocumentType = "aadhaar_card"
	DocFSSAILicense    DocumentType = "fssai_license"
	DocFoodSafetyCert  DocumentType = "food_safety_cert"
	DocCancelledCheque DocumentType = "cancelled_cheque"
	DocKitchenPhoto1   DocumentType = "kitchen_photo_1"
	DocKitchenPhoto2   DocumentType = "kitchen_photo_2"
	DocKitchenPhoto3   DocumentType = "kitchen_photo_3"
	DocProfileImage    DocumentType = "profile_image"
)

type DocumentStatus string

const (
	DocStatusPending  DocumentStatus = "pending"
	DocStatusVerified DocumentStatus = "verified"
	DocStatusRejected DocumentStatus = "rejected"
)

type ChefDocument struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID          uuid.UUID      `gorm:"type:uuid;not null;index;index:idx_chef_doc_fssai,priority:3" json:"chefId"`
	Type            DocumentType   `gorm:"type:varchar(50);not null;index;index:idx_chef_doc_fssai,priority:1" json:"type"`
	FileName        string         `gorm:"not null" json:"fileName"`
	FilePath        string         `gorm:"not null" json:"-"`          // GCS object path (never exposed)
	FileURL         string         `gorm:"-" json:"fileUrl,omitempty"` // Computed: public URL or signed URL
	Bucket          string         `gorm:"not null" json:"-"`          // Which bucket (public/private)
	ContentType     string         `gorm:"" json:"contentType"`
	FileSize        int64          `gorm:"" json:"fileSize"`
	Status          DocumentStatus `gorm:"type:varchar(20);default:'pending';index:idx_chef_doc_fssai,priority:2" json:"status"`
	RejectionReason string         `gorm:"" json:"rejectionReason,omitempty"`
	// ExpiryDate is the document's expiry date (nullable — not all document types
	// have an expiry, and existing rows will have NULL until re-uploaded).
	// Used by the FSSAI expiry-reminder endpoint.
	// Composite index idx_chef_doc_fssai (type, status, chef_id, expiry_date)
	// backs the FSSAI lockout hot path (ExcludeFSSAILocked / IsChefFSSAIExpired).
	ExpiryDate *time.Time `gorm:"index:idx_chef_doc_fssai,priority:4" json:"expiryDate,omitempty"`
	// ImagePHash is the perceptual (difference) hash of an uploaded document
	// image, hex-encoded — used to detect the same document reused across
	// accounts. Empty for PDFs and pre-existing rows.
	ImagePHash string    `gorm:"type:varchar(16);index" json:"-"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Chef ChefProfile `gorm:"foreignKey:ChefID" json:"-"`
}

// IsPhotoDoc returns true if this is a photo/image type (kitchen photos, profile image).
// Photos accept JPEG, PNG, WebP. Non-photo docs accept JPEG, PNG, PDF.
func IsPhotoDoc(docType DocumentType) bool {
	switch docType {
	case DocKitchenPhoto1, DocKitchenPhoto2, DocKitchenPhoto3, DocProfileImage:
		return true
	default:
		return false
	}
}

// IsPrivateDoc returns true if this document type should be stored in the private bucket.
// Only profile_image and menu item photos are public — everything else (ID docs, kitchen
// photos used for verification) goes to the private bucket.
func IsPrivateDoc(docType DocumentType) bool {
	switch docType {
	case DocProfileImage:
		return false // Public: shown on chef's storefront
	default:
		return true // Private: ID docs, kitchen photos (for internal verification)
	}
}

type ChefDocumentResponse struct {
	ID              uuid.UUID      `json:"id"`
	Type            DocumentType   `json:"type"`
	FileName        string         `json:"fileName"`
	FileURL         string         `json:"fileUrl,omitempty"`
	Status          DocumentStatus `json:"status"`
	RejectionReason string         `json:"rejectionReason,omitempty"`
	ExpiryDate      *time.Time     `json:"expiryDate,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
}

func (d *ChefDocument) ToResponse() ChefDocumentResponse {
	return ChefDocumentResponse{
		ID:              d.ID,
		Type:            d.Type,
		FileName:        d.FileName,
		FileURL:         d.FileURL,
		Status:          d.Status,
		RejectionReason: d.RejectionReason,
		ExpiryDate:      d.ExpiryDate,
		CreatedAt:       d.CreatedAt,
	}
}
