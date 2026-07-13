# Etherscan CLI

A cross-platform Go CLI and interactive explorer for the [Etherscan V2 API](https://docs.etherscan.io/). It maps supported Etherscan endpoints into commands for accounts, tokens, contracts, logs, gas, stats, proxy/RPC-style methods, API usage, address metadata, across multiple EVM chains.

Etherscan's API documentation remains as the main reference for endpoint parameters, responses, rate limits, supported chains, and errors.

## Installation

Download a prebuilt binary from [GitHub Releases](https://github.com/etherscan/etherscan-cli/releases)

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

<!-- BEGIN GENERATED COMMAND INDEX -->
### Account

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan account balance` | Get native balance | [balance](https://docs.etherscan.io/api-reference/endpoint/balance.md) |
| `etherscan account balancemulti` | Get native balances for multiple addresses | [balancemulti](https://docs.etherscan.io/api-reference/endpoint/balancemulti.md) |
| `etherscan account txlist` | List normal transactions | [txlist](https://docs.etherscan.io/api-reference/endpoint/txlist.md), [advanced-filter-txlist](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-txlist.md) |
| `etherscan account txlistinternal` | List internal transactions | [txlistinternal](https://docs.etherscan.io/api-reference/endpoint/txlistinternal.md), [txlistinternal-blockrange](https://docs.etherscan.io/api-reference/endpoint/txlistinternal-blockrange.md), [txlistinternal-txhash](https://docs.etherscan.io/api-reference/endpoint/txlistinternal-txhash.md), [advanced-filter-txlistinternal](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-txlistinternal.md) |
| `etherscan account tokentx` | List ERC-20 transfers | [tokentx](https://docs.etherscan.io/api-reference/endpoint/tokentx.md), [advanced-filter-tokentx](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-tokentx.md) |
| `etherscan account tokennfttx` | List ERC-721 transfers | [tokennfttx](https://docs.etherscan.io/api-reference/endpoint/tokennfttx.md), [advanced-filter-tokennfttx](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-tokennfttx.md) |
| `etherscan account token1155tx` | List ERC-1155 transfers | [token1155tx](https://docs.etherscan.io/api-reference/endpoint/token1155tx.md), [advanced-filter-token1155tx](https://docs.etherscan.io/api-reference/endpoint/advanced-filter-token1155tx.md) |
| `etherscan account getminedblocks` | List mined blocks | [getminedblocks](https://docs.etherscan.io/api-reference/endpoint/getminedblocks.md) |
| `etherscan account balancehistory` | Get native balance at block | [balancehistory](https://docs.etherscan.io/api-reference/endpoint/balancehistory.md) |
| `etherscan account tokenbalance` | Get ERC-20 token balance | [tokenbalance](https://docs.etherscan.io/api-reference/endpoint/tokenbalance.md) |
| `etherscan account tokenbalancehistory` | Get token balance at block | [tokenbalancehistory](https://docs.etherscan.io/api-reference/endpoint/tokenbalancehistory.md) |
| `etherscan account addresstokenbalance` | List ERC-20 holdings | [addresstokenbalance](https://docs.etherscan.io/api-reference/endpoint/addresstokenbalance.md) |
| `etherscan account addresstokennftbalance` | List NFT holdings | [addresstokennftbalance](https://docs.etherscan.io/api-reference/endpoint/addresstokennftbalance.md) |
| `etherscan account addresstokennftinventory` | List NFT inventory | [addresstokennftinventory](https://docs.etherscan.io/api-reference/endpoint/addresstokennftinventory.md) |
| `etherscan account getdeposittxs` | List L2 deposits | [getdeposittxs](https://docs.etherscan.io/api-reference/endpoint/getdeposittxs.md) |
| `etherscan account getwithdrawaltxs` | List L2 withdrawals | [getwithdrawaltxs](https://docs.etherscan.io/api-reference/endpoint/getwithdrawaltxs.md) |
| `etherscan account txsBeaconWithdrawal` | List beacon withdrawals | [txsbeaconwithdrawal](https://docs.etherscan.io/api-reference/endpoint/txsbeaconwithdrawal.md) |
| `etherscan account fundedby` | Get likely funder | [fundedby](https://docs.etherscan.io/api-reference/endpoint/fundedby.md) |
| `etherscan account txnbridge` | List bridge transactions | [txnbridge](https://docs.etherscan.io/api-reference/endpoint/txnbridge.md) |

### Contract

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan contract getabi` | Get contract ABI | [getabi](https://docs.etherscan.io/api-reference/endpoint/getabi.md) |
| `etherscan contract getsourcecode` | Get contract source metadata | [getsourcecode](https://docs.etherscan.io/api-reference/endpoint/getsourcecode.md) |
| `etherscan contract getcontractcreation` | Get contract creation data | [getcontractcreation](https://docs.etherscan.io/api-reference/endpoint/getcontractcreation.md) |
| `etherscan contract verify` | Submit source verification | [verifysourcecode](https://docs.etherscan.io/api-reference/endpoint/verifysourcecode.md) |
| `etherscan contract verify-status` | Check verification status | [checkverifystatus](https://docs.etherscan.io/api-reference/endpoint/checkverifystatus.md) |
| `etherscan contract verify-proxy` | Submit proxy verification | [verifyproxycontract](https://docs.etherscan.io/api-reference/endpoint/verifyproxycontract.md) |
| `etherscan contract check-proxy` | Check proxy verification | [checkproxyverification](https://docs.etherscan.io/api-reference/endpoint/checkproxyverification.md) |

### Transaction

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan transaction status` | Get transaction execution status | [getstatus](https://docs.etherscan.io/api-reference/endpoint/getstatus.md) |
| `etherscan transaction receipt-status` | Get transaction receipt status | [gettxreceiptstatus](https://docs.etherscan.io/api-reference/endpoint/gettxreceiptstatus.md) |

### Block

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan block reward` | Get block reward | [getblockreward](https://docs.etherscan.io/api-reference/endpoint/getblockreward.md) |
| `etherscan block countdown` | Get block countdown | [getblockcountdown](https://docs.etherscan.io/api-reference/endpoint/getblockcountdown.md) |
| `etherscan block txcount` | Get block transaction count | [getblocktxnscount](https://docs.etherscan.io/api-reference/endpoint/getblocktxnscount.md) |
| `etherscan block bytime` | Find block by timestamp | [getblocknobytime](https://docs.etherscan.io/api-reference/endpoint/getblocknobytime.md) |

### Logs

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan logs get` | Query event logs | [getlogs](https://docs.etherscan.io/api-reference/endpoint/getlogs.md), [getlogs-address-topics](https://docs.etherscan.io/api-reference/endpoint/getlogs-address-topics.md), [getlogs-topics](https://docs.etherscan.io/api-reference/endpoint/getlogs-topics.md) |

### Stats

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan stats ethsupply` | Get ETH supply | [ethsupply](https://docs.etherscan.io/api-reference/endpoint/ethsupply.md) |
| `etherscan stats ethsupply2` | Get extended ETH supply | [ethsupply2](https://docs.etherscan.io/api-reference/endpoint/ethsupply2.md) |
| `etherscan stats ethprice` | Get ETH price | [ethprice](https://docs.etherscan.io/api-reference/endpoint/ethprice.md) |
| `etherscan stats chainsize` | Get chain size | [chainsize](https://docs.etherscan.io/api-reference/endpoint/chainsize.md) |
| `etherscan stats nodecount` | Get node count | [nodecount](https://docs.etherscan.io/api-reference/endpoint/nodecount.md) |
| `etherscan stats tokensupply` | Get token supply | [tokensupply](https://docs.etherscan.io/api-reference/endpoint/tokensupply.md) |
| `etherscan stats tokensupplyhistory` | Get token supply at block | [tokensupplyhistory](https://docs.etherscan.io/api-reference/endpoint/tokensupplyhistory.md) |
| `etherscan stats ethdailyprice` | Get ethdailyprice series | [ethdailyprice](https://docs.etherscan.io/api-reference/endpoint/ethdailyprice.md) |
| `etherscan stats dailytx` | Get dailytx series | [dailytx](https://docs.etherscan.io/api-reference/endpoint/dailytx.md) |
| `etherscan stats dailynewaddress` | Get dailynewaddress series | [dailynewaddress](https://docs.etherscan.io/api-reference/endpoint/dailynewaddress.md) |
| `etherscan stats dailyavgblocksize` | Get dailyavgblocksize series | [dailyavgblocksize](https://docs.etherscan.io/api-reference/endpoint/dailyavgblocksize.md) |
| `etherscan stats dailyavgblocktime` | Get dailyavgblocktime series | [dailyavgblocktime](https://docs.etherscan.io/api-reference/endpoint/dailyavgblocktime.md) |
| `etherscan stats dailyavggasprice` | Get dailyavggasprice series | [dailyavggasprice](https://docs.etherscan.io/api-reference/endpoint/dailyavggasprice.md) |
| `etherscan stats dailyavggaslimit` | Get dailyavggaslimit series | [dailyavggaslimit](https://docs.etherscan.io/api-reference/endpoint/dailyavggaslimit.md) |
| `etherscan stats dailygasused` | Get dailygasused series | [dailygasused](https://docs.etherscan.io/api-reference/endpoint/dailygasused.md) |
| `etherscan stats dailyblockrewards` | Get dailyblockrewards series | [dailyblockrewards](https://docs.etherscan.io/api-reference/endpoint/dailyblockrewards.md) |
| `etherscan stats dailyblkcount` | Get dailyblkcount series | [dailyblkcount](https://docs.etherscan.io/api-reference/endpoint/dailyblkcount.md) |
| `etherscan stats dailytxnfee` | Get dailytxnfee series | [dailytxnfee](https://docs.etherscan.io/api-reference/endpoint/dailytxnfee.md) |
| `etherscan stats dailynetutilization` | Get dailynetutilization series | [dailynetutilization](https://docs.etherscan.io/api-reference/endpoint/dailynetutilization.md) |
| `etherscan stats dailyuncleblkcount` | Get dailyuncleblkcount series | [dailyuncleblkcount](https://docs.etherscan.io/api-reference/endpoint/dailyuncleblkcount.md) |
| `etherscan stats dailyavghashrate` | Get dailyavghashrate series | [dailyavghashrate](https://docs.etherscan.io/api-reference/endpoint/dailyavghashrate.md) |
| `etherscan stats dailyavgnetdifficulty` | Get dailyavgnetdifficulty series | [dailyavgnetdifficulty](https://docs.etherscan.io/api-reference/endpoint/dailyavgnetdifficulty.md) |
| `etherscan stats dailyensregister` | Get dailyensregister series | [dailyensregister](https://docs.etherscan.io/api-reference/endpoint/dailyensregister.md) |
| `etherscan stats nodecounthistory` | Get nodecounthistory series | [nodecounthistory](https://docs.etherscan.io/api-reference/endpoint/nodecounthistory.md) |

### Token

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan token info` | Get token metadata | [tokeninfo](https://docs.etherscan.io/api-reference/endpoint/tokeninfo.md) |
| `etherscan token tokenholderlist` | List token holders | [tokenholderlist](https://docs.etherscan.io/api-reference/endpoint/tokenholderlist.md) |
| `etherscan token tokenholdercount` | Get token holder count | [tokenholdercount](https://docs.etherscan.io/api-reference/endpoint/tokenholdercount.md) |
| `etherscan token topholders` | Get top holders | [topholders](https://docs.etherscan.io/api-reference/endpoint/topholders.md) |

### Gas Tracker

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan gastracker oracle` | Get gas oracle | [gasoracle](https://docs.etherscan.io/api-reference/endpoint/gasoracle.md) |
| `etherscan gastracker estimate` | Estimate gas confirmation time | [gasestimate](https://docs.etherscan.io/api-reference/endpoint/gasestimate.md) |

### Nametag

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan nametag getaddresstag` | Get address name tags and metadata (Pro Plus) | [getaddresstag](https://docs.etherscan.io/api-reference/endpoint/getaddresstag.md) |

### Proxy

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan proxy eth_blockNumber` | Get latest block number | [ethblocknumber](https://docs.etherscan.io/api-reference/endpoint/ethblocknumber.md) |
| `etherscan proxy eth_getBlockByNumber` | Get block by number | [ethgetblockbynumber](https://docs.etherscan.io/api-reference/endpoint/ethgetblockbynumber.md) |
| `etherscan proxy eth_getTransactionByHash` | Get transaction by hash | [ethgettransactionbyhash](https://docs.etherscan.io/api-reference/endpoint/ethgettransactionbyhash.md) |
| `etherscan proxy eth_getTransactionByBlockNumberAndIndex` | Get transaction by block/index | [ethgettransactionbyblocknumberandindex](https://docs.etherscan.io/api-reference/endpoint/ethgettransactionbyblocknumberandindex.md) |
| `etherscan proxy eth_getTransactionCount` | Get account nonce | [ethgettransactioncount](https://docs.etherscan.io/api-reference/endpoint/ethgettransactioncount.md) |
| `etherscan proxy eth_getBlockTransactionCountByNumber` | Get block tx count | [ethgetblocktransactioncountbynumber](https://docs.etherscan.io/api-reference/endpoint/ethgetblocktransactioncountbynumber.md) |
| `etherscan proxy eth_getUncleByBlockNumberAndIndex` | Get uncle by block/index | [ethgetunclebyblocknumberandindex](https://docs.etherscan.io/api-reference/endpoint/ethgetunclebyblocknumberandindex.md) |
| `etherscan proxy eth_sendRawTransaction` | Broadcast signed transaction | [ethsendrawtransaction](https://docs.etherscan.io/api-reference/endpoint/ethsendrawtransaction.md) |
| `etherscan proxy eth_call` | Execute eth_call | [ethcall](https://docs.etherscan.io/api-reference/endpoint/ethcall.md) |
| `etherscan proxy eth_estimateGas` | Estimate gas | [ethestimategas](https://docs.etherscan.io/api-reference/endpoint/ethestimategas.md) |
| `etherscan proxy eth_getTransactionReceipt` | Get receipt | [ethgettransactionreceipt](https://docs.etherscan.io/api-reference/endpoint/ethgettransactionreceipt.md) |
| `etherscan proxy eth_getCode` | Get code | [ethgetcode](https://docs.etherscan.io/api-reference/endpoint/ethgetcode.md) |
| `etherscan proxy eth_getStorageAt` | Get storage | [ethgetstorageat](https://docs.etherscan.io/api-reference/endpoint/ethgetstorageat.md) |
| `etherscan proxy eth_gasPrice` | Get gas price | [ethgasprice](https://docs.etherscan.io/api-reference/endpoint/ethgasprice.md) |

### API Usage

| Command | Description | API documentation |
| --- | --- | --- |
| `etherscan apilimit` | Show API credit usage | [getapilimit](https://docs.etherscan.io/api-reference/endpoint/getapilimit.md) |
<!-- END GENERATED COMMAND INDEX -->
