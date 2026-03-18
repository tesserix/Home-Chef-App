package models

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryStatus string

const (
	DeliveryPending    DeliveryStatus = "pending"
	DeliveryAssigned   DeliveryStatus = "assigned"
	DeliveryAtPickup   DeliveryStatus = "at_pickup"
	DeliveryPickedUp   DeliveryStatus = "picked_up"
	DeliveryInTransit  DeliveryStatus = "in_transit"
	DeliveryAtDropoff  DeliveryStatus = "at_dropoff"
	DeliveryDelivered  DeliveryStatus = "delivered"
	DeliveryFailed     DeliveryStatus = "failed"
	DeliveryReturned   DeliveryStatus = "returned"
	DeliveryCancelled  DeliveryStatus = "cancelled"
)

type AgentType string

const (
	AgentInternal   AgentType = "internal"
	AgentFreelance  AgentType = "freelance"
	AgentThirdParty AgentType = "third_party"
)

type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationInReview VerificationStatus = "in_review"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

type AssignmentType string

const (
	AssignmentManual     AssignmentType = "manual"
	AssignmentAuto       AssignmentType = "auto"
	AssignmentThirdParty AssignmentType = "third_party"
)

type DeliveryPartner struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	VehicleType     string     `gorm:"" json:"vehicleType"` // bike, scooter, car
	VehicleNumber   string     `gorm:"" json:"vehicleNumber"`
	LicenseNumber   string     `gorm:"" json:"licenseNumber"`
	IsVerified      bool       `gorm:"default:false" json:"verified"`
	VerifiedAt      *time.Time `gorm:"" json:"verifiedAt"`
	IsActive        bool       `gorm:"default:true" json:"isActive"`
	IsOnline        bool       `gorm:"default:false" json:"isOnline"`
	CurrentLatitude float64    `gorm:"" json:"currentLatitude"`
	CurrentLongitude float64   `gorm:"" json:"currentLongitude"`
	Rating          float64    `gorm:"default:0" json:"rating"`
	TotalDeliveries int        `gorm:"default:0" json:"totalDeliveries"`
	TotalReviews    int        `gorm:"default:0" json:"totalReviews"`

	// Agent classification
	AgentType  AgentType `gorm:"type:varchar(20);default:'freelance'" json:"agentType"`
	EmployeeID string    `gorm:"" json:"employeeId,omitempty"` // For internal agents

	// Shift management
	ShiftStart *time.Time `gorm:"" json:"shiftStart,omitempty"`
	ShiftEnd   *time.Time `gorm:"" json:"shiftEnd,omitempty"`

	// Capacity
	MaxConcurrent int `gorm:"default:1" json:"maxConcurrent"`

	// Performance metrics
	AcceptanceRate float64 `gorm:"default:0" json:"acceptanceRate"`
	OnTimeRate     float64 `gorm:"default:0" json:"onTimeRate"`
	CSATScore      float64 `gorm:"default:0" json:"csatScore"`
	OfferedCount   int     `gorm:"default:0" json:"offeredCount"`
	AcceptedCount  int     `gorm:"default:0" json:"acceptedCount"`
	CompletedOnTime int    `gorm:"default:0" json:"completedOnTime"`

	// Verification
	VerificationStatus VerificationStatus `gorm:"type:varchar(20);default:'pending'" json:"verificationStatus"`
	VerifiedByID       *uuid.UUID         `gorm:"type:uuid" json:"verifiedById,omitempty"`
	RejectionReason    string             `gorm:"" json:"rejectionReason,omitempty"`

	// Personal details
	City             string     `gorm:"" json:"city"`
	EmergencyContact string     `gorm:"" json:"emergencyContact"`
	EmergencyPhone   string     `gorm:"" json:"emergencyPhone"`
	DateOfBirth      *time.Time `gorm:"" json:"dateOfBirth,omitempty"`

	// Extended vehicle details
	VehicleMake         string `gorm:"" json:"vehicleMake"`
	VehicleModel        string `gorm:"" json:"vehicleModel"`
	VehicleYear         int    `gorm:"" json:"vehicleYear"`
	VehicleColor        string `gorm:"" json:"vehicleColor"`
	HasDeliveryBoxSpace bool   `gorm:"default:false" json:"hasDeliveryBoxSpace"` // For bicycle partners

	// Payout info
	BankAccountNumber string `gorm:"" json:"-"`
	BankIFSC          string `gorm:"" json:"-"`
	BankAccountName   string `gorm:"" json:"-"`
	UpiID             string `gorm:"" json:"-"`
	PayoutMethod      string `gorm:"type:varchar(20)" json:"payoutMethod"` // bank_transfer, upi

	// Onboarding tracking
	OnboardingStep     int        `gorm:"default:0" json:"onboardingStep"`
	OnboardingComplete bool       `gorm:"default:false" json:"onboardingComplete"`
	TermsAcceptedAt    *time.Time `gorm:"" json:"termsAcceptedAt,omitempty"`

	// Referral
	ReferralCode string     `gorm:"" json:"referralCode,omitempty"`
	ReferredByID *uuid.UUID `gorm:"type:uuid" json:"referredById,omitempty"`

	// Payment gateway linked accounts
	StripeAccountID    string `gorm:"" json:"-"`
	RazorpayAccountID  string `gorm:"" json:"-"` // Razorpay Route linked account ID

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	// Relationships
	User       User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Deliveries []Delivery `gorm:"foreignKey:DeliveryPartnerID" json:"deliveries,omitempty"`
	Documents  []DeliveryPartnerDocument `gorm:"foreignKey:PartnerID" json:"documents,omitempty"`
	VerifiedBy *User      `gorm:"foreignKey:VerifiedByID" json:"verifiedBy,omitempty"`
}

