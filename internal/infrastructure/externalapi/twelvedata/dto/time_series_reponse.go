package dto

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
