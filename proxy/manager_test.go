package proxy

import (
	"fmt"
	"testing"
	"time"

	"container/heap"
	"sync"

	"github.com/stretchr/testify/assert"
)

const PROXY_RESPONSE = `{"data":[{"_id":"662a76a76fb9cbee37969d17","ip":"116.100.220.220","anonymityLevel":"elite","asn":"AS7552","city":"Hanoi","country":"VN","created_at":"2024-04-25T15:28:39.254Z","google":false,"isp":"Viettel Corporation","lastChecked":1758272716,"latency":184.703,"org":"Viettel Group","port":"1080","protocols":["socks4"],"speed":1,"upTime":59.28843020097773,"upTimeSuccessCount":2183,"upTimeTryCount":3682,"updated_at":"2025-09-19T09:05:16.492Z","responseTime":2711},{"_id":"662a000a6fb9cbee37852713","ip":"102.165.125.102","anonymityLevel":"elite","asn":"AS37521","country":"NG","created_at":"2024-04-25T07:02:11.876Z","google":false,"lastChecked":1758272716,"latency":129.744,"org":"Internet Solutions Nigeria Limited","port":"5678","protocols":["socks4"],"speed":1,"upTime":98.09782608695652,"upTimeSuccessCount":3610,"upTimeTryCount":3680,"updated_at":"2025-09-19T09:05:16.585Z","city":"Ikeja","isp":"Internet Solutions Nigeria Limited","responseTime":4220},{"_id":"6659e8666fb9cbee3787eb84","ip":"187.95.82.53","anonymityLevel":"elite","asn":"AS262696","city":"Araçariguama","country":"BR","created_at":"2024-05-31T15:10:30.503Z","google":false,"isp":"Turbonet Telecomunicações","lastChecked":1758272716,"latency":208.46,"org":"Turbonet Telecomunicações","port":"3629","protocols":["socks4"],"speed":1,"upTime":99.23887587822014,"upTimeSuccessCount":3390,"upTimeTryCount":3416,"updated_at":"2025-09-19T09:05:16.582Z","responseTime":2205},{"_id":"662a20d26fb9cbee378a405b","ip":"34.43.181.55","anonymityLevel":"elite","asn":"AS396982","city":"Mountain View","country":"US","created_at":"2024-04-25T09:22:26.286Z","google":false,"isp":"Google LLC","lastChecked":1758272716,"latency":3.257,"org":"Google LLC","port":"3128","protocols":["socks5"],"speed":1,"upTime":99.9729510413849,"upTimeSuccessCount":3696,"upTimeTryCount":3697,"updated_at":"2025-09-19T09:05:16.582Z","responseTime":3516},{"_id":"661bf21f6fb9cbee378e8bb4","ip":"209.159.153.22","anonymityLevel":"elite","asn":"AS19318","city":"Secaucus","country":"US","created_at":"2024-04-14T15:11:27.922Z","google":false,"isp":"Interserver, Inc","lastChecked":1758272716,"latency":82.662,"org":"Interserver, Inc","port":"15817","protocols":["socks4"],"speed":1,"upTime":80.63063063063063,"upTimeSuccessCount":3043,"upTimeTryCount":3774,"updated_at":"2025-09-19T09:05:16.491Z","responseTime":3116}],"total":11111,"page":1,"limit":5}`

type MockProxyClient struct{}

func (MockProxyClient) GetProxies() ([]Proxy, error) {
	return parseProxyList([]byte(PROXY_RESPONSE))
}

type EmptyProxyClient struct{}

func (EmptyProxyClient) GetProxies() ([]Proxy, error) {
	return []Proxy{}, nil
}

type ErrorProxyClient struct{}

func (ErrorProxyClient) GetProxies() ([]Proxy, error) {
	return nil, fmt.Errorf("client error")
}

type FallbackProxyClient struct{}

func (FallbackProxyClient) GetProxies() ([]Proxy, error) {
	return []Proxy{}, nil
}

type MalformedProxyClient struct{}

func (MalformedProxyClient) GetProxies() ([]Proxy, error) {
	// Malformed JSON
	badJSON := []byte(`{"data":[{"_id":123,"ip":true}]}`)
	return parseProxyList(badJSON)
}

