package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/glifio/go-pools/sdk"
	"github.com/glifio/go-pools/types"
	"github.com/glifio/go-pools/util"
	"github.com/xuri/excelize/v2"
)

var minerStrs = []string{
	"f03137648",
	"f033463",
}

var DIAL_ADDR = ""
var TOKEN = ""

func main() {
	ctx := context.Background()

	sdk, err := sdk.New(ctx, big.NewInt(314), types.Extern{
		LotusDialAddr: DIAL_ADDR,
		LotusToken:    TOKEN,
	})
	if err != nil {
		log.Fatal("node connection error: ", err)
	}

	miners := make([]address.Address, len(minerStrs))
	for i, idStr := range minerStrs {
		addr, err := address.NewFromString(idStr)
		if err != nil {
			log.Fatal("Miner address not valid", idStr)
			panic(err)
		}
		miners[i] = addr
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			panicWithMsg("Error closing file", err)
		}
	}()

	lapi, closer, err := sdk.Extern().ConnectLotusClient()
	if err != nil {
		log.Println("error connecting to lotus client:", err)
		return
	}
	defer closer()

	// get tipset height
	tipset, err := lapi.ChainHead(ctx)
	if err != nil {
		panic(err)
	}

	dateTime := util.EpochHeightToTimestamp(big.NewInt(int64(tipset.Height())), big.NewInt(314)).Format(time.RFC3339)

	writeCell(f, "A1", "Miner Evaluation")
	writeCell(f, "A2", "Compute epoch")
	writeCell(f, "B2", tipset.Height().String())
	writeCell(f, "C2", dateTime)

	writeCell(f, "A4", "Miner")
	writeCell(f, "B4", "IPledge")
	writeCell(f, "C4", "TF")
	writeCell(f, "D4", "TF/IPledge")
	writeCell(f, "E4", "QAP")
	writeCell(f, "F4", "RBP")
	writeCell(f, "G4", "SSIZE")

	for i, miner := range miners {
		offset := 5
		fmt.Println("Simulating miner liquidation: ", miner)
		ts, err := terminateMiner(ctx, lapi, miner, uint64(tipset.Height()))
		if err != nil {
			panic(err)
		}
		writeCell(f, "A"+fmt.Sprint(i+offset), miner.String())
		writeFILVal(f, "B"+fmt.Sprint(i+offset), ts.InitialPledge)
		writeFILVal(f, "C"+fmt.Sprint(i+offset), ts.SectorStats.TerminationPenalty)
		ratioFloat := new(big.Rat).SetFrac(ts.SectorStats.TerminationPenalty, ts.InitialPledge)
		ratio := fmt.Sprintf("%s%%", new(big.Rat).Mul(ratioFloat, big.NewRat(100, 1)).FloatString(2))
		fmt.Println("Termination complete: ", miner)
		fmt.Println("Initial pledge: ", ts.InitialPledge)
		fmt.Println("Termination fee: ", ts.SectorStats.TerminationPenalty)
		fmt.Println("Termination fee / Initial pledge: ", ratio)
		writeCell(f, "D"+fmt.Sprint(i+offset), ratio)

		pow, err := lapi.StateMinerPower(ctx, miner, tipset.Key())
		if err != nil {
			panic(err)
		}

		writeCell(f, "E"+fmt.Sprint(i+offset), fmt.Sprintf("%v", pow.MinerPower.QualityAdjPower.Int))
		writeCell(f, "F"+fmt.Sprint(i+offset), fmt.Sprintf("%v", pow.MinerPower.RawBytePower.Int))
		writeCell(f, "G"+fmt.Sprint(i+offset), fmt.Sprintf("%v", ts.MinerInfo.SectorSize))
	}

	if err := f.SaveAs("minereval.xlsx"); err != nil {
		panicWithMsg("Error saving file", err)
	}
}

func writeCell(f *excelize.File, cell, value string) {
	if err := f.SetCellValue("Sheet1", cell, value); err != nil {
		panic(err)
	}
}

func writeFILVal(f *excelize.File, cell string, value *big.Int) {
	filVal := util.ToFIL(value)
	writeCell(f, cell, fmt.Sprintf("%0.04f", filVal))
}

func panicWithMsg(msg string, err error) {
	panic(fmt.Sprintf("%s: %s", msg, err))
}
