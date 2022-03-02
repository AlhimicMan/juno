package types

type WasmTransferMsg struct {
	TxHash    string
	Sender    string
	Contract  string
	Amount    string
	Recipient string
}
