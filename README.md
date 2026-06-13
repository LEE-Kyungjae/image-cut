# Imagecut

Imagecut is a small Go web app for cutting grid images into separate files.

Current MVP:

- Upload PNG or JPEG grid images.
- Configure rows, columns, margin, and gutter.
- Preview the grid overlay before cutting.
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
- OpenAI image generation mock mode.
- OpenAI image generation adapter behind explicit cost controls.
