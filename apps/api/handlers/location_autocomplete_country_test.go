package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func stubPhoton(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"features":[
			{"geometry":{"coordinates":[144.96,-37.81]},"properties":{"name":"Collins St","city":"Melbourne","state":"Victoria","postcode":"3000","countrycode":"AU"}},
			{"geometry":{"coordinates":[174.76,-36.85]},"properties":{"name":"Queen St","city":"Auckland","state":"Auckland","postcode":"1010","countrycode":"NZ"}},
			{"geometry":{"coordinates":[77.59,12.97]},"properties":{"name":"MG Road","city":"Bengaluru","state":"Karnataka","postcode":"560001","countrycode":"IN"}}
		]}`))
	}))
	t.Cleanup(srv.Close)
	prev := photonAPI
	photonAPI = srv.URL
	t.Cleanup(func() { photonAPI = prev })
}

func autocomplete(t *testing.T, query string) []AddressSuggestion {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/locations/autocomplete?"+query, nil)
	(&LocationHandler{}).AutocompleteAddresses(c)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data []AddressSuggestion `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Data
}

func TestAutocomplete_ScopesSuggestionsToTheRequestedCountry(t *testing.T) {
	stubPhoton(t)

	au := autocomplete(t, "q=Collins+St&country=AU")
	require.Len(t, au, 1)
	require.Equal(t, "Melbourne", au[0].City)

	nz := autocomplete(t, "q=Queen+St&country=nz")
	require.Len(t, nz, 1)
	require.Equal(t, "NZ", nz[0].Country)

	in := autocomplete(t, "q=MG+Road")
	require.Len(t, in, 1, "no country is India, as before")
	require.Equal(t, "IN", in[0].Country)
}
