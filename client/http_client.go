package client

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/paologalligit/go-extractor/entities"
)

type Extractor interface {
	CallShowings(url string, headers map[string]string) (*entities.ShowingResponse, error)
	CallSeats(url string, headers map[string]string) (*entities.Response, error)
}

type ExtractorClient struct {
	client *http.Client
}

func New() *ExtractorClient {
	return &ExtractorClient{
		client: &http.Client{},
	}
}

// CallShowings fetches showings and unmarshals into ShowingResponse
func (c *ExtractorClient) CallShowings(url string, headers map[string]string) (*entities.ShowingResponse, error) {
	body, err := c.doGet(url, headers)
	if err != nil {
		return nil, err
	}
	var resp entities.ShowingResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CallSeats fetches seat data and unmarshals into Response
func (c *ExtractorClient) CallSeats(url string, headers map[string]string) (*entities.Response, error) {
	body, err := c.doGet(url, headers)
	if err != nil {
		return nil, err
	}
	var resp entities.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// doGet is an internal helper for GET requests
func (c *ExtractorClient) doGet(url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
