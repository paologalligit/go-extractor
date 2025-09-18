package header

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	COOKIE = "vuecinemas-it#lang=it-IT; ASP.NET_SessionId=4sie3vepgw5qkbcne3bfmuzr; shell#lang=en; cinemaCurrency=EUR; isSecondaryMarket=false; cinemaId=1018; cinemaName=torino; analyticsCinemaName=Torino; __RequestVerificationToken=DfqenUDuwKPzmOUtfBKU1mkmH7d2GJLNUbORQyD89nAvVJXlgb1IW6iBpUa04vQBlNnKk7SvMWM3PrY5YKRJeT09uEIPwiRoBFCUhVodHys1; microservicesToken=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiI1MDczZDYwNS1kZjY2LTRmYzItODNiYy00YjE1M2YyNTcwOGEiLCJDb3VudHJ5IjoiSVQiLCJBdXRoIjoiMyIsIlNob3dpbmciOiIzIiwiQm9va2luZyI6IjMiLCJQYXltZW50IjoiMyIsIlBhcnRuZXIiOiIwIiwiTG95YWx0eSI6IjMiLCJDYW1wYWlnblRyYWNraW5nQ29kZSI6IiIsIkNsaWVudE5hbWUiOiIiLCJuYmYiOjE3NTgyMDU3NDcsImV4cCI6MTc1ODI0ODk0NywiaXNzIjoiUHJvZCJ9.woKTTNqLYK9itznQkqGWGLSudFfM3bII021O1fqksms; microservicesRefreshToken=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiI1MDczZDYwNS1kZjY2LTRmYzItODNiYy00YjE1M2YyNTcwOGEiLCJDb3VudHJ5IjoiSVQiLCJJc0Fub255bW91cyI6IlRydWUiLCJuYmYiOjE3NTgyMDU3NDcsImV4cCI6MTc1ODgxMDU0NywiaXNzIjoiQXV0aFByb2QifQ.OTTMd3P2EgJ7SI9_VcdubTOIXJox0QRbW9vb6OqETw0; accessTokenExpirationTime=2025-09-19T02%3A29%3A07Z; refreshTokenExpirationTime=2025-09-25T14%3A29%3A07Z; __cflb=02DiuE2nAGMFu3TxDikSow61ukPuq1im33X6HPRK1TRUx; SC_ANALYTICS_GLOBAL_COOKIE=e1b05fd31d7e46d89d29e8da925c5bdb|True; hasLayout=true"
	EXP    = "2025-09-19T02:29:07Z"
)

type mockTimeProvider struct {
	now time.Time
}

func (m mockTimeProvider) Now() time.Time {
	return m.now
}

func TestCookiesManager_IsExpired(t *testing.T) {
	exp, _ := time.Parse(time.RFC3339, EXP)
	tests := []struct {
		name           string
		cookies        string
		timeProvider   TimeProvider
		expectsExpired bool
	}{
		{
			name:           "expired (now after EXP)",
			cookies:        COOKIE,
			timeProvider:   mockTimeProvider{now: exp.Add(1 * time.Second)},
			expectsExpired: true,
		},
		{
			name:           "not expired (now before EXP)",
			cookies:        COOKIE,
			timeProvider:   mockTimeProvider{now: exp.Add(-1 * time.Second)},
			expectsExpired: false,
		},
		{
			name:           "missing accessTokenExpirationTime",
			cookies:        "foo=bar; other=val",
			timeProvider:   mockTimeProvider{now: exp},
			expectsExpired: true, // fallback: should be expired
		},
		{
			name:           "malformed accessTokenExpirationTime",
			cookies:        "accessTokenExpirationTime=not-a-date",
			timeProvider:   mockTimeProvider{now: exp},
			expectsExpired: true, // fallback: should be expired
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cm := &CookiesManager{
				cookies:      tc.cookies,
				timeProvider: tc.timeProvider,
			}
			assert.Equal(t, tc.expectsExpired, cm.IsExpired())
		})
	}
}

func TestExtractAccessTokenExpirationTime(t *testing.T) {
	cases := []struct {
		name        string
		cookies     string
		expectsErr  bool
		expectsTime string
	}{
		{
			name:        "valid accessTokenExpirationTime",
			cookies:     COOKIE,
			expectsErr:  false,
			expectsTime: EXP,
		},
		{
			name:        "missing accessTokenExpirationTime",
			cookies:     "foo=bar; other=val",
			expectsErr:  true,
			expectsTime: "",
		},
		{
			name:        "malformed accessTokenExpirationTime (not url encoded)",
			cookies:     "accessTokenExpirationTime=not-a-date",
			expectsErr:  false, // url.QueryUnescape will succeed, but time.Parse will fail elsewhere
			expectsTime: "not-a-date",
		},
		{
			name:        "malformed accessTokenExpirationTime (bad encoding)",
			cookies:     "accessTokenExpirationTime=%ZZZ",
			expectsErr:  true,
			expectsTime: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			val, err := extractAccessTokenExpirationTime(tc.cookies)
			if tc.expectsErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectsTime, val)
			}
		})
	}
}
