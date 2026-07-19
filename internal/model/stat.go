package model

// Stats represents the current webhook processing counts in the database.
// The counts are limited to the past 7 days.
type Stats struct {
	Delivered int `json:"delivered"`
	Failed    int `json:"failed"`
	Retrying  int `json:"retrying"`
	Queued    int `json:"queued"`
}
