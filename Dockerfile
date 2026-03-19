FROM golang:1.24-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /release-service ./cmd/server

FROM alpine:3.22
WORKDIR /app
RUN adduser -D -g '' appuser
COPY --from=build /release-service /app/release-service
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/release-service"]
