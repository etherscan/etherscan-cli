<h1 align="center">Etherscan CLI</h1>

<p align="center">
  <strong>Explore EVM chains from your terminal.</strong><br>
  One API key for balances, transactions, tokens, contracts, logs, gas, stats, and more.
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#get-started">Get started</a> ·
  <a href="#practical-workflows">Examples</a> ·
  <a href="#command-reference">Command reference</a> ·
  <a href="https://docs.etherscan.io/">API documentation</a>
</p>

<p align="center">
  <img width="100%" alt="Etherscan CLI interactive explorer" src="https://github.com/user-attachments/assets/98332d40-dda7-415c-8664-8e9c16fa71f4">
</p>

The official command-line client and interactive explorer for the [Etherscan V2 API](https://docs.etherscan.io/). Use it interactively, pipe clean JSON into scripts, export transactions to CSV, or give an AI agent a predictable interface to on-chain data.

## Why Etherscan CLI?

- **Interactive explorer** — browse endpoints, fill parameters, and inspect results without memorizing commands.
- **Multichain by default** — switch between Ethereum and supported EVM chains by name or chain ID.
- **Made for the terminal** — readable tables for humans, plus JSON and CSV for automation.
- **Automatic pagination** — collect multi-page transaction and token results with one flag.
- **Broad API coverage** — accounts, contracts, tokens, logs, blocks, gas, stats, name tags, and proxy/RPC methods.
- **Agent-friendly output** — deterministic commands, clean stdout, compact JSON, and explicit errors.

## Install

### npm — macOS, Linux, and Windows

```sh
npm install --global @etherscan/cli
etherscan version
```

No global install is required when using `npx`:

```sh
npx @etherscan/cli version
```

The npm package installs only the native binary for the current platform. Node.js 18 or newer is required.

### Shell — macOS and Linux

```sh
curl -fsSL https://raw.githubusercontent.com/etherscan/etherscan-cli/master/install.sh | sh
```

The installer detects amd64 or arm64, verifies the release checksum, and installs to `/usr/local/bin`. To use a different location:

```sh
curl -fsSL https://raw.githubusercontent.com/etherscan/etherscan-cli/master/install.sh | INSTALL_DIR="$HOME/.local/bin" sh
```

<details>
<summary><strong>Windows binaries, Go install, and manual installation</strong></summary>

### Windows binary

Download `windows_amd64.zip` or `windows_arm64.zip` from [GitHub Releases](https://github.com/etherscan/etherscan-cli/releases/latest), extract `etherscan.exe`, and place it in a directory included in `PATH`.

### Go

With Go 1.25 or newer:

```sh
go install github.com/etherscan/etherscan-cli/cmd/etherscan@latest
```

### Manual

Download the archive for your operating system and architecture from [GitHub Releases](https://github.com/etherscan/etherscan-cli/releases/latest), verify it against `checksums.txt`, and place the extracted binary in `PATH`.

</details>

## Get started

### 1. Add your API key

Get an [Etherscan API key](https://etherscan.io/apis), then validate and save it:

```sh
etherscan login
etherscan whoami
```

For CI or temporary sessions, use an environment variable instead:

```sh
export ETHERSCAN_API_KEY="YOUR_API_KEY"
```

### 2. Make your first request

```sh
etherscan account balance 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045
```

### 3. Explore interactively

Run the CLI without a command to open the full-screen endpoint explorer:

```sh
etherscan
```

Use `etherscan tui` when you want to launch it explicitly.

## Practical workflows

### Follow wallet activity

```sh
# Recent normal transactions
etherscan account txlist 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --page 1 --offset 10

# ERC-20 transfers as JSON
etherscan account tokentx 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --json

# Collect multiple pages for analysis
etherscan account txlist 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --all --max-pages 50 --csv
```

### Inspect a smart contract

```sh
# WETH contract ABI and verified source metadata
etherscan contract getabi 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2
etherscan contract getsourcecode 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2 --json
```

### Switch chains

```sh
# Chain names and numeric IDs both work
etherscan --chain base gastracker oracle
etherscan --chain 8453 account balance 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045

# See every chain built into this release
etherscan chains list
```

### Build scripts and agent workflows

```sh
# Clean JSON for jq, Python, or an AI agent
etherscan account txlist 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --json --compact | jq '.[0]'

# CSV for spreadsheets and data pipelines
etherscan account tokentx 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --csv > token-transfers.csv
```

Tables are the default. Use `--json`, `--json --compact`, `--csv`, or `--output <table|json|csv>` to select another format. Diagnostics go to stderr so stdout remains suitable for pipelines.

For paginated commands, `--all` follows subsequent pages and `--max-pages <n>` sets a safety limit. Results may be truncated when that limit is reached.

## Command reference

Every command includes built-in parameter and usage help:

```sh
etherscan --help
etherscan account txlist --help
etherscan contract verify --help
```

<details>
<summary><strong>Browse all commands</strong></summary>

### CLI utilities

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

</details>

## Development

```sh
# Run the Go test suite
go test ./...

# Build the CLI
go build -o etherscan ./cmd/etherscan

# Run the npm launcher tests
npm test
```

API behavior, parameters, rate limits, supported chains, and error definitions are documented in the [Etherscan API documentation](https://docs.etherscan.io/).
