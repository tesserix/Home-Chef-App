package handlers

// admin_easy_split_rollout_test.go — #1084. The rollout switch is only useful
// if an operator can drive it, and only safe if a typo cannot drive it: an
// unrecognised value would read back as "follow the platform flag" and move a
// chef's money onto a rail nobody chose. The operator UI itself lives in
// tesserix-home; this is the API it drives.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func seedRolloutChef(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, mode, business_name) VALUES (?, 'live', 'Ammas Kitchen')`,
		id.String()).Error)
	return id
}

func putEasySplitMode(t *testing.T, r http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func chefEasySplitMode(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", id.String()).Error)
	return chef.EasySplitMode
}

func TestSetChefEasySplitMode_FlipsOneChef(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := seedRolloutChef(t, db)

	w := putEasySplitMode(t, r, "/admin/chefs/"+chefID.String()+"/easy-split-mode", `{"value":"on"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, services.PayoutAutoOn, chefEasySplitMode(t, db, chefID))

	var resp struct {
		EasySplitMode string `json:"easySplitMode"`
		Effective     bool   `json:"effective"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, services.PayoutAutoOn, resp.EasySplitMode)
	require.True(t, resp.Effective, "the platform flag is off, so only the override can explain this")
}

// Clearing the override returns the chef to the platform flag.
func TestSetChefEasySplitMode_AcceptsAnEmptyValueAsInherit(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := seedRolloutChef(t, db)

	require.Equal(t, http.StatusOK,
		putEasySplitMode(t, r, "/admin/chefs/"+chefID.String()+"/easy-split-mode", `{"value":"off"}`).Code)
	require.Equal(t, http.StatusOK,
		putEasySplitMode(t, r, "/admin/chefs/"+chefID.String()+"/easy-split-mode", `{"value":""}`).Code)

	require.Empty(t, chefEasySplitMode(t, db, chefID))
}

func TestSetChefEasySplitMode_RejectsAnythingElse(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := seedRolloutChef(t, db)
	require.Equal(t, http.StatusOK,
		putEasySplitMode(t, r, "/admin/chefs/"+chefID.String()+"/easy-split-mode", `{"value":"off"}`).Code)

	for _, bad := range []string{`{"value":"ON "}`, `{"value":"true"}`, `{"value":"enabled"}`} {
		w := putEasySplitMode(t, r, "/admin/chefs/"+chefID.String()+"/easy-split-mode", bad)
		require.Equal(t, http.StatusBadRequest, w.Code, bad)
	}
	require.Equal(t, services.PayoutAutoOff, chefEasySplitMode(t, db, chefID),
		"a rejected value must not disturb what was there")
}

func TestSetChefEasySplitMode_UnknownChefIs404(t *testing.T) {
	setupChefPayoutProfileDB(t)
	r := profileRouter()

	w := putEasySplitMode(t, r, "/admin/chefs/"+uuid.NewString()+"/easy-split-mode", `{"value":"on"}`)
	require.Equal(t, http.StatusNotFound, w.Code)
}

// A cohort is how a rollout actually proceeds — ten chefs at a time, not one
// request per chef with a partial state if the operator's tab closes halfway.
func TestSetEasySplitModeBulk_FlipsACohort(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	first, second := seedRolloutChef(t, db), seedRolloutChef(t, db)

	w := putEasySplitMode(t, r, "/admin/chefs/easy-split-mode",
		`{"value":"on","chefIds":["`+first.String()+`","`+second.String()+`"]}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, services.PayoutAutoOn, chefEasySplitMode(t, db, first))
	require.Equal(t, services.PayoutAutoOn, chefEasySplitMode(t, db, second))

	var resp struct {
		Updated int `json:"updated"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 2, resp.Updated)
}

// The operator screen needs both halves of the decision: what this chef is set
// to, and what that resolves to once the platform flag is taken into account.
func TestGetChefPayoutProfile_ReportsTheRolloutDecision(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := seedRolloutChef(t, db)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET easy_split_mode = 'on' WHERE id = ?`,
		chefID.String()).Error)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/chefs/"+chefID.String()+"/payout-profile", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		EasySplit struct {
			Enabled   bool   `json:"enabled"`
			Mode      string `json:"mode"`
			Effective bool   `json:"effective"`
		} `json:"easySplit"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.EasySplit.Enabled, "the platform flag is still off")
	require.Equal(t, services.PayoutAutoOn, resp.EasySplit.Mode)
	require.True(t, resp.EasySplit.Effective)
}

func TestSetEasySplitModeBulk_RejectsAnEmptyCohortOrABadValue(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := seedRolloutChef(t, db)

	require.Equal(t, http.StatusBadRequest,
		putEasySplitMode(t, r, "/admin/chefs/easy-split-mode", `{"value":"on","chefIds":[]}`).Code)
	require.Equal(t, http.StatusBadRequest,
		putEasySplitMode(t, r, "/admin/chefs/easy-split-mode",
			`{"value":"yes","chefIds":["`+chefID.String()+`"]}`).Code)
	require.Empty(t, chefEasySplitMode(t, db, chefID))
}
