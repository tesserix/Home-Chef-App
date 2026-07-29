package handlers

// chef_penalty.go — admin + chef surfaces for the cancellation levy (#834 item 6).
//
// The admin queue exists because the levy is only defensible WITH a waiver: an auto-fine with
// no human recourse punishes a genuine emergency exactly as hard as a careless cancellation.
// The chef-side list exists so a levy is never a surprise deduction discovered on a statement.

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// ChefPenaltyHandler owns the levy endpoints.
type ChefPenaltyHandler struct{}

func NewChefPenaltyHandler() *ChefPenaltyHandler { return &ChefPenaltyHandler{} }

// chefPenaltyRow is one levy as the admin and chef surfaces render it.
type chefPenaltyRow struct {
	ID          string  `json:"id"`
	ChefID      string  `json:"chefId"`
	ChefName    string  `json:"chefName,omitempty"`
	Status      string  `json:"status"`
	Kind        string  `json:"kind"`
	Reference   string  `json:"reference,omitempty"`
	BasisAmount float64 `json:"basisAmount"`
	RatePercent float64 `json:"ratePercent"`
	Amount      float64 `json:"amount"`
	LeadHours   float64 `json:"leadHours"`
	Reason      string  `json:"reason,omitempty"`
	WaiveReason string  `json:"waiveReason,omitempty"`
	OccurredAt  string  `json:"occurredAt"`
}

func toChefPenaltyRow(p *models.ChefPenalty, chefName string) chefPenaltyRow {
	return chefPenaltyRow{
		ID: p.ID.String(), ChefID: p.ChefID.String(), ChefName: chefName,
		Status: string(p.Status), Kind: string(p.Kind), Reference: p.Reference,
		BasisAmount: p.BasisAmount, RatePercent: p.RatePercent, Amount: p.Amount,
		LeadHours: p.LeadHours, Reason: p.Reason, WaiveReason: p.WaiveReason,
		OccurredAt: p.OccurredAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ListChefPenalties — GET /admin/chef-penalties?status=pending. Platform-wide levy queue.
func (h *ChefPenaltyHandler) ListChefPenalties(c *gin.Context) {
	q := database.DB.Model(&models.ChefPenalty{}).Order("occurred_at DESC").Limit(200)
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []models.ChefPenalty
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load penalties"})
		return
	}
	out := make([]chefPenaltyRow, 0, len(rows))
	for i := range rows {
		var chef models.ChefProfile
		database.DB.Select("id", "business_name").First(&chef, "id = ?", rows[i].ChefID)
		out = append(out, toChefPenaltyRow(&rows[i], chef.BusinessName))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// WaiveChefPenalty — POST /admin/chef-penalties/:id/waive {"reason":"..."}. Cancels a pending
// levy. A reason is REQUIRED: a waiver with no recorded justification is indistinguishable from
// a mistake when the deduction is later queried.
func (h *ChefPenaltyHandler) WaiveChefPenalty(c *gin.Context) {
	penaltyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid penalty id"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A reason is required to waive a penalty"})
		return
	}
	adminID, _ := middleware.GetUserID(c)
	if err := services.WaiveChefPenalty(database.DB, penaltyID, adminID, req.Reason); err != nil {
		if errors.Is(err, services.ErrPenaltyNotPending) {
			c.JSON(http.StatusConflict, gin.H{"error": "This penalty has already been waived or deducted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to waive the penalty"})
		return
	}
	services.LogAudit(c, "chef.penalty.waive", "chef_penalty", penaltyID.String(), nil,
		gin.H{"reason": req.Reason})
	c.JSON(http.StatusOK, gin.H{"status": "waived"})
}

// GetMyChefPenalties — GET /chef/penalties. The chef's own levy history plus the total their
// next settlement will deduct, so a deduction is never a surprise.
func (h *ChefPenaltyHandler) GetMyChefPenalties(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	var rows []models.ChefPenalty
	database.DB.Where("chef_id = ?", chef.ID).Order("occurred_at DESC").Limit(100).Find(&rows)
	out := make([]chefPenaltyRow, 0, len(rows))
	for i := range rows {
		out = append(out, toChefPenaltyRow(&rows[i], ""))
	}
	pending, err := services.PendingChefPenaltyTotal(database.DB, chef.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load penalties"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "pendingTotal": pending})
}

// ListCreditNotes — GET /admin/credit-notes. The GST credit notes issued against refunds, the
// artefact the filing adjustment is built from (#834 item 2).
func (h *ChefPenaltyHandler) ListCreditNotes(c *gin.Context) {
	var rows []models.CreditNote
	if err := database.DB.Order("issued_at DESC").Limit(500).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load credit notes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}
