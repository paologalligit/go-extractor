package proxy

import (
	"container/heap"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockAlgo struct{}

func (m *mockAlgo) CalculateScore(proxy *Proxy, lastUsedAgoSeconds int64) int {
	// Simple deterministic score: higher speed, lower latency, older last used is better
	return int(1000*float64(proxy.Speed) - 10*proxy.Latency + float64(lastUsedAgoSeconds))
}

func makeProxy(ip string, speed int, latency float64, lastUsedAgoSeconds int64) Proxy {
	return Proxy{
		IP:      ip,
		Speed:   speed,
		Latency: latency,
	}
}

func TestProxyHeapOrdering(t *testing.T) {
	t.Parallel()
	algo := &mockAlgo{}
	now := time.Now()
	proxies := []Proxy{
		makeProxy("2.2.2.2", 5, 100, 1800), // medium
		makeProxy("3.3.3.3", 1, 200, 60),   // low speed, high latency, recent
		makeProxy("1.1.1.1", 10, 50, 3600), // high speed, low latency, old last used
	}

	h := NewProxyHeap(proxies, algo)
	// Set LastUsedAt to simulate different last used times
	h.elements[0].LastUsedAt = now.Add(-1800 * time.Second)
	h.elements[1].LastUsedAt = now.Add(-60 * time.Second)
	h.elements[2].LastUsedAt = now.Add(-3600 * time.Second)
	for _, e := range h.elements {
		_ = e.UpdateScore(algo)
	}
	heap.Init(h)
	for i, p := range h.elements {
		t.Logf("index %d: ip=%s, score=%d", i, p.Proxy.IP, p.Score)
	}
	e := heap.Pop(h).(*ProxyElement)
	assert.Equal(t, "1.1.1.1", e.Proxy.IP)
	e = heap.Pop(h).(*ProxyElement)
	assert.Equal(t, "2.2.2.2", e.Proxy.IP)
	e = heap.Pop(h).(*ProxyElement)
	assert.Equal(t, "3.3.3.3", e.Proxy.IP)
}

func TestProxyHeapGetElements(t *testing.T) {
	t.Parallel()
	algo := &mockAlgo{}
	now := time.Now()
	proxies := []Proxy{
		makeProxy("3.3.3.3", 1, 200, 60),
		makeProxy("2.2.2.2", 5, 100, 1800),
		makeProxy("1.1.1.1", 10, 50, 3600),
	}

	h := NewProxyHeap(proxies, algo)
	h.elements[0].LastUsedAt = now.Add(-60 * time.Second)
	h.elements[1].LastUsedAt = now.Add(-1800 * time.Second)
	h.elements[2].LastUsedAt = now.Add(-3600 * time.Second)
	for _, e := range h.elements {
		_ = e.UpdateScore(algo)
	}
	heap.Init(h)
	bestTwo := h.GetElements(2)
	assert.Equal(t, 2, len(bestTwo), "expected 2 elements, got %d", len(bestTwo))
	assert.Equal(t, "1.1.1.1", bestTwo[0].Proxy.IP)
	assert.Equal(t, "2.2.2.2", bestTwo[1].Proxy.IP)
}

func TestProxyHeapRecentlyUsedPenalty(t *testing.T) {
	t.Parallel()
	algo := &mockAlgo{}
	now := time.Now()
	proxies := []Proxy{
		makeProxy("1.1.1.1", 10, 50, 3600),
		makeProxy("2.2.2.2", 10, 50, 10), // recently used
	}

	h := NewProxyHeap(proxies, algo)
	h.elements[0].LastUsedAt = now.Add(-3600 * time.Second)
	h.elements[1].LastUsedAt = now.Add(-10 * time.Second)
	for _, e := range h.elements {
		_ = e.UpdateScore(algo)
	}
	heap.Init(h)
	best := h.elements[0]
	assert.Equal(t, "1.1.1.1", best.Proxy.IP, "expected best proxy to be 1.1.1.1, got %s", best.Proxy.IP)
}
