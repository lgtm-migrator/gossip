package main

import (
	"log"
	"net/http"
	"regexp"

	_ "github.com/joho/godotenv/autoload"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
)

func serve() {
	goji.DefaultMux.Compile()
	// Install our handler at the root of the standard net/http default mux.
	// This allows packages like expvar to continue working as expected.
	http.Handle("/", goji.DefaultMux)

	listener := bind.Socket(bind.Sniff())
	log.Println("Starting Goji on", listener.Addr())

	graceful.HandleSignals()
	bind.Ready()
	graceful.PreHook(func() { log.Printf("Goji received signal, gracefully stopping") })
	graceful.PostHook(func() {
		log.Printf("Goji stopped")
		log.Printf("Shutting down the server")
		handler.DB.Close()
		log.Printf("Database shut down. Terminating the process.")
	})

	err := graceful.Serve(listener, http.DefaultServeMux)

	if err != nil {
		log.Fatal(err)
	}

	graceful.Wait()
}

func main() {
	goji.Use(TokenAuthHandler)
	goji.Get("/", handler.SayHello)
	pattern := regexp.MustCompile(`^/api/messages/(?P<id>[0-9]+)$`)
	goji.Get(pattern, handler.FindMessageById)
	goji.Get("/api/messages/latest", handler.FetchLatestMessages)
	goji.Post("/api/messages/log", handler.StoreMessage)
	serve()
}