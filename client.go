package auctioneer

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/cfhttp"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter -o auctioneerfakes/fake_client.go . Client
type Client interface {
	RequestLRPAuctions(lrpStart []*LRPStartRequest) error
	RequestTaskAuctions(tasks []*TaskStartRequest) error
}

type auctioneerClient struct {
	httpClient *http.Client
	url        string
}

func NewClient(auctioneerURL string) Client {
	return &auctioneerClient{
		httpClient: cfhttp.NewClient(),
		url:        auctioneerURL,
	}
}

func NewSecureClient(url, caFile, certFile, keyFile string, clientSessionCacheSize, maxIdleConnsPerHost int) Client {
	client := &auctioneerClient{
		httpClient: cfhttp.NewClient(),
		url:        url,
	}
	tlsConfig, err := cfhttp.NewTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		return nil
	}
	tlsConfig.ClientSessionCache = tls.NewLRUClientSessionCache(clientSessionCacheSize)
	tlsConfig.InsecureSkipVerify = false

	if tr, ok := client.httpClient.Transport.(*http.Transport); ok {
		tr.TLSClientConfig = tlsConfig
		tr.MaxIdleConnsPerHost = maxIdleConnsPerHost
	} else {
		return nil
	}

	return client
}

// func NewSecureClient(url, caFile, certFile, keyFile string, clientSessionCacheSize, maxIdleConnsPerHost int) (InternalClient, error) {
// 	return newSecureClient(url, caFile, certFile, keyFile, clientSessionCacheSize, maxIdleConnsPerHost, false)
// }

func (c *auctioneerClient) RequestLRPAuctions(lrpStarts []*LRPStartRequest) error {
	reqGen := rata.NewRequestGenerator(c.url, Routes)

	payload, err := json.Marshal(lrpStarts)
	if err != nil {
		return err
	}

	req, err := reqGen.CreateRequest(CreateLRPAuctionsRoute, rata.Params{}, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("http error: status code %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return nil
}

func (c *auctioneerClient) RequestTaskAuctions(tasks []*TaskStartRequest) error {
	reqGen := rata.NewRequestGenerator(c.url, Routes)

	payload, err := json.Marshal(tasks)
	if err != nil {
		return err
	}

	req, err := reqGen.CreateRequest(CreateTaskAuctionsRoute, rata.Params{}, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("http error: status code %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return nil
}
