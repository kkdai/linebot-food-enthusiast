package main

func exchangeRate(currencyDate string,
	currencyFrom string, currencyTo string) map[string]any {
	// This hypothetical API returns a JSON such as:
	// {"base":"USD","date":"2024-04-17","rates":{"SEK": 0.091}}
	return map[string]any{
		"base":  currencyFrom,
		"date":  currencyDate,
		"rates": map[string]any{currencyTo: 0.091}}
}
