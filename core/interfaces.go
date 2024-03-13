package core

type Utxo struct {
	Hash   string `json:"hsh"`
	Index  uint32 `json:"ind"`
	Amount uint64 `json:"amount"`
}

type ITxSubmitter interface {
	// SubmitTx submits transaction - txSigned should be cbor serialized signed transaction
	SubmitTx(txSigned []byte) error
}

type ITxRetriever interface {
	GetTxByHash(hash string) (map[string]interface{}, error)
}

type ITxDataRetriever interface {
	GetSlot() (uint64, error)
	GetProtocolParameters() ([]byte, error)
}

type IUTxORetriever interface {
	GetUtxos(addr string) ([]Utxo, error)
}

type ITxProvider interface {
	ITxSubmitter
	ITxDataRetriever
	IUTxORetriever
	Dispose()
}

type ISigner interface {
	GetSigningKey() []byte
	GetVerificationKey() []byte
}

type IStakeSigner interface {
	ISigner
	GetStakeSigningKey() []byte
	GetStakeVerificationKey() []byte
}

type IWallet interface {
	IStakeSigner
	GetKeyHash() string
}

type IWalletManager interface {
	// Create creates new wallet
	Create(directory string, forceCreate bool) (IWallet, error)
	// Load loads wallet
	Load(directory string) (IWallet, error)
}

type IPolicyScript interface {
	GetPolicyScript() []byte
	GetCount() int
}
