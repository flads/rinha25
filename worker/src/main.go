package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/valyala/fasthttp"
)

const (
	defaultURL  = "http://payment-processor-default:8080/payments"
	fallbackURL = "http://payment-processor-fallback:8080/payments"
)

type App struct {
	redis *redis.Client
	http  *fasthttp.Client
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
		time.Sleep(50 * time.Millisecond)

		var requests []string

		for i := 0; i < 250; i++ {
			item, err := a.redis.LPop(context.Background(), "requests").Result()
			if err == redis.Nil {
				break
			}

			if err != nil {
				log.Println("Error at LPOP:", err)
				break
			}
			requests = append(requests, item)
		}

		for _, raw := range requests {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &data); err != nil {
				continue
			}

			a.makePayment(data)
		}
	}
}

func (a *App) makePayment(data map[string]interface{}) {
	body, _ := json.Marshal(data)

	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	req.SetRequestURI(defaultURL)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBody(body)

	err := a.http.Do(req, res)
	if err == nil && res.StatusCode() == 200 {
		a.addToRequestsLists("default_requests", data)
		return
	}

	req.SetRequestURI(fallbackURL)
	err = a.http.Do(req, res)
	if err == nil && res.StatusCode() == 200 {
		a.addToRequestsLists("fallback_requests", data)
	}
}

func (a *App) addToRequestsLists(listName string, data map[string]interface{}) {
	requestedAt, ok := data["requestedAt"].(string)
	if !ok {
		return
	}

	score, err := StrToTimeWithMicro(requestedAt)
	if err != nil {
		log.Println("Error in timestamp conversion:", err)
		return
	}

	value, _ := json.Marshal(data)

	a.redis.ZAdd(context.Background(), listName, &redis.Z{
		Score:  float64(score),
		Member: value,
	})
}

func StrToTimeWithMicro(dateTime string) (int64, error) {
	layouts := []string{
		"2006-01-02T15:04:05.000000Z07:00",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}

	var t time.Time
	var err error

	for _, layout := range layouts {
		t, err = time.Parse(layout, dateTime)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, err
	}

	secs := t.Unix()
	micros := t.Nanosecond() / 1000

	return secs*1_000_000 + int64(micros), nil
}

func main() {
	app := NewApp()
	app.execute()
}
