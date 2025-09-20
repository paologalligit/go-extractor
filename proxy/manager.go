package proxy

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/paologalligit/go-extractor/constant"
)

type ProxyClient interface {
	GetProxies() ([]Proxy, error)
}

type ProxyClientImpl struct {
	client *http.Client
}

func NewProxyClientImpl(client *http.Client) *ProxyClientImpl {
	return &ProxyClientImpl{client: client}
}

func (p *ProxyClientImpl) GetProxies() ([]Proxy, error) {
	body, err := p.callProxyList()
	if err != nil {
		return nil, err
	}

	return parseProxyList(body)
}

// TODO: this should be a local list of tested proxies
func (p *ProxyClientImpl) callProxyList() ([]byte, error) {
	resp, err := p.client.Get(constant.PROXY_LIST_URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func parseProxyList(body []byte) ([]Proxy, error) {
	var proxies ProxyResponse
	if err := json.Unmarshal(body, &proxies); err != nil {
		return nil, err
	}
	return proxies.Data, nil
}

type ProxyResponse struct {
	Data  []Proxy `json:"data"`
	Total int     `json:"total"`
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
}

type Proxy struct {
	ID                 string   `json:"_id"`
	IP                 string   `json:"ip"`
	Port               string   `json:"port"`
	CreatedAt          string   `json:"created_at"`
	LastChecked        int64    `json:"lastChecked"`
	Latency            float64  `json:"latency"`
	Protocols          []string `json:"protocols"`
	Speed              int      `json:"speed"`
	UpTime             float64  `json:"upTime"`
	UpTimeSuccessCount int      `json:"upTimeSuccessCount"`
	UpTimeTryCount     int      `json:"upTimeTryCount"`
	UpdatedAt          string   `json:"updated_at"`
	ResponseTime       int      `json:"responseTime"`
}

// TODO: we need to consider also a fallback proxy which is the local ip address
type ProxyManager struct {
	proxyHeap *ProxyHeap
}

type ProxyManagerOptions struct {
	Client ProxyClient
	Algo   ProxyScoreAlgo
}

func New(options *ProxyManagerOptions) (*ProxyManager, error) {
	proxies, err := options.Client.GetProxies()
	if err != nil {
		return nil, err
	}

	return &ProxyManager{proxyHeap: NewProxyHeap(proxies, options.Algo)}, nil
}

func (p *ProxyManager) GetBestProxies(n uint16) []*ProxyElement {
	return p.proxyHeap.GetElements(n)
}
