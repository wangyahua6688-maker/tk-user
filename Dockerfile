FROM golang:1.24 AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o tk-user .

FROM debian:stable-slim

WORKDIR /app
COPY --from=builder /app/tk-user .
COPY etc/user.yaml ./etc/user.yaml

EXPOSE 9103

CMD ["./tk-user","-f","etc/user.yaml"]