package header

import (
	"fmt"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type CookiesManager struct {
	cookies   string
	fetchedAt time.Time
}

const BASE_URL = "https://www.thespacecinema.it/"

func New() (*CookiesManager, error) {
	cookies, err := getCookiesFromBaseURL(BASE_URL)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}
	return &CookiesManager{
		cookies:   cookies,
		fetchedAt: time.Now(),
	}, nil
}

func (c *CookiesManager) GetCookies() (string, error) {
	if time.Since(c.fetchedAt) > 1*time.Hour {
		cookies, err := getCookiesFromBaseURL(BASE_URL)
		if err != nil {
			return "", fmt.Errorf("failed to get cookies: %w", err)
		}
		c.cookies = cookies
		c.fetchedAt = time.Now()
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
	if _, err := page.Goto(baseURL); err != nil {
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
