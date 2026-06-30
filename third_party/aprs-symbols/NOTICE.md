# APRS Symbols — Third-Party Asset Notice

CQOps uses APRS symbol graphics from the **aprs.fi APRS symbol set** by Heikki Hannikainen, OH7LZB.

## Source

- **Upstream repository:** https://github.com/hessu/aprs-symbols
- **Author:** Heikki Hannikainen, OH7LZB (aprs.fi)
- **Released:** 2015-12-03 to the APRS community

## Embedded Files

The following raster sprite sheets are embedded in the CQOps binary and served by the CQOps Live dashboard:

| File | Description |
|---|---|
| `aprs-symbols-24-0.png` | Primary symbol table (table `/`) |
| `aprs-symbols-24-1.png` | Secondary symbol table (table `\`) |
| `aprs-symbols-24-2.png` | Overlay characters |

These are the 24×24 px raster renderings from the upstream project.

## Licensing Status

The symbol set is provided **for APRS applications for free** according to the upstream README. The upstream project asks users to **provide a pointer to the source** (`http://github.com/hessu/aprs-symbols/`).

The graphics have **mixed, per-symbol copyright status** documented in the upstream [`COPYRIGHT.md`](COPYRIGHT.md):

- Many symbols are **VEC-OH7LZB**: vectorized by OH7LZB based on the original APRS symbol set by Roger Barker G4IDE, Steve Dimse KH4G, or Stephen Smith WA8LMF. The original symbols have **unknown licensing**.
- Some symbols are **OH7LZB**: original vector designs by Heikki Hannikainen, licensed under **CC BY-SA 2.0**.
- Some symbols are from **public domain** sources (OpenClipArt, Wikipedia, etc.).
- Some symbols represent **brand/product logos** (Apple, Microsoft, Kenwood, etc.) whose copyrights are owned by the respective companies.

**These graphics are third-party assets and are NOT covered by the CQOps project license (Apache 2.0).** The upstream copyright information in `COPYRIGHT.md` must be preserved.

CQOps does not claim that the entire symbol set is MIT, Apache, or public domain.
