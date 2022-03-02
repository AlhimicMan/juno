package main

import (
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/types/module"
	ibc "github.com/cosmos/ibc-go/v2/modules/core"
	"os"

	dewebapp "github.com/deweb-services/deweb/app"
	"github.com/forbole/juno/v2/cmd/parse"

	"github.com/forbole/juno/v2/modules/messages"
	"github.com/forbole/juno/v2/modules/registrar"

	"github.com/forbole/juno/v2/cmd"
)

func main() {
	// JunoConfig the runner
	config := cmd.NewConfig("juno").
		WithParseConfig(parse.NewConfig().
			WithEncodingConfigBuilder(parse.MakeEncodingConfig(getBasicManagers())).
			WithRegistrar(registrar.NewDefaultRegistrar(
				messages.CosmosMessageAddressesParser,
			)),
		)

	// Run the commands and panic on any error
	exec := cmd.BuildDefaultExecutor(config)
	err := exec.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// getBasicManagers returns the various basic managers that are used to register the encoding to
// support custom messages.
// This should be edited by custom implementations if needed.
func getBasicManagers() []module.BasicManager {
	return []module.BasicManager{
		simapp.ModuleBasics,
		module.NewBasicManager(
			ibc.AppModule{},
		),
		dewebapp.ModuleBasics,
	}
}
