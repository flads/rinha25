package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/valyala/fasthttp"
)

const (
	defaultURL     = "http://payment-processor-default:8080/payments"
	fallbackURL    = "http://payment-processor-fallback:8080/payments"
	databaseURLDef = "http://database:8081/processor-default"
	databaseURLFb  = "http://database:8081/processor-fallback"
)

type App struct {
	redis                      *redis.Client
	http                       *fasthttp.Client
	hasFailedRequestsToProcess bool
}

func NewApp() *App {
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	return &App{
		redis: rdb,
		http:  &fasthttp.Client{},
	}
}

func (a *App) execute() {
	time.Sleep(250 * time.Millisecond)

	for {
		time.Sleep(10 * time.Millisecond)

		items, err := a.redis.LPopCount(context.Background(), "requests", 250).Result()
		if err != nil && err != redis.Nil {
			log.Println("Erro ao fazer LPOP COUNT:", err)
			continue
		}

		for _, raw := range items {
			data, err := parseRequest(raw)
			if err != nil {
				log.Println("Erro ao processar request:", err)
				continue
			}

			a.callDefaultProcessor(raw, data)
		}

		if len(items) == 0 && a.hasFailedRequestsToProcess {
			a.processFailedRequests()
		}
	}
}

// converte string "requestedAt@{json}" em map[string]interface{}
func parseRequest(raw string) (map[string]interface{}, error) {
	parts := strings.SplitN(raw, "@", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidFormat
	}

	requestedAt := parts[0]
	jsonPart := parts[1]

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonPart), &data); err != nil {
		return nil, err
	}

	// injeta o campo requestedAt
	data["requestedAt"] = requestedAt
	return data, nil
}

var ErrInvalidFormat = fmt.Errorf("formato inválido, esperado requestedAt@json")

func (a *App) callDefaultProcessor(raw string, data map[string]interface{}) bool {
	if a.sendRequest(defaultURL, data) {
		a.sendToDatabase(databaseURLDef, data)
		return true
	}

	// Falhou → reenvia para lista de falhas no mesmo formato "requestedAt@json"
	a.redis.RPush(context.Background(), "failed_requests", raw)
	a.redis.Set(context.Background(), "default_failed_10_secs_ago", "true", 10*time.Second)
	a.hasFailedRequestsToProcess = true
	return false
}

func (a *App) callFallbackProcessor(data map[string]interface{}) bool {
	if a.sendRequest(fallbackURL, data) {
		a.sendToDatabase(databaseURLFb, data)
		return true
	}
	return false
}

func (a *App) sendRequest(url string, data map[string]interface{}) bool {
	body, _ := json.Marshal(data)

	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBody(body)

	err := a.http.Do(req, res)
	if err == nil && res.StatusCode() == 200 {
		return true
	}
	return false
}

func (a *App) sendToDatabase(url string, data map[string]interface{}) {
	body, _ := json.Marshal(data)

	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBody(body)

	err := a.http.Do(req, res)
	if err != nil || res.StatusCode() != 200 {
		log.Println("Erro ao enviar para database:", err, "status:", res.StatusCode())
	}
}

func (a *App) processFailedRequests() {
	val, _ := a.redis.Get(context.Background(), "default_failed_10_secs_ago").Result()
	if val != "" {
		return // ainda no cooldown
	}

	items, err := a.redis.LPopCount(context.Background(), "failed_requests", 250).Result()
	if err != nil && err != redis.Nil {
		log.Println("Erro ao processar falhas:", err)
		return
	}

	if len(items) == 0 {
		a.hasFailedRequestsToProcess = false
		return
	}

	for _, raw := range items {
		data, err := parseRequest(raw)
		if err != nil {
			log.Println("Erro ao decodificar failed_request:", err)
			continue
		}

		if a.callDefaultProcessor(raw, data) {
			continue
		}
		a.callFallbackProcessor(data)
	}
}

func main() {
	app := NewApp()
	app.execute()
}
