package tokens

import (
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"

	"github.com/forbole/juno/v2/database"
	"github.com/forbole/juno/v2/types"
)

func ProcessWasmContractExecuteMessage(rawMsg []byte, tx *types.Tx, db database.Database) error {
	executeMsg := &WasmMsgExecuteContract{}
	err := json.Unmarshal(rawMsg, executeMsg)
	if err != nil {
		return fmt.Errorf("error processing contract execute msg: %w", err)
	}
	msgMap := executeMsg.Msg.(map[string]interface{})
	transferRawMsg, ok := msgMap["transfer"]
	if ok {
		transferMsgBody := transferRawMsg.(map[string]interface{})
		transferMsg, err := ProcessTokenTransferMsg(transferMsgBody)
		if err != nil {
			return fmt.Errorf("error processing contract execute msg as transfer msg: %w", err)
		}
		transferMsg.Contract = executeMsg.Contract
		transferMsg.Sender = executeMsg.Sender
		transferMsg.TxHash = tx.TxHash
		return db.SaveTokenTransfer(transferMsg)
	}

	return nil
}

func ProcessTokenTransferMsg(transferMsgBody map[string]interface{}) (*types.WasmTransferMsg, error) {
	amount, converted := transferMsgBody["amount"].(string)
	if !converted {
		return nil, fmt.Errorf("error converting amount from message")
	}
	transferMsg := &types.WasmTransferMsg{
		Amount:    amount,
		Recipient: transferMsgBody["recipient"].(string),
	}
	return transferMsg, nil
}

// HandleMsg represents a message handler that stores the given message inside the proper database table
func HandleMsg(
	msg sdk.Msg, tx *types.Tx, cdc codec.Codec, db database.Database,
) error {
	// Marshal the value properly
	bz, err := cdc.MarshalJSON(msg)
	if err != nil {
		return err
	}

	messageType := proto.MessageName(msg)
	if messageType == "cosmwasm.wasm.v1.MsgExecuteContract" {
		err := ProcessWasmContractExecuteMessage(bz, tx, db)
		return err
	}
	return nil
}
