<h1 align="center">Etherscan CLI</h1>

<p align="center">
  <strong>Explore EVM chains from your terminal.</strong><br>
  One API key for balances, transactions, tokens, contracts, logs, gas, stats, and more.
</p>

<p align="center">
  <img width="100%" alt="Etherscan CLI interactive explorer" src="https://github.com/user-attachments/assets/98332d40-dda7-415c-8664-8e9c16fa71f4">
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#get-started">Get started</a> ·
  <a href="#practical-workflows">Examples</a> ·
  <a href="#command-reference">Command reference</a> ·
  <a href="https://docs.etherscan.io/">API documentation</a>
</p>

The official command-line client and interactive explorer for the [Etherscan V2 API](https://docs.etherscan.io/). Use it interactively, pipe clean JSON into scripts, export transactions to CSV, or give an AI agent a predictable interface to on-chain data.

## Why Etherscan CLI?

- **Explore interactively** — browse endpoints, fill parameters, switch chains, and inspect results without memorizing commands.
- **Use one multichain interface** — query Ethereum and supported EVM chains by name or chain ID.
- **Work with humans or machines** — emit clean JSON by default for scripts and agents, or switch to tables and CSV when a human or a spreadsheet is reading.
- **Reach broad API coverage** — access accounts, contracts, tokens, logs, blocks, gas, stats, name tags, and proxy methods, with automatic pagination for list endpoints.

## Install

Prebuilt releases support macOS, Linux, and Windows on amd64 and arm64. After installing with any persistent method, run `etherscan version` to verify that the binary is on your `PATH`.

### Homebrew — macOS and Linux

```sh
brew install etherscan/etherscan-cli/etherscan
etherscan version
```

### npm — macOS, Linux, and Windows

Install the CLI globally:

```sh
npm install -g @etherscan-npm/cli
etherscan version
```

Or run it once without keeping a global installation:

```sh
npx @etherscan-npm/cli version
```

The npm package downloads the matching native release archive and verifies its SHA-256 checksum during installation. Lifecycle scripts must be enabled.

> The package is currently published as `@etherscan-npm/cli` while the `@etherscan` npm scope is being transferred. The `etherscan` command is unchanged.

### Installation script — macOS and Linux

```sh
curl -fsSL https://raw.githubusercontent.com/etherscan/etherscan-cli/master/scripts/install.sh | sh
```

The script selects the correct amd64 or arm64 archive, verifies its SHA-256 checksum, installs to `~/.local/bin` by default, and adds that directory to your shell profile when needed. Open a new terminal if `etherscan` is not immediately available. Run the installer with `--help` to see version, install-directory, and `PATH` options.

### PowerShell — Windows

Run in PowerShell:

```powershell
irm https://raw.githubusercontent.com/etherscan/etherscan-cli/master/scripts/install.ps1 | iex
```

Or from Command Prompt:

```bat
powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/etherscan/etherscan-cli/master/scripts/install.ps1 | iex"
```

The installer selects the correct x64 or arm64 archive, verifies its SHA-256 checksum, installs to `%LOCALAPPDATA%\Programs\Etherscan\bin` by default, and adds that directory to your user `PATH`. Open a new terminal if `etherscan` is not immediately available.

### Go

Go 1.25 or newer is required.

```sh
go install github.com/etherscan/etherscan-cli/cmd/etherscan@latest
```

Ensure your Go binary directory (`GOBIN`, or `GOPATH/bin` by default) is on `PATH`.

### Manual

Download the archive for your operating system and architecture plus `checksums.txt` from [GitHub Releases](https://github.com/etherscan/etherscan-cli/releases/latest). Verify the archive's SHA-256 checksum, extract it, and place `etherscan` (or `etherscan.exe`) on your `PATH`.

## Get started

### 1. Create an API key

Create an API key in your [Etherscan API dashboard](https://etherscan.io/myapikey). One Etherscan V2 key works across supported chains, subject to your API plan.

### 2. Authenticate

Validate and save the key locally, then confirm the active identity:

```sh
etherscan login
etherscan whoami
```

For CI or a temporary shell session, set `ETHERSCAN_API_KEY` instead of saving the key.

macOS or Linux:

```sh
export ETHERSCAN_API_KEY="YOUR_API_KEY"
```

PowerShell:

```powershell
$env:ETHERSCAN_API_KEY = "YOUR_API_KEY"
```

Command Prompt:

```bat
set "ETHERSCAN_API_KEY=YOUR_API_KEY"
```

### 3. Make your first request

```sh
etherscan account balance 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045
```

### Explore interactively

Run `etherscan tui` to open the full-screen endpoint explorer:

```sh
etherscan tui
```

The explorer can be opened before authentication and asks you to validate and save a key when you submit an API-backed endpoint. Running `etherscan` with no command prints the Quick Start guide.

## Practical workflows

### Follow wallet activity

```sh
# Recent normal transactions
etherscan account txlist 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --page 1 --offset 10

# ERC-20 transfers as a readable table instead of the default JSON
etherscan account tokentx 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 -o table

# Collect multiple pages for analysis
etherscan account txlist 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --all --max-pages 50 -o csv
```

### Inspect a smart contract

```sh
# WETH contract ABI and verified source metadata
etherscan contract getabi 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2
etherscan contract getsourcecode 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2
```

### Switch chains

```sh
# Chain names and numeric IDs both work
etherscan --chain base gastracker oracle
etherscan --chain 8453 account balance 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045

# See every chain built into this release
etherscan chains
```

### Build scripts and agent workflows

```sh
# Clean JSON for jq, Python, or an AI agent
etherscan account txlist 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 --compact | jq '.[0]'

# CSV for spreadsheets and data pipelines
etherscan account tokentx 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 -o csv > token-transfers.csv
```

## Output and Pagination

JSON is the default, matching the Etherscan API's own `application/json` responses. Use `-o table` for a readable terminal view of row-shaped results. API results are written to stdout; progress messages, warnings, diagnostics, and errors go to stderr so stdout remains suitable for pipelines and redirection.

| Flag | Purpose |
| --- | --- |
| `--output <format>`, `-o <format>` | Select `json` (default), `table`, or `csv` |
| `--compact` | Remove indentation from JSON output |
| `--all` | Automatically follow pages for supported list commands |
| `--max-pages <n>` | Stop `--all` after at most `n` pages (default: `20`) |

If `--all` reaches `--max-pages`, the CLI prints a warning to stderr and the result may be truncated. Increase the limit or narrow the request with supported filters when you need more results.

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

| Command | Description |
| --- | --- |
| `etherscan` | Show the Quick Start guide |
| `etherscan tui` | Launch the interactive explorer |
| `etherscan login` | Validate and store an API key |
| `etherscan logout` | Remove the stored API key |
| `etherscan uninstall` | Remove all CLI configuration |
| `etherscan update` | Update a Homebrew or installer-script installation |
| `etherscan whoami` | Show the active chain and masked API key |
| `etherscan config` | Get, list, or set CLI configuration |
| `etherscan chains` | List chains built into this CLI release |
| `etherscan completion` | Generate shell completion |
| `etherscan version` | Print the CLI version |
| `etherscan --help` | Show command usage and available options |

<!-- BEGIN GENERATED COMMAND INDEX -->
### Account

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan account balance` | Get the native balance of an address | [balance](https://docs.etherscan.io/api-reference/endpoint/balance.md) |
| `etherscan account balancemulti` | Get native balances for multiple addresses | [balancemulti](https://docs.etherscan.io/api-reference/endpoint/balancemulti.md) |
| `etherscan account txlist` | List normal transactions for an address or advanced filter | [txlist](https://docs.etherscan.io/api-reference/endpoint/txlist.md), [advanced-filter-txlist](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-txlist.md) |
| `etherscan account txlistinternal` | List internal transactions by address, transaction hash, block range, or advanced filter | [txlistinternal](https://docs.etherscan.io/api-reference/endpoint/txlistinternal.md), [txlistinternal-blockrange](https://docs.etherscan.io/api-reference/endpoint/txlistinternal-blockrange.md), [txlistinternal-txhash](https://docs.etherscan.io/api-reference/endpoint/txlistinternal-txhash.md), [advanced-filter-txlistinternal](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-txlistinternal.md) |
| `etherscan account tokentx` | List ERC-20 token transfers | [tokentx](https://docs.etherscan.io/api-reference/endpoint/tokentx.md), [advanced-filter-tokentx](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-tokentx.md) |
| `etherscan account tokennfttx` | List ERC-721 token transfers | [tokennfttx](https://docs.etherscan.io/api-reference/endpoint/tokennfttx.md), [advanced-filter-tokennfttx](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-tokennfttx.md) |
| `etherscan account token1155tx` | List ERC-1155 token transfers | [token1155tx](https://docs.etherscan.io/api-reference/endpoint/token1155tx.md), [advanced-filter-token1155tx](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-token1155tx.md) |
| `etherscan account getminedblocks` | List blocks or uncles mined by an address | [getminedblocks](https://docs.etherscan.io/api-reference/endpoint/getminedblocks.md) |
| `etherscan account balancehistory` | Get an address's native balance at a block | [balancehistory](https://docs.etherscan.io/api-reference/endpoint/balancehistory.md) |
| `etherscan account tokenbalance` | Get an address's ERC-20 token balance | [tokenbalance](https://docs.etherscan.io/api-reference/endpoint/tokenbalance.md) |
| `etherscan account tokenbalancehistory` | Get an address's token balance at a block | [tokenbalancehistory](https://docs.etherscan.io/api-reference/endpoint/tokenbalancehistory.md) |
| `etherscan account addresstokenbalance` | List ERC-20 holdings for an address | [addresstokenbalance](https://docs.etherscan.io/api-reference/endpoint/addresstokenbalance.md) |
| `etherscan account addresstokennftbalance` | List NFT holdings for an address | [addresstokennftbalance](https://docs.etherscan.io/api-reference/endpoint/addresstokennftbalance.md) |
| `etherscan account addresstokennftinventory` | List an address's inventory for an NFT contract | [addresstokennftinventory](https://docs.etherscan.io/api-reference/endpoint/addresstokennftinventory.md) |
| `etherscan account getdeposittxs` | List L2 deposit transactions | [getdeposittxs](https://docs.etherscan.io/api-reference/endpoint/getdeposittxs.md) |
| `etherscan account getwithdrawaltxs` | List L2 withdrawal transactions | [getwithdrawaltxs](https://docs.etherscan.io/api-reference/endpoint/getwithdrawaltxs.md) |
| `etherscan account txsBeaconWithdrawal` | List Ethereum beacon withdrawals | [txsbeaconwithdrawal](https://docs.etherscan.io/api-reference/endpoint/txsbeaconwithdrawal.md) |
| `etherscan account fundedby` | Find the address that likely funded an account | [fundedby](https://docs.etherscan.io/api-reference/endpoint/fundedby.md) |
| `etherscan account txnbridge` | List bridge transactions for an address | [txnbridge](https://docs.etherscan.io/api-reference/endpoint/txnbridge.md) |

### Contract

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan contract getabi` | Get a verified contract's ABI | [getabi](https://docs.etherscan.io/api-reference/endpoint/getabi.md) |
| `etherscan contract getsourcecode` | Get verified source code and contract metadata | [getsourcecode](https://docs.etherscan.io/api-reference/endpoint/getsourcecode.md) |
| `etherscan contract getcontractcreation` | Get creator and creation transaction data for contracts | [getcontractcreation](https://docs.etherscan.io/api-reference/endpoint/getcontractcreation.md) |
| `etherscan contract verify` | Submit contract source code for verification | [verifysourcecode](https://docs.etherscan.io/api-reference/endpoint/verifysourcecode.md) |
| `etherscan contract verify-status` | Check a source verification submission | [checkverifystatus](https://docs.etherscan.io/api-reference/endpoint/checkverifystatus.md) |
| `etherscan contract verify-proxy` | Submit a proxy contract for verification | [verifyproxycontract](https://docs.etherscan.io/api-reference/endpoint/verifyproxycontract.md) |
| `etherscan contract check-proxy` | Check a proxy verification submission | [checkproxyverification](https://docs.etherscan.io/api-reference/endpoint/checkproxyverification.md) |

### Transaction

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan transaction status` | Get a transaction's execution status and error description | [getstatus](https://docs.etherscan.io/api-reference/endpoint/getstatus.md) |
| `etherscan transaction receipt-status` | Get a transaction receipt's success or failure status | [gettxreceiptstatus](https://docs.etherscan.io/api-reference/endpoint/gettxreceiptstatus.md) |

### Block

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan block reward` | Get block and uncle rewards | [getblockreward](https://docs.etherscan.io/api-reference/endpoint/getblockreward.md) |
| `etherscan block countdown` | Estimate the time remaining until a block | [getblockcountdown](https://docs.etherscan.io/api-reference/endpoint/getblockcountdown.md) |
| `etherscan block txcount` | Get the number of transactions in a block | [getblocktxnscount](https://docs.etherscan.io/api-reference/endpoint/getblocktxnscount.md) |
| `etherscan block bytime` | Find the closest block before or after a timestamp | [getblocknobytime](https://docs.etherscan.io/api-reference/endpoint/getblocknobytime.md) |

### Logs

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan logs get` | Query event logs by block range, address, and topics | [getlogs](https://docs.etherscan.io/api-reference/endpoint/getlogs.md), [getlogs-address-topics](https://docs.etherscan.io/api-reference/endpoint/getlogs-address-topics.md), [getlogs-topics](https://docs.etherscan.io/api-reference/endpoint/getlogs-topics.md) |

### Stats

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan stats ethsupply` | Get the total ETH supply | [ethsupply](https://docs.etherscan.io/api-reference/endpoint/ethsupply.md) |
| `etherscan stats ethsupply2` | Get the extended ETH supply breakdown | [ethsupply2](https://docs.etherscan.io/api-reference/endpoint/ethsupply2.md) |
| `etherscan stats ethprice` | Get the latest ETH price | [ethprice](https://docs.etherscan.io/api-reference/endpoint/ethprice.md) |
| `etherscan stats chainsize` | Get historical Ethereum chain size data | [chainsize](https://docs.etherscan.io/api-reference/endpoint/chainsize.md) |
| `etherscan stats nodecount` | Get the total Ethereum node count | [nodecount](https://docs.etherscan.io/api-reference/endpoint/nodecount.md) |
| `etherscan stats tokensupply` | Get an ERC-20 token's total supply | [tokensupply](https://docs.etherscan.io/api-reference/endpoint/tokensupply.md) |
| `etherscan stats tokensupplyhistory` | Get a token's total supply at a block | [tokensupplyhistory](https://docs.etherscan.io/api-reference/endpoint/tokensupplyhistory.md) |
| `etherscan stats ethdailyprice` | Get historical daily ETH prices | [ethdailyprice](https://docs.etherscan.io/api-reference/endpoint/ethdailyprice.md) |
| `etherscan stats dailytx` | Get historical daily transaction counts | [dailytx](https://docs.etherscan.io/api-reference/endpoint/dailytx.md) |
| `etherscan stats dailynewaddress` | Get historical daily new-address counts | [dailynewaddress](https://docs.etherscan.io/api-reference/endpoint/dailynewaddress.md) |
| `etherscan stats dailyavgblocksize` | Get historical average daily block size | [dailyavgblocksize](https://docs.etherscan.io/api-reference/endpoint/dailyavgblocksize.md) |
| `etherscan stats dailyavgblocktime` | Get historical average daily block time | [dailyavgblocktime](https://docs.etherscan.io/api-reference/endpoint/dailyavgblocktime.md) |
| `etherscan stats dailyavggasprice` | Get historical average daily gas price | [dailyavggasprice](https://docs.etherscan.io/api-reference/endpoint/dailyavggasprice.md) |
| `etherscan stats dailyavggaslimit` | Get historical average daily gas limit | [dailyavggaslimit](https://docs.etherscan.io/api-reference/endpoint/dailyavggaslimit.md) |
| `etherscan stats dailygasused` | Get historical total daily gas used | [dailygasused](https://docs.etherscan.io/api-reference/endpoint/dailygasused.md) |
| `etherscan stats dailyblockrewards` | Get historical daily block rewards | [dailyblockrewards](https://docs.etherscan.io/api-reference/endpoint/dailyblockrewards.md) |
| `etherscan stats dailyblkcount` | Get historical daily block counts | [dailyblkcount](https://docs.etherscan.io/api-reference/endpoint/dailyblkcount.md) |
| `etherscan stats dailytxnfee` | Get historical daily transaction fees | [dailytxnfee](https://docs.etherscan.io/api-reference/endpoint/dailytxnfee.md) |
| `etherscan stats dailynetutilization` | Get historical daily network utilization | [dailynetutilization](https://docs.etherscan.io/api-reference/endpoint/dailynetutilization.md) |
| `etherscan stats dailyuncleblkcount` | Get historical daily uncle block counts | [dailyuncleblkcount](https://docs.etherscan.io/api-reference/endpoint/dailyuncleblkcount.md) |
| `etherscan stats dailyavghashrate` | Get historical average daily network hash rate | [dailyavghashrate](https://docs.etherscan.io/api-reference/endpoint/dailyavghashrate.md) |
| `etherscan stats dailyavgnetdifficulty` | Get historical average daily network difficulty | [dailyavgnetdifficulty](https://docs.etherscan.io/api-reference/endpoint/dailyavgnetdifficulty.md) |
| `etherscan stats dailyensregister` | Get historical daily ENS registration counts | [dailyensregister](https://docs.etherscan.io/api-reference/endpoint/dailyensregister.md) |
| `etherscan stats nodecounthistory` | Get historical Ethereum node counts | [nodecounthistory](https://docs.etherscan.io/api-reference/endpoint/nodecounthistory.md) |

### Token

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan token info` | Get token metadata such as name, symbol, type, and supply | [tokeninfo](https://docs.etherscan.io/api-reference/endpoint/tokeninfo.md) |
| `etherscan token tokenholderlist` | List token holders and their balances | [tokenholderlist](https://docs.etherscan.io/api-reference/endpoint/tokenholderlist.md) |
| `etherscan token tokenholdercount` | Get a token's holder count | [tokenholdercount](https://docs.etherscan.io/api-reference/endpoint/tokenholdercount.md) |
| `etherscan token topholders` | Get the largest token holders | [topholders](https://docs.etherscan.io/api-reference/endpoint/topholders.md) |

### Gas Tracker

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan gastracker oracle` | Get safe, proposed, and fast gas prices | [gasoracle](https://docs.etherscan.io/api-reference/endpoint/gasoracle.md) |
| `etherscan gastracker estimate` | Estimate confirmation time for a gas price | [gasestimate](https://docs.etherscan.io/api-reference/endpoint/gasestimate.md) |

### Nametag

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan nametag getaddresstag` | Get name tags and metadata for addresses (Pro Plus) | [getaddresstag](https://docs.etherscan.io/api-reference/endpoint/getaddresstag.md) |

### Proxy

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan proxy eth_blockNumber` | Get the latest block number | [ethblocknumber](https://docs.etherscan.io/api-reference/endpoint/ethblocknumber.md) |
| `etherscan proxy eth_getBlockByNumber` | Get a block by number or tag | [ethgetblockbynumber](https://docs.etherscan.io/api-reference/endpoint/ethgetblockbynumber.md) |
| `etherscan proxy eth_getTransactionByHash` | Get a transaction by hash | [ethgettransactionbyhash](https://docs.etherscan.io/api-reference/endpoint/ethgettransactionbyhash.md) |
| `etherscan proxy eth_getTransactionByBlockNumberAndIndex` | Get a transaction by block number and index | [ethgettransactionbyblocknumberandindex](https://docs.etherscan.io/api-reference/endpoint/ethgettransactionbyblocknumberandindex.md) |
| `etherscan proxy eth_getTransactionCount` | Get an address's transaction count (nonce) | [ethgettransactioncount](https://docs.etherscan.io/api-reference/endpoint/ethgettransactioncount.md) |
| `etherscan proxy eth_getBlockTransactionCountByNumber` | Get a block's transaction count | [ethgetblocktransactioncountbynumber](https://docs.etherscan.io/api-reference/endpoint/ethgetblocktransactioncountbynumber.md) |
| `etherscan proxy eth_getUncleByBlockNumberAndIndex` | Get an uncle by block number and index | [ethgetunclebyblocknumberandindex](https://docs.etherscan.io/api-reference/endpoint/ethgetunclebyblocknumberandindex.md) |
| `etherscan proxy eth_sendRawTransaction` | Broadcast a signed raw transaction | [ethsendrawtransaction](https://docs.etherscan.io/api-reference/endpoint/ethsendrawtransaction.md) |
| `etherscan proxy eth_call` | Execute a read-only contract call | [ethcall](https://docs.etherscan.io/api-reference/endpoint/ethcall.md) |
| `etherscan proxy eth_estimateGas` | Estimate the gas required for a transaction | [ethestimategas](https://docs.etherscan.io/api-reference/endpoint/ethestimategas.md) |
| `etherscan proxy eth_getTransactionReceipt` | Get a transaction receipt by hash | [ethgettransactionreceipt](https://docs.etherscan.io/api-reference/endpoint/ethgettransactionreceipt.md) |
| `etherscan proxy eth_getCode` | Get the code stored at an address | [ethgetcode](https://docs.etherscan.io/api-reference/endpoint/ethgetcode.md) |
| `etherscan proxy eth_getStorageAt` | Get a value from a contract storage position | [ethgetstorageat](https://docs.etherscan.io/api-reference/endpoint/ethgetstorageat.md) |
| `etherscan proxy eth_gasPrice` | Get the current gas price | [ethgasprice](https://docs.etherscan.io/api-reference/endpoint/ethgasprice.md) |

### API Usage

| Command | Description | API docs |
| --- | --- | --- |
| `etherscan apilimit` | Show used, available, and total API credits | [getapilimit](https://docs.etherscan.io/api-reference/endpoint/getapilimit.md) |
<!-- END GENERATED COMMAND INDEX -->

</details>

## Configuration and authentication

Authentication is resolved in this order: `--api-key` for the current command, `ETHERSCAN_API_KEY`, then the key saved by `etherscan login`.

`etherscan login` and the TUI key-setup prompt store the key as plaintext in `$XDG_CONFIG_HOME/etherscan/config.toml` when `XDG_CONFIG_HOME` is set, or `~/.etherscan/config.toml` otherwise. The directory and file are created with restrictive permissions where the operating system supports them. Treat this file as a secret, never commit API keys, and prefer the environment variable or `--api-key` for CI and temporary sessions.

`etherscan logout` removes only the saved key. If `ETHERSCAN_API_KEY` is set, it remains active until you unset it in that shell or environment.

Manage non-secret defaults with:

```sh
etherscan config list
etherscan config set default_chain=base
etherscan config set default_output=table
```

For the active chain, `--chain` takes precedence over `ETHERSCAN_CHAIN`, which takes precedence over `default_chain` in the configuration file.

## Updating

Update through the same channel used to install the CLI:

| Installation channel | Update command |
| --- | --- |
| Homebrew | `brew upgrade etherscan/etherscan-cli/etherscan` |
| npm | `npm install -g @etherscan-npm/cli@latest` |
| macOS/Linux or Windows installer script | `etherscan update` |
| Go | `go install github.com/etherscan/etherscan-cli/cmd/etherscan@latest` |
| Manual release archive | Download and verify the new archive from [GitHub Releases](https://github.com/etherscan/etherscan-cli/releases/latest) |

Use the channel-specific command above rather than mixing update mechanisms. For an npm-managed installation, `etherscan update` prints the npm command without modifying files under `node_modules`.

## Shell completion

Generate completion for bash, zsh, fish, or PowerShell, then load or save the output according to your shell's completion setup:

```sh
etherscan completion bash
etherscan completion zsh
etherscan completion fish
etherscan completion powershell
```

## Development

Go 1.25 or newer is required to build the CLI.

```sh
# Run the Go test suite
go test ./...

# Build the CLI
go build -o etherscan ./cmd/etherscan
```

Installer changes can be checked with `sh scripts/test-install.sh` on macOS/Linux or `./scripts/test-install.ps1` in PowerShell on Windows.

The npm distribution can be checked with `sh scripts/test-npm.sh` on macOS/Linux or `./scripts/test-npm.ps1` in PowerShell. These tests pack and install the package against local fixture release archives; they do not publish to npm.

For the first npm release, publish the GitHub release assets before publishing the package. From an exact release-tag checkout, run `npm version --no-git-tag-version <version>` followed by `npm publish --access public`. Then configure npm trusted publishing for `etherscan/etherscan-cli` and `.github/workflows/release.yml`, and set the `NPM_PUBLISH_ENABLED` repository variable to `true` for later tagged releases.

## API coverage and support

Endpoint, chain, and plan availability can differ. The Etherscan documentation is authoritative for [supported chains](https://docs.etherscan.io/supported-chains), [rate limits](https://docs.etherscan.io/resources/rate-limits), [PRO endpoints](https://docs.etherscan.io/resources/pro-endpoints), parameters, responses, and [API errors](https://docs.etherscan.io/resources/common-error-messages).

Report CLI bugs and feature requests in [GitHub Issues](https://github.com/etherscan/etherscan-cli/issues). For API-key, account, billing, or endpoint-support questions, use the [Etherscan support form](https://etherscan.io/contactus?id=11).

## License

[MIT](LICENSE)
