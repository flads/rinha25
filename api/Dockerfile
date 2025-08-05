FROM golang:1.24.5 AS builder

WORKDIR /src

COPY ./src/ /src/
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /src/app *.go

FROM alpine:latest

COPY --from=builder /src/app /home/app

WORKDIR /home/app

RUN mkdir -p /tmp/sockets 
RUN chmod -R 770 /tmp/sockets
RUN chown -R 1000:1001 /tmp/sockets

EXPOSE 8080

CMD ["./main"]