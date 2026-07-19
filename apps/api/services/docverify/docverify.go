// Package docverify performs upload-time document genuineness checks: given the
// OCR text of an uploaded document and the claimed type, it decides whether the
// image looks like a genuine, self-consistent document of that type (right
// anchor text + valid number format/checksum) — not a random photo, screenshot,
// or self-inconsistent fake. Registry existence of the number is verified
// separately (manually / see FSSAI+identity verification issue).
package docverify

import (
	"regexp"
	"strings"
)

type Status string

const (
	StatusGenuine  Status = "looks_genuine"
	StatusRejected Status = "rejected"
	StatusUnknown  Status = "unknown" // could not assess (no readable text)
)

type Verdict struct {
	Status       Status          `json:"status"`
	ClaimedType  string          `json:"claimedType"`
	DetectedType string          `json:"detectedType,omitempty"`
	Reason       string          `json:"reason,omitempty"`
	Signals      map[string]bool `json:"signals"`
}

// anchors holds the mandatory identifying text per document type (lowercased).
// Distinctive per-type anchor text only — deliberately excludes generic phrases
// like "government of india" that appear on multiple documents and would cause
// false anchor hits (e.g. an Aadhaar claimed as a PAN).
var anchors = map[string][]string{
	"pan_card":         {"income tax department", "permanent account number"},
	"aadhaar_card":     {"aadhaar", "आधार", "unique identification", "uidai"},
	"passport":         {"republic of india", "passport"},
	"fssai_license":    {"food safety", "fssai", "food safety and standards"},
	"food_safety_cert": {"food safety", "fssai"},
}

var prettyNames = map[string]string{
	"pan_card":         "PAN card",
	"aadhaar_card":     "Aadhaar card",
	"passport":         "passport",
	"fssai_license":    "FSSAI licence",
	"food_safety_cert": "food safety certificate",
}

var (
	panRe        = regexp.MustCompile(`[A-Z]{5}[0-9]{4}[A-Z]`)
	aadhaarRe    = regexp.MustCompile(`\b(\d{4}\s?\d{4}\s?\d{4})\b`)
	fssaiRe      = regexp.MustCompile(`\b\d{14}\b`)
	passportNoRe = regexp.MustCompile(`\b[A-Z][0-9]{7}\b`)
	mrzLineRe    = regexp.MustCompile(`(?m)^[A-Z0-9<]{40,44}$`)
)

// idProofType is a generic "any recognised ID" claim used by onboarding flows
// that collect a single ID proof rather than a specific PAN/Aadhaar/passport.
const idProofType = "id_proof"

// IsVerifiable reports whether a document type has genuineness signals we check.
func IsVerifiable(docType string) bool {
	if docType == idProofType {
		return true
	}
	_, ok := anchors[docType]
	return ok
}

// Verhoeff validates a 12-digit Aadhaar number's checksum (final digit included).
func Verhoeff(num string) bool {
	if len(num) != 12 {
		return false
	}
	c := 0
	for i := 0; i < len(num); i++ {
		ch := num[len(num)-1-i]
		if ch < '0' || ch > '9' {
			return false
		}
		c = verhoeffD[c][verhoeffP[i%8][ch-'0']]
	}
	return c == 0
}

// ClassifyType returns the document types whose anchor text appears in ocrText.
func ClassifyType(ocrText string) []string {
	text := strings.ToLower(ocrText)
	var out []string
	for docType, keys := range anchors {
		for _, k := range keys {
			if strings.Contains(text, k) {
				out = append(out, docType)
				break
			}
		}
	}
	return out
}

// Assess decides whether ocrText looks like a genuine document of claimedType.
// A doc passes only when the claimed type's anchor text is present AND the
// identifying number's format/checksum validates. Non-verifiable types return
// StatusUnknown (nothing to check).
func Assess(claimedType, ocrText string) Verdict {
	if claimedType == idProofType {
		return assessAnyID(ocrText)
	}
	v := Verdict{ClaimedType: claimedType, Signals: map[string]bool{}}
	if !IsVerifiable(claimedType) {
		v.Status = StatusUnknown
		v.Reason = "no genuineness signals for this document type"
		return v
	}
	if strings.TrimSpace(ocrText) == "" {
		v.Status = StatusUnknown
		v.Reason = "no readable text in the image"
		return v
	}

	lower := strings.ToLower(ocrText)
	anchorHit := false
	for _, a := range anchors[claimedType] {
		if strings.Contains(lower, a) {
			anchorHit = true
			break
		}
	}
	v.Signals["anchorText"] = anchorHit

	numberOK := assessNumber(claimedType, ocrText, v.Signals)
	v.Signals["numberFormat"] = numberOK

	switch {
	case anchorHit && numberOK:
		v.Status = StatusGenuine
	case !anchorHit:
		v.Status = StatusRejected
		if others := otherTypes(ocrText, claimedType); len(others) > 0 {
			v.DetectedType = others[0]
			v.Reason = "this looks like a " + pretty(others[0]) + ", not a " + pretty(claimedType)
		} else {
			v.Reason = "this does not look like a " + pretty(claimedType) + " (expected identifying text not found)"
		}
	default:
		v.Status = StatusRejected
		v.Reason = "the " + pretty(claimedType) + " number could not be validated on the image"
	}
	return v
}

