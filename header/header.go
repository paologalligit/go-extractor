package header

func GetHeaders(cookiesManager *CookiesManager) (map[string]string, error) {
	cookies, err := cookiesManager.GetCookies()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"cookie": cookies,
	}, nil
}
