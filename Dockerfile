FROM golang:1-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o linebackerr .

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ffmpeg && ffmpeg --version
COPY --from=builder /app/linebackerr .
EXPOSE 6666
CMD ["./linebackerr"]
