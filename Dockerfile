# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache wget
RUN adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=build /out/server /app/server
# Non-secret per-env config (config_{env}.yaml). Secrets ride in compiled
# config/secret_{env}.go files or MONGO_URI / JWT_SECRET env vars.
COPY config/*.yaml /app/config/
EXPOSE 8080 50051
ENTRYPOINT ["/app/server"]