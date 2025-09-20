package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"log"

	"net"

	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/header"
	ext_proxy "github.com/paologalligit/go-extractor/proxy"
	"golang.org/x/net/proxy"
)

// ClientWithProxy associates an http.Client with its proxy info.
type ClientWithProxy struct {
	Client Extractor
	Proxy  *ext_proxy.Proxy
}

// ClientPool manages a pool of reusable *http.Client instances, each using a proxy.
type ClientPool struct {
	pool         chan *ClientWithProxy
	proxyManager *ext_proxy.ProxyManager
}

func newTransportForProxy(proxyElem *ext_proxy.Proxy) *http.Transport {
	scheme := "http"
	if len(proxyElem.Protocols) > 0 && proxyElem.Protocols[0] != "" {
		scheme = proxyElem.Protocols[0]
	}
	proxyAddr := net.JoinHostPort(proxyElem.IP, proxyElem.Port)

	switch scheme {
	case "http", "https":
		proxyURLStr := fmt.Sprintf("%s://%s", scheme, proxyAddr)
		u, err := url.Parse(proxyURLStr)
		if err != nil {
			return nil // or handle error
		}
		return &http.Transport{
			Proxy:               http.ProxyURL(u),
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	case "socks4", "socks5":
		var auth *proxy.Auth
		// If you have username/password, set auth here
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
		if err != nil {
			return nil // or handle error
		}
		return &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	default:
		// Fallback: no proxy
		return &http.Transport{}
	}
}

// NewClientPool creates a new pool with the given size, using proxies from the ProxyManager.
func NewClientPool(size uint16, proxyManager *ext_proxy.ProxyManager, cookieManager *header.CookiesManager) *ClientPool {
	pool := make(chan *ClientWithProxy, size)
	bestProxies := proxyManager.GetBestProxies(size)
	for i := range bestProxies {
		proxyElem := bestProxies[i]
		transport := newTransportForProxy(proxyElem.Proxy)
		if transport == nil {
			continue
		}
		client := &ExtractorClient{
			client: &http.Client{
				Transport: transport,
				Timeout:   30 * time.Second,
			},
			cookieManager: cookieManager,
		}
		log.Printf("[POOL] Created client for proxy %s:%s (%v)", proxyElem.Proxy.IP, proxyElem.Proxy.Port, proxyElem.Proxy.Protocols)
		pool <- &ClientWithProxy{Client: client, Proxy: proxyElem.Proxy}
	}
	return &ClientPool{pool: pool, proxyManager: proxyManager}
}

// Get retrieves a client from the pool.
func (p *ClientPool) Get() *ClientWithProxy {
	cwp := <-p.pool
	log.Printf("[POOL] Borrowed client for proxy %s:%s", cwp.Proxy.IP, cwp.Proxy.Port)
	return cwp
}

// Put returns a client to the pool.
func (p *ClientPool) Put(cwp *ClientWithProxy) {
	log.Printf("[POOL] Returned client for proxy %s:%s", cwp.Proxy.IP, cwp.Proxy.Port)
	p.pool <- cwp
}

// CallShowings fetches showings and unmarshals into ShowingResponse
func (c *ClientPool) CallShowings(url string) (*entities.ShowingResponse, error) {
	cwp := c.Get()
	defer c.Put(cwp)

	return cwp.Client.CallShowings(url)
}

// CallSeats fetches seat data and unmarshals into Response
func (c *ClientPool) CallSeats(url string) (*entities.Response, error) {
	cwp := c.Get()
	defer c.Put(cwp)

	return cwp.Client.CallSeats(url)
}

func (c *ClientPool) GetCinemas() (*entities.CinemasFile, error) {
	cwp := c.Get()
	defer c.Put(cwp)

	return cwp.Client.GetCinemas()
}

func (c *ClientPool) GetFilms() (*entities.FilmsFile, error) {
	cwp := c.Get()
	defer c.Put(cwp)

	return cwp.Client.GetFilms()
}
