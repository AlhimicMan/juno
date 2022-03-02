package parse

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types/module"
	"os"

	"github.com/forbole/juno/v2/types/config"

	"github.com/spf13/cobra"

	"github.com/forbole/juno/v2/types"
)

// ReadConfig parses the configuration file for the executable having the give name using
// the provided configuration parser
func ReadConfig(cfg *Config) types.CobraCmdFunc {
	return func(_ *cobra.Command, _ []string) error {
		file := config.GetConfigFilePath()

		// Make sure the path exists
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("config file does not exist. Make sure you have run the init command")
		}

		// Read the config
		cfg, err := config.Read(file, cfg.GetConfigParser())
		if err != nil {
			return err
		}

		// Set the global configuration
		config.Cfg = cfg
		return nil
	}
}

// MakeEncodingConfig creates an EncodingConfig to properly handle all the messages
func MakeEncodingConfig(managers []module.BasicManager) func() params.EncodingConfig {
	return func() params.EncodingConfig {
		encodingConfig := params.MakeTestEncodingConfig()
		std.RegisterLegacyAminoCodec(encodingConfig.Amino)
		std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
		manager := mergeBasicManagers(managers)
		manager.RegisterLegacyAminoCodec(encodingConfig.Amino)
		manager.RegisterInterfaces(encodingConfig.InterfaceRegistry)
		return encodingConfig
	}
}

// mergeBasicManagers merges the given managers into a single module.BasicManager
func mergeBasicManagers(managers []module.BasicManager) module.BasicManager {
	var union = module.BasicManager{}
	for _, manager := range managers {
		for k, v := range manager {
			union[k] = v
		}
	}
	return union
}
