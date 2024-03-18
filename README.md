# cardano-wallet-tx
A Go wrapper around the Cardano CLI that offers the following features:
- Create simple wallets (signing/verifying keys) and addresses
- Create multisig addresses via policy script
- Create transactions
- Transaction hash signing and assembling signatures into the final transaction
- Query UTXOs, slot, protocol parameters, and submit transactions with Blockfrost or CLI

Note: It only supports simple money transfer transactions; there is no functionality for smart contracts, etc.