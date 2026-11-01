package loadtest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

var (
	sortOptions       = []string{"price_asc", "price_desc", "rating", "newest"}
	searchTerms       = []string{"pro", "smart", "classic", "ultra", "eco", "premium", "compact", "wireless"}
	reviewTexts       = []string{"Great value for the price.", "Arrived quickly and works as described.", "Solid build quality.", "Would buy again.", "Decent, but not outstanding."}
	virtualUserHeader = "X-Virtual-User"
)

func (r *Runner) virtualUser(ctx context.Context, id int, state *runState, workers *sync.WaitGroup) {
	defer workers.Done()
	defer state.active.Add(-1)

	random := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)*7919))
	who := state.identities[id%len(state.identities)]
	label := "lt-" + strconv.Itoa(id)

	stagger := time.Duration(random.Int63n(int64(state.plan.ThinkMax) + 1))
	if !sleep(ctx, stagger) {
		return
	}
	for ctx.Err() == nil {
		r.step(ctx, random, who, label, state)
		span := int64(state.plan.ThinkMax - state.plan.ThinkMin)
		think := state.plan.ThinkMin
		if span > 0 {
			think += time.Duration(random.Int63n(span + 1))
		}
		if !sleep(ctx, think) {
			return
		}
	}
}

func sleep(ctx context.Context, duration time.Duration) bool {
	if duration <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (r *Runner) step(ctx context.Context, random *rand.Rand, who identity, label string, state *runState) {
	product := state.products[random.Intn(len(state.products))]
	roll := random.Intn(100)
	switch {
	case roll < 22:
		r.send(ctx, state, who, label, http.MethodGet,
			"/api/v1/products?limit=20&sort="+sortOptions[random.Intn(len(sortOptions))], nil)
	case roll < 38:
		r.send(ctx, state, who, label, http.MethodGet,
			"/api/v1/products/category/"+url.PathEscape(product.Category)+"?limit=20&sort=price_asc", nil)
	case roll < 48:
		r.send(ctx, state, who, label, http.MethodGet,
			"/api/v1/products?limit=20&q="+searchTerms[random.Intn(len(searchTerms))], nil)
	case roll < 63:
		r.send(ctx, state, who, label, http.MethodGet, "/api/v1/products/"+product.ID.Hex(), nil)
	case roll < 76:
		r.send(ctx, state, who, label, http.MethodGet, "/api/v1/reviews/product/"+product.ID.Hex()+"?limit=10", nil)
	case roll < 84:
		r.send(ctx, state, who, label, http.MethodGet, "/api/v1/reviews/product/"+product.ID.Hex()+"/summary", nil)
	case roll < 88:
		r.send(ctx, state, who, label, http.MethodPost, "/api/v1/reviews", map[string]any{
			"productId": product.ID.Hex(),
			"rating":    1 + random.Intn(5),
			"comment":   reviewTexts[random.Intn(len(reviewTexts))],
		})
	case roll < 93:
		r.send(ctx, state, who, label, http.MethodPost, "/api/v1/orders", buildOrder(random, state))
	default:
		r.send(ctx, state, who, label, http.MethodGet, "/api/v1/orders?limit=10", nil)
	}
}

func buildOrder(random *rand.Rand, state *runState) map[string]any {
	count := 1 + random.Intn(3)
	chosen := make(map[string]bool, count)
	items := make([]map[string]any, 0, count)
	for len(items) < count {
		product := state.products[random.Intn(len(state.products))]
		key := product.ID.Hex()
		if chosen[key] {
			continue
		}
		chosen[key] = true
		items = append(items, map[string]any{"productId": key, "quantity": 1 + random.Intn(2)})
	}
	return map[string]any{
		"items": items,
		"shippingAddress": map[string]string{
			"fullName":   "Load Test User",
			"line1":      "12 Benchmark Street",
			"city":       "Bengaluru",
			"state":      "Karnataka",
			"postalCode": "560001",
			"country":    "India",
		},
	}
}

func (r *Runner) send(ctx context.Context, state *runState, who identity, label, method, path string, payload any) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, body)
	if err != nil {
		return
	}
	request.Header.Set("Authorization", "Bearer "+who.token)
	request.Header.Set(virtualUserHeader, label)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	started := time.Now()
	response, err := r.client.Do(request)
	elapsed := time.Since(started)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		state.requests.Add(1)
		state.failures.Add(1)
		state.histogram.record(elapsed)
		return
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()

	state.requests.Add(1)
	state.histogram.record(elapsed)
	switch {
	case response.StatusCode == http.StatusTooManyRequests:
		state.throttled.Add(1)
	case response.StatusCode >= 500:
		state.failures.Add(1)
	}
}
