package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type marketGeocoderTransport func(*http.Request) (*http.Response, error)

func (f marketGeocoderTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAutocompleteUsesSelectedCountry(t *testing.T) {
	old := photonClient
	t.Cleanup(func() { photonClient = old })
	for _, country := range []string{"AU", "NZ"} {
		t.Run(country, func(t *testing.T) {
			calls := 0
			photonClient = &http.Client{Transport: marketGeocoderTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, strings.ToLower(country), r.URL.Query().Get("countrycode"))
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"features":[{"properties":{"countrycode":"au","city":"Melbourne","postcode":"3000"}},{"properties":{"countrycode":"nz","city":"Auckland","postcode":"1010"}},{"properties":{"countrycode":"in","city":"Delhi"}}]}`)), Header: make(http.Header)}, nil
			})}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/locations/autocomplete?q=Test&country="+country, nil)
			NewLocationHandler().AutocompleteAddresses(c)
			require.Equal(t, 200, w.Code)
			require.Equal(t, 1, calls)
			require.Contains(t, w.Body.String(), `"country":"`+country+`"`)
			require.NotContains(t, w.Body.String(), "Delhi")
		})
	}
}

func TestAutocompleteRejectsUnsupportedCountry(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/locations/autocomplete?q=Test&country=US", nil)
	NewLocationHandler().AutocompleteAddresses(c)
	require.Equal(t, 400, w.Code)
}
