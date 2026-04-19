# syntax=docker/dockerfile:1.7

# ---------- Stage 1: frontend build ----------
FROM node:20-alpine AS web
WORKDIR /web
RUN corepack enable
COPY web/package.json web/package-lock.json* web/pnpm-lock.yaml* ./
RUN (pnpm install --frozen-lockfile 2>/dev/null || npm install)
COPY web/ ./
RUN (pnpm build 2>/dev/null || npm run build)

# ---------- Stage 2: backend build ----------
FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy built frontend into embed path.
RUN rm -rf internal/frontend/dist && mkdir -p internal/frontend/dist
COPY --from=web /web/build internal/frontend/dist
RUN CGO_ENABLED=1 go build -tags="sqlite_fts5" -ldflags="-s -w" -o /out/frames ./cmd/frames

# ---------- Stage 3: runtime ----------
FROM alpine:3.20
RUN apk add --no-cache \
    ca-certificates tzdata \
    sqlite-libs \
    vips-tools \
    libraw-tools \
    ffmpeg \
    exiftool
WORKDIR /app
COPY --from=build /out/frames /app/frames
EXPOSE 8080
ENTRYPOINT ["/app/frames"]
