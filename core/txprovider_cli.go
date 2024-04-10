package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

type TxProviderCli struct {
	baseDirectory string
	testNetMagic  uint
	socketPath    string
}

func NewTxProviderCli(testNetMagic uint, socketPath string) (*TxProviderCli, error) {
	baseDirectory, err := os.MkdirTemp("", "cardano-txs")
	if err != nil {
		return nil, err
	}

	return &TxProviderCli{
		baseDirectory: baseDirectory,
		testNetMagic:  testNetMagic,
		socketPath:    socketPath,
	}, nil
}

func (b *TxProviderCli) Dispose() {
	os.RemoveAll(b.baseDirectory)
	os.Remove(b.baseDirectory)
}

func (b *TxProviderCli) GetProtocolParameters(_ context.Context) ([]byte, error) {
	args := append([]string{
		"query", "protocol-parameters",
		"--socket-path", b.socketPath,
	}, getTestNetMagicArgs(b.testNetMagic)...)

	response, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return nil, err
	}

	return []byte(response), nil
}

func (b *TxProviderCli) GetUtxos(_ context.Context, addr string) ([]Utxo, error) {
	args := append([]string{
		"query", "utxo",
		"--socket-path", b.socketPath,
		"--address", addr,
	}, getTestNetMagicArgs(b.testNetMagic)...)

	output, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return nil, err
	}

	rows := strings.Split(strings.Trim(output, "\n"), "\n")[2:]
	inputs := make([]Utxo, len(rows))

	for i, x := range rows {
		cnt := 0

	exitloop:
		for _, val := range strings.Split(x, " ") {
			if val == "" {
				continue
			}

			cnt++
			switch cnt {
			case 1:
				inputs[i].Hash = val
			case 2:
				intVal, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					return nil, err
				}

				inputs[i].Index = uint32(intVal)
			case 3:
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

func (b *TxProviderCli) GetSlot(_ context.Context) (uint64, error) {
	args := append([]string{
		"query", "tip",
		"--socket-path", b.socketPath,
	}, getTestNetMagicArgs(b.testNetMagic)...)

	res, err := runCommand(resolveCardanoCliBinary(), args)
	if err != nil {
		return 0, err
	}

	var legder struct {
		Block           uint64 `json:"block"`
		Epoch           uint64 `json:"epoch"`
		Era             string `json:"era"`
		Hash            string `json:"hash"`
		Slot            uint64 `json:"slot"`
		SlotInEpoch     uint64 `json:"slotInEpoch"`
		SlotsToEpochEnd uint64 `json:"slotsToEpochEnd"`
		SyncProgress    string `json:"syncProgress"`
	}

	if err := json.Unmarshal([]byte(res), &legder); err != nil {
		return 0, err
	}

	return legder.Slot, nil
}

func (b *TxProviderCli) SubmitTx(_ context.Context, txSigned []byte) error {
	txFilePath := path.Join(b.baseDirectory, "tx.send")

	txBytes, err := TransactionWitnessedRaw(txSigned).ToJSON()
	if err != nil {
		return err
	}

	if err := os.WriteFile(txFilePath, txBytes, FilePermission); err != nil {
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
