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

package serve

import (
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunHandler(t *testing.T) {
	spec.Run(t, "RunHandler", testRunHandler, spec.Report(report.Terminal{}))
}

func testRunHandler(t *testing.T, describe spec.G, it spec.S) {
	var req *http.Request
	var recorder *httptest.ResponseRecorder
	var err error
	var mux *http.ServeMux

	describe("RunHandler", func() {
		it.Before(func() {
			recorder = httptest.NewRecorder()
			mux = http.NewServeMux()
			req, err = http.NewRequest("POST", "/run", nil)
			assert.NoError(t, err)

			mux.HandleFunc("/run", RunHandler)
			mux.ServeHTTP(recorder, req)
		})

		describe("headers", func() {
			it("has status 200 OK", func() {
				assert.Equal(t, http.StatusOK, recorder.Code)
			})

			it("sets the content-type to JSON", func() {
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			})
		})

		describe("response body", func() {
			it("contains a trace", func() {
				assert.JSONEq(t, `{"foo":  "bar"}`, recorder.Body.String())
			})
		})
	})
}