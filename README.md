# Imagecut

Imagecut is a small Go web app for cutting grid images into separate files.

Current MVP:

- Upload PNG or JPEG grid images.
- Generate a local sample grid for API-free testing.
- Configure rows, columns, margin, and gutter.
- Preview the grid overlay before cutting.
- Review per-cut thumbnails before downloading the ZIP.
- Select a cell, drag it, resize it with the corner handle, or nudge its crop area before export.
- See source size, output count, and per-cell size warnings.
- Export cuts as original format, PNG, or JPEG with quality control.
- Download all cuts as a ZIP.
- No OpenAI API key is required.
- Optional OpenAI generation is guarded by explicit environment variables and confirmation text.
- The UI shows request count, output size, and the current GPT-Image-2 pricing basis before generation.

## Run

```bash
go run ./cmd/server
```

Open http://localhost:8080.

## Test

```bash
go test ./...
```

## OpenAI generation guard

The app defaults to mock generation. Real OpenAI image generation only runs when all of these are true:

- `OPENAI_API_KEY` is set.
- `IMAGECUT_OPENAI_ENABLED=true` is set.
- The request uses `provider=openai`.
- The form confirmation field is exactly `ALLOW_COST`.

Example:

```bash
OPENAI_API_KEY=... IMAGECUT_OPENAI_ENABLED=true go run ./cmd/server
```

The UI displays the GPT-Image-2 pricing basis from OpenAI's public pricing page: text input is priced per 1M tokens, and image output is priced per 1M tokens. The exact final bill depends on OpenAI's token accounting for the generated image, so the app treats the display as a guardrail, not an invoice.

## Planned

- Cost estimation from a maintained pricing table.
- Batch export presets and named projects.
