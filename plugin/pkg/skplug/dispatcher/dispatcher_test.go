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
	var partition string
	partition = "0"

	it("Dispatcher registered one plugin", func() {
		assert.Len(t, rawSubject.pluginsServers, 1)
		assert.Len(t, rawSubject.pluginsClients, 1)
		assert.Len(t, rawSubject.capabilityToPlugins[proto.Capability_EVENT], 1)
		assert.Len(t, rawSubject.capabilityToPlugins[proto.Capability_HORIZONTAL_RECOMMENDATION], 1)
		assert.Len(t, rawSubject.capabilityToPlugins[proto.Capability_VERTICAL_RECOMMENDATION], 1)
		assert.Len(t, rawSubject.capabilityToPlugins[proto.Capability_STAT], 1)
	})
	describe("Event", func() {
		it("create autoscaler", func() {
			err := subject.Event(partition, time.Now().UnixNano(), proto.EventType_CREATE, &skplug.Autoscaler{})
			assert.Nil(t, err)
		})

		it("create pod", func() {
			err := subject.Event(partition, time.Now().UnixNano(), proto.EventType_CREATE, &skplug.Pod{})
			assert.Nil(t, err)
		})

		it("delete pod", func() {
			err := subject.Event(partition, time.Now().UnixNano(), proto.EventType_DELETE, &skplug.Pod{})
			assert.Nil(t, err)
		})

		it("delete autoscaler", func() {
			err := subject.Event(partition, time.Now().UnixNano(), proto.EventType_DELETE, &skplug.Autoscaler{})
			assert.Nil(t, err)
		})
	})
	it("Stat", func() {
		err := subject.Stat(partition, []*proto.Stat{})
		assert.Nil(t, err)
	})

	it("HorizontalRecommendation", func() {
		var rec int32
		var err error

		rec, err = subject.HorizontalRecommendation("1", time.Now().UnixNano())
		assert.Nil(t, err)
		assert.Equal(t, rec, int32(1))

		rec, err = subject.HorizontalRecommendation("2", time.Now().UnixNano())
		assert.Nil(t, err)
		assert.Equal(t, rec, int32(2))
	})

	it("VerticalRecommendation", func() {
		var rec []*proto.RecommendedPodResources
		var err error

		rec, err = subject.VerticalRecommendation(partition, time.Now().UnixNano())
		assert.Nil(t, err)
		assert.Len(t, rec, 1)
		assert.Equal(t, rec[0].Target, int64(50))
		assert.Equal(t, rec[0].UpperBound, int64(100))
		assert.Equal(t, rec[0].LowerBound, int64(1))
		assert.Equal(t, rec[0].PodName, "Pod1")

	})
}