// DeliveryPartnerDocument stores verification documents uploaded by delivery partners
type DeliveryPartnerDocument struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PartnerID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"partnerId"`
	Type            PartnerDocType `gorm:"type:varchar(50);not null;index" json:"type"`
	FileName        string         `gorm:"not null" json:"fileName"`
	FilePath        string         `gorm:"not null" json:"-"`
	FileURL         string         `gorm:"-" json:"fileUrl,omitempty"`
	Bucket          string         `gorm:"not null" json:"-"`
	ContentType     string         `gorm:"" json:"contentType"`
	FileSize        int64          `gorm:"" json:"fileSize"`
	Status          DocumentStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	RejectionReason string         `gorm:"" json:"rejectionReason,omitempty"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`

	Partner DeliveryPartner `gorm:"foreignKey:PartnerID" json:"-"`
}

type PartnerDocType string

const (
	PartnerDocDrivingLicense      PartnerDocType = "driving_license"
	PartnerDocVehicleRC           PartnerDocType = "vehicle_rc"
	PartnerDocInsurance           PartnerDocType = "insurance"
	PartnerDocAadhaar             PartnerDocType = "aadhaar"
	PartnerDocPanCard             PartnerDocType = "pan_card"
	PartnerDocPhoto               PartnerDocType = "photo"
	PartnerDocPoliceVerification  PartnerDocType = "police_verification"
	PartnerDocVehicleFront        PartnerDocType = "vehicle_front"
	PartnerDocVehicleBack         PartnerDocType = "vehicle_back"
	PartnerDocVehicleLeft         PartnerDocType = "vehicle_left"
	PartnerDocVehicleRight        PartnerDocType = "vehicle_right"
	PartnerDocVehicleTop          PartnerDocType = "vehicle_top"
	PartnerDocVehicleNumberPlate  PartnerDocType = "vehicle_number_plate"
)

// IsVehiclePhoto returns true for vehicle photo types (stored in public bucket like profile photo)
func IsVehiclePhoto(docType PartnerDocType) bool {
	switch docType {
	case PartnerDocVehicleFront, PartnerDocVehicleBack, PartnerDocVehicleLeft,
		PartnerDocVehicleRight, PartnerDocVehicleTop, PartnerDocVehicleNumberPlate:
		return true
	default:
		return false
	}
}

type PartnerDocumentResponse struct {
	ID              uuid.UUID      `json:"id"`
	Type            PartnerDocType `json:"type"`
	FileName        string         `json:"fileName"`
	FileURL         string         `json:"fileUrl,omitempty"`
	Status          DocumentStatus `json:"status"`
	RejectionReason string         `json:"rejectionReason,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
}

