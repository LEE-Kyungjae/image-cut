# Imagecut

Imagecut is a small Go web app for cutting grid images into separate files.

Current MVP:

- Upload PNG or JPEG grid images.
- Generate a local sample grid for API-free testing.
- Configure rows, columns, margin, and gutter.
- Preview the grid overlay before cutting.
- Select a cell and nudge its crop area before export.
- See source size, output count, and per-cell size warnings.
- Download all cuts as a ZIP.
- No OpenAI API key is required.

## Run

```bash
go run ./cmd/server
```

Open http://localhost:8080.

## Test

```bash
go test ./...
```

## Planned

- Manual crop adjustment per cell.
- OpenAI image generation adapter behind explicit cost controls.
