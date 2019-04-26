package serve

import (
	"context"
	"github.com/NYTimes/gziphandler"
	"log"
	"net/http"
	"time"
)

type SkenarioServer struct {
	srv *http.Server
	mux *http.ServeMux
}

func (ss *SkenarioServer) Serve() {
	ss.mux = http.NewServeMux()

	index := http.FileServer(http.Dir("pkg/serve"))
	ss.mux.Handle("/", index)

	runHandler := http.HandlerFunc(RunHandler)
	gzipRunHandler := gziphandler.GzipHandler(runHandler)
	ss.mux.Handle("/run", gzipRunHandler)

	ss.srv = &http.Server{
		Addr:    ":3000",
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
