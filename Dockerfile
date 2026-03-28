FROM golang:1.24-alpine AS builder

ENV GOPROXY=https://proxy.golang.org,direct
ENV GOOS=linux
ENV CGO_ENABLED=0

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o tk-user .
RUN chmod +x tk-user

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/tk-user .
COPY etc/user.yaml ./etc/user.yaml
EXPOSE 9103
CMD ["./tk-user", "-f", "etc/user.yaml"]