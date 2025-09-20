package client

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/paologalligit/go-extractor/constant"
	"github.com/paologalligit/go-extractor/entities"
	"github.com/paologalligit/go-extractor/header"
)

type Extractor interface {
	CallShowings(url string) (*entities.ShowingResponse, error)
	CallSeats(url string) (*entities.Response, error)
	GetCinemas() (*entities.CinemasFile, error)
	GetFilms() (*entities.FilmsFile, error)
}

type ExtractorClient struct {
	client        *http.Client
	cookieManager *header.CookiesManager
}

func New(cookieManager *header.CookiesManager) *ExtractorClient {
	return &ExtractorClient{
		client:        &http.Client{},
		cookieManager: cookieManager,
	}
}

// CallShowings fetches showings and unmarshals into ShowingResponse
func (c *ExtractorClient) CallShowings(url string) (*entities.ShowingResponse, error) {
	body, err := c.doGet(url)
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
func (c *ExtractorClient) CallSeats(url string) (*entities.Response, error) {
	body, err := c.doGet(url)
	if err != nil {
		return nil, err
	}
	var resp entities.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *ExtractorClient) GetCinemas() (*entities.CinemasFile, error) {
	body, err := c.doGet(constant.CINEMAS_URL)
	if err != nil {
		return nil, err
	}
	var cinemas entities.CinemasFile
	if err := json.Unmarshal(body, &cinemas); err != nil {
		return nil, err
	}
	return &cinemas, nil
}

func (c *ExtractorClient) GetFilms() (*entities.FilmsFile, error) {
	body, err := c.doGet(constant.FILMS_URL)
	if err != nil {
		return nil, err
	}
	var films entities.FilmsFile
	if err := json.Unmarshal(body, &films); err != nil {
		return nil, err
	}
	return &films, nil
}

func (c *ExtractorClient) HTTPClient() *http.Client {
	return c.client
}

// doGet is an internal helper for GET requests
func (c *ExtractorClient) doGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	headers, err := header.GetHeaders(c.cookieManager)
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
