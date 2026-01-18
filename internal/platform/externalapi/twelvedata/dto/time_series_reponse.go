// Package dto defines data transfer objects for the Twelve Data API responses.
package dto

// TimeSeriesResponse represents the JSON response from the Twelve Data time_series endpoint.
type TimeSeriesResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	Symbol   string `json:"symbol"`
	Interval string `json:"interval"`
	Values   []struct {
		Datetime string `json:"datetime"`
		Open     string `json:"open"`
		High     string `json:"high"`
		Low      string `json:"low"`
		Close    string `json:"close"`
		Volume   string `json:"volume"`
	} `json:"values"`
}
