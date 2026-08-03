// Package models holds the sample data types.
package models

// Widget is the primary domain type.
type Widget struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Labels []string `json:"labels,omitempty"`
	Owner  Owner    `json:"owner"`
	secret string
}

// Owner describes a widget owner.
type Owner struct {
	Email string `json:"email"`
	Team  string `json:"team,omitempty"`
}

// Display returns a human-readable owner label.
func (o Owner) Display() string { return o.Email }

// WidgetRequest is the request body for widget lookups.
type WidgetRequest struct {
	ID string `json:"id"`
}
