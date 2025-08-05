package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"src/datetime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
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
	router := gin.Default()

	router.POST("/payments", func(ctx *gin.Context) {
		a.payments(ctx)
	})

	router.GET("/payments-summary", func(ctx *gin.Context) {
		a.paymentsSummary(ctx)
	})

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

	http.Serve(listener, router)
}

func (a *App) payments(ctx *gin.Context) {
	var data map[string]interface{}
	if err := json.NewDecoder(ctx.Request.Body).Decode(&data); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Error"})
		return
	}

	data["requestedAt"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	ctx.Status(http.StatusNoContent)

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

func (a *App) paymentsSummary(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")

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

	c.JSON(http.StatusOK, resp)
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
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(item), &data); err == nil {
			if amt, ok := data["amount"].(float64); ok {
				sum += amt
			}
		}
	}

	return float64(int(sum*100)) / 100.0
}
