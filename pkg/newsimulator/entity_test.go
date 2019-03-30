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

package newsimulator

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
)

func TestEntity(t *testing.T) {
	spec.Run(t, "Entity spec", testEntity, spec.Report(report.Terminal{}))
}

func testEntity(t *testing.T, describe spec.G, it spec.S) {
	var subject Entity

	it.Before(func() {
		subject = NewEntity("test entity name", "test entity kind")
	})

	it("creates an entity", func() {
		assert.Equal(t, subject.Name(), EntityName("test entity name"))
		assert.Equal(t, subject.Kind(), EntityKind("test entity kind"))
	})
}
