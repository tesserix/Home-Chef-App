package markets

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Policy struct {
	CountryCode     string `json:"country_code"`
	Name            string `json:"name"`
	Currency        string `json:"currency"`
	PaymentProvider string `json:"payment_provider"`
	Status          string `json:"status"`
}

func catalog() [3]Policy {
	return [3]Policy{
		{CountryCode: "IN", Name: "India", Currency: "INR", PaymentProvider: "cashfree", Status: "active"},
		{CountryCode: "AU", Name: "Australia", Currency: "AUD", PaymentProvider: "stripe", Status: "planned"},
		{CountryCode: "NZ", Name: "New Zealand", Currency: "NZD", PaymentProvider: "stripe", Status: "planned"},
	}
}

func Lookup(country string) (Policy, bool) {
	code := strings.ToUpper(strings.TrimSpace(country))
	for _, policy := range catalog() {
		if policy.CountryCode == code {
			return policy, true
		}
	}
	return Policy{}, false
}

// RegisterRoutes exposes discovery only; it never authorizes payment creation.
func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/markets", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"markets": catalog()})
	})
	v1.GET("/markets/:country", func(c *gin.Context) {
		policy, ok := Lookup(c.Param("country"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "Unsupported market", "code": "unsupported_market"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"market": policy})
	})
}
