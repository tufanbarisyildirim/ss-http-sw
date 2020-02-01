package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var webServer = NewWebServer("127.0.0.1:8090", "stats.txt")

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	webServer.Router.HandleFunc("/", mainHandlerFunc)

	go webServer.Start()

	<-sigs
	webServer.Stop()
}

func mainHandlerFunc(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(200)
	_, err := fmt.Fprintf(writer, "Total : %d, Avg: %.2f req/sec in %s", webServer.Stats.Total(), webServer.Stats.Avg(), webServer.Stats.WindowSize)
	if err != nil {
		webServer.Logger.Printf("error printing response %s\n", err)
	}
}
