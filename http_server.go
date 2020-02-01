package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"
)

type WebServer struct {
	Addr   string
	Logger *log.Logger
	server *http.Server
	Router *http.ServeMux
	Stats  *StatsWorker
}

func NewWebServer(addr string, statsFile string) *WebServer {
	logger := log.New(os.Stdout, "webserver: ", log.LstdFlags)

	ws := &WebServer{
		Addr:   addr,
		Logger: logger,
		Router: http.NewServeMux(),
		Stats:  NewStatsWorker(statsFile, 500, 5*time.Minute),
	}

	return ws
}

func (ws *WebServer) Stop() {
	ws.Logger.Println("stopping web server gracefully")
	err := ws.server.Shutdown(context.Background())
	if err != nil {
		ws.Logger.Fatalf("error stopping http server %s", err)
	}
	ws.Stats.Stop()
}

func (ws *WebServer) middlewareStats(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws.Stats.CountRequest()
		ws.Logger.Printf("HTTP %s %s\n", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func (ws *WebServer) Start() {
	ws.Stats.Start()

	ws.server = &http.Server{
		Addr:     ws.Addr,
		Handler:  ws.middlewareStats(ws.Router),
		ErrorLog: ws.Logger,
	}

	log.Println("starting http server  http://127.0.0.1:8090")

	if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		ws.Logger.Fatalf("Could not listen on %s: %v\n", ws.Addr, err)
	}
}
