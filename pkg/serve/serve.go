package serve

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type SkenarioServer struct {
	IndexRoot string
	srv       *http.Server
}

func (ss *SkenarioServer) Serve() {

	router := chi.NewRouter()
	router.Use(middleware.NoCache)
	router.Use(middleware.DefaultCompress)

	router.Mount("/debug", middleware.Profiler())
	router.Mount("/", http.FileServer(http.Dir(ss.IndexRoot)))
	router.HandleFunc("/run", RunHandler)

	ss.srv = &http.Server{
		Addr:    "0.0.0.0:3000",
		Handler: router,
	}

	go func() {
		log.Println("Listening ...")
		log.Fatal(ss.srv.ListenAndServe())
	}()
}

func (ss *SkenarioServer) Shutdown() {
	log.Println("Shutting down ...")

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	err := ss.srv.Shutdown(ctx)
	if err != nil {
		log.Fatalf("shutdown error: %s", err.Error())
	}

	log.Println("Done.")
}
