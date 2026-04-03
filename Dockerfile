# Stage 1: Build Go binary
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /gateway .

# Stage 2: Build webui
FROM node:22-alpine AS webui

WORKDIR /webui
COPY webui/package.json webui/package-lock.json* ./
RUN npm ci || npm install
COPY webui/ ./
RUN npm run build

# Stage 3: Final image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /gateway /gateway
COPY --from=webui /webui/dist /webui/dist

ENV TZ=Asia/Shanghai
ENV UPSTREAM_URL=http://192.168.1.237:8317
ENV PORT=9900
ENV ADMIN_PORT=9911
ENV PROXY_PORT=9901
ENV ADMIN_PASSWORD=admin123
ENV DATABASE_URL=/data/gateway.db

EXPOSE 9900 9911 9901

RUN mkdir /data

CMD ["/gateway"]