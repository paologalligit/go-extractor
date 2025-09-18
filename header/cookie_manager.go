package header

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type TimeProvider interface {
	Now() time.Time
}

type realTimeProvider struct{}

func (realTimeProvider) Now() time.Time {
	return time.Now()
}

type CookiesManager struct {
	cookies      string
	timeProvider TimeProvider
}

const BASE_URL = "https://www.thespacecinema.it/"

func New() (*CookiesManager, error) {
	fmt.Println("Retrieving cookies for the first time...")
	cookies, err := getCookiesFromBaseURL(BASE_URL)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}
	return &CookiesManager{
		cookies:      cookies,
		timeProvider: realTimeProvider{},
	}, nil
}

func NewWithTimeProvider(tp TimeProvider) (*CookiesManager, error) {
	fmt.Println("Retrieving cookies for the first time...")
	cookies, err := getCookiesFromBaseURL(BASE_URL)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}
	return &CookiesManager{
		cookies:      cookies,
		timeProvider: tp,
	}, nil
}

func (c *CookiesManager) GetCookies() (string, error) {
	if c.IsExpired() {
		fmt.Println("Cookies expired! Time to fetch them brand new...")
		cookies, err := getCookiesFromBaseURL(BASE_URL)
		if err != nil {
			return "", fmt.Errorf("failed to get cookies: %w", err)
		}
		c.cookies = cookies
	}
	return c.cookies, nil
}

func getCookiesFromBaseURL(baseURL string) (string, error) {
	pw, err := playwright.Run()
	if err != nil {
		return "", fmt.Errorf("could not launch playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	context, err := browser.NewContext()
	if err != nil {
		return "", fmt.Errorf("could not create browser context: %w", err)
	}
	page, err := context.NewPage()
	if err != nil {
		return "", fmt.Errorf("could not create page: %w", err)
	}
	if _, err := page.Goto(baseURL, playwright.PageGotoOptions{Timeout: playwright.Float(45000)}); err != nil {
		return "", fmt.Errorf("could not navigate to baseURL: %w", err)
	}

	// Wait for network to be idle (optional, can adjust as needed)
	page.WaitForLoadState(
		playwright.PageWaitForLoadStateOptions{
			State:   playwright.LoadStateNetworkidle,
			Timeout: playwright.Float(10000),
		},
	)

	// Extract cookies
	cookies, err := context.Cookies()
	if err != nil {
		return "", fmt.Errorf("could not get cookies: %w", err)
	}

	// Format cookies for HTTP header
	var cookiePairs []string
	for _, c := range cookies {
		cookiePairs = append(cookiePairs, c.Name+"="+c.Value)
	}
	cookieHeader := strings.Join(cookiePairs, "; ")
	return cookieHeader, nil
}

func (c *CookiesManager) IsExpired() bool {
	decodedValue, err := extractAccessTokenExpirationTime(c.cookies)
	if err != nil {
		return true
	}
	// Parse the time (layout: 2006-01-02T15:04:05Z)
	t, err := time.Parse(time.RFC3339, decodedValue)
	if err != nil {
		return true
	}
	// Convert to local time
	localTime := t.Local()
	// If now is after the expiration time, it's expired
	return c.timeProvider.Now().After(localTime)
}

func extractAccessTokenExpirationTime(cookies string) (string, error) {
	cookiePairs := strings.SplitSeq(cookies, "; ")
	for pair := range cookiePairs {
		if after, ok := strings.CutPrefix(pair, "accessTokenExpirationTime="); ok {
			value := after
			// URL decode the value
			decodedValue, err := url.QueryUnescape(value)
			if err != nil {
				return "", err
			}
			return decodedValue, nil
		}
	}
	return "", fmt.Errorf("accessTokenExpirationTime not found")
}
