package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Payment struct {
	Timestamp int64   `json:"timestamp"`
	Amount    float64 `json:"amount"`
}

type Request struct {
	Amount        float64 `json:"amount"`
	RequestedAt   string  `json:"requestedAt"`
	CorrelationID string  `json:"correlationId"`
}

var (
	requests         []Request
	defaultRequests  []Payment
	fallbackRequests []Payment
	mu               sync.Mutex
)

func getRequestsSummary(requests []Payment, from, to int64) (totalAmount float64, totalRequests int) {
	log.Printf("Calculating summary from %d to %d", from, to)
	log.Printf("Total requests: %d", len(requests))
	log.Printf("Total requests: %v", requests)

	start := sort.Search(len(requests), func(i int) bool {
		return requests[i].Timestamp >= from
	})
	end := sort.Search(len(requests), func(i int) bool {
		return requests[i].Timestamp > to
	})
	selected := requests[start:end]
	for _, p := range selected {
		totalAmount += p.Amount
	}
	totalRequests = len(selected)
	return
}

func paymentsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	switch r.Method {
	case http.MethodPost:
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Preenche RequestedAt com horário do servidor
		req.RequestedAt = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
		requests = append(requests, req)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "received"})

	case http.MethodGet:
		limit := 250
		if len(requests) < limit {
			limit = len(requests)
		}
		resp := requests[:limit]

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func paymentsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var from, to int64
	if fromStr != "" {
		parsedFrom, err := time.Parse("2006-01-02T15:04:05.000Z07:00", fromStr)
		if err != nil {
			http.Error(w, "Invalid 'from' parameter", http.StatusBadRequest)
			return
		}
		from = parsedFrom.UnixMicro()
	}
	if toStr != "" {
		parsedTo, err := time.Parse("2006-01-02T15:04:05.000Z07:00", toStr)
		if err != nil {
			http.Error(w, "Invalid 'to' parameter", http.StatusBadRequest)
			return
		}
		to = parsedTo.UnixMicro()
	} else {
		to = time.Now().UnixMicro()
	}

	mu.Lock()
	defer mu.Unlock()

	defaultAmount, defaultTotal := getRequestsSummary(defaultRequests, from, to)
	fallbackAmount, fallbackTotal := getRequestsSummary(fallbackRequests, from, to)

	resp := map[string]interface{}{
		"default": map[string]interface{}{
			"totalRequests": defaultTotal,
			"totalAmount":   defaultAmount,
		},
		"fallback": map[string]interface{}{
			"totalRequests": fallbackTotal,
			"totalAmount":   fallbackAmount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func processorDefaultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var p Payment
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("Received default payment: %+v", p)

	mu.Lock()
	defer mu.Unlock()
	defaultRequests = append(defaultRequests, p)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
}

func processorFallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var p Payment
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	fallbackRequests = append(fallbackRequests, p)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
}

func populateTestPayments() {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().UnixMicro()

	for i := 0; i < 1000; i++ {
		p := Payment{
			Timestamp: now + int64(i), // para diferenciar timestamps
			Amount:    float64(i+1) * 10,
		}
		defaultRequests = append(defaultRequests, p)
	}

	for i := 0; i < 200; i++ {
		p := Payment{
			Timestamp: now + int64(i),
			Amount:    float64(i+1) * 5,
		}
		fallbackRequests = append(fallbackRequests, p)
	}

	fmt.Printf("Inseridos %d registros na defaultRequests e %d na fallbackRequests\n", len(defaultRequests), len(fallbackRequests))
}

func main() {
	// populateTestPayments() // se quiser popular com dados de teste

	http.HandleFunc("/payments", paymentsHandler)
	http.HandleFunc("/payments-summary", paymentsSummaryHandler)
	http.HandleFunc("/processor-default", processorDefaultHandler)
	http.HandleFunc("/processor-fallback", processorFallbackHandler)

	log.Println("API rodando em :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
