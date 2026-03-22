FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=builder /app/api .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]
