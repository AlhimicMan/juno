package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/angelorc/desmos-parser/client"
	"github.com/angelorc/desmos-parser/config"
	"github.com/angelorc/desmos-parser/db/postgresql"
	"github.com/angelorc/desmos-parser/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/pkg/errors"
)

var configPath string

func main() {
	flag.StringVar(&configPath, "config", "", "Configuration file")
	flag.Parse()

	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		panic(err)
	}

	cp, err := client.New(cfg.RPCNode, cfg.ClientNode)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to start RPC client"))
	}

	defer cp.Stop() // nolint: errcheck

	codec := simapp.MakeCodec()
	db, err := postgresql.Builder(*cfg, codec)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to open database connection"))
	}

	defer db.Sql.Close()

	if err := db.Sql.Ping(); err != nil {
		log.Fatal(errors.Wrap(err, "failed to ping database"))
	}

	lastHeight, err := db.LastBlockHeight()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to get latest block height"))
	}

	for i := int64(2); i < lastHeight; i++ {
		block, err := cp.Block(i)
		if err != nil {
			log.Printf("failed to get block %d: %s", i, err)
			continue
		}

		for _, tmTx := range block.Block.Txs {
			txHash := fmt.Sprintf("%X", tmTx.Hash())

			tx, err := cp.Tx(txHash)
			if err != nil {
				log.Printf("failed to get tx %s: %s", txHash, err)
				continue
			}

			if i%10 == 0 {
				log.Printf("migrated transaction %s\n", txHash)
			}

			convTx, err := types.NewTx(tx)
			if err != nil {
				panic(err)
			}

			if err := db.SaveTx(*convTx); err != nil {
				log.Printf("failed to persist transaction %s: %s", txHash, err)
			}
		}
	}
}
