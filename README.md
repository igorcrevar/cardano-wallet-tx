# go-cardano-tx

A comprehensive library for creating, signing, and submitting Cardano transactions with a focus on ease of use and flexibility. The library offers the following key functionalities:

- **Transaction Creation**:  
   - Build transactions using the Cardano CLI.  
   - Supports **lovelace** and **native assets/tokens**.  
   - *(Note: Smart contracts and advanced functionalities are currently not supported.)*

- **Transaction Signing**:  
   - Sign transactions and assemble multiple signatures into a finalized transaction.

- **Blockchain Queries**:  
   - Query UTXOs, current slot, protocol parameters, and submit transactions using **Ogmios**, **Blockfrost**, or the Cardano CLI.

- **Address Management**:  
   - Generate and manipulate Cardano addresses.  

- **Multisig Support**:  
   - Create multisig addresses using policy scripts.