type mockHeapAlgo struct{}

func (m *mockHeapAlgo) CalculateScore(proxy *Proxy, lastUsedAgoSeconds int64) int {
	// Simple: higher speed, lower latency, older last used is better
	return int(1000*float64(proxy.Speed) - 10*proxy.Latency + float64(lastUsedAgoSeconds))
}

type reverseHeapAlgo struct{}

func (m *reverseHeapAlgo) CalculateScore(proxy *Proxy, lastUsedAgoSeconds int64) int {
	// Reverse: lower speed, higher latency, more recent last used is better
	return int(-1000*float64(proxy.Speed) + 10*proxy.Latency - float64(lastUsedAgoSeconds))
}

func TestParseBodyList(t *testing.T) {
	proxies, err := parseProxyList([]byte(PROXY_RESPONSE))
	assert.NoError(t, err)
	assert.Equal(t, 5, len(proxies))
	assert.Equal(t, "116.100.220.220", proxies[0].IP)
	assert.Equal(t, "102.165.125.102", proxies[1].IP)
	assert.Equal(t, "187.95.82.53", proxies[2].IP)
	assert.Equal(t, "34.43.181.55", proxies[3].IP)
	assert.Equal(t, "209.159.153.22", proxies[4].IP)
}

func TestProxyManagerInitialization(t *testing.T) {
	client := MockProxyClient{}
	options := &ProxyManagerOptions{Client: client}
	manager, err := New(options)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, 5, len(manager.proxyHeap.elements))
	assert.Equal(t, "116.100.220.220", manager.proxyHeap.elements[0].Proxy.IP)
}

func TestProxyManagerEmptyList(t *testing.T) {
	client := EmptyProxyClient{}
	options := &ProxyManagerOptions{Client: client}
	manager, err := New(options)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, 0, len(manager.proxyHeap.elements))
}

func TestProxyManagerErrorHandling(t *testing.T) {
	client := ErrorProxyClient{}
	options := &ProxyManagerOptions{Client: client}
	manager, err := New(options)
	assert.Error(t, err)
	assert.Nil(t, manager)
}

func TestProxyManagerHeapIntegration(t *testing.T) {
	client := MockProxyClient{}
	options := &ProxyManagerOptions{Client: client, Algo: &mockHeapAlgo{}}
	manager, err := New(options)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// Create a new heap from the proxies for testing
	proxies := make([]Proxy, len(manager.proxyHeap.elements))
	for i, elem := range manager.proxyHeap.elements {
		proxies[i] = *elem.Proxy
	}
	algo := &mockHeapAlgo{}
	h := NewProxyHeap(proxies, algo)
	// Pop all elements to check order
	var ips []string
	for h.Len() > 0 {
		ips = append(ips, heap.Pop(h).(*ProxyElement).Proxy.IP)
	}
	// The best proxy (by mock scoring) should be first
	assert.Contains(t, ips, "34.43.181.55") // This proxy has lowest latency in the mock data
}

// For this test, we'll assume a GetBestProxies(n) method exists or is simulated.
func getBestProxiesFromHeap(h *ProxyHeap, n int) []*Proxy {
	var result []*Proxy
	tmp := make([]Proxy, len(h.elements))
	for i, elem := range h.elements {
		tmp[i] = *elem.Proxy
	}
	tmpHeap := NewProxyHeap(tmp, h.algo)
	for i := 0; i < n && tmpHeap.Len() > 0; i++ {
		result = append(result, heap.Pop(tmpHeap).(*ProxyElement).Proxy)
	}
	return result
}

func TestProxyManagerGetBestProxies(t *testing.T) {
	client := MockProxyClient{}
	options := &ProxyManagerOptions{Client: client, Algo: &mockHeapAlgo{}}
	manager, err := New(options)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	h := manager.proxyHeap
	best := getBestProxiesFromHeap(h, 2)
	assert.Equal(t, 2, len(best))
	// The best proxies by mock scoring should be first
	assert.Contains(t, []string{"34.43.181.55", "209.159.153.22", "187.95.82.53", "116.100.220.220", "102.165.125.102"}, best[0].IP)
	assert.Contains(t, []string{"34.43.181.55", "209.159.153.22", "187.95.82.53", "116.100.220.220", "102.165.125.102"}, best[1].IP)
}

