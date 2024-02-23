package core

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	MinUtxoValue = uint64(1000000)

	draftTxFile   = "tx.draft"
	witnessTxFile = "witness.tx"
)

type TxInput struct {
	Hash  string `json:"hsh"`
	Index uint32 `json:"ind"`
}

func (i TxInput) String() string {
	return fmt.Sprintf("%s#%d", i.Hash, i.Index)
}

type TxOutput struct {
	Addr   string `json:"addr"`
	Amount uint64 `json:"amount"`
}

func (o TxOutput) String() string {
	return fmt.Sprintf("%s+%d", o.Addr, o.Amount)
}

type TxBuilder struct {
	baseDirectory      string
	inputs             []TxInput
	outputs            []TxOutput
	metadata           []byte
	policy             []byte
	witnessCount       int
	protocolParameters []byte
	timeToLive         uint64
	testNetMagic       uint
	fee                uint64
}

func NewTxBuilder() (*TxBuilder, error) {
	baseDirectory, err := os.MkdirTemp("", "cardano-txs")
	if err != nil {
		return nil, err
	}

	return &TxBuilder{
		baseDirectory: baseDirectory,
	}, nil
}

func (b *TxBuilder) Dispose() {
	os.RemoveAll(b.baseDirectory)
}

func (b *TxBuilder) SetTestNetMagic(testNetMagic uint) *TxBuilder {
	b.testNetMagic = testNetMagic

	return b
}

func (b *TxBuilder) SetFee(fee uint64) *TxBuilder {
	b.fee = fee

	return b
}

func (b *TxBuilder) AddInputs(inputs ...TxInput) *TxBuilder {
	b.inputs = append(b.inputs, inputs...)

	return b
}

func (b *TxBuilder) AddOutputs(outputs ...TxOutput) *TxBuilder {
	b.outputs = append(b.outputs, outputs...)

	return b
}

func (b *TxBuilder) UpdateLastOutputAmount(amount uint64) *TxBuilder {
	b.outputs[len(b.outputs)-1].Amount = amount

	return b
}

func (b *TxBuilder) SetPolicy(policy []byte, witnessCount int) *TxBuilder {
	b.policy = policy
	b.witnessCount = witnessCount

	return b
}

func (b *TxBuilder) SetMetaData(metadata []byte) *TxBuilder {
	b.metadata = metadata

	return b
}

func (b *TxBuilder) SetProtocolParameters(protocolParameters []byte) *TxBuilder {
	b.protocolParameters = protocolParameters

	return b
}

func (b *TxBuilder) SetTimeToLive(timeToLive uint64) *TxBuilder {
	b.timeToLive = timeToLive

	return b
}

func (b *TxBuilder) CalculateFee() (uint64, error) {
	if b.protocolParameters == nil {
		return 0, errors.New("protocol parameters not set")
	}

	protocolParamsFilePath := path.Join(b.baseDirectory, "protocol-parameters.json")
	if err := os.WriteFile(protocolParamsFilePath, b.protocolParameters, 0755); err != nil {
		return 0, err
	}

	if err := b.buildRawTx(protocolParamsFilePath, 0); err != nil {
		return 0, err
	}

	witnessCount := b.witnessCount
	if witnessCount == 0 {
		witnessCount = 1
	}

	feeOutput, err := runCommand(resolveCardanoCliBinary(), append([]string{
		"transaction", "calculate-min-fee",
		"--tx-body-file", path.Join(b.baseDirectory, draftTxFile),
		"--tx-in-count", strconv.Itoa(len(b.inputs)),
		"--tx-out-count", strconv.Itoa(len(b.outputs)),
		"--witness-count", strconv.FormatUint(uint64(witnessCount), 10),
		"--byron-witness-count", "0",
		"--protocol-params-file", protocolParamsFilePath,
	}, getTestNetMagicArgs(b.testNetMagic)...))
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(strings.Split(feeOutput, " ")[0], 10, 64)
}

func (b *TxBuilder) Build() ([]byte, error) {
	if b.protocolParameters == nil {
		return nil, errors.New("protocol parameters not set")
	}

	protocolParamsFilePath := path.Join(b.baseDirectory, "protocol-parameters.json")
	if err := os.WriteFile(protocolParamsFilePath, b.protocolParameters, 0755); err != nil {
		return nil, err
	}

	if err := b.buildRawTx(protocolParamsFilePath, b.fee); err != nil {
		return nil, err
	}

	return os.ReadFile(path.Join(b.baseDirectory, draftTxFile))
}

