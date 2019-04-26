package serve

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
)

type SkenarioServer struct {
	IndexRoot string
	srv       *http.Server
	mux       *http.ServeMux
}

func (ss *SkenarioServer) Serve() {
	ss.mux = http.NewServeMux()

	indexHandler := http.FileServer(http.Dir(ss.IndexRoot))
	indexHandler = middleware.NoCache(indexHandler)
	indexHandler = middleware.DefaultCompress(indexHandler)
	ss.mux.Handle("/", indexHandler)

	runHandler := http.HandlerFunc(RunHandler)
	gzipRunHandler := middleware.DefaultCompress(runHandler)
	ss.mux.Handle("/run", gzipRunHandler)

	ss.srv = &http.Server{
		Addr:    "0.0.0.0:3000",
		Handler: ss.mux,
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
