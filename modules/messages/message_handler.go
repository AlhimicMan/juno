package messages

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"

	"github.com/forbole/juno/v2/database"
	"github.com/forbole/juno/v2/types"
)

// HandleMsg represents a message handler that stores the given message inside the proper database table
func HandleMsg(
	index int, msg sdk.Msg, tx *types.Tx,
	parseAddresses MessageAddressesParser, cdc codec.Marshaler, db database.Database,
) error {
	msgPartitionID, err := db.CreatePartition("message", tx.Height)
	if err != nil {
		return err
	}

	// Get the involved addresses
	addresses, err := parseAddresses(cdc, msg)
	if err != nil {
		return err
	}

	// Marshal the value properly
	bz, err := cdc.MarshalJSON(msg)
	if err != nil {
		return err
	}

	fmt.Println("tx_hash, index, partition_id: ", tx.TxHash, index, msgPartitionID)

	db.UpdateMessage(types.NewMessage(
		tx.TxHash,
		index,
		proto.MessageName(msg),
		string(bz),
		addresses,
		msgPartitionID,
		tx.Height,
	))

	return db.SaveMessage(types.NewMessage(
		tx.TxHash,
		index,
		proto.MessageName(msg),
		string(bz),
		addresses,
		msgPartitionID,
		tx.Height,
	))
}
