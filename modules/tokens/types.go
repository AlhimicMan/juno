package tokens

type WasmMsgExecuteContract struct {
	Sender   string
	Contract string
	Msg      interface{}
}
