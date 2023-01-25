FROM golang:1.19-alpine as builder
WORKDIR /build

COPY go.* .
RUN go mod download

COPY *.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine
WORKDIR /app

COPY --from=builder /build/main .

ENTRYPOINT ["./main"]