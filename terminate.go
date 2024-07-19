package main

import (
	"context"
	"fmt"
	"log"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/glifio/go-pools/terminate"
)

func terminateMiner(ctx context.Context, lapi *api.FullNodeStruct, miner address.Address, epoch uint64) (*terminate.PreviewTerminateSectorsReturn, error) {
	errorCh := make(chan error)
	resultCh := make(chan *terminate.PreviewTerminateSectorsReturn)
	epochStr := fmt.Sprintf("@%v", epoch)
	go terminate.PreviewTerminateSectors(ctx, lapi, miner, epochStr, 0, 0, 0,
		false, false, false, 0, errorCh, nil, resultCh)

	for {
		select {
		case result := <-resultCh:
			return result, nil
		case err := <-errorCh:
			log.Fatal(err)
		}
	}
}