func (d *DeliveryPartnerDocument) ToResponse() PartnerDocumentResponse {
	return PartnerDocumentResponse{
		ID:              d.ID,
		Type:            d.Type,
		FileName:        d.FileName,
		FileURL:         d.FileURL,
		Status:          d.Status,
		RejectionReason: d.RejectionReason,
		CreatedAt:       d.CreatedAt,
	}
}

type Delivery struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID           uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"orderId"`
	DeliveryPartnerID uuid.UUID      `gorm:"type:uuid;not null;index" json:"deliveryPartnerId"`
	Status            DeliveryStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`

	// Pickup Location (Chef)
	PickupAddressLine1 string  `gorm:"" json:"pickupAddressLine1"`
	PickupAddressCity  string  `gorm:"" json:"pickupAddressCity"`
	PickupLatitude     float64 `gorm:"" json:"pickupLatitude"`
	PickupLongitude    float64 `gorm:"" json:"pickupLongitude"`

	// Dropoff Location (Customer)
	DropoffAddressLine1 string  `gorm:"" json:"dropoffAddressLine1"`
	DropoffAddressCity  string  `gorm:"" json:"dropoffAddressCity"`
	DropoffLatitude     float64 `gorm:"" json:"dropoffLatitude"`
	DropoffLongitude    float64 `gorm:"" json:"dropoffLongitude"`

	// Tracking
	Distance          float64 `gorm:"" json:"distance"` // in km
	EstimatedDuration int     `gorm:"" json:"estimatedDuration"` // in minutes
	ActualDuration    int     `gorm:"" json:"actualDuration"`

	// Retry / failure tracking
	AttemptNumber int    `gorm:"default:1" json:"attemptNumber"`
	MaxAttempts   int    `gorm:"default:3" json:"maxAttempts"`
	FailureReason string `gorm:"" json:"failureReason,omitempty"` // customer_unavailable, wrong_address, refused, etc.

	// Assignment tracking
	AssignmentType AssignmentType `gorm:"type:varchar(20);default:'manual'" json:"assignmentType"`
	AssignedByID   *uuid.UUID     `gorm:"type:uuid" json:"assignedById,omitempty"`
	OfferExpiresAt *time.Time     `gorm:"" json:"offerExpiresAt,omitempty"`

	// Third-party provider fields (only set when AssignmentType = "third_party")
	ProviderID          *uuid.UUID `gorm:"type:uuid" json:"providerId,omitempty"`
	ExternalDeliveryID  string     `gorm:"" json:"externalDeliveryId,omitempty"`
	ExternalTrackingID  string     `gorm:"" json:"externalTrackingId,omitempty"`
	ExternalTrackingURL string     `gorm:"" json:"externalTrackingUrl,omitempty"`
	ProviderCost        float64    `gorm:"default:0" json:"providerCost"` // what Fe3dr pays the provider

	// Earnings
	DeliveryFee float64 `gorm:"default:0" json:"deliveryFee"`
	Tip         float64 `gorm:"default:0" json:"tip"`
	TotalPayout float64 `gorm:"default:0" json:"totalPayout"`

	// Timestamps
	AssignedAt   time.Time  `gorm:"autoCreateTime" json:"assignedAt"`
	PickedUpAt   *time.Time `gorm:"" json:"pickedUpAt,omitempty"`
	DeliveredAt  *time.Time `gorm:"" json:"deliveredAt,omitempty"`
	CancelledAt  *time.Time `gorm:"" json:"cancelledAt,omitempty"`
	CancelReason string     `gorm:"" json:"cancelReason,omitempty"`

	// Relationships
	Order           Order           `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	DeliveryPartner DeliveryPartner  `gorm:"foreignKey:DeliveryPartnerID" json:"deliveryPartner,omitempty"`
	AssignedBy      *User            `gorm:"foreignKey:AssignedByID" json:"assignedBy,omitempty"`
	Provider        *DeliveryProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

// DTOs
type DeliveryPartnerResponse struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"userId"`
	VehicleType     string    `json:"vehicleType"`
	IsVerified      bool      `json:"verified"`
	IsOnline        bool      `json:"isOnline"`
	Rating          float64   `json:"rating"`
	TotalDeliveries int       `json:"totalDeliveries"`
	AgentType       AgentType `json:"agentType"`
	VerificationStatus VerificationStatus `json:"verificationStatus"`
}

