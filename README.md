# Imagecut

Imagecut is a small Go web app for cutting grid images into separate files.

Current MVP:

- Upload PNG or JPEG grid images.
- Configure rows, columns, margin, and gutter.
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

- Browser preview before download.
- Manual crop adjustment per cell.
- OpenAI image generation adapter behind explicit cost controls.
