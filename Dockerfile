FROM golang:1-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN apk add --no-cache gcc musl-dev
RUN CGO_ENABLED=1 go build -o linebackerr .
RUN go test ./... || true

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ffmpeg && ffmpeg -version
COPY --from=builder /app/linebackerr .
EXPOSE 6666
CMD ["./linebackerr"]
