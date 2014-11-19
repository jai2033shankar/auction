package auction_http_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

type AuctionHTTPClient struct {
	client           *http.Client
	repAddress       auctiontypes.RepAddress
	requestGenerator *rata.RequestGenerator
	logger           lager.Logger
}

type Response struct {
	Body []byte
}

func New(client *http.Client, repAddress auctiontypes.RepAddress, logger lager.Logger) *AuctionHTTPClient {
	return &AuctionHTTPClient{
		client:           client,
		repAddress:       repAddress,
		requestGenerator: rata.NewRequestGenerator(repAddress.Address, routes.Routes),
		logger:           logger,
	}
}

func (c *AuctionHTTPClient) State() (auctiontypes.RepState, error) {
	logger := c.logger.Session("fetching-state", lager.Data{
		"rep": c.repAddress.RepGuid,
	})

	logger.Debug("requesting")

	req, err := c.requestGenerator.CreateRequest(routes.State, nil, nil)
	if err != nil {
		logger.Error("failed-to-create-request", err)
		return auctiontypes.RepState{}, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logger.Error("failed-to-perform-request", err)
		return auctiontypes.RepState{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("invalid-status-code", fmt.Errorf("%d", resp.StatusCode))
		return auctiontypes.RepState{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var state auctiontypes.RepState
	err = json.NewDecoder(resp.Body).Decode(&state)
	if err != nil {
		logger.Error("failed-to-decode-rep-state", err)
		return auctiontypes.RepState{}, err
	}

	logger.Debug("done")

	return state, nil
}

func (c *AuctionHTTPClient) Perform(work auctiontypes.Work) (auctiontypes.Work, error) {
	logger := c.logger.Session("sending-work", lager.Data{
		"rep":    c.repAddress.RepGuid,
		"starts": len(work.Starts),
		"stops":  len(work.Stops),
	})

	logger.Debug("requesting")

	body, err := json.Marshal(work)
	if err != nil {
		logger.Error("failed-to-marshal-work", err)
		return auctiontypes.Work{}, err
	}

	req, err := c.requestGenerator.CreateRequest(routes.Perform, nil, bytes.NewReader(body))
	if err != nil {
		logger.Error("failed-to-create-request", err)
		return auctiontypes.Work{}, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logger.Error("failed-to-perform-request", err)
		return auctiontypes.Work{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("invalid-status-code", fmt.Errorf("%d", resp.StatusCode))
		return auctiontypes.Work{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var failedWork auctiontypes.Work
	err = json.NewDecoder(resp.Body).Decode(&failedWork)
	if err != nil {
		logger.Error("failed-to-decode-failed-work", err)
		return auctiontypes.Work{}, err
	}

	logger.Debug("done")

	return failedWork, nil
}

func (c *AuctionHTTPClient) Reset() error {
	logger := c.logger.Session("SIM-reseting", lager.Data{
		"rep": c.repAddress.RepGuid,
	})

	logger.Debug("requesting")

	req, err := c.requestGenerator.CreateRequest(routes.Sim_Reset, nil, nil)
	if err != nil {
		logger.Error("failed-to-create-request", err)
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logger.Error("failed-to-perform-request", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("invalid-status-code", fmt.Errorf("%d", resp.StatusCode))
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	logger.Debug("done")
	return nil
}