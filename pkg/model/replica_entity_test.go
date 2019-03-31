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

package model

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/informers"
	v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	k8sfakes "k8s.io/client-go/kubernetes/fake"

	"knative-simulator/pkg/simulator"
)

func TestReplicaEntity(t *testing.T) {
	spec.Run(t, "Replica Entity", testReplicaEntity, spec.Report(report.Terminal{}))
}

func testReplicaEntity(t *testing.T, describe spec.G, it spec.S) {
	var subject ReplicaEntity
	var rawSubject *replicaEntity
	var fakeClient kubernetes.Interface
	var endpointsInformer v1.EndpointsInformer

	it.Before(func() {
		fakeClient = k8sfakes.NewSimpleClientset()
		informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)
		endpointsInformer = informerFactory.Core().V1().Endpoints()

		newEndpoints := &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Skenario Revision",
			},
			Subsets: []corev1.EndpointSubset{{
				Addresses: []corev1.EndpointAddress{},
			}},
		}

		fakeClient.CoreV1().Endpoints("skenario").Create(newEndpoints)
		endpointsInformer.Informer().GetIndexer().Add(newEndpoints)

		subject = NewReplicaEntity(fakeClient, endpointsInformer, "1.2.3.4")
		assert.NotNil(t, subject)

		rawSubject = subject.(*replicaEntity)
	})

	describe("NewReplicaEntity()", func() {
		var address corev1.EndpointAddress

		it.Before(func() {
			address = corev1.EndpointAddress{
				IP:       "1.2.3.4",
				Hostname: "Replica",
			}
		})

		it("sets a kubernetes client", func() {
			assert.Equal(t, fakeClient, rawSubject.kubernetesClient)
		})

		it("sets an endpoints informer", func() {
			assert.Equal(t, endpointsInformer, rawSubject.endpointsInformer)
		})

		it("sets its EndpointsAddress", func() {
			assert.Equal(t, address, rawSubject.endpointAddress)
		})
	})

	describe("Entity interface", func() {
		it("implements Name()", func() {
			assert.Equal(t, simulator.EntityName("Replica"), subject.Name())
		})

		it("implements Kind()", func() {
			assert.Equal(t, simulator.EntityKind("Replica"), subject.Kind())
		})
	})

	describe("Activate()", func() {
		var endpoints *corev1.Endpoints
		var epSubsets []corev1.EndpointSubset
		var epAddresses []corev1.EndpointAddress
		var err error

		it.Before(func() {
			subject.Activate()
			endpoints, err = fakeClient.CoreV1().Endpoints("skenario").Get("Skenario Revision", metav1.GetOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, endpoints)

			epSubsets = endpoints.Subsets
			assert.NotNil(t, epSubsets)

			epAddresses = epSubsets[0].Addresses
			assert.NotNil(t, epAddresses)
		})

		it("Adds its EndpointAddress to the Endpoints", func() {
			assert.Contains(t, epAddresses, rawSubject.endpointAddress)
		})
	})

	describe("Deactivate()", func() {
		var endpoints *corev1.Endpoints
		var epSubsets []corev1.EndpointSubset
		var epAddresses []corev1.EndpointAddress
		var err error

		it.Before(func() {
			subject.Activate()
			subject.Deactivate()

			endpoints, err = fakeClient.CoreV1().Endpoints("skenario").Get("Skenario Revision", metav1.GetOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, endpoints)

			epSubsets = endpoints.Subsets
			assert.NotNil(t, epSubsets)

			epAddresses = epSubsets[0].Addresses
			assert.NotNil(t, epAddresses)
		})

		it("Removes its EndpointAddress from the Endpoints", func() {
			assert.NotContains(t, epAddresses, rawSubject.endpointAddress)
		})
	})
}
