package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/valyala/fasthttp"
)

type App struct {
	client *redis.Client
}

func NewApp() *App {
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	return &App{
		client: rdb,
	}
}

func (a *App) ListenAndServe() {
	socketPath := os.Getenv("SOCKET_PATH")
	if socketPath == "" {
		socketPath = "/sockets/api-default.sock"
	}

	if err := os.RemoveAll(socketPath); err != nil {
		log.Fatal("Erro ao remover socket antigo:", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}

	defer listener.Close()

	if err := os.Chmod(socketPath, 0660); err != nil {
		log.Fatal("Erro ao definir permissões do socket:", err)
	}

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		if ctx.IsPost() {
			switch string(ctx.Path()) {
			case "/payments":
				a.payments(ctx)
				return
			}
		}

		if ctx.IsGet() {
			switch string(ctx.Path()) {
			case "/payments-summary":
				a.paymentsSummary(ctx)
				return
			}
		}

		ctx.Error("Not found", fasthttp.StatusNotFound)
	}

	fasthttp.Serve(listener, requestHandler)
}

func (a *App) payments(ctx *fasthttp.RequestCtx) {
	body := ctx.PostBody()
	requestedAt := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	body = append([]byte(requestedAt+"@"), body...)

	ctx.Response.SetStatusCode(fasthttp.StatusNoContent)

	go func() {
		err := a.client.RPush(context.Background(), "requests", body).Err()
		if err != nil {
			log.Println("Erro ao gravar no Redis:", err)
		}
	}()
}

func (a *App) paymentsSummary(ctx *fasthttp.RequestCtx) {
	// Pegar query params
	args := ctx.QueryArgs()
	query := ""

	if from := string(args.Peek("from")); from != "" {
		query += "from=" + from
	}

	if to := string(args.Peek("to")); to != "" {
		if query != "" {
			query += "&"
		}
		query += "to=" + to
	}

	url := "http://database:8081/payments-summary"
	if query != "" {
		url += "?" + query
	}

	// Fazer requisição GET
	status, body, err := fasthttp.Get(nil, url)
	if err != nil {
		ctx.Error(fmt.Sprintf("Erro ao chamar serviço externo: %v", err), fasthttp.StatusInternalServerError)
		return
	}

	if status != fasthttp.StatusOK {
		ctx.Error(fmt.Sprintf("Serviço externo retornou status %d", status), status)
		return
	}

	// Retornar diretamente o corpo da outra API
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	ctx.Response.SetBody(body)
}
