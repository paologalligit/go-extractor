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

func (p *ProxyClientImpl) GetProxies() ([]Proxy, error) {
	body, err := p.callProxyList()
	if err != nil {
		return nil, err
	}

	return parseProxyList(body)
}

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
	AnonymityLevel     string   `json:"anonymityLevel"`
	ASN                string   `json:"asn"`
	City               string   `json:"city,omitempty"`
	Country            string   `json:"country"`
	CreatedAt          string   `json:"created_at"`
	ISP                string   `json:"isp,omitempty"`
	LastChecked        int64    `json:"lastChecked"`
	Latency            float64  `json:"latency"`
	Org                string   `json:"org"`
	Port               string   `json:"port"`
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
	proxies []Proxy
}

type ProxyManagerOptions struct {
	Client ProxyClient
}

func New(options *ProxyManagerOptions) (*ProxyManager, error) {
	proxies, err := options.Client.GetProxies()
	if err != nil {
		return nil, err
	}

	return &ProxyManager{proxies: proxies}, nil
}
