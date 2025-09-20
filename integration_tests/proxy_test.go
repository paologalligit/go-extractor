package integrationtests

import (
	"encoding/json"
	"log"
	"net/http"
	"testing"

	"github.com/paologalligit/go-extractor/client"
	"github.com/paologalligit/go-extractor/proxy"
)

type httpBinIPResponse struct {
	Origin string `json:"origin"`
}

type MockProxyClient struct{}

func (MockProxyClient) GetProxies() ([]proxy.Proxy, error) {
	return []proxy.Proxy{
		{
			ID:                 "661656326fb9cbee37c78800",
			IP:                 "164.92.172.206",
			Port:               "8888",
			CreatedAt:          "2024-04-10T09:04:50.351Z",
			LastChecked:        1758297164,
			Latency:            11.343,
			Protocols:          []string{"http"},
			Speed:              1,
			UpTime:             71.76377537569206,
			UpTimeSuccessCount: 2722,
			UpTimeTryCount:     3793,
			UpdatedAt:          "2025-09-19T15:52:44.472Z",
			ResponseTime:       1671,
		},
	}, nil
}

func TestClientPoolUsesProxy(t *testing.T) {
	t.Parallel()
	// Setup a minimal ProxyManager and ClientPool for the test
	proxyAlgo := &proxy.ProxyScoreAlgoNaive{}
	proxyManager, err := proxy.New(&proxy.ProxyManagerOptions{
		Client: MockProxyClient{},
		Algo:   proxyAlgo,
	})
	if err != nil {
		t.Fatalf("failed to create proxy manager: %v", err)
	}
	pool := client.NewClientPool(5, proxyManager, nil)

	for range 2 {
		cwp := pool.Get()

		proxyIP := cwp.Proxy.IP
		log.Printf("[TEST] Using proxy IP: %s", proxyIP)

		ec, ok := cwp.Client.(*client.ExtractorClient)
		if !ok {
			t.Fatalf("client is not an ExtractorClient")
		}
		// Use http.NewRequest and Do for robust proxy support
		req, err := http.NewRequest("GET", "https://httpbin.org/ip", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		resp, err := ec.HTTPClient().Do(req)
		if err != nil {
			t.Fatalf("failed to GET httpbin.org/ip: %v", err)
		}
		defer resp.Body.Close()

		var ipResp httpBinIPResponse
		if err := json.NewDecoder(resp.Body).Decode(&ipResp); err != nil {
			t.Fatalf("failed to decode httpbin response: %v", err)
		}
		log.Printf("[TEST] httpbin.org/ip returned: %s", ipResp.Origin)

		if ipResp.Origin == "" {
			t.Fatalf("httpbin.org/ip returned empty origin")
		}
		if ipResp.Origin != proxyIP {
			t.Errorf("expected proxy IP %s, got %s (proxy not used?)", proxyIP, ipResp.Origin)
		}
		pool.Put(cwp)
	}
}
