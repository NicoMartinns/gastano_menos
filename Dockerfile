FROM golang:1.26.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o gastando-menos ./cmd/api

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/gastando-menos .

EXPOSE 8080
CMD ["./gastando-menos"]