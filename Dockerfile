FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /gateway .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /gateway /gateway

ENV TZ=Asia/Shanghai
ENV UPSTREAM_URL=http://192.168.1.237:8317
ENV PORT=8080
ENV DATABASE_URL=/data/gateway.db
ENV JWT_SECRET=change-this-secret
ENV ADMIN_PASSWORD=admin123

EXPOSE 8080

RUN mkdir /data

CMD ["/gateway"]