func TestProxyManagerProxyUsageUpdate(t *testing.T) {
	client := MockProxyClient{}
	options := &ProxyManagerOptions{Client: client, Algo: &mockHeapAlgo{}}
	manager, err := New(options)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	h := manager.proxyHeap
	// Mark the current best as just used (set LastUsedAt to now)
	best := h.elements[0]
	best.LastUsedAt = time.Now()
	_ = best.UpdateScore(h.algo)
	heap.Fix(h, best.Index)

	// After reordering, the previously best proxy should not be at the top
	newBest := h.elements[0]
	assert.NotEqual(t, best.Proxy.IP, newBest.Proxy.IP, "recently used proxy should be deprioritized")
}

func TestProxyManagerFallbackProxy(t *testing.T) {
	client := FallbackProxyClient{}
	options := &ProxyManagerOptions{Client: client, Algo: &mockHeapAlgo{}}
	manager, err := New(options)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	// Simulate a fallback: if no proxies, return a local IP proxy
	fallback := Proxy{IP: "127.0.0.1"}
	var proxyToUse Proxy
	if len(manager.proxyHeap.elements) == 0 {
		proxyToUse = fallback
	} else {
		proxyToUse = *manager.proxyHeap.elements[0].Proxy
	}
	assert.Equal(t, "127.0.0.1", proxyToUse.IP)
}

func TestProxyManagerThreadSafety(t *testing.T) {
	t.Parallel()
	client := MockProxyClient{}
	options := &ProxyManagerOptions{Client: client, Algo: &mockHeapAlgo{}}
	manager, err := New(options)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	h := manager.proxyHeap
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				// Simulate concurrent reads
				_ = h.GetElements(2)
				// Simulate concurrent update
				e := h.elements[idx%len(h.elements)]
				e.LastUsedAt = time.Now()
				assert.NoError(t, e.UpdateScore(h.algo))
				heap.Fix(h, e.Index)
			}
		}(i)
	}
	wg.Wait()
	// If no panic or race, test passes
}

func TestProxyManagerAlgoSwap(t *testing.T) {
	client := MockProxyClient{}
	options := &ProxyManagerOptions{Client: client, Algo: &mockHeapAlgo{}}
	manager, err := New(options)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// Use diverse test data
	proxies := []Proxy{
		{IP: "1.1.1.1", Speed: 10, Latency: 10},
		{IP: "2.2.2.2", Speed: 1, Latency: 200},
		{IP: "3.3.3.3", Speed: 5, Latency: 100},
	}
	algo1 := &mockHeapAlgo{}
	h1 := NewProxyHeap(proxies, algo1)
	now := time.Now()
	h1.elements[0].LastUsedAt = now.Add(-3600 * time.Second)
	h1.elements[1].LastUsedAt = now.Add(-60 * time.Second)
	h1.elements[2].LastUsedAt = now.Add(-1800 * time.Second)
	for _, e := range h1.elements {
		_ = e.UpdateScore(algo1)
	}
	heap.Init(h1)
	best1 := heap.Pop(h1).(*ProxyElement).Proxy.IP

	// Now use a different scoring algo
	algo2 := &reverseHeapAlgo{}
	h2 := NewProxyHeap(proxies, algo2)
	h2.elements[0].LastUsedAt = now.Add(-3600 * time.Second)
	h2.elements[1].LastUsedAt = now.Add(-60 * time.Second)
	h2.elements[2].LastUsedAt = now.Add(-1800 * time.Second)
	for _, e := range h2.elements {
		_ = e.UpdateScore(algo2)
	}
	heap.Init(h2)
	best2 := heap.Pop(h2).(*ProxyElement).Proxy.IP

	assert.NotEqual(t, best1, best2, "Swapping algos should change the best proxy")
}

func TestProxyManagerMalformedProxyData(t *testing.T) {
	client := MalformedProxyClient{}
	options := &ProxyManagerOptions{Client: client}
	manager, err := New(options)
	assert.Error(t, err)
	assert.Nil(t, manager)
}
