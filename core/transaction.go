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
	MinUTxODefaultValue = uint64(1000000)
	draftTxFile         = "tx.draft"
)

type TxInput struct {
	Hash  string `json:"hsh"`
	Index uint32 `json:"ind"`
}

func (i TxInput) String() string {
	return fmt.Sprintf("%s#%d", i.Hash, i.Index)
}

type TxInputWithPolicyScript struct {
	Input        TxInput
	PolicyScript IPolicyScript
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
	inputs             []TxInputWithPolicyScript
	outputs            []TxOutput
	metadata           []byte
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
	os.Remove(b.baseDirectory)
}

func (b *TxBuilder) SetTestNetMagic(testNetMagic uint) *TxBuilder {
	b.testNetMagic = testNetMagic

	return b
}

func (b *TxBuilder) SetFee(fee uint64) *TxBuilder {
	b.fee = fee

	return b
}

func (b *TxBuilder) AddInputsWithScript(script IPolicyScript, inputs ...TxInput) *TxBuilder {
	for _, inp := range inputs {
		b.inputs = append(b.inputs, TxInputWithPolicyScript{
			Input:        inp,
			PolicyScript: script,
		})
	}

	return b
}

func (b *TxBuilder) AddInputsWithScripts(inputs []TxInput, scripts []IPolicyScript) *TxBuilder {
	cnt := len(inputs)
	if l := len(scripts); cnt > l {
		cnt = l
	}

	for i, inp := range inputs[:cnt] {
		b.inputs = append(b.inputs, TxInputWithPolicyScript{
			Input:        inp,
			PolicyScript: scripts[i],
		})
	}

	return b
}

func (b *TxBuilder) AddInputs(inputs ...TxInput) *TxBuilder {
	for _, inp := range inputs {
		b.inputs = append(b.inputs, TxInputWithPolicyScript{
			Input: inp,
		})
	}

	return b
}

func (b *TxBuilder) AddOutputs(outputs ...TxOutput) *TxBuilder {
	b.outputs = append(b.outputs, outputs...)

	return b
}

func (b *TxBuilder) UpdateOutputAmount(index int, amount uint64) *TxBuilder {
	if index < 0 {
		index = len(b.outputs) + index
	}

	b.outputs[index].Amount = amount

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

func (b *TxBuilder) CalculateFee(witnessCount int) (uint64, error) {
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

	if witnessCount == 0 {
		for _, x := range b.inputs {
			witnessCount += x.PolicyScript.GetCount()
		}

		if witnessCount == 0 {
			witnessCount = 1
		}
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

	bytes, err := os.ReadFile(path.Join(b.baseDirectory, draftTxFile))
	if err != nil {
		return nil, err
	}

	return NewTransactionUnwitnessedRawFromJson(bytes)
}

func (b *TxBuilder) Sign(tx []byte, wallets []ISigningKeyRetriver) ([]byte, error) {
	outFilePath := path.Join(b.baseDirectory, "tx.sig")
	txFilePath := path.Join(b.baseDirectory, "tx.raw")

	txBytes, err := TransactionUnwitnessedRaw(tx).ToJSON()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(txFilePath, txBytes, 0755); err != nil {
		return nil, err
	}

	args := append([]string{
		"transaction", "sign",
		"--tx-body-file", txFilePath,
		"--out-file", outFilePath},
		getTestNetMagicArgs(b.testNetMagic)...)

	for i, wallet := range wallets {
		signingKeyPath := path.Join(b.baseDirectory, fmt.Sprintf("tx_%d.skey", i))
		if err := SaveKeyBytesToFile(wallet.GetSigningKey(), signingKeyPath, true, false); err != nil {
			return nil, err
		}

		args = append(args, "--signing-key-file", signingKeyPath)
	}

	if _, err = runCommand(resolveCardanoCliBinary(), args); err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(outFilePath)
	if err != nil {
		return nil, err
	}

	return NewTransactionWitnessedRawFromJson(bytes)
}

func (b *TxBuilder) AddWitness(tx []byte, wallet ISigningKeyRetriver) ([]byte, error) {
	outFilePath := path.Join(b.baseDirectory, "tx.wit")
	txFilePath := path.Join(b.baseDirectory, "tx.raw")
	signingKeyPath := path.Join(b.baseDirectory, "tx.skey")

	txBytes, err := TransactionUnwitnessedRaw(tx).ToJSON()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(txFilePath, txBytes, 0755); err != nil {
		return nil, err
	}

	if err := SaveKeyBytesToFile(wallet.GetSigningKey(), signingKeyPath, true, false); err != nil {
		return nil, err
	}

	args := append([]string{
		"transaction", "witness",
		"--signing-key-file", signingKeyPath,
		"--tx-body-file", txFilePath,
		"--out-file", outFilePath},
		getTestNetMagicArgs(b.testNetMagic)...)

	if _, err = runCommand(resolveCardanoCliBinary(), args); err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(outFilePath)
	if err != nil {
		return nil, err
	}

	return NewWitnessRawFromJson(bytes)
}

func (b *TxBuilder) AssembleWitnesses(tx []byte, witnesses [][]byte) ([]byte, error) {
	outFilePath := path.Join(b.baseDirectory, "tx.sig")
	txFilePath := path.Join(b.baseDirectory, "tx.raw")
	witnessesFilePaths := make([]string, len(witnesses))

	for i, witness := range witnesses {
		witnessesFilePaths[i] = path.Join(b.baseDirectory, fmt.Sprintf("witness-%d", i+1))

		content, err := TxWitnessRaw(witness).ToJSON()
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(witnessesFilePaths[i], content, 0755); err != nil {
			return nil, err
		}
	}

	txBytes, err := TransactionUnwitnessedRaw(tx).ToJSON()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(txFilePath, txBytes, 0755); err != nil {
		return nil, err
	}

	args := []string{
		"transaction", "assemble",
		"--tx-body-file", txFilePath,
		"--out-file", outFilePath}

	for _, fp := range witnessesFilePaths {
		args = append(args, "--witness-file", fp)
	}

	if _, err = runCommand(resolveCardanoCliBinary(), args); err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(outFilePath)
	if err != nil {
		return nil, err
	}

	return NewTransactionWitnessedRawFromJson(bytes)
}

func (b *TxBuilder) GetTxHash(tx []byte) (string, error) {
	txFilePath := path.Join(b.baseDirectory, "tx.tmp")

	txBytes, err := TransactionUnwitnessedRaw(tx).ToJSON()
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(txFilePath, txBytes, 0755); err != nil {
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

	args := []string{
		"transaction", "build-raw",
		"--protocol-params-file", protocolParamsFilePath,
		"--fee", strconv.FormatUint(fee, 10),
		"--out-file", path.Join(b.baseDirectory, draftTxFile),
	}

	for i, inp := range b.inputs {
		args = append(args, "--tx-in", inp.Input.String())

		if inp.PolicyScript != nil {
			policyFilePath = path.Join(b.baseDirectory, fmt.Sprintf("policy_%d.json", i))
			if err := os.WriteFile(policyFilePath, inp.PolicyScript.GetPolicyScript(), 0755); err != nil {
				return err
			}

			args = append(args, "--tx-in-script-file", policyFilePath)
		}
	}

	for _, out := range b.outputs {
		args = append(args, "--tx-out", out.String())
	}

	if metaDataFilePath != "" {
		args = append(args, "--metadata-json-file", metaDataFilePath)
	}

	_, err := runCommand(resolveCardanoCliBinary(), args)
	return err
}
