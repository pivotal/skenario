package serve

import (
	"github.com/NYTimes/gziphandler"
	"log"
	"net/http"
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

	log.Println("Listening ...")
	log.Fatal(ss.srv.ListenAndServe())
}

