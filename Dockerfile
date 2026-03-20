FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1-alpine AS backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN apk add --no-cache gcc musl-dev
RUN CGO_ENABLED=1 go build -o linebackerr .

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ffmpeg && ffmpeg -version
COPY --from=backend /app/linebackerr ./linebackerr
COPY --from=frontend /app/frontend/dist ./dist
EXPOSE 6666
CMD ["./linebackerr"]