// assessAnyID passes when the image looks like any recognised Indian ID
// (PAN, Aadhaar, or passport) — used for a generic "id_proof" upload.
func assessAnyID(ocrText string) Verdict {
	v := Verdict{ClaimedType: idProofType, Signals: map[string]bool{}}
	if strings.TrimSpace(ocrText) == "" {
		v.Status = StatusUnknown
		v.Reason = "no readable text in the image"
		return v
	}
	for _, t := range []string{"pan_card", "aadhaar_card", "passport"} {
		if sub := Assess(t, ocrText); sub.Status == StatusGenuine {
			v.Status = StatusGenuine
			v.DetectedType = t
			v.Signals = sub.Signals
			return v
		}
	}
	v.Status = StatusRejected
	v.Reason = "this does not look like a recognised ID document (PAN, Aadhaar, or passport)"
	return v
}

func assessNumber(claimedType, ocrText string, signals map[string]bool) bool {
	switch claimedType {
	case "pan_card":
		return panRe.MatchString(strings.ToUpper(ocrText))
	case "aadhaar_card":
		if m := aadhaarRe.FindString(ocrText); m != "" {
			ok := Verhoeff(strings.ReplaceAll(m, " ", ""))
			signals["aadhaarChecksum"] = ok
			return ok
		}
		signals["aadhaarChecksum"] = false
		return false
	case "fssai_license", "food_safety_cert":
		return fssaiRe.MatchString(ocrText)
	case "passport":
		up := strings.ToUpper(ocrText)
		hasNo := passportNoRe.MatchString(up)
		mrzLines := mrzLineRe.FindAllString(up, -1)
		hasMRZ := len(mrzLines) > 0
		signals["mrz"] = hasMRZ
		signals["mrzCheckDigit"] = mrzPassportCheckOK(mrzLines)
		return hasNo || hasMRZ
	default:
		return false
	}
}

// icaoCheckDigit computes the ICAO 9303 MRZ check digit for a field (weights
// 7,3,1; digit = value, A-Z = 10..35, filler '<' = 0; sum mod 10). Returns -1
// on an invalid character.
func icaoCheckDigit(s string) int {
	weights := [3]int{7, 3, 1}
	sum := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		var v int
		switch {
		case ch >= '0' && ch <= '9':
			v = int(ch - '0')
		case ch >= 'A' && ch <= 'Z':
			v = int(ch-'A') + 10
		case ch == '<':
			v = 0
		default:
			return -1
		}
		sum += v * weights[i%3]
	}
	return sum % 10
}

// mrzPassportCheckOK returns true if any MRZ line's first 9 characters (the
// passport number) match the check digit in position 10 — a strong signal the
// MRZ is a genuine, uncorrupted machine-readable zone.
func mrzPassportCheckOK(lines []string) bool {
	for _, ln := range lines {
		if len(ln) < 10 {
			continue
		}
		field := ln[0:9]
		check := ln[9]
		if check < '0' || check > '9' {
			continue
		}
		if cd := icaoCheckDigit(field); cd >= 0 && cd == int(check-'0') {
			return true
		}
	}
	return false
}

func otherTypes(ocrText, exclude string) []string {
	var out []string
	for _, t := range ClassifyType(ocrText) {
		if t != exclude {
			out = append(out, t)
		}
	}
	return out
}

func pretty(t string) string {
	if p, ok := prettyNames[t]; ok {
		return p
	}
	return t
}

var verhoeffD = [10][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 2, 3, 4, 0, 6, 7, 8, 9, 5},
	{2, 3, 4, 0, 1, 7, 8, 9, 5, 6},
	{3, 4, 0, 1, 2, 8, 9, 5, 6, 7},
	{4, 0, 1, 2, 3, 9, 5, 6, 7, 8},
	{5, 9, 8, 7, 6, 0, 4, 3, 2, 1},
	{6, 5, 9, 8, 7, 1, 0, 4, 3, 2},
	{7, 6, 5, 9, 8, 2, 1, 0, 4, 3},
	{8, 7, 6, 5, 9, 3, 2, 1, 0, 4},
	{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
}

var verhoeffP = [8][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 5, 7, 6, 2, 8, 3, 0, 9, 4},
	{5, 8, 0, 3, 7, 9, 6, 1, 4, 2},
	{8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
	{9, 4, 5, 3, 1, 2, 6, 8, 7, 0},
	{4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
	{2, 7, 9, 3, 8, 0, 6, 4, 1, 5},
	{7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
}
