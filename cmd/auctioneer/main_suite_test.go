package main_test

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"code.cloudfoundry.org/auctioneer"
	"code.cloudfoundry.org/bbs"
	bbstestrunner "code.cloudfoundry.org/bbs/cmd/bbs/testrunner"
	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/bbs/test_helpers"
	"code.cloudfoundry.org/bbs/test_helpers/sqlrunner"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/consuladapter/consulrunner"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"testing"
	"time"
)

var (
	auctioneerProcess ifrit.Process

	auctioneerPath string

	dotNetStack           = "dot-net"
	dotNetRootFSURL       = models.PreloadedRootFS(dotNetStack)
	linuxStack            = "linux"
	linuxRootFSURL        = models.PreloadedRootFS(linuxStack)
	dotNetCell, linuxCell *FakeCell

	auctioneerServerPort int
	auctioneerLocation   string
	runner               *ginkgomon.Runner
	auctioneerArgs       []string

	consulRunner *consulrunner.ClusterRunner
	consulClient consuladapter.Client

	auctioneerClient auctioneer.Client

	bbsArgs    bbstestrunner.Args
	bbsBinPath string
	bbsURL     *url.URL
	bbsRunner  *ginkgomon.Runner
	bbsProcess ifrit.Process
	bbsClient  bbs.InternalClient

	sqlProcess ifrit.Process
	sqlRunner  sqlrunner.SQLRunner

	logger lager.Logger
)

func TestAuctioneer(t *testing.T) {
	// these integration tests can take a bit, especially under load;
	// 1 second is too harsh
	SetDefaultEventuallyTimeout(10 * time.Second)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Auctioneer Cmd Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	bbsConfig, err := gexec.Build("code.cloudfoundry.org/bbs/cmd/bbs", "-race")
	Expect(err).NotTo(HaveOccurred())

	compiledAuctioneerPath, err := gexec.Build("code.cloudfoundry.org/auctioneer/cmd/auctioneer", "-race")
	Expect(err).NotTo(HaveOccurred())
	return []byte(strings.Join([]string{compiledAuctioneerPath, bbsConfig}, ","))
}, func(pathsByte []byte) {
	path := string(pathsByte)
	compiledAuctioneerPath := strings.Split(path, ",")[0]
	bbsBinPath = strings.Split(path, ",")[1]

	bbsBinPath = strings.Split(path, ",")[1]
	auctioneerPath = string(compiledAuctioneerPath)

	auctioneerServerPort = 1800 + GinkgoParallelNode()
	auctioneerLocation = fmt.Sprintf("127.0.0.1:%d", auctioneerServerPort)

	dbName := fmt.Sprintf("diego_%d", GinkgoParallelNode())
	sqlRunner = test_helpers.NewSQLRunner(dbName)
	sqlProcess = ginkgomon.Invoke(sqlRunner)

	consulRunner = consulrunner.NewClusterRunner(
		9001+GinkgoParallelNode()*consulrunner.PortOffsetLength,
		1,
		"http",
	)

	auctioneerClient = auctioneer.NewClient("http://" + auctioneerLocation)

	logger = lagertest.NewTestLogger("test")

	consulRunner.Start()
	consulRunner.WaitUntilReady()

	bbsPort := 13000 + GinkgoParallelNode()*2
	healthPort := bbsPort + 1
	bbsAddress := fmt.Sprintf("127.0.0.1:%d", bbsPort)
	healthAddress := fmt.Sprintf("127.0.0.1:%d", healthPort)

	bbsURL = &url.URL{
		Scheme: "http",
		Host:   bbsAddress,
	}

	bbsClient = bbs.NewClient(bbsURL.String())

	bbsArgs = bbstestrunner.Args{
		Address:           bbsAddress,
		AdvertiseURL:      bbsURL.String(),
		AuctioneerAddress: "http://" + auctioneerLocation,
		ConsulCluster:     consulRunner.ConsulCluster(),
		HealthAddress:     healthAddress,

		EncryptionKeys:           []string{"label:key"},
		ActiveKeyLabel:           "label",
		DatabaseDriver:           sqlRunner.DriverName(),
		DatabaseConnectionString: sqlRunner.ConnectionString(),
	}
})

var _ = BeforeEach(func() {
	auctioneerArgs = []string{}

	consulRunner.Reset()

	bbsRunner = bbstestrunner.New(bbsBinPath, bbsArgs)
	bbsProcess = ginkgomon.Invoke(bbsRunner)

	consulClient = consulRunner.NewClient()

	serviceClient := bbs.NewServiceClient(consulClient, clock.NewClock())

	dotNetCell = SpinUpFakeCell(serviceClient, "dot-net-cell", "", dotNetStack)
	linuxCell = SpinUpFakeCell(serviceClient, "linux-cell", "", linuxStack)
})

var _ = JustBeforeEach(func() {
	auctioneerArgs = append([]string{
		"-bbsAddress", bbsURL.String(),
		"-listenAddr", auctioneerLocation,
		"-lockRetryInterval", "1s",
		"-consulCluster", consulRunner.ConsulCluster(),
	}, auctioneerArgs...)

	runner = ginkgomon.New(ginkgomon.Config{
		Name: "auctioneer",
		Command: exec.Command(
			auctioneerPath,
			auctioneerArgs...,
		),
		StartCheck: "auctioneer.started",
	})
})

var _ = AfterEach(func() {
	ginkgomon.Kill(auctioneerProcess)
	ginkgomon.Kill(bbsProcess)
	dotNetCell.Stop()
	linuxCell.Stop()

	sqlRunner.Reset()
})

var _ = SynchronizedAfterSuite(func() {
	if consulRunner != nil {
		consulRunner.Stop()
	}

	ginkgomon.Kill(sqlProcess)
}, func() {
	gexec.CleanupBuildArtifacts()
})
