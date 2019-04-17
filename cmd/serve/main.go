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
 *
 */

package main

import (
	"log"
	"net/http"

	"github.com/NYTimes/gziphandler"

	"skenario/pkg/serve"
)

func main() {
	index := http.FileServer(http.Dir("pkg/serve"))
	http.Handle("/", index)

	runHandler :=	http.HandlerFunc(serve.RunHandler)
	gzipRunHandler := gziphandler.GzipHandler(runHandler)
	http.Handle("/run", gzipRunHandler)

	log.Println("Listening ...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
