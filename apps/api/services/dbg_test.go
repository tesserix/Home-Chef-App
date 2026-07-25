package services

import (
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/homechef/api/models"
)

func TestDbgPurge(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleChef)
	chefID := seedAccountChef(t, db, user.ID)

	_, err := RequestDeletion(db, &user, "")
	require.NoError(t, err)

	var n int64
	db.Raw(`SELECT COUNT(*) FROM menu_items WHERE chef_id = ?`, chefID).Scan(&n)
	t.Logf("menu_items after RequestDeletion=%d", n)

	err = PurgeUser(db, user.ID, models.RoleChef)
	t.Logf("PurgeUser err=%v", err)

	db.Raw(`SELECT COUNT(*) FROM menu_items WHERE chef_id = ?`, chefID).Scan(&n)
	t.Logf("menu_items after purge=%d", n)
	db.Raw(`SELECT COUNT(*) FROM chef_profiles WHERE id = ?`, chefID).Scan(&n)
	t.Logf("chef_profiles after purge=%d", n)
	db.Raw(`SELECT COUNT(*) FROM chef_documents WHERE chef_id = ?`, chefID).Scan(&n)
	t.Logf("chef_documents after purge=%d", n)
}
