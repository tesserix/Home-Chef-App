package services

// The brand mark as it appears on the app icon, embedded so a generated
// document never depends on a file being present in the container image.

import _ "embed"

//go:embed assets/brand-mark.png
var brandMark []byte

// brandMarkPNG returns the Fe3dr app mark for a document masthead.
func brandMarkPNG() []byte { return brandMark }
