package client

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/paologalligit/go-extractor/header"
	"github.com/paologalligit/go-extractor/proxy"
)

// ClientWithProxy associates an http.Client with its proxy info.
type ClientWithProxy struct {
	Client Extractor
	Proxy  *proxy.Proxy
}

// ClientPool manages a pool of reusable *http.Client instances, each using a proxy.
type ClientPool struct {
	pool         chan *ClientWithProxy
	proxyManager *proxy.ProxyManager
}

// NewClientPool creates a new pool with the given size, using proxies from the ProxyManager.
func NewClientPool(size uint16, proxyManager *proxy.ProxyManager, cookieManager *header.CookiesManager) *ClientPool {
	pool := make(chan *ClientWithProxy, size)
	bestProxies := proxyManager.GetBestProxies(size)
	for i := range bestProxies {
		proxyElem := bestProxies[i]
		scheme := "http"
		if len(proxyElem.Proxy.Protocols) > 0 && proxyElem.Proxy.Protocols[0] != "" {
			scheme = proxyElem.Proxy.Protocols[0]
		}
		proxyURLStr := fmt.Sprintf("%s://%s:%s", scheme, proxyElem.Proxy.IP, proxyElem.Proxy.Port)
		u, err := url.Parse(proxyURLStr)
		if err != nil {
			// Skip this proxy if URL is invalid
			continue
		}
		transport := &http.Transport{
			Proxy:               http.ProxyURL(u),
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			TLSHandshakeTimeout: 10 * time.Second,
		}
		client := &ExtractorClient{
			client: &http.Client{
				Transport: transport,
				Timeout:   30 * time.Second,
			},
			cookieManager: cookieManager,
		}
		pool <- &ClientWithProxy{Client: client, Proxy: proxyElem.Proxy}
	}
	return &ClientPool{pool: pool, proxyManager: proxyManager}
}

// Get retrieves a client from the pool.
func (p *ClientPool) Get() *ClientWithProxy {
	return <-p.pool
}

// Put returns a client to the pool.
func (p *ClientPool) Put(cwp *ClientWithProxy) {
	p.pool <- cwp
}
