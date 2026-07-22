package handlers

// menu_image_race_test.go — concurrency proof for the menu-image slot claim.
//
// This test needs real Postgres. sqlite in-memory serialises writes and treats
// SELECT ... FOR UPDATE as a no-op, so it cannot demonstrate the race in either
// direction — a green run there would be meaningless.
//
// Set HOMECHEF_TEST_PG to run, e.g.
//   HOMECHEF_TEST_PG='host=127.0.0.1 port=55432 user=postgres dbname=racetest sslmode=disable'
//
// The defect: UploadMenuItemImage counted existing images, then inserted, with
// no lock. The picker uploads a multi-selection concurrently, so several
// requests read the same count and all pass — exceeding the 5-image cap, giving
// more than one row IsPrimary (so item.image_url is written twice), and
// colliding sort_order.

import (
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func racePG(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("HOMECHEF_TEST_PG")
	if dsn == "" {
		t.Skip("HOMECHEF_TEST_PG not set — concurrency proof needs real Postgres")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	return db
}

// seedRaceItem creates an isolated item with `existing` images already attached.
func seedRaceItem(t *testing.T, db *gorm.DB, existing int) uuid.UUID {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS menu_items (
		id uuid PRIMARY KEY, chef_id uuid, name text, price numeric,
		image_url text DEFAULT '', created_at timestamptz, updated_at timestamptz,
		deleted_at timestamptz)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS menu_item_images (
		id uuid PRIMARY KEY DEFAULT gen_random_uuid(), menu_item_id uuid NOT NULL,
		url text NOT NULL, is_primary boolean DEFAULT false, sort_order int DEFAULT 0,
		created_at timestamptz, updated_at timestamptz, deleted_at timestamptz)`).Error)

	itemID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO menu_items (id, chef_id, name, price) VALUES (?,?,?,?)`,
		itemID, uuid.New(), "Dal Fry", 120).Error)
	for i := 0; i < existing; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO menu_item_images (id, menu_item_id, url, is_primary, sort_order)
			 VALUES (?,?,?,?,?)`,
			uuid.New(), itemID, "https://example/seed.jpg", i == 0, i).Error)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM menu_item_images WHERE menu_item_id = ?`, itemID)
		db.Exec(`DELETE FROM menu_items WHERE id = ?`, itemID)
	})
	return itemID
}

// claimConcurrently fires n goroutines at the same item and returns how many
// succeeded.
func claimConcurrently(t *testing.T, db *gorm.DB, itemID uuid.UUID, n int) int {
	t.Helper()
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // release them together to maximise overlap
			if _, err := claimMenuItemImageSlot(db, itemID, "https://example/new.jpg"); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(i)
	}
	close(start)
	wg.Wait()
	return ok
}

func imageStats(t *testing.T, db *gorm.DB, itemID uuid.UUID) (total, primaries, distinctSort int) {
	t.Helper()
	require.NoError(t, db.Raw(`SELECT count(*) FROM menu_item_images WHERE menu_item_id = ?`, itemID).Scan(&total).Error)
	require.NoError(t, db.Raw(`SELECT count(*) FROM menu_item_images WHERE menu_item_id = ? AND is_primary`, itemID).Scan(&primaries).Error)
	require.NoError(t, db.Raw(`SELECT count(DISTINCT sort_order) FROM menu_item_images WHERE menu_item_id = ?`, itemID).Scan(&distinctSort).Error)
	return
}

func TestMenuImageRace_CapIsNeverExceeded(t *testing.T) {
	db := racePG(t)
	// One slot left. Eight racers want it.
	itemID := seedRaceItem(t, db, maxMenuItemImages-1)

	succeeded := claimConcurrently(t, db, itemID, 8)

	total, _, _ := imageStats(t, db, itemID)
	require.LessOrEqual(t, total, maxMenuItemImages,
		"concurrent uploads must not push the gallery past the cap")
	require.Equal(t, 1, succeeded,
		"exactly one racer should win the last slot")
}

func TestMenuImageRace_ExactlyOnePrimary(t *testing.T) {
	db := racePG(t)
	// Empty item: every racer reads count==0 and claims IsPrimary unless locked.
	// Two primaries means item.image_url gets written twice, last write wins.
	itemID := seedRaceItem(t, db, 0)

	claimConcurrently(t, db, itemID, 8)

	total, primaries, _ := imageStats(t, db, itemID)
	require.Equal(t, 1, primaries,
		"exactly one image may be primary, whatever the interleaving (got %d of %d rows)", primaries, total)
}

func TestMenuImageRace_SortOrderNeverCollides(t *testing.T) {
	db := racePG(t)
	itemID := seedRaceItem(t, db, 0)

	claimConcurrently(t, db, itemID, 5)

	total, _, distinct := imageStats(t, db, itemID)
	require.Equal(t, total, distinct,
		"every image needs its own sort_order or gallery order is undefined")
}

func TestMenuImageRace_FullItemAdmitsNoOne(t *testing.T) {
	db := racePG(t)
	itemID := seedRaceItem(t, db, maxMenuItemImages)

	succeeded := claimConcurrently(t, db, itemID, 6)

	total, _, _ := imageStats(t, db, itemID)
	require.Equal(t, 0, succeeded, "a full gallery must reject every racer")
	require.Equal(t, maxMenuItemImages, total)
}
