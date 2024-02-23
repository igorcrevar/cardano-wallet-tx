package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

type Utxo struct {
	Hash   string `json:"hsh"`
	Index  uint32 `json:"ind"`
	Amount uint64 `json:"amount"`
	// Era    string `json:"era"`
}

type LedgerTip struct {
	Block           uint64 `json:"block"`
	Epoch           uint64 `json:"epoch"`
	Era             string `json:"era"`
	Hash            string `json:"hash"`
	Slot            uint64 `json:"slot"`
	SlotInEpoch     uint64 `json:"slotInEpoch"`
	SlotsToEpochEnd uint64 `json:"slotsToEpochEnd"`
	SyncProgress    string `json:"syncProgress"`
}

type TxDataRetrieverCli struct {
	baseDirectory string
	testNetMagic  uint
	socketPath    string
}

func NewTxDataRetrieverCli(testNetMagic uint, socketPath string) (*TxDataRetrieverCli, error) {
	baseDirectory, err := os.MkdirTemp("", "cardano-txs")
	if err != nil {
		return nil, err
	}

	return &TxDataRetrieverCli{
		baseDirectory: baseDirectory,
		testNetMagic:  testNetMagic,
		socketPath:    socketPath,
	}, nil
}

func (b *TxDataRetrieverCli) Dispose() {
	os.RemoveAll(b.baseDirectory)
}

func (b *TxDataRetrieverCli) GetProtocolParameters() ([]byte, error) {
	outFile := path.Join(b.baseDirectory, "protocol.json")

	args := append([]string{
		"query", "protocol-parameters",
		"--socket-path", b.socketPath,
		"--out-file", outFile,
	}, getTestNetMagicArgs(b.testNetMagic)...)

	_, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(outFile)
	if err != nil {
		return nil, err
	}

	_ = os.Remove(outFile)

	return bytes, nil
}

func (b *TxDataRetrieverCli) GetUtxos(addr string) ([]Utxo, error) {
	args := append([]string{
		"query", "utxo",
		"--socket-path", b.socketPath,
		"--address", addr,
	}, getTestNetMagicArgs(b.testNetMagic)...)

	output, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return nil, err
	}

	rows := strings.Split(output, "\n")
	rows = rows[2 : len(rows)-1]
	inputs := make([]Utxo, len(rows))

	for i, x := range rows {
		cnt := 0
		inputs[i] = Utxo{}

	exitloop:
		for _, val := range strings.Split(x, " ") {
			if val == "" {
				continue
			}

			switch cnt {
			case 0:
				inputs[i].Hash = val
				cnt++
			case 1:
				intVal, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					return nil, err
				}

				inputs[i].Index = uint32(intVal)
				cnt++
			case 2:
				intVal, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					return nil, err
				}

				inputs[i].Amount = intVal

				break exitloop
			}
		}
	}

	return inputs, nil
}

func (b *TxDataRetrieverCli) GetSlot() (uint64, error) {
	args := append([]string{
		"query", "tip",
		"--socket-path", b.socketPath,
	}, getTestNetMagicArgs(b.testNetMagic)...)

	res, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return 0, err
	}

	var legder LedgerTip

	if err := json.Unmarshal([]byte(res), &legder); err != nil {
		return 0, err
	}

	return legder.Slot, nil
}

func (b *TxDataRetrieverCli) SubmitTx(tx []byte) error {
	txFilePath := path.Join(b.baseDirectory, "tx.send")

	if err := os.WriteFile(txFilePath, tx, 0755); err != nil {
		return err
	}

	args := append([]string{
		"transaction", "submit",
		"--socket-path", b.socketPath,
		"--tx-file", txFilePath,
	}, getTestNetMagicArgs(b.testNetMagic)...)

	res, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return err
	}

	if strings.Contains(res, "Transaction successfully submitted.") {
		return nil
	}

	return fmt.Errorf("unknown error submiting tx: %s", res)
}
