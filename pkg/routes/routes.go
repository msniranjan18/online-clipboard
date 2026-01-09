package routes

import (
	"net/http"

	"github.com/msniranjan18/online-clipboard/pkg/handlers"
	"github.com/msniranjan18/online-clipboard/pkg/hub"
)

// NewRouter sets up the application routes and returns a mux
func NewRouter(h *hub.Hub) *http.ServeMux {
	mux := http.NewServeMux()

	// 1. Static Assets
	// Important: The path "./static" is relative to where the binary runs (root)
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// 2. WebSocket Endpoint
	// The handler now lives in the handlers package
	mux.HandleFunc("/ws/", handlers.HandleWS(h))

	// 3. SPA Catch-All Route
	// This serves index.html for any path, allowing for dynamic room names
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	return mux
}
