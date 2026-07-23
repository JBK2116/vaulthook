package model

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// SearchType is the type of search to execute when querying for events
type SearchType string

const (
	LookUp SearchType = "lookup"
	Filter SearchType = "filter"
)

// SearchRequest contains options to be used when querying for events
type SearchRequest struct {
	Type             SearchType `json:"type"`
	WebhookID        *string    `json:"webhook_id"`
	EventID          *string    `json:"event_id"`
	Providers        []string   `json:"providers"`
	EventType        *string    `json:"event_type"`
	DeliveryStatuses []string   `json:"delivery_statuses"`
	ResponseCode     *int       `json:"response_code"`
	FromTime         *string    `json:"from_time"`
	ToTime           *string    `json:"to_time"`
	PayloadSearch    *string    `json:"payload_search"`
	HasRetries       bool       `json:"has_retries"`
	HasError         bool       `json:"has_error"`
	Offset           int        `json:"offset"`
	Limit            int        `json:"limit"`
}

// LookupOpts contains options to be used when executing lookup queries for events
type LookupOpts struct {
	WebhookID *string
	EventID   *string
	Offset    int
	Limit     int
}

// FilterOpts contains the options to be used when executing filter queries for events
type FilterOpts struct {
	Providers        []string
	EventType        *string
	DeliveryStatuses []string
	ResponseCode     *int
	FromTime         *time.Time
	ToTime           *time.Time
	PayloadSearch    *string
	HasRetries       bool
	HasError         bool
	Offset           int
	Limit            int
}

// SearchResponse wraps a page of webhook events together with a flag indicating
// whether more results are available.
type SearchResponse struct {
	Events  []Webhook `json:"events"`
	HasMore bool      `json:"has_more"`
}

// Validation errors for SearchPayload.
var (
	ErrMissingType       = errors.New("search type must be either lookup or filter")
	ErrMissingLookup     = errors.New("either webhook id or event id must be provided")
	ErrIDTooLong         = errors.New("webhook id and event id must be 255 characters or fewer")
	ErrNoProviders       = errors.New("at least one provider must be selected")
	ErrNoStatuses        = errors.New("at least one delivery status must be selected")
	ErrResponseCodeRange = errors.New("response code must be between 100 and 511")
	ErrDatePairRequired  = errors.New("both from time and to time must be set together")
	ErrInvalidTime       = errors.New("from time or to time is not a valid ISO 8601 datetime")
	ErrDateOrder         = errors.New("to time must be after from time")
	ErrSearchTooLong     = errors.New("payload search must be 255 characters or fewer")
	ErrEventTypeTooLong  = errors.New("event type must be 255 characters or fewer")
)

// Validate verifies that all fields are set to valid values
func (p *SearchRequest) Validate() error {
	switch p.Type {
	case LookUp:
		if err := p.validateLookup(); err != nil {
			return err
		}
		return nil
	case Filter:
		if err := p.validateFilter(); err != nil {
			return err
		}
		return nil
	default:
		return ErrMissingType
	}
}

// validateLookup ensures that the SearchPayload is valid for lookup queries
func (p *SearchRequest) validateLookup() error {
	id := strings.TrimSpace(derefStr(p.WebhookID))
	evID := strings.TrimSpace(derefStr(p.EventID))
	if id == "" && evID == "" {
		return ErrMissingLookup
	}
	if utf8.RuneCountInString(id) > 255 || utf8.RuneCountInString(evID) > 255 {
		return ErrIDTooLong
	}
	return nil
}

// validateFilter ensures that the SearchPayload is valid for filter queries
func (p *SearchRequest) validateFilter() error {
	if len(p.Providers) == 0 {
		return ErrNoProviders
	}
	if len(p.DeliveryStatuses) == 0 {
		return ErrNoStatuses
	}
	if p.ResponseCode != nil {
		if *p.ResponseCode < 100 || *p.ResponseCode > 511 {
			return ErrResponseCodeRange
		}
	}
	if (p.FromTime != nil && p.ToTime == nil) || (p.FromTime == nil && p.ToTime != nil) {
		return ErrDatePairRequired
	}
	if p.FromTime != nil && p.ToTime != nil {
		fromTime, fromErr := time.Parse(time.RFC3339, *p.FromTime)
		if fromErr != nil {
			return ErrInvalidTime
		}
		toTime, toErr := time.Parse(time.RFC3339, *p.ToTime)
		if toErr != nil {
			return ErrInvalidTime
		}
		if !fromTime.Before(toTime) {
			return ErrDateOrder
		}
	}
	if p.PayloadSearch != nil {
		strVal := derefStr(p.PayloadSearch)
		if utf8.RuneCountInString(strVal) > 255 {
			return ErrSearchTooLong
		}
	}
	if p.EventType != nil {
		strVal := derefStr(p.EventType)
		if utf8.RuneCountInString(strVal) > 255 {
			return ErrEventTypeTooLong
		}
	}
	return nil
}

// derefStr returns the string value of the provided pointer to string
func derefStr(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}
