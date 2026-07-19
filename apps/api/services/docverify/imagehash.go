package docverify

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math/bits"
	"strconv"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// DHash computes a 64-bit difference hash of an image — a perceptual fingerprint
// that stays stable across rescaling/recompression, used to detect the same
// document image reused across different accounts.
func DHash(imageBytes []byte) (uint64, error) {
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return 0, fmt.Errorf("decode image: %w", err)
	}
	const w, h = 9, 8
	small := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(small, small.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	var gray [h][w]uint32
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := small.At(x, y).RGBA()
			gray[y][x] = (299*r + 587*g + 114*b) / 1000
		}
	}
	var hash uint64
	bit := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w-1; x++ {
			if gray[y][x] < gray[y][x+1] {
				hash |= 1 << uint(bit)
			}
			bit++
		}
	}
	return hash, nil
}

// Hamming returns the number of differing bits between two hashes (0 = identical).
func Hamming(a, b uint64) int { return bits.OnesCount64(a ^ b) }

// ReuseThreshold is the max Hamming distance at which two document images are
// treated as the same document (identical = 0; minor recompression stays small).
const ReuseThreshold = 6

// HashHex / ParseHashHex convert a hash to/from the 16-char hex stored on the row.
func HashHex(h uint64) string { return fmt.Sprintf("%016x", h) }

func ParseHashHex(s string) (uint64, bool) {
	if len(s) != 16 {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
