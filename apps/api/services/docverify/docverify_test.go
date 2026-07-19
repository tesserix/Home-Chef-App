package docverify

import "testing"

func TestVerhoeff(t *testing.T) {
	// 999999990019 is a documented valid Verhoeff/Aadhaar-format number.
	if !Verhoeff("999999990019") {
		t.Error("expected 999999990019 to be a valid Verhoeff number")
	}
	// Corrupting the check digit must fail.
	if Verhoeff("999999990018") {
		t.Error("expected corrupted number to fail Verhoeff")
	}
	// Transposition should be caught by Verhoeff.
	if Verhoeff("123456789012") {
		t.Error("expected random 12 digits to fail Verhoeff")
	}
	if Verhoeff("99999999001") || Verhoeff("abcd99990019") {
		t.Error("wrong length / non-digit must fail")
	}
}

func TestAssess_PAN(t *testing.T) {
	genuine := "INCOME TAX DEPARTMENT GOVT. OF INDIA\nPermanent Account Number\nABCDE1234F\nRAHUL SHARMA"
	if v := Assess("pan_card", genuine); v.Status != StatusGenuine {
		t.Errorf("genuine PAN: got %s (%s)", v.Status, v.Reason)
	}
	// Right anchors but no PAN-format number → rejected.
	noNum := "INCOME TAX DEPARTMENT Permanent Account Number RAHUL"
	if v := Assess("pan_card", noNum); v.Status != StatusRejected {
		t.Errorf("PAN without number should be rejected, got %s", v.Status)
	}
	// A random selfie's OCR (garbage) → rejected, no anchor.
	if v := Assess("pan_card", "happy birthday to you"); v.Status != StatusRejected {
		t.Errorf("random image should be rejected, got %s", v.Status)
	}
}

func TestAssess_WrongTypeDetected(t *testing.T) {
	// User claims PAN but uploads an Aadhaar.
	aadhaar := "Government of India\nAadhaar\n9999 9999 0019\nUnique Identification Authority"
	v := Assess("pan_card", aadhaar)
	if v.Status != StatusRejected {
		t.Fatalf("expected rejected, got %s", v.Status)
	}
	if v.DetectedType != "aadhaar_card" {
		t.Errorf("expected detectedType aadhaar_card, got %q", v.DetectedType)
	}
}

func TestAssess_Aadhaar(t *testing.T) {
	genuine := "Government of India\nAadhaar\n9999 9999 0019\nUIDAI"
	if v := Assess("aadhaar_card", genuine); v.Status != StatusGenuine {
		t.Errorf("genuine Aadhaar: got %s (%s)", v.Status, v.Reason)
	}
	// Anchors present but the 12-digit number fails Verhoeff → rejected.
	badChecksum := "Government of India Aadhaar 1234 5678 9012 UIDAI"
	if v := Assess("aadhaar_card", badChecksum); v.Status != StatusRejected {
		t.Errorf("bad Aadhaar checksum should be rejected, got %s", v.Status)
	}
}

func TestAssess_FSSAI(t *testing.T) {
	genuine := "Food Safety and Standards Authority of India\nLicense No. 12345678901234\nValid Upto 01/01/2027"
	if v := Assess("fssai_license", genuine); v.Status != StatusGenuine {
		t.Errorf("genuine FSSAI: got %s (%s)", v.Status, v.Reason)
	}
	if v := Assess("fssai_license", "FSSAI license number missing here"); v.Status != StatusRejected {
		t.Errorf("FSSAI without 14-digit number should be rejected, got %s", v.Status)
	}
}

func TestAssess_Passport(t *testing.T) {
	genuine := "REPUBLIC OF INDIA\nPASSPORT\nA1234567\nP<INDSHARMA<<RAHUL<<<<<<<<<<<<<<<<<<<<<<<<<<"
	if v := Assess("passport", genuine); v.Status != StatusGenuine {
		t.Errorf("genuine passport: got %s (%s)", v.Status, v.Reason)
	}
}

func TestAssess_IDProof(t *testing.T) {
	// A genuine PAN uploaded as the generic "id_proof" passes, detected as PAN.
	pan := "INCOME TAX DEPARTMENT Permanent Account Number ABCDE1234F"
	v := Assess("id_proof", pan)
	if v.Status != StatusGenuine || v.DetectedType != "pan_card" {
		t.Errorf("id_proof PAN: got %s detected=%q (%s)", v.Status, v.DetectedType, v.Reason)
	}
	// A genuine Aadhaar as id_proof also passes.
	aadhaar := "Government of India Aadhaar 9999 9999 0019 UIDAI"
	if v := Assess("id_proof", aadhaar); v.Status != StatusGenuine || v.DetectedType != "aadhaar_card" {
		t.Errorf("id_proof Aadhaar: got %s detected=%q", v.Status, v.DetectedType)
	}
	// A random image as id_proof is rejected.
	if v := Assess("id_proof", "cute cat photo 2026"); v.Status != StatusRejected {
		t.Errorf("id_proof random image should be rejected, got %s", v.Status)
	}
	if !IsVerifiable("id_proof") {
		t.Error("id_proof must be verifiable")
	}
}

func TestICAOCheckDigit(t *testing.T) {
	// Standard ICAO 9303 example: passport number L898902C3 → check digit 6.
	if got := icaoCheckDigit("L898902C3"); got != 6 {
		t.Errorf("icaoCheckDigit(L898902C3) = %d, want 6", got)
	}
	// Date 740812 → 2 (documented ICAO example).
	if got := icaoCheckDigit("740812"); got != 2 {
		t.Errorf("icaoCheckDigit(740812) = %d, want 2", got)
	}
	if got := icaoCheckDigit("bad!"); got != -1 {
		t.Errorf("invalid char should return -1, got %d", got)
	}
}

func TestMRZPassportCheck(t *testing.T) {
	// Genuine ICAO line 2 — passport L898902C3 + check digit 6.
	line2 := "L898902C36UTO7408122F1204159ZE184226B<<<<<10"
	if !mrzPassportCheckOK([]string{line2}) {
		t.Error("expected valid MRZ line to pass the check digit")
	}
	// Corrupt the check digit.
	bad := "L898902C30UTO7408122F1204159ZE184226B<<<<<10"
	if mrzPassportCheckOK([]string{bad}) {
		t.Error("corrupted MRZ check digit must fail")
	}
}

func TestAssess_NonVerifiableType(t *testing.T) {
	if v := Assess("cancelled_cheque", "any text"); v.Status != StatusUnknown {
		t.Errorf("non-verifiable type should be unknown, got %s", v.Status)
	}
}

func TestAssess_NoText(t *testing.T) {
	if v := Assess("pan_card", "   "); v.Status != StatusUnknown {
		t.Errorf("empty OCR should be unknown, got %s", v.Status)
	}
}
