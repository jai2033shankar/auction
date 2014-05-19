package auctiondistributor

import (
	"fmt"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/cloudfoundry-incubator/auction/auctioneer"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/simulation/visualization"
)

type AuctionCommunicator func(auctiontypes.AuctionRequest) auctiontypes.AuctionResult

type AuctionDistributor struct {
	client        auctiontypes.TestRepPoolClient
	communicator  AuctionCommunicator
	maxConcurrent int
}

func NewInProcessAuctionDistributor(client auctiontypes.TestRepPoolClient, maxConcurrent int) *AuctionDistributor {
	return &AuctionDistributor{
		client:        client,
		maxConcurrent: maxConcurrent,
		communicator: func(auctionRequest auctiontypes.AuctionRequest) auctiontypes.AuctionResult {
			return auctioneer.Auction(client, auctionRequest)
		},
	}
}

func NewRemoteAuctionDistributor(hosts []string, client auctiontypes.TestRepPoolClient, maxConcurrent int) *AuctionDistributor {
	return &AuctionDistributor{
		client:        client,
		maxConcurrent: maxConcurrent,
		communicator:  newHttpRemoteAuctions(hosts).RemoteAuction,
	}
}

func (ad *AuctionDistributor) HoldAuctionsFor(instances []auctiontypes.Instance, representatives []string, rules auctiontypes.AuctionRules) *visualization.Report {
	fmt.Printf("\nStarting Auctions\n\n")
	bar := pb.StartNew(len(instances))

	t := time.Now()
	semaphore := make(chan bool, ad.maxConcurrent)
	c := make(chan auctiontypes.AuctionResult)
	for _, inst := range instances {
		go func(inst auctiontypes.Instance) {
			semaphore <- true
			result := ad.communicator(auctiontypes.AuctionRequest{
				Instance: inst,
				RepGuids: representatives,
				Rules:    rules,
			})
			result.Duration = time.Since(t)
			c <- result
			<-semaphore
		}(inst)
	}

	results := []auctiontypes.AuctionResult{}
	for _ = range instances {
		results = append(results, <-c)
		bar.Increment()
	}

	bar.Finish()

	duration := time.Since(t)
	report := &visualization.Report{
		RepGuids:        representatives,
		AuctionResults:  results,
		InstancesByRep:  visualization.FetchAndSortInstances(ad.client, representatives),
		AuctionDuration: duration,
	}

	return report
}
