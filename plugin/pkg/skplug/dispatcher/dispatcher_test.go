package dispatcher

import (
	"github.com/josephburnett/sk-plugin/pkg/skplug"
	"github.com/josephburnett/sk-plugin/pkg/skplug/proto"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var subject Dispatcher
var rawSubject *dispatcher

func TestDispatcher(t *testing.T) {
	subject = GetInstance()
	subject.Init([]string{"../../../../build/plugin-fake"})
	rawSubject = subject.(*dispatcher)
	spec.Run(t, "Dispatcher", testDispatcher, spec.Report(report.Terminal{}))
}
func testDispatcher(t *testing.T, describe spec.G, it spec.S) {
	noErrorPartition := "noErrorPartition"
	errorPartition := "errorPartition"
	concurrentPartition1 := "concurrentPartition1"
	concurrentPartition2 := "concurrentPartition2"

	it("Dispatcher registered one plugin", func() {
		assert.Len(t, rawSubject.pluginsServers, 1)
		assert.Len(t, rawSubject.pluginsClients, 1)
		assert.Len(t, rawSubject.capabilityToPlugins[proto.Capability_EVENT], 1)
		assert.Len(t, rawSubject.capabilityToPlugins[proto.Capability_HORIZONTAL_RECOMMENDATION], 1)
		assert.Len(t, rawSubject.capabilityToPlugins[proto.Capability_VERTICAL_RECOMMENDATION], 1)
		assert.Len(t, rawSubject.capabilityToPlugins[proto.Capability_STAT], 1)
	})
	describe("Event", func() {
		it("create autoscaler with an existent partition, no error", func() {
			err := subject.Event(noErrorPartition, time.Now().UnixNano(), proto.EventType_CREATE, &skplug.Autoscaler{})
			assert.Nil(t, err)
		})

		it("create pod with an existent partition, no error", func() {
			err := subject.Event(noErrorPartition, time.Now().UnixNano(), proto.EventType_CREATE, &skplug.Pod{})
			assert.Nil(t, err)
		})

		it("delete pod with an existent partition, no error", func() {
			err := subject.Event(noErrorPartition, time.Now().UnixNano(), proto.EventType_DELETE, &skplug.Pod{})
			assert.Nil(t, err)
		})

		it("delete autoscaler with an existent partition, no error", func() {
			err := subject.Event(noErrorPartition, time.Now().UnixNano(), proto.EventType_DELETE, &skplug.Autoscaler{})
			assert.Nil(t, err)
		})
		it("create autoscaler with non-existent partition, produce an error", func() {
			err := subject.Event(errorPartition, time.Now().UnixNano(), proto.EventType_CREATE, &skplug.Autoscaler{})
			assert.NotNil(t, err)
		})

		it("create pod with non-existent partition, produce an error", func() {
			err := subject.Event(errorPartition, time.Now().UnixNano(), proto.EventType_CREATE, &skplug.Pod{})
			assert.NotNil(t, err)
		})

		it("delete pod with non-existent partition, produce an error", func() {
			err := subject.Event(errorPartition, time.Now().UnixNano(), proto.EventType_DELETE, &skplug.Pod{})
			assert.NotNil(t, err)
		})

		it("delete autoscaler with non-existent partition, produce an error", func() {
			err := subject.Event(errorPartition, time.Now().UnixNano(), proto.EventType_DELETE, &skplug.Autoscaler{})
			assert.NotNil(t, err)
		})
	})
	describe("Stat", func() {
		it("call with an existent partition, no error", func() {
			err := subject.Stat(noErrorPartition, []*proto.Stat{})
			assert.Nil(t, err)
		})
		it("call with non-existent partition, produce an error", func() {
			err := subject.Stat(errorPartition, []*proto.Stat{})
			assert.NotNil(t, err)
		})
	})

	describe("HorizontalRecommendation", func() {
		var rec int32
		var err error
		it("case with two concurrent partitions", func() {
			rec, err = subject.HorizontalRecommendation(concurrentPartition1, time.Now().UnixNano())
			assert.Nil(t, err)
			assert.Equal(t, rec, int32(1))

			rec, err = subject.HorizontalRecommendation(concurrentPartition2, time.Now().UnixNano())
			assert.Nil(t, err)
			assert.Equal(t, rec, int32(2))
		})
		it("call with an existent partition, no error", func() {
			rec, err = subject.HorizontalRecommendation(noErrorPartition, time.Now().UnixNano())
			assert.Nil(t, err)
		})
		it("call with non-existent partition, produce an error", func() {
			rec, err = subject.HorizontalRecommendation(errorPartition, time.Now().UnixNano())
			assert.NotNil(t, err)
		})
	})

	describe("VerticalRecommendation", func() {
		var rec []*proto.RecommendedPodResources
		var err error
		it("case with two concurrent partitions", func() {
			rec, err = subject.VerticalRecommendation(concurrentPartition1, time.Now().UnixNano())
			assert.Nil(t, err)
			assert.Len(t, rec, 1)
			assert.Equal(t, rec[0].Target, int64(50))
			assert.Equal(t, rec[0].UpperBound, int64(100))
			assert.Equal(t, rec[0].LowerBound, int64(1))
			assert.Equal(t, rec[0].PodName, "Pod1")

			rec, err = subject.VerticalRecommendation(concurrentPartition2, time.Now().UnixNano())
			assert.Nil(t, err)
			assert.Len(t, rec, 1)
			assert.Equal(t, rec[0].Target, int64(100))
			assert.Equal(t, rec[0].UpperBound, int64(200))
			assert.Equal(t, rec[0].LowerBound, int64(100))
			assert.Equal(t, rec[0].PodName, "Pod1")
		})
		it("call with an existent partition, no error", func() {
			rec, err = subject.VerticalRecommendation(noErrorPartition, time.Now().UnixNano())
			assert.Nil(t, err)
		})
		it("call with non-existent partition, produce an error", func() {
			rec, err = subject.VerticalRecommendation(errorPartition, time.Now().UnixNano())
			assert.NotNil(t, err)
		})
	})
}