func (b *TxBuilder) Sign(tx []byte, signingKeyPath string) ([]byte, error) {
	outFilePath := path.Join(b.baseDirectory, "tx.sign")
	txFilePath := path.Join(b.baseDirectory, "tx.tmp")

	if err := os.WriteFile(txFilePath, tx, 0755); err != nil {
		return nil, err
	}

	args := append([]string{
		"transaction", "sign",
		"--signing-key-file", signingKeyPath,
		"--tx-body-file", txFilePath,
		"--out-file", outFilePath},
		getTestNetMagicArgs(b.testNetMagic)...)

	_, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(outFilePath)
}

func (b *TxBuilder) AddWitness(tx []byte, signingKeyPath string) ([]byte, error) {
	outFilePath := path.Join(b.baseDirectory, "witness.tmp")
	txFilePath := path.Join(b.baseDirectory, "tx.tmp")

	if err := os.WriteFile(txFilePath, tx, 0755); err != nil {
		return nil, err
	}

	args := append([]string{
		"transaction", "witness",
		"--signing-key-file", signingKeyPath,
		"--tx-body-file", txFilePath,
		"--out-file", outFilePath},
		getTestNetMagicArgs(b.testNetMagic)...)

	_, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(outFilePath)
}

func (b *TxBuilder) AssembleWitnesses(tx []byte, witnesses [][]byte) ([]byte, error) {
	outFilePath := path.Join(b.baseDirectory, "tx.signed")
	txFilePath := path.Join(b.baseDirectory, "tx.tmp")
	witnessesFilePaths := make([]string, len(witnesses))

	for i, content := range witnesses {
		witnessesFilePaths[i] = path.Join(b.baseDirectory, fmt.Sprintf("witness-%d", i+1))

		if err := os.WriteFile(witnessesFilePaths[i], content, 0755); err != nil {
			return nil, err
		}
	}

	if err := os.WriteFile(txFilePath, tx, 0755); err != nil {
		return nil, err
	}

	args := []string{
		"transaction", "assemble",
		"--tx-body-file", txFilePath,
		"--out-file", outFilePath}

	for _, fp := range witnessesFilePaths {
		args = append(args, "--witness-file", fp)
	}

	_, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(outFilePath)
}

func (b *TxBuilder) GetTxHash(tx []byte) (string, error) {
	txFilePath := path.Join(b.baseDirectory, "tx.tmp")

	if err := os.WriteFile(txFilePath, tx, 0755); err != nil {
		return "", err
	}

	args := []string{
		"transaction", "txid",
		"--tx-body-file", txFilePath}

	res, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return "", err
	}

	return strings.Trim(res, "\n"), err
}

func (b *TxBuilder) buildRawTx(protocolParamsFilePath string, fee uint64) error {
	metaDataFilePath := ""
	policyFilePath := ""

	if b.metadata != nil {
		metaDataFilePath = path.Join(b.baseDirectory, "metadata.json")
		if err := os.WriteFile(metaDataFilePath, b.metadata, 0755); err != nil {
			return err
		}
	}

	if b.policy != nil {
		policyFilePath = path.Join(b.baseDirectory, "policy.json")
		if err := os.WriteFile(policyFilePath, b.policy, 0755); err != nil {
			return err
		}
	}

	args := []string{
		"transaction", "build-raw",
	}

	for _, inp := range b.inputs {
		args = append(args, "--tx-in", inp.String())
	}

	for _, out := range b.outputs {
		args = append(args, "--tx-out", out.String())
	}

	if metaDataFilePath != "" {
		args = append(args, "--metadata-json-file", metaDataFilePath)
	}

	if protocolParamsFilePath != "" {
		args = append(args, "--protocol-params-file", protocolParamsFilePath)
	}

	if policyFilePath != "" {
		args = append(args, "--tx-in-script-file", policyFilePath)
	}

	args = append(args,
		"--invalid-hereafter", strconv.FormatUint(b.timeToLive, 10),
		"--fee", strconv.FormatUint(fee, 10),
		"--out-file", path.Join(b.baseDirectory, draftTxFile))

	_, err := runCommand(resolveCardanoCliBinary(), args)
	return err
}