// DeliveryResponse is the customer-facing delivery tracking response.
// Does NOT include driver earnings (TotalPayout) or driver identity (DeliveryPartnerID).
type DeliveryResponse struct {
	ID                uuid.UUID      `json:"id"`
	OrderID           uuid.UUID      `json:"orderId"`
	Status            DeliveryStatus `json:"status"`
	Distance          float64        `json:"distance"`
	EstimatedDuration int            `json:"estimatedDuration"`
	DeliveryFee       float64        `json:"deliveryFee"`
	AssignedAt        time.Time      `json:"assignedAt"`
	PickedUpAt        *time.Time     `json:"pickedUpAt,omitempty"`
	DeliveredAt       *time.Time     `json:"deliveredAt,omitempty"`
}

func (d *Delivery) ToResponse() DeliveryResponse {
	return DeliveryResponse{
		ID:                d.ID,
		OrderID:           d.OrderID,
		Status:            d.Status,
		Distance:          d.Distance,
		EstimatedDuration: d.EstimatedDuration,
		DeliveryFee:       d.DeliveryFee,
		AssignedAt:        d.AssignedAt,
		PickedUpAt:        d.PickedUpAt,
		DeliveredAt:       d.DeliveredAt,
	}
}

// DeliveryPartnerDetailResponse includes more info than the basic response
type DeliveryPartnerDetailResponse struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"userId"`
	VehicleType      string     `json:"vehicleType"`
	VehicleNumber    string     `json:"vehicleNumber"`
	LicenseNumber    string     `json:"licenseNumber"`
	IsVerified       bool       `json:"verified"`
	VerifiedAt       *time.Time `json:"verifiedAt,omitempty"`
	IsActive         bool       `json:"isActive"`
	IsOnline         bool       `json:"isOnline"`
	CurrentLatitude  float64    `json:"currentLatitude"`
	CurrentLongitude float64    `json:"currentLongitude"`
	Rating           float64    `json:"rating"`
	TotalDeliveries  int        `json:"totalDeliveries"`
	TotalReviews     int        `json:"totalReviews"`
	CreatedAt        time.Time  `json:"createdAt"`
	Name             string     `json:"name,omitempty"`
	Email            string     `json:"email,omitempty"`
	Phone            string     `json:"phone,omitempty"`
	Avatar           string     `json:"avatar,omitempty"`

	// Extended fields
	AgentType          AgentType          `json:"agentType"`
	EmployeeID         string             `json:"employeeId,omitempty"`
	MaxConcurrent      int                `json:"maxConcurrent"`
	AcceptanceRate     float64            `json:"acceptanceRate"`
	OnTimeRate         float64            `json:"onTimeRate"`
	CSATScore          float64            `json:"csatScore"`
	OfferedCount       int                `json:"offeredCount"`
	AcceptedCount      int                `json:"acceptedCount"`
	CompletedOnTime    int                `json:"completedOnTime"`
	VerificationStatus VerificationStatus `json:"verificationStatus"`
	RejectionReason    string             `json:"rejectionReason,omitempty"`
	Documents          []PartnerDocumentResponse `json:"documents,omitempty"`

	// Onboarding fields
	City               string `json:"city,omitempty"`
	EmergencyContact   string `json:"emergencyContact,omitempty"`
	EmergencyPhone     string `json:"emergencyPhone,omitempty"`
	VehicleMake         string `json:"vehicleMake,omitempty"`
	VehicleModel        string `json:"vehicleModel,omitempty"`
	VehicleYear         int    `json:"vehicleYear,omitempty"`
	VehicleColor        string `json:"vehicleColor,omitempty"`
	HasDeliveryBoxSpace bool   `json:"hasDeliveryBoxSpace,omitempty"`
	PayoutMethod        string `json:"payoutMethod,omitempty"`
	OnboardingStep     int    `json:"onboardingStep"`
	OnboardingComplete bool   `json:"onboardingComplete"`
	ReferralCode       string `json:"referralCode,omitempty"`
}

func (p *DeliveryPartner) ToDetailResponse() DeliveryPartnerDetailResponse {
	resp := DeliveryPartnerDetailResponse{
		ID:                 p.ID,
		UserID:             p.UserID,
		VehicleType:        p.VehicleType,
		VehicleNumber:      p.VehicleNumber,
		LicenseNumber:      p.LicenseNumber,
		IsVerified:         p.IsVerified,
		VerifiedAt:         p.VerifiedAt,
		IsActive:           p.IsActive,
		IsOnline:           p.IsOnline,
		CurrentLatitude:    p.CurrentLatitude,
		CurrentLongitude:   p.CurrentLongitude,
		Rating:             p.Rating,
		TotalDeliveries:    p.TotalDeliveries,
		TotalReviews:       p.TotalReviews,
		CreatedAt:          p.CreatedAt,
		AgentType:          p.AgentType,
		EmployeeID:         p.EmployeeID,
		MaxConcurrent:      p.MaxConcurrent,
		AcceptanceRate:     p.AcceptanceRate,
		OnTimeRate:         p.OnTimeRate,
		CSATScore:          p.CSATScore,
		OfferedCount:       p.OfferedCount,
		AcceptedCount:      p.AcceptedCount,
		CompletedOnTime:    p.CompletedOnTime,
		VerificationStatus: p.VerificationStatus,
		RejectionReason:    p.RejectionReason,
		City:               p.City,
		EmergencyContact:   p.EmergencyContact,
		EmergencyPhone:     p.EmergencyPhone,
		VehicleMake:        p.VehicleMake,
		VehicleModel:       p.VehicleModel,
		VehicleYear:        p.VehicleYear,
		VehicleColor:        p.VehicleColor,
		HasDeliveryBoxSpace: p.HasDeliveryBoxSpace,
		PayoutMethod:        p.PayoutMethod,
		OnboardingStep:     p.OnboardingStep,
		OnboardingComplete: p.OnboardingComplete,
		ReferralCode:       p.ReferralCode,
	}
	if p.User.ID != uuid.Nil {
		resp.Name = p.User.FirstName + " " + p.User.LastName
		resp.Email = p.User.Email
		resp.Phone = p.User.Phone
		resp.Avatar = p.User.Avatar
	}
	if len(p.Documents) > 0 {
		resp.Documents = make([]PartnerDocumentResponse, len(p.Documents))
		for i, doc := range p.Documents {
			resp.Documents[i] = doc.ToResponse()
		}
	}
	return resp
}

// Referral types

type ReferralStatus string

const (
	ReferralPending   ReferralStatus = "pending"
	ReferralCompleted ReferralStatus = "completed"
	ReferralPaid      ReferralStatus = "paid"
	ReferralExpired   ReferralStatus = "expired"
)

type DriverReferral struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ReferrerID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"referrerId"`
	RefereeID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"refereeId"`
	ReferralCode string         `gorm:"not null" json:"referralCode"`
	Status       ReferralStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	BonusAmount  float64        `gorm:"default:0" json:"bonusAmount"`
	PaidAt       *time.Time     `gorm:"" json:"paidAt,omitempty"`
	ExpiresAt    time.Time      `gorm:"not null" json:"expiresAt"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`

	Referrer DeliveryPartner `gorm:"foreignKey:ReferrerID" json:"referrer,omitempty"`
	Referee  DeliveryPartner `gorm:"foreignKey:RefereeID" json:"referee,omitempty"`
}
