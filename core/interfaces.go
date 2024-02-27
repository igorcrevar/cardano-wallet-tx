package core

type Utxo struct {
	Hash   string `json:"hsh"`
	Index  uint32 `json:"ind"`
	Amount uint64 `json:"amount"`
}

type ITxSubmitter interface {
	SubmitTx(tx []byte) error
}

type ITxDataRetriever interface {
	GetSlot() (uint64, error)
	GetUtxos(addr string) ([]Utxo, error)
	GetProtocolParameters() ([]byte, error)
}

type ITxProvider interface {
	ITxSubmitter
	ITxDataRetriever
	Dispose()
}

type IWallet interface {
	GetAddress() string
	GetVerificationKey() []byte
	GetSigningKey() []byte
	GetKeyHash() string
}

type IWalletBuilder interface {
	Create(directory string, forceCreate bool) (IWallet, error)
}
