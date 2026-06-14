FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/imagecut ./cmd/server

FROM alpine:3.22

RUN addgroup -S imagecut && adduser -S imagecut -G imagecut
WORKDIR /app

COPY --from=build /out/imagecut /app/imagecut
COPY web /app/web

USER imagecut
EXPOSE 8080

ENV ADDR=:8080
ENTRYPOINT ["/app/imagecut"]
