/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package serve

import (
	"context"
	"github.com/josephburnett/sk-plugin/pkg/skplug/plugindispatcher"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type SkenarioServer struct {
	IndexRoot string
	srv       *http.Server
}

func (ss *SkenarioServer) Serve() {
	plugindispatcher.Init(os.Args[1:])
	router := chi.NewRouter()
	router.Use(middleware.NoCache)
	router.Use(middleware.DefaultCompress)
	router.Use(middleware.Logger)

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

	log.Println("Shutting down autoscaler plugins")
	plugindispatcher.Shutdown()

	log.Println("Done.")
}
