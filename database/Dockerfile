FROM golang:1.24.5 AS builder

WORKDIR /src

COPY ./src/ /src/
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /src/app *.go

FROM alpine:latest

RUN mkdir -p /home/app

COPY --from=builder /src/app /home/app

WORKDIR /home/app

EXPOSE 8081

CMD ["./app"]