package services

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	vision "cloud.google.com/go/vision/apiv1"

	"github.com/homechef/api/services/docverify"
)

// visionClient is the Cloud Vision OCR client. Initialised at startup via
// InitVision using the same default (workload-identity) credentials as GCS.
var visionClient *vision.ImageAnnotatorClient

// InitVision creates the Cloud Vision client. Callers should treat a failure
// as non-fatal — OCR is a convenience (pre-fill), never a hard dependency.
func InitVision() error {
	ctx := context.Background()
	c, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Vision client: %w", err)
	}
	visionClient = c
	return nil
}

// CloseVision releases the Vision client.
func CloseVision() {
	if visionClient != nil {
		visionClient.Close()
	}
}

// OCRResult holds the fields we try to lift off a document image. Both are
// best-effort — the chef always confirms/edits before anything is saved.
type OCRResult struct {
	FSSAINumber string `json:"fssaiNumber"`
	ExpiryDate  string `json:"expiryDate"` // ISO YYYY-MM-DD
}

var (
	// FSSAI licence numbers are exactly 14 digits.
	fssaiRe = regexp.MustCompile(`\b(\d{14})\b`)
	// Numeric dates: 01/01/2027, 1-1-27, 01.01.2027 (Indian DD/MM/YYYY).
	numDateRe = regexp.MustCompile(`(\d{1,2})\s*[/\-.]\s*(\d{1,2})\s*[/\-.]\s*(\d{2,4})`)
	// Month-name dates: 01 Jan 2027 / 1 January 2027.
	monDateRe = regexp.MustCompile(`(?i)(\d{1,2})\s+([A-Za-z]{3,9})\.?\s+(\d{4})`)
)

var monthAbbrev = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

// DetectText runs document-text OCR on an image and returns the raw text.
func DetectText(ctx context.Context, imageBytes []byte) (string, error) {
	if visionClient == nil {
		return "", fmt.Errorf("vision client not initialised")
	}
	img, err := vision.NewImageFromReader(bytes.NewReader(imageBytes))
	if err != nil {
		return "", fmt.Errorf("invalid image: %w", err)
	}
	ann, err := visionClient.DetectDocumentText(ctx, img, nil)
	if err != nil {
		return "", fmt.Errorf("vision OCR failed: %w", err)
	}
	if ann == nil {
		return "", nil
	}
	return ann.GetText(), nil
}

// DetectDocumentFields runs document-text OCR on an image and extracts the
// FSSAI number + expiry date. Returns a zero-value result (no error) when the
// fields aren't found, so the caller can degrade to manual entry.
func DetectDocumentFields(ctx context.Context, imageBytes []byte) (OCRResult, error) {
	var res OCRResult
	text, err := DetectText(ctx, imageBytes)
	if err != nil {
		return res, err
	}
	if m := fssaiRe.FindString(text); m != "" {
		res.FSSAINumber = m
	}
	res.ExpiryDate = extractExpiry(text)
	return res, nil
}

// AssessDocumentImage OCRs an image and returns a genuineness verdict for the
// claimed document type. Errors when OCR is unavailable so the caller can
// decide to fail open.
func AssessDocumentImage(ctx context.Context, imageBytes []byte, claimedType string) (docverify.Verdict, error) {
	text, err := DetectText(ctx, imageBytes)
	if err != nil {
		return docverify.Verdict{}, err
	}
	return docverify.Assess(claimedType, text), nil
}

type dateMatch struct {
	iso string
	idx int
	t   time.Time
}

// extractExpiry finds the most likely expiry date in the OCR text: it prefers
// a date sitting just after a "valid upto / validity / expiry" keyword and in
// the future, falling back to the latest plausible date on the page.
func extractExpiry(text string) string {
	lower := strings.ToLower(text)
	var matches []dateMatch

	for _, m := range numDateRe.FindAllStringSubmatchIndex(text, -1) {
		if iso, t, ok := buildISO(text[m[2]:m[3]], text[m[4]:m[5]], text[m[6]:m[7]]); ok {
			matches = append(matches, dateMatch{iso, m[0], t})
		}
	}
	for _, m := range monDateRe.FindAllStringSubmatchIndex(text, -1) {
		mon := strings.ToLower(text[m[4]:m[5]])
		moNum, ok := monthAbbrev[mon[:min(3, len(mon))]]
		if !ok {
			continue
		}
		if iso, t, ok2 := buildISO(text[m[2]:m[3]], strconv.Itoa(moNum), text[m[6]:m[7]]); ok2 {
			matches = append(matches, dateMatch{iso, m[0], t})
		}
	}
	if len(matches) == 0 {
		return ""
	}

	keywords := []string{"valid upto", "valid up to", "validity", "valid till", "expiry", "expires", "upto"}
	best, bestScore := "", -1
	for _, dm := range matches {
		score := 0
		if dm.t.After(time.Now()) {
			score += 2
		}
		start := dm.idx - 40
		if start < 0 {
			start = 0
		}
		before := lower[start:dm.idx]
		for _, k := range keywords {
			if strings.Contains(before, k) {
				score += 5
				break
			}
		}
		if score > bestScore {
			bestScore, best = score, dm.iso
		}
	}
	if bestScore <= 0 {
		// No keyword/future signal — take the latest date on the page.
		latest := matches[0]
		for _, dm := range matches[1:] {
			if dm.t.After(latest.t) {
				latest = dm
			}
		}
		best = latest.iso
	}
	return best
}

// buildISO validates a D/M/Y triple and returns it as YYYY-MM-DD. Rejects
// impossible dates (e.g. 31/02) by checking the round-trip.
func buildISO(dStr, moStr, yStr string) (string, time.Time, bool) {
	d, e1 := strconv.Atoi(strings.TrimSpace(dStr))
	mo, e2 := strconv.Atoi(strings.TrimSpace(moStr))
	y, e3 := strconv.Atoi(strings.TrimSpace(yStr))
	if e1 != nil || e2 != nil || e3 != nil {
		return "", time.Time{}, false
	}
	if y < 100 {
		y += 2000
	}
	if mo < 1 || mo > 12 || d < 1 || d > 31 {
		return "", time.Time{}, false
	}
	t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
	if t.Day() != d || int(t.Month()) != mo {
		return "", time.Time{}, false
	}
	return t.Format("2006-01-02"), t, true
}
