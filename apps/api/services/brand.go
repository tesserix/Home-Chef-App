package services

// The one place the product is named. Every customer- and chef-facing document
// (invoice, statement, TDS certificate, email) reads from here, so the brand can
// never be half-renamed across surfaces again.
const (
	BrandName      = "Fe3dr"
	BrandLegalName = "Fe3dr Marketplace"
	BrandWebsite   = "www.fe3dr.com"
)
