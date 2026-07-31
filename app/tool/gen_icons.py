#!/usr/bin/env python3
"""Render the legacy (pre-adaptive) launcher PNGs from the ZlefRemote mark.

Android 26+ uses the adaptive icon in res/mipmap-anydpi-v26; API 24-25 needs
real bitmaps, and those are what this writes. The geometry is the same 32x32
grid as lib/ui/mark.dart and res/drawable/ic_launcher_foreground.xml — change
one, re-run this, keep the three in step.

    python3 tool/gen_icons.py
"""

from pathlib import Path

from PIL import Image, ImageDraw

BG = (6, 6, 10, 255)  # --zl-bg
INK = (233, 234, 226, 255)  # --zl-text
OLIVE = (189, 206, 116, 255)  # --zl-olive-bright

# 32x32 design grid, drawn inside the icon's 80% safe area
SCREEN_OUTER = [(3, 5), (22, 5), (29, 12), (29, 23), (3, 23)]
SCREEN_INNER = [(5.4, 7.4), (21, 7.4), (26.6, 13), (26.6, 20.6), (5.4, 20.6)]
POINTER = [(13, 10), (22, 16), (17.6, 17), (19.6, 20.6), (17, 21.8), (15.1, 18.2), (12, 20.6)]
STAND = [(13, 26), (21, 26), (21, 28), (13, 28)]

DENSITIES = {
    "mdpi": 48,
    "hdpi": 72,
    "xhdpi": 96,
    "xxhdpi": 144,
    "xxxhdpi": 192,
}

SUPERSAMPLE = 4
INSET = 0.80  # keep the mark clear of round/squircle masks


def render(size: int) -> Image.Image:
    big = size * SUPERSAMPLE
    image = Image.new("RGBA", (big, big), BG)
    draw = ImageDraw.Draw(image)

    scale = big * INSET / 32
    offset = (big - 32 * scale) / 2

    def place(points):
        return [(offset + x * scale, offset + y * scale) for x, y in points]

    draw.polygon(place(SCREEN_OUTER), fill=INK)
    draw.polygon(place(SCREEN_INNER), fill=BG)
    draw.polygon(place(POINTER), fill=OLIVE)
    draw.polygon(place(STAND), fill=INK)

    return image.resize((size, size), Image.LANCZOS)


def main() -> None:
    res = Path(__file__).resolve().parent.parent / "android/app/src/main/res"
    for density, size in DENSITIES.items():
        icon = render(size)
        directory = res / f"mipmap-{density}"
        directory.mkdir(parents=True, exist_ok=True)
        icon.save(directory / "ic_launcher.png")
        icon.save(directory / "ic_launcher_round.png")
        print(f"mipmap-{density}: {size}x{size}")


if __name__ == "__main__":
    main()
