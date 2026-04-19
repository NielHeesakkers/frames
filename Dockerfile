# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /out/frames ./cmd/frames

FROM alpine:3.20
RUN apk add --no-cache ca-certificates sqlite-libs tzdata
WORKDIR /app
COPY --from=build /out/frames /app/frames
EXPOSE 8080
ENTRYPOINT ["/app/frames"]
