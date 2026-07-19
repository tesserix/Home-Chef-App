package docverify

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// gradient builds a test image whose brightness ramps left→right (flip reverses
// it) so two variants produce clearly different difference-hashes.
func gradient(w, h int, flip bool) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(255 * x / w)
			if flip {
				v = 255 - v
			}
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func TestDHash_IdenticalImages(t *testing.T) {
	img := gradient(200, 160, false)
	h1, err := DHash(img)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := DHash(img)
	if Hamming(h1, h2) != 0 {
		t.Errorf("identical images must hash identically, got distance %d", Hamming(h1, h2))
	}
}

func TestDHash_RecompressedStaysClose(t *testing.T) {
	src := gradient(300, 240, false)
	h1, _ := DHash(src)

	// Decode, re-encode as JPEG at lower quality + different size = "re-saved".
	img, _, _ := image.Decode(bytes.NewReader(src))
	var jb bytes.Buffer
	_ = jpeg.Encode(&jb, img, &jpeg.Options{Quality: 60})
	h2, err := DHash(jb.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if d := Hamming(h1, h2); d > ReuseThreshold {
		t.Errorf("recompressed image should stay within reuse threshold, got %d", d)
	}
}

func TestDHash_DifferentImages(t *testing.T) {
	h1, _ := DHash(gradient(200, 160, false))
	h2, _ := DHash(gradient(200, 160, true))
	if d := Hamming(h1, h2); d <= ReuseThreshold {
		t.Errorf("clearly different images should exceed reuse threshold, got %d", d)
	}
}

func TestDHash_BadImage(t *testing.T) {
	if _, err := DHash([]byte("not an image")); err == nil {
		t.Error("expected error decoding non-image bytes")
	}
}

func TestHashHexRoundTrip(t *testing.T) {
	h := uint64(0xdeadbeefcafe0011)
	s := HashHex(h)
	if len(s) != 16 {
		t.Fatalf("want 16 hex chars, got %q", s)
	}
	back, ok := ParseHashHex(s)
	if !ok || back != h {
		t.Errorf("round-trip failed: %x -> %s -> %x (%v)", h, s, back, ok)
	}
	if _, ok := ParseHashHex("xyz"); ok {
		t.Error("bad hex should fail to parse")
	}
}
