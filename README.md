# Imagecut

Imagecut is a small Go web app for cutting grid images into separate files.

Current MVP:

- Upload PNG or JPEG grid images.
- Generate a local sample grid for API-free testing.
- Configure rows, columns, margin, and gutter.
- Preview the grid overlay before cutting.
- Review per-cut thumbnails before downloading the ZIP.
- Download an individual cut directly from its thumbnail.
- Select a cell, drag it, resize it with the corner handle, or nudge its crop area before export.
- Edit selected crop coordinates directly with x/y/w/h pixel inputs.
- Undo and redo crop edits from the UI or keyboard shortcuts.
- Save and load grid/crop/output settings as a JSON project file.
- Name a project and reuse that name for JSON, ZIP, and cut downloads.
- See source size, output count, and per-cell size warnings.
- Export cuts as original format, PNG, or JPEG with quality control.
- Export PNG and JPEG batches in one ZIP when both formats are needed.
- Download all cuts as a ZIP.
- No OpenAI API key is required.
- Optional OpenAI generation is guarded by explicit environment variables and confirmation text.
- The UI shows request count, output size, and current GPT-Image-2 pricing rates from `/pricing` before generation.

## Run

```bash
go run ./cmd/server
```

Open http://localhost:8080.

Or use Make:

```bash
make run
```

Or run with Docker:

```bash
docker compose up --build
```

## Test

```bash
go test ./...
```

Full local check:

```bash
make check
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

The UI displays the GPT-Image-2 pricing basis from OpenAI's public pricing page through the local `/pricing` JSON endpoint. As of 2026-06-14, GPT-Image-2 image input is $8.00 per 1M tokens, cached image input is $2.00 per 1M tokens, and image output is $30.00 per 1M tokens. Text input is $5.00 per 1M tokens and cached text input is $1.25 per 1M tokens. The exact final bill depends on OpenAI's token accounting for the generated image, so the app treats the display as a guardrail, not an invoice.
