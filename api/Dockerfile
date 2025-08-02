FROM golang:1.24.5 AS builder

WORKDIR /src

COPY ./src/ /src/
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /src/app *.go

FROM alpine:latest

RUN adduser -D api
USER api

COPY --from=builder /src/app /home/api/app

WORKDIR /home/api/app

EXPOSE 8080

CMD ["./main"]