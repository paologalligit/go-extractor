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

func makeProxy(ip string, speed int, latency float64, lastUsedAgoSeconds int64) *ProxyElement {
	return &ProxyElement{
		Proxy: &Proxy{
			IP:      ip,
			Speed:   speed,
			Latency: latency,
		},
		LastUsedAt: time.Now().Add(-time.Duration(lastUsedAgoSeconds) * time.Second),
	}
}

func TestProxyHeapOrdering(t *testing.T) {
	t.Parallel()
	algo := &mockAlgo{}
	proxies := []*ProxyElement{
		makeProxy("2.2.2.2", 5, 100, 1800), // medium
		makeProxy("3.3.3.3", 1, 200, 60),   // low speed, high latency, recent
		makeProxy("1.1.1.1", 10, 50, 3600), // high speed, low latency, old last used
	}
	for _, p := range proxies {
		assert.NoError(t, p.UpdateScore(algo))
	}
	h := NewProxyHeap(proxies, algo)
	for i, p := range h.elements {
		t.Logf("index %d: ip=%s, score=%d", i, p.Proxy.IP, p.Score)
	}
	// After heap.Init, the e proxy should be at index 0 (root of the heap)
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
	proxies := []*ProxyElement{
		makeProxy("3.3.3.3", 1, 200, 60),
		makeProxy("2.2.2.2", 5, 100, 1800),
		makeProxy("1.1.1.1", 10, 50, 3600),
	}
	for _, p := range proxies {
		assert.NoError(t, p.UpdateScore(algo))
	}
	h := NewProxyHeap(proxies, algo)
	bestTwo := h.GetElements(2)
	assert.Equal(t, 2, len(bestTwo), "expected 2 elements, got %d", len(bestTwo))
	assert.Equal(t, "1.1.1.1", bestTwo[0].Proxy.IP)
	assert.Equal(t, "2.2.2.2", bestTwo[1].Proxy.IP)
}

func TestProxyHeapRecentlyUsedPenalty(t *testing.T) {
	t.Parallel()
	algo := &mockAlgo{}
	proxies := []*ProxyElement{
		makeProxy("1.1.1.1", 10, 50, 3600),
		makeProxy("2.2.2.2", 10, 50, 10), // recently used
	}
	for _, p := range proxies {
		assert.NoError(t, p.UpdateScore(algo))
	}
	h := NewProxyHeap(proxies, algo)
	best := h.elements[0]
	assert.Equal(t, "1.1.1.1", best.Proxy.IP, "expected best proxy to be 1.1.1.1, got %s", best.Proxy.IP)
}
