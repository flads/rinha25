package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"src/datetime"
	"strconv"
	"strings"
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
	var data map[string]interface{}
	if err := json.NewDecoder(bytes.NewReader(ctx.PostBody())).Decode(&data); err != nil {
		ctx.Error("Invalid JSON", fasthttp.StatusBadRequest)
		return
	}

	data["requestedAt"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	ctx.Response.SetStatusCode(fasthttp.StatusNoContent)

	go func() {
		encoded, err := json.Marshal(data)
		if err != nil {
			log.Println("Erro ao codificar JSON:", err)
			return
		}

		err = a.client.RPush(context.Background(), "requests", encoded).Err()
		if err != nil {
			log.Println("Erro ao gravar no Redis:", err)
		}
	}()
}

func (a *App) paymentsSummary(ctx *fasthttp.RequestCtx) {
	postArgs := ctx.QueryArgs()
	from := string(postArgs.Peek("from"))
	to := string(postArgs.Peek("to"))

	defaultList := a.getRequests("default_requests", from, to)
	fallbackList := a.getRequests("fallback_requests", from, to)

	resp := map[string]interface{}{
		"default": map[string]interface{}{
			"totalRequests": len(defaultList),
			"totalAmount":   a.getRequestsAmountSum(defaultList),
		},
		"fallback": map[string]interface{}{
			"totalRequests": len(fallbackList),
			"totalAmount":   a.getRequestsAmountSum(fallbackList),
		},
	}

	ctx.Response.Header.Set("Content-Type", "application/json")
	json.NewEncoder(ctx.Response.BodyWriter()).Encode(resp)
}

func (a *App) getRequests(listName string, from string, to string) []string {
	ctx := context.Background()

	if from != "" && to != "" {
		fromInt, err1 := datetime.StrToTimeWithMicro(from)
		toInt, err2 := datetime.StrToTimeWithMicro(to)

		if err1 == nil && err2 == nil {
			vals, _ := a.client.ZRangeByScore(ctx, listName, &redis.ZRangeBy{
				Min: fmt.Sprintf("%d", fromInt),
				Max: fmt.Sprintf("%d", toInt),
			}).Result()

			return vals
		}
	}

	vals, _ := a.client.ZRange(ctx, listName, 0, -1).Result()
	return vals
}

func (a *App) getRequestsAmountSum(requests []string) float64 {
	sum := 0.0

	for _, item := range requests {
		parts := strings.Split(item, "|")
		if len(parts) > 0 {
			if amt, err := strconv.ParseFloat(parts[0], 64); err == nil {
				sum += (amt / 100.0)
			}
		}
	}

	return float64(int(sum*100)) / 100.0
}
