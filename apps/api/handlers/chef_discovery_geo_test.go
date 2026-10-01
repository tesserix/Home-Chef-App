package handlers

import (
	"math"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// Melbourne: far enough from the equator that ignoring cos(lat) on longitude
// misjudges an east-west distance by ~25%.
const melLat, melLng = -37.8136, 144.9631

func offsetKm(northKm, eastKm float64) (float64, float64) {
	return melLat + northKm/111.0, melLng + eastKm/(111.0*math.Cos(melLat*math.Pi/180))
}

func setupGeoDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, latitude REAL, longitude REAL)`).Error)
	place := func(id string, northKm, eastKm float64) {
		lat, lng := offsetKm(northKm, eastKm)
		require.NoError(t, db.Exec(`INSERT INTO chef_profiles VALUES (?, ?, ?)`, id, lat, lng).Error)
	}
	place("far-north-25km", 25, 0)
	place("east-19km", 0, 19)
	place("diagonal-24km", 17, 17)
	place("north-15km", 15, 0)
	place("here", 0, 0)
	return db
}

func nearbyIDs(t *testing.T, db *gorm.DB, radiusKm float64) []string {
	t.Helper()
	var ids []string
	require.NoError(t, db.Table("chef_profiles").
		Scopes(withinRadius(melLat, melLng, radiusKm)).
		Order(nearestFirst(melLat, melLng, "", "")).
		Pluck("id", &ids).Error)
	return ids
}

func TestDiscoveryShowsOnlyKitchensWithinTheCircleNearestFirst(t *testing.T) {
	db := setupGeoDB(t)

	require.Equal(t, []string{"here", "north-15km", "east-19km"}, nearbyIDs(t, db, discoveryRadiusKm))
}

func TestDiscoveryExcludesTheCornersOfTheBoundingSquare(t *testing.T) {
	db := setupGeoDB(t)

	require.NotContains(t, nearbyIDs(t, db, 20), "diagonal-24km",
		"17km north + 17km east is 24km away and must not ride in on the square's corner")
}

func TestDiscoveryKeepsPaidPlacementAboveDistance(t *testing.T) {
	db := setupGeoDB(t)
	var ids []string
	require.NoError(t, db.Table("chef_profiles").
		Scopes(withinRadius(melLat, melLng, discoveryRadiusKm)).
		Order(nearestFirst(melLat, melLng, "CASE WHEN id = 'east-19km' THEN 0 ELSE 1 END ASC", "")).
		Pluck("id", &ids).Error)

	require.Equal(t, []string{"east-19km", "here", "north-15km"}, ids)
}

func TestDiscoveryHonoursASmallerRequestedRadius(t *testing.T) {
	db := setupGeoDB(t)

	require.Equal(t, []string{"here"}, nearbyIDs(t, db, 10))
}

func TestDiscoveryRadiusParsing(t *testing.T) {
	cases := map[string]float64{
		"":      20,
		"5":     5,
		"20":    20,
		"30":    20,
		"20000": 20, // legacy clients sent metres
		"5000":  5,
		"0":     20,
		"-3":    20,
		"abc":   20,
	}
	for raw, want := range cases {
		require.Equal(t, want, requestedDiscoveryRadius(raw), "radius=%q", raw)
	}
}

func TestDiscoveryBreaksDistanceTiesWithTheGivenOrder(t *testing.T) {
	db := setupGeoDB(t)
	lat, lng := offsetKm(0, 0)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles VALUES ('here-too', ?, ?)`, lat, lng).Error)
	var ids []string
	require.NoError(t, db.Table("chef_profiles").
		Scopes(withinRadius(melLat, melLng, 1)).
		Order(nearestFirst(melLat, melLng, "", "id DESC")).
		Pluck("id", &ids).Error)

	require.Equal(t, []string{"here-too", "here"}, ids)
}

func TestQueryCoordsRejectsImpossibleLocations(t *testing.T) {
	cases := map[string]bool{
		"lat=-37.81&lng=144.96": true,
		"lat=-91&lng=144.96":    false,
		"lat=-37.81&lng=181":    false,
		"lat=NaN&lng=144.96":    false,
		"lat=-37.81&lng=Inf":    false,
		"lat=-37.81":            false,
		"":                      false,
	}
	for query, want := range cases {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/v1/chefs?"+query, nil)
		_, _, ok := queryCoords(c)
		require.Equal(t, want, ok, query)
	}
}
