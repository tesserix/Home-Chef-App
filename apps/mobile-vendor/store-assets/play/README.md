# Fe3dr Vendor — Google Play listing assets

| File | Spec | Status |
|---|---|---|
| `icon-512.png` | 512 × 512, opaque PNG, 30 KB | Derived from `assets/icon.png` (1024²) by a clean 2× downscale |
| `feature-graphic-1024x500.png` | 1024 × 500 PNG | **Provisional — see below** |

Play requires both before you can publish. Neither can be a screenshot.

## Both are engineering output, not design work

`docs/store-release/README.md` §8 already flags the vendor app icon: it was
regenerated in-repo so the two apps did not ship byte-identical artwork (an
Apple 4.3 flag), and it wants a designer's sign-off. `icon-512.png` is derived
from that same mark, so it carries the same caveat.

The **feature graphic is the more exposed of the two** — it is the banner across
the top of the Play listing, larger and more prominent than the icon. It was
generated here so Android is not blocked on it, following the tokens in
`.impeccable.md` (paper ground, ink headline, single persimmon accent, hairline
rule not a bordered card, asymmetric rather than centred). It is deliberately
restrained and should read as intentional, but it is not designed work. Replace
it before you care about the listing.

Fonts are a known compromise: the brand calls for Geist Sans display, which is
not installed locally, so it renders in the system UI face. A designed
replacement should use the real typeface.

## Regenerating

Both are produced by short PIL scripts (see the session that added them). The
feature graphic samples its background straight out of `assets/icon.png`
(`(253, 250, 246)`) so the mark sits on its own ground with no visible seam —
if you swap the icon, resample rather than hardcoding the paper colour.

## Screenshots

Play phone screenshots live in `../screenshots/android-phone/`. Note Play
rejects any screenshot whose long edge is more than twice its short edge, so
they are captured at 1080 × 1920 (ratio 1.78) rather than the Pixel 8 Pro's
native 1344 × 2992 (ratio 2.23), which would fail on upload.
