package auctionmetricemitterdelegate_test

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auctioneer/auctionmetricemitterdelegate"
	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/rep"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auction Metric Emitter Delegate", func() {
	var delegate auctiontypes.AuctionMetricEmitterDelegate
	var metricSender *fake.FakeMetricSender

	BeforeEach(func() {
		metricSender = fake.NewFakeMetricSender()
		metrics.Initialize(metricSender, nil)

		delegate = auctionmetricemitterdelegate.New()
	})

	Describe("AuctionCompleted", func() {
		It("should adjust the metric counters", func() {
			resource := rep.NewResource(10, 10, "linux")
			delegate.AuctionCompleted(auctiontypes.AuctionResults{
				SuccessfulLRPs: []auctiontypes.LRPAuction{
					{
						LRP: rep.NewLRP(models.NewActualLRPKey("successful-start", 0, "domain"), resource),
					},
				},
				SuccessfulTasks: []auctiontypes.TaskAuction{
					{
						Task: rep.NewTask("successful-task", resource),
					},
				},
				FailedLRPs: []auctiontypes.LRPAuction{
					{
						LRP:           rep.NewLRP(models.NewActualLRPKey("insufficient-capacity", 0, "domain"), resource),
						AuctionRecord: auctiontypes.AuctionRecord{PlacementError: rep.ErrorInsufficientResources.Error()},
					},
					{
						LRP:           rep.NewLRP(models.NewActualLRPKey("incompatible-stacks", 0, "domain"), resource),
						AuctionRecord: auctiontypes.AuctionRecord{PlacementError: auctiontypes.ErrorCellMismatch.Error()},
					},
				},
				FailedTasks: []auctiontypes.TaskAuction{
					{
						Task:          rep.NewTask("failed-task", resource),
						AuctionRecord: auctiontypes.AuctionRecord{PlacementError: rep.ErrorInsufficientResources.Error()},
					},
				},
			})

			Expect(metricSender.GetCounter("AuctioneerLRPAuctionsStarted")).To(BeNumerically("==", 1))
			Expect(metricSender.GetCounter("AuctioneerTaskAuctionsStarted")).To(BeNumerically("==", 1))
			Expect(metricSender.GetCounter("AuctioneerLRPAuctionsFailed")).To(BeNumerically("==", 2))
			Expect(metricSender.GetCounter("AuctioneerTaskAuctionsFailed")).To(BeNumerically("==", 1))
		})
	})

	Describe("FetchStatesCompleted", func() {
		It("should adjust the metric counters", func() {
			delegate.FetchStatesCompleted(1 * time.Second)

			sentMetric := metricSender.GetValue("AuctioneerFetchStatesDuration")
			Expect(sentMetric.Value).To(Equal(1e+09))
			Expect(sentMetric.Unit).To(Equal("nanos"))
		})
	})
})
