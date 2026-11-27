#!/usr/bin/env python3
"""Build the Android launcher icons from ZlefRemote's own icon.

The source of truth is public/app/icons/icon-512.png — the icon the product has
always had (a phone with a cursor in it, signalling to the right). This script
only re-sizes it; it does not invent artwork. Adaptive icons get the same glyph
inset into the 108dp safe zone so no launcher mask clips the waves.

    python3 tool/gen_icons.py
"""

from pathlib import Path

from PIL import Image

REPO = Path(__file__).resolve().parent.parent.parent
SOURCE = REPO / "public/app/icons/icon-512.png"
RES = Path(__file__).resolve().parent.parent / "android/app/src/main/res"

# legacy launcher bitmaps (API 24-25 have no adaptive icons)
LEGACY = {"mdpi": 48, "hdpi": 72, "xhdpi": 96, "xxhdpi": 144, "xxxhdpi": 192}

# adaptive foreground: 108dp canvas, glyph kept inside the 66dp safe circle
ADAPTIVE = {"mdpi": 108, "hdpi": 162, "xhdpi": 216, "xxhdpi": 324, "xxxhdpi": 432}
SAFE_FRACTION = 0.62


def main() -> None:
    source = Image.open(SOURCE).convert("RGBA")

    for density, size in LEGACY.items():
        icon = source.resize((size, size), Image.LANCZOS)
        directory = RES / f"mipmap-{density}"
        directory.mkdir(parents=True, exist_ok=True)
        icon.save(directory / "ic_launcher.png")
        icon.save(directory / "ic_launcher_round.png")

    for density, canvas in ADAPTIVE.items():
        glyph_size = int(canvas * SAFE_FRACTION)
        glyph = source.resize((glyph_size, glyph_size), Image.LANCZOS)
        foreground = Image.new("RGBA", (canvas, canvas), (0, 0, 0, 0))
        offset = (canvas - glyph_size) // 2
        foreground.paste(glyph, (offset, offset), glyph)
        directory = RES / f"drawable-{density}"
        directory.mkdir(parents=True, exist_ok=True)
        foreground.save(directory / "ic_launcher_foreground.png")

    print(f"launcher icons rebuilt from {SOURCE.relative_to(REPO)}")


if __name__ == "__main__":
    main()
