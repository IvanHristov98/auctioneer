package auctionrunnerdelegate

import (
	"net/http"

	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_client"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

type AuctionRunnerDelegate struct {
	client *http.Client
	bbs    bbs.AuctioneerBBS
	logger lager.Logger
}

func New(client *http.Client, bbs bbs.AuctioneerBBS, logger lager.Logger) *AuctionRunnerDelegate {
	return &AuctionRunnerDelegate{
		client: client,
		bbs:    bbs,
		logger: logger,
	}
}

func (a *AuctionRunnerDelegate) FetchCellReps() (map[string]auctiontypes.CellRep, error) {
	cells, err := a.bbs.Cells()
	cellReps := map[string]auctiontypes.CellRep{}
	if err != nil {
		return cellReps, err
	}

	for _, cell := range cells {
		cellReps[cell.CellID] = auction_http_client.New(a.client, cell.CellID, cell.RepAddress, a.logger.Session(cell.RepAddress))
	}

	return cellReps, nil
}

func (a *AuctionRunnerDelegate) AuctionCompleted(results auctiontypes.AuctionResults) {
	for _, vol := range results.FailedVolumes {
		err := a.bbs.FailVolume(a.logger, vol.VolumeSetGuid, vol.Index, vol.PlacementError)
		if err != nil {
			a.logger.Error("failed-to-fail-volume", err, lager.Data{
				"volume":         vol,
				"auction-result": "failed",
			})
		}
	}
	for _, task := range results.FailedTasks {
		err := a.bbs.FailTask(a.logger, task.Identifier(), task.PlacementError)
		if err != nil {
			a.logger.Error("failed-to-fail-task", err, lager.Data{
				"task":           task,
				"auction-result": "failed",
			})
		}
	}

	for _, lrp := range results.FailedLRPs {
		err := a.bbs.FailActualLRP(a.logger, models.NewActualLRPKey(lrp.DesiredLRP.ProcessGuid, lrp.Index, lrp.DesiredLRP.Domain), lrp.PlacementError)
		if err != nil {
			a.logger.Error("failed-to-fail-LRP", err, lager.Data{
				"lrp":            lrp,
				"auction-result": "failed",
			})
		}
	}
}
