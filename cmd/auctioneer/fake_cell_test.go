package main_test

import (
	"net/http/httptest"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/auction/simulation/simulationrep"

	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_handlers"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/rata"
)

type FakeCell struct {
	cellID      string
	stack       string
	server      *httptest.Server
	heartbeater ifrit.Process

	SimulationRep auctiontypes.SimulationCellRep
}

func SpinUpFakeCell(cellID string, stack string) *FakeCell {
	fakeRep := &FakeCell{
		cellID: cellID,
		stack:  stack,
	}

	fakeRep.SpinUp()

	return fakeRep
}

func (f *FakeCell) LRPs() ([]auctiontypes.LRP, error) {
	state, err := f.SimulationRep.State()
	if err != nil {
		return nil, err
	}
	return state.LRPs, nil
}

func (f *FakeCell) Tasks() ([]auctiontypes.Task, error) {
	state, err := f.SimulationRep.State()
	if err != nil {
		return nil, err
	}
	return state.Tasks, nil
}

func (f *FakeCell) SpinUp() {
	//make a test-friendly AuctionRepDelegate using the auction package's SimulationRepDelegate
	f.SimulationRep = simulationrep.New(f.stack, "Z0", auctiontypes.Resources{
		DiskMB:     100,
		MemoryMB:   100,
		Containers: 100,
	})

	//spin up an http auction server
	logger := lager.NewLogger(f.cellID)
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))
	handlers := auction_http_handlers.New(f.SimulationRep, logger)
	router, err := rata.NewRouter(routes.Routes, handlers)
	Ω(err).ShouldNot(HaveOccurred())
	f.server = httptest.NewServer(router)

	//start hearbeating to ETCD (via global test bbs)
	f.heartbeater = ifrit.Invoke(bbs.NewCellHeartbeat(models.CellPresence{
		CellID:     f.cellID,
		Stack:      f.stack,
		RepAddress: f.server.URL,
	}, time.Second))
}

func (f *FakeCell) Stop() {
	f.server.Close()
	f.heartbeater.Signal(os.Interrupt)
	Eventually(f.heartbeater.Wait()).Should(Receive(BeNil()))
}
