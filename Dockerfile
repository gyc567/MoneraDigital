# ============================================================
# Stage 1: Build Frontend (Node.js)
# ============================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# ============================================================
# Stage 2: Build Backend (Go)
# ============================================================
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

COPY --from=frontend-builder /app/dist ./dist

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o server ./cmd/server/main.go

# ============================================================
# Stage 3: Runtime
# ============================================================
FROM alpine:3.19 AS runtime

WORKDIR /app

RUN apk --no-cache add ca-certificates curl

RUN addgroup -S app && adduser -S -G app app

COPY --from=backend-builder /app/server .

COPY --from=backend-builder /app/dist ./dist

RUN chown -R app:app /app

USER app

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:5000/health || exit 1

ENV PORT=5000
ENV GIN_MODE=release

CMD ["./server"]
