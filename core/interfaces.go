package core

type ITxSubmitter interface {
	SubmitTx(tx []byte) error
}

type ITxDataRetriever interface {
	GetSlot() (uint64, error)
	GetUtxos(addr string) ([]Utxo, error)
	GetProtocolParameters() ([]byte, error)
}
