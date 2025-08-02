package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"

	"src/datetime"
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

func (a *App) SetupRoutes() {
	http.HandleFunc("/payments", a.payments)
	http.HandleFunc("/payments-summary", a.paymentsSummary)
}

func (a *App) payments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	data["requestedAt"] = time.Now().Format(time.RFC3339)

	encoded, _ := json.Marshal(data)
	a.client.RPush(context.Background(), "requests", encoded)

	w.WriteHeader(http.StatusNoContent)
}

func (a *App) paymentsSummary(w http.ResponseWriter, r *http.Request) {
	defaultList := a.getRequests("default_requests", r)
	fallbackList := a.getRequests("fallback_requests", r)

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *App) getRequests(listName string, r *http.Request) []string {
	ctx := context.Background()
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

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
