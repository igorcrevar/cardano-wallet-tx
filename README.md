# cardano-wallet-tx
A Go wrapper around the Cardano CLI that offers the following features:
- Create wallets/addresses
- Create multisig addresses
- Create transactions
- Single transaction signing or multi-witness transaction signing
- Query UTXOs, slot, protocol parameters, and submit transactions
- Optionally query UTXOs, slot, protocol parameters, and submit transactions with Blockfrost

Note: It only supports simple money transfer transactions; there is no functionality for smart contracts, etc.