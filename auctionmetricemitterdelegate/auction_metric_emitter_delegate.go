package auctionmetricemitterdelegate

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auctioneer"
)

type auctionMetricEmitterDelegate struct{}

func New() auctionMetricEmitterDelegate {
	return auctionMetricEmitterDelegate{}
}

func (_ auctionMetricEmitterDelegate) FetchStatesCompleted(fetchStatesDuration time.Duration) {
	auctioneer.FetchStatesDuration.Send(fetchStatesDuration)
}

func (_ auctionMetricEmitterDelegate) AuctionCompleted(results auctiontypes.AuctionResults) {
	auctioneer.VolumeAuctionsStarted.Add(uint64(len(results.SuccessfulVolumes)))
	auctioneer.LRPAuctionsStarted.Add(uint64(len(results.SuccessfulLRPs)))
	auctioneer.TaskAuctionsStarted.Add(uint64(len(results.SuccessfulTasks)))

	auctioneer.VolumeAuctionsFailed.Add(uint64(len(results.FailedVolumes)))
	auctioneer.LRPAuctionsFailed.Add(uint64(len(results.FailedLRPs)))
	auctioneer.TaskAuctionsFailed.Add(uint64(len(results.FailedTasks)))
}
