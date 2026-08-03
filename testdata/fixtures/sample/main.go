// Command sample is a small HTTP service used as a deterministic analysis
// fixture for repomap's golden tests.
package main

import (
	"encoding/json"
	"net/http"

	"example.com/sample/models"
	"example.com/sample/service"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/widgets/{id}", getWidget)
	_ = http.ListenAndServe(":8080", mux)
}

// getWidget decodes a WidgetRequest, fetches the widget, and encodes it back.
func getWidget(w http.ResponseWriter, r *http.Request) {
	var req models.WidgetRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	widget := service.FetchWidget(req.ID)
	_ = json.NewEncoder(w).Encode(widget)
}
