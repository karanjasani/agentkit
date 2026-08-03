// Package service contains the sample business logic.
package service

import (
	"encoding/json"
	"net/http"

	"example.com/sample/models"
)

// widgetAPI is the upstream endpoint used to fetch widgets.
const widgetAPI = "https://api.example.com/widgets"

// FetchWidget retrieves a widget from the upstream service.
func FetchWidget(id string) models.Widget {
	resp, err := http.Get(widgetAPI)
	if err != nil {
		return models.Widget{ID: id}
	}
	defer resp.Body.Close()
	var w models.Widget
	_ = json.NewDecoder(resp.Body).Decode(&w)
	return w
}

// Helper is a trivial helper used to exercise caller analysis.
func Helper() string { return "help" }

// UsesHelper calls Helper.
func UsesHelper() string { return Helper() }
