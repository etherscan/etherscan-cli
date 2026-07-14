# Etherscan CLI

A cross-platform Go CLI and interactive explorer for the [Etherscan V2 API](https://docs.etherscan.io/). It maps supported Etherscan endpoints into commands for accounts, tokens, contracts, logs, gas, stats, proxy/RPC-style methods, API usage, address metadata, across multiple EVM chains.

Etherscan's API documentation remains as the main reference for endpoint parameters, responses, rate limits, supported chains, and errors.

## Installation

### macOS and Linux

Install the latest release with the installer script:

```sh
curl -fsSL https://raw.githubusercontent.com/etherscan/etherscan-cli/master/install.sh | sh
```

The script detects macOS or Linux and amd64 or arm64, verifies the release checksum, and installs `etherscan` to `/usr/local/bin`. If that directory is not writable, either run the command with the permissions appropriate for your system or choose a user-owned directory:

```sh
curl -fsSL https://raw.githubusercontent.com/etherscan/etherscan-cli/master/install.sh | INSTALL_DIR="$HOME/.local/bin" sh
```

Make sure a custom install directory is included in your `PATH`.

### Windows

Download the latest `windows_amd64.zip` or `windows_arm64.zip` from [GitHub Releases](https://github.com/etherscan/etherscan-cli/releases/latest), extract `etherscan.exe`, and place it in a directory included in your `PATH`.

PowerShell example for the current v1.0.0 release on a typical Intel/AMD Windows computer:

```powershell
Invoke-WebRequest https://github.com/etherscan/etherscan-cli/releases/download/v1.0.0/etherscan_1.0.0_windows_amd64.zip -OutFile etherscan.zip
Expand-Archive .\etherscan.zip -DestinationPath .\etherscan
.\etherscan\etherscan.exe version
```

### Go

With Go 1.25 or newer:

```sh
go install github.com/etherscan/etherscan-cli/cmd/etherscan@latest
```

### npm

npm installation is not currently supported. This project is a Go CLI and does not publish an official npm package; use a release binary, the installer script, or `go install`.

## Quickstart

Store and validate an Etherscan API key, then make your first request:

```sh
etherscan login
etherscan whoami
etherscan account balance 0x0000000000000000000000000000000000000000
```

Select a chain by name or chain ID:

```sh
etherscan --chain base account txlist 0x0000000000000000000000000000000000000000 --page 1 --offset 5
etherscan --chain 8453 gastracker oracle
```

Use JSON when handing results to another program or agent:

```sh
etherscan account txlist 0x0000000000000000000000000000000000000000 --json
```

## Interactive Explorer

Run `etherscan` in an interactive terminal, allowing you to explore, pick and fill in the parameters of each endpoint.

```sh
etherscan
```

<img width="1918" height="1054" alt="image" src="https://github.com/user-attachments/assets/98332d40-dda7-415c-8664-8e9c16fa71f4" />

## Output and Scripting

Tables are the default. Use JSON or CSV for scripts and pipelines:

```sh
etherscan account balance 0x... --json
etherscan account balance 0x... --json --compact
etherscan account txlist 0x... --csv
etherscan account txlist 0x... --all --max-pages 50 --json
```

| Flag | Purpose |
| --- | --- |
| `--output <format>` | Select `table`, `json`, or `csv` |
| `--json` | Print the raw API result as JSON |
| `--compact` | Print compact JSON |
| `--csv` | Print list-style results as CSV |
| `--all` | Automatically paginate supported list commands |
| `--max-pages <n>` | Stop automatic pagination after at most `n` pages |

If `--all` reaches `--max-pages`, the result may be truncated.


## Command Index

### CLI utilities

| Command | Description |
| --- | --- |
| `etherscan` | Launch the explorer in an interactive terminal |
| `etherscan tui` | Explicitly launch the interactive explorer |
| `etherscan login` | Validate and store an API key |
| `etherscan logout` | Remove the stored API key |
| `etherscan uninstall` | Remove all CLI configuration |
| `etherscan whoami` | Show the active chain and saved API key |
| `etherscan config` | Get, list, or set CLI configuration |
| `etherscan chains list` | List chains built into this CLI release |
| `etherscan completion` | Generate shell completion |
| `etherscan version` | Print build information |
| `etherscan --help` | Helpful tips and command usage |

### Account

| Command | Description |
| --- | --- |
| `etherscan account balance` | Get the native balance of an address |
| `etherscan account balancemulti` | Get native balances for multiple addresses |
| `etherscan account txlist` | List normal transactions for an address or advanced filter |
| `etherscan account txlistinternal` | List internal transactions by address, transaction hash, block range, or advanced filter |
| `etherscan account tokentx` | List ERC-20 token transfers |
| `etherscan account tokennfttx` | List ERC-721 token transfers |
| `etherscan account token1155tx` | List ERC-1155 token transfers |
| `etherscan account getminedblocks` | List blocks or uncles mined by an address |
| `etherscan account balancehistory` | Get an address's native balance at a block |
| `etherscan account tokenbalance` | Get an address's ERC-20 token balance |
| `etherscan account tokenbalancehistory` | Get an address's token balance at a block |
| `etherscan account addresstokenbalance` | List ERC-20 holdings for an address |
| `etherscan account addresstokennftbalance` | List NFT holdings for an address |
| `etherscan account addresstokennftinventory` | List an address's inventory for an NFT contract |
| `etherscan account getdeposittxs` | List L2 deposit transactions |
| `etherscan account getwithdrawaltxs` | List L2 withdrawal transactions |
| `etherscan account txsBeaconWithdrawal` | List Ethereum beacon withdrawals |
| `etherscan account fundedby` | Find the address that likely funded an account |
| `etherscan account txnbridge` | List bridge transactions for an address |

### Contract

| Command | Description |
| --- | --- |
| `etherscan contract getabi` | Get a verified contract's ABI |
| `etherscan contract getsourcecode` | Get verified source code and contract metadata |
| `etherscan contract getcontractcreation` | Get creator and creation transaction data for contracts |
| `etherscan contract verify` | Submit contract source code for verification |
| `etherscan contract verify-status` | Check a source verification submission |
| `etherscan contract verify-proxy` | Submit a proxy contract for verification |
| `etherscan contract check-proxy` | Check a proxy verification submission |

### Transaction

| Command | Description |
| --- | --- |
| `etherscan transaction status` | Get a transaction's execution status and error description |
| `etherscan transaction receipt-status` | Get a transaction receipt's success or failure status |

### Block

| Command | Description |
| --- | --- |
| `etherscan block reward` | Get block and uncle rewards |
| `etherscan block countdown` | Estimate the time remaining until a block |
| `etherscan block txcount` | Get the number of transactions in a block |
| `etherscan block bytime` | Find the closest block before or after a timestamp |

### Logs

| Command | Description |
| --- | --- |
| `etherscan logs get` | Query event logs by block range, address, and topics |

### Stats

| Command | Description |
| --- | --- |
| `etherscan stats ethsupply` | Get the total ETH supply |
| `etherscan stats ethsupply2` | Get the extended ETH supply breakdown |
| `etherscan stats ethprice` | Get the latest ETH price |
| `etherscan stats chainsize` | Get historical Ethereum chain size data |
| `etherscan stats nodecount` | Get the total Ethereum node count |
| `etherscan stats tokensupply` | Get an ERC-20 token's total supply |
| `etherscan stats tokensupplyhistory` | Get a token's total supply at a block |
| `etherscan stats ethdailyprice` | Get historical daily ETH prices |
| `etherscan stats dailytx` | Get historical daily transaction counts |
| `etherscan stats dailynewaddress` | Get historical daily new-address counts |
| `etherscan stats dailyavgblocksize` | Get historical average daily block size |
| `etherscan stats dailyavgblocktime` | Get historical average daily block time |
| `etherscan stats dailyavggasprice` | Get historical average daily gas price |
| `etherscan stats dailyavggaslimit` | Get historical average daily gas limit |
| `etherscan stats dailygasused` | Get historical total daily gas used |
| `etherscan stats dailyblockrewards` | Get historical daily block rewards |
| `etherscan stats dailyblkcount` | Get historical daily block counts |
| `etherscan stats dailytxnfee` | Get historical daily transaction fees |
| `etherscan stats dailynetutilization` | Get historical daily network utilization |
| `etherscan stats dailyuncleblkcount` | Get historical daily uncle block counts |
| `etherscan stats dailyavghashrate` | Get historical average daily network hash rate |
| `etherscan stats dailyavgnetdifficulty` | Get historical average daily network difficulty |
| `etherscan stats dailyensregister` | Get historical daily ENS registration counts |
| `etherscan stats nodecounthistory` | Get historical Ethereum node counts |

### Token

| Command | Description |
| --- | --- |
| `etherscan token info` | Get token metadata such as name, symbol, type, and supply |
| `etherscan token tokenholderlist` | List token holders and their balances |
| `etherscan token tokenholdercount` | Get a token's holder count |
| `etherscan token topholders` | Get the largest token holders |

### Gas Tracker

| Command | Description |
| --- | --- |
| `etherscan gastracker oracle` | Get safe, proposed, and fast gas prices |
| `etherscan gastracker estimate` | Estimate confirmation time for a gas price |

### Nametag

| Command | Description |
| --- | --- |
| `etherscan nametag getaddresstag` | Get name tags and metadata for addresses (Pro Plus) |

### Proxy

| Command | Description |
| --- | --- |
| `etherscan proxy eth_blockNumber` | Get the latest block number |
| `etherscan proxy eth_getBlockByNumber` | Get a block by number or tag |
| `etherscan proxy eth_getTransactionByHash` | Get a transaction by hash |
| `etherscan proxy eth_getTransactionByBlockNumberAndIndex` | Get a transaction by block number and index |
| `etherscan proxy eth_getTransactionCount` | Get an address's transaction count (nonce) |
| `etherscan proxy eth_getBlockTransactionCountByNumber` | Get a block's transaction count |
| `etherscan proxy eth_getUncleByBlockNumberAndIndex` | Get an uncle by block number and index |
| `etherscan proxy eth_sendRawTransaction` | Broadcast a signed raw transaction |
| `etherscan proxy eth_call` | Execute a read-only contract call |
| `etherscan proxy eth_estimateGas` | Estimate the gas required for a transaction |
| `etherscan proxy eth_getTransactionReceipt` | Get a transaction receipt by hash |
| `etherscan proxy eth_getCode` | Get the code stored at an address |
| `etherscan proxy eth_getStorageAt` | Get a value from a contract storage position |
| `etherscan proxy eth_gasPrice` | Get the current gas price |

### API Usage

| Command | Description |
| --- | --- |
| `etherscan apilimit` | Show used, available, and total API credits |
