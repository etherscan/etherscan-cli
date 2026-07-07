# Etherscan CLI

Etherscan's CLI gives you command line access to our API V2. 

These structured inputs have proven beneficial to AI agents, by encapsulating authentication and individual API methods to simple --flags. 

This official CLI will also be maintained to be 1:1 to the existing APIs as soon as any changes are made.

## Installation

Download the build from our [Release page](https://github.com/etherscan/etherscan-cli/releases) for your environment. 

Start by providing your API key for authentication.

```
etherscan login
```

From here you can handover to your agent, who will have access to all the API V2 endpoints. Plus utility tools to check available rate limits, supported chains etc below.

For full documentation on each endpoint, refer to our [docs](https://docs.etherscan.io/) or [llms.txt](https://docs.etherscan.io/llms.txt) reference.

## Output and Scripting

Tables are the default. Use JSON or CSV for scripts:

```sh
etherscan account balance 0x... --json
etherscan account balance 0x... --json --compact
etherscan account txlist 0x... --csv
etherscan account txlist 0x... --all --csv
etherscan account txlist 0x... --all --max-pages 50 --json
```

Common output flags:

| Flag | Purpose |
| --- | --- |
| `--output <format>` | Choose `table`, `json`, or `csv` |
| `--json` | Print raw JSON |
| `--compact` | Print compact JSON |
| `--csv` | Print CSV for list-style results |
| `--all` | Auto-paginate list commands |
| `--max-pages <n>` | Stop `--all` after a maximum number of pages |

`--all` works on paginated list commands. If the command reaches `--max-pages`, results may be truncated.

## Configuration and API Keys

API keys are resolved in this order:

1. `--api-key`
2. `ETHERSCAN_API_KEY`
3. OS keyring, saved by `etherscan login`
4. Plaintext config fallback

Supported config keys:

```sh
etherscan config set api_key=...
etherscan config set default_chain=ethereum
etherscan config set default_output=table
etherscan config set base_url=https://api.etherscan.io/v2/api
```

## Command reference

| Command | Description |
| --- | --- |
| `etherscan account` | Etherscan account commands |
| `etherscan apilimit` | Show API credit usage |
| `etherscan block` | Etherscan block commands |
| `etherscan chains` | Manage supported chains |
| `etherscan completion` | Generate shell completion |
| `etherscan config` | Manage CLI configuration |
| `etherscan contract` | Etherscan contract commands |
| `etherscan gastracker` | Etherscan gastracker commands |
| `etherscan login` | Store an Etherscan API key |
| `etherscan logout` | Remove the stored API key |
| `etherscan logs` | Etherscan logs commands |
| `etherscan proxy` | Etherscan proxy commands |
| `etherscan stats` | Etherscan stats commands |
| `etherscan token` | Etherscan token commands |
| `etherscan transaction` | Etherscan transaction commands |
| `etherscan version` | Print version |
| `etherscan whoami` | Show the active chain and saved API key |

## `etherscan login`

```text
Usage: etherscan login
```

Global flags: `--api-key`, `--chain`, `--base-url`, `--timeout`, `--rate-limit`

## `etherscan logout`

```text
Usage: etherscan logout
```

## `etherscan whoami`

```text
Usage: etherscan whoami
```

Global flags: `--api-key`, `--chain`

## `etherscan config`

```text
Usage: etherscan config
```

| Subcommand | Description |
| --- | --- |
| `etherscan config get` | Get a config value |
| `etherscan config list` | Show configuration |
| `etherscan config set` | Set a config value |

### `etherscan config get`

```text
Usage: etherscan config get key
```

| Argument | Description | Required |
| --- | --- | --- |
| `key` | Config key: `api_key`, `default_chain`, `default_output`, or `base_url` | Yes |

### `etherscan config list`

```text
Usage: etherscan config list
```

### `etherscan config set`

```text
Usage: etherscan config set key=value
```

| Argument | Description | Required |
| --- | --- | --- |
| `key=value` | Config assignment for `api_key`, `default_chain`, `default_output`, or `base_url` | Yes |

## `etherscan chains`

```text
Usage: etherscan chains
```

Global flags: `--output`/`--json`/`--csv`

| Subcommand | Description |
| --- | --- |
| `etherscan chains list` | List built-in chains |

### `etherscan chains list`

```text
Usage: etherscan chains list
```

Global flags: `--output`/`--json`/`--csv`

## `etherscan completion`

```text
Usage: etherscan completion [bash|zsh|fish|powershell]
```

| Argument | Description | Required |
| --- | --- | --- |
| `shell` | `bash`, `zsh`, `fish`, or `powershell` | Yes |

## `etherscan version`

```text
Usage: etherscan version
```

## `etherscan apilimit`

```text
Usage: etherscan apilimit
```

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

## `etherscan account`

```text
Usage: etherscan account
```

| Subcommand | Description |
| --- | --- |
| `etherscan account addresstokenbalance` | List ERC-20 holdings |
| `etherscan account addresstokennftbalance` | List NFT holdings |
| `etherscan account addresstokennftinventory` | List NFT inventory |
| `etherscan account balance` | Get native balance |
| `etherscan account balancehistory` | Get native balance at block |
| `etherscan account balancemulti` | Get native balances for multiple addresses |
| `etherscan account fundedby` | Get likely funder |
| `etherscan account getdeposittxs` | List L2 deposits |
| `etherscan account getminedblocks` | List mined blocks |
| `etherscan account getwithdrawaltxs` | List L2 withdrawals |
| `etherscan account token1155tx` | List ERC-1155 transfers |
| `etherscan account tokenbalance` | Get ERC-20 token balance |
| `etherscan account tokenbalancehistory` | Get token balance at block |
| `etherscan account tokennfttx` | List ERC-721 transfers |
| `etherscan account tokentx` | List ERC-20 transfers |
| `etherscan account txlist` | List normal transactions |
| `etherscan account txlistinternal` | List internal transactions |
| `etherscan account txnbridge` | List bridge transactions |
| `etherscan account txsBeaconWithdrawal` | List beacon withdrawals |

### `etherscan account addresstokenbalance`

```text
Usage: etherscan account addresstokenbalance <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account addresstokennftbalance`

```text
Usage: etherscan account addresstokennftbalance <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account addresstokennftinventory`

```text
Usage: etherscan account addresstokennftinventory <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--contractaddress` | NFT contract | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account balance`

```text
Usage: etherscan account balance <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan account balancehistory`

```text
Usage: etherscan account balancehistory <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--blockno` | block number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan account balancemulti`

```text
Usage: etherscan account balancemulti <addr1,addr2,...> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | comma-separated addresses | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | comma-separated addresses | `-` |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan account fundedby`

```text
Usage: etherscan account fundedby <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan account getdeposittxs`

```text
Usage: etherscan account getdeposittxs <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |
| `--sort` | asc or desc | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account getminedblocks`

```text
Usage: etherscan account getminedblocks <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--blocktype` | blocks or uncles | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account getwithdrawaltxs`

```text
Usage: etherscan account getwithdrawaltxs <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |
| `--sort` | asc or desc | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account token1155tx`

```text
Usage: etherscan account token1155tx <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination. Supports advanced `--from`, `--to`, and `--fromto-opr` filtering.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--contractaddress` | token contract | `-` |
| `--endblock` | end block | `-` |
| `--from` | filter by sender address | `-` |
| `--fromto-opr` | combine from/to: and or or | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |
| `--sort` | asc or desc | `-` |
| `--startblock` | start block | `-` |
| `--to` | filter by recipient address | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account tokenbalance`

```text
Usage: etherscan account tokenbalance <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--contractaddress` | token contract | `-` |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan account tokenbalancehistory`

```text
Usage: etherscan account tokenbalancehistory <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--blockno` | block number | `-` |
| `--contractaddress` | token contract | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan account tokennfttx`

```text
Usage: etherscan account tokennfttx <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination. Supports advanced `--from`, `--to`, and `--fromto-opr` filtering.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--contractaddress` | token contract | `-` |
| `--endblock` | end block | `-` |
| `--from` | filter by sender address | `-` |
| `--fromto-opr` | combine from/to: and or or | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |
| `--sort` | asc or desc | `-` |
| `--startblock` | start block | `-` |
| `--to` | filter by recipient address | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account tokentx`

```text
Usage: etherscan account tokentx <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination. Supports advanced `--from`, `--to`, and `--fromto-opr` filtering.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--contractaddress` | token contract | `-` |
| `--endblock` | end block | `-` |
| `--from` | filter by sender address | `-` |
| `--fromto-opr` | combine from/to: and or or | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |
| `--sort` | asc or desc | `-` |
| `--startblock` | start block | `-` |
| `--to` | filter by recipient address | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account txlist`

```text
Usage: etherscan account txlist <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination. Supports advanced `--from`, `--to`, and `--fromto-opr` filtering.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--endblock` | end block | `-` |
| `--from` | filter by sender address | `-` |
| `--fromto-opr` | combine from/to: and or or | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |
| `--sort` | asc or desc | `-` |
| `--startblock` | start block | `-` |
| `--to` | filter by recipient address | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account txlistinternal`

```text
Usage: etherscan account txlistinternal [flags]
```

Notes: Supports `--all` automatic pagination. Supports advanced `--from`, `--to`, and `--fromto-opr` filtering.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--endblock` | end block | `-` |
| `--from` | filter by sender address | `-` |
| `--fromto-opr` | combine from/to: and or or | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |
| `--sort` | asc or desc | `-` |
| `--startblock` | start block | `-` |
| `--to` | filter by recipient address | `-` |
| `--txhash` | transaction hash | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account txnbridge`

```text
Usage: etherscan account txnbridge <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan account txsBeaconWithdrawal`

```text
Usage: etherscan account txsBeaconWithdrawal <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Supports `--all` automatic pagination. Ethereum mainnet only.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--endblock` | end block | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |
| `--sort` | asc or desc | `-` |
| `--startblock` | start block | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

## `etherscan contract`

```text
Usage: etherscan contract
```

| Subcommand | Description |
| --- | --- |
| `etherscan contract check-proxy` | Check proxy verification |
| `etherscan contract getabi` | Get contract ABI |
| `etherscan contract getcontractcreation` | Get contract creation data |
| `etherscan contract getsourcecode` | Get contract source metadata |
| `etherscan contract verify` | Submit source verification |
| `etherscan contract verify-proxy` | Submit proxy verification |
| `etherscan contract verify-status` | Check verification status |

### `etherscan contract check-proxy`

```text
Usage: etherscan contract check-proxy <guid> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `guid` | verification GUID | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--guid` | verification GUID | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan contract getabi`

```text
Usage: etherscan contract getabi <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan contract getcontractcreation`

```text
Usage: etherscan contract getcontractcreation <addr1,...> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `contractaddresses` | comma-separated addresses | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--contractaddresses` | comma-separated addresses | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan contract getsourcecode`

```text
Usage: etherscan contract getsourcecode <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan contract verify`

```text
Usage: etherscan contract verify <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `contractaddress` | address | Yes |

Notes: Prompts for confirmation unless `--yes` is passed.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--codeformat` | code format | `-` |
| `--compilerversion` | compiler version | `-` |
| `--constructorArguments` | constructor args | `-` |
| `--contractaddress` | address | `-` |
| `--contractname` | contract name | `-` |
| `--evmVersion` | EVM version | `-` |
| `--file` | read source/payload content from file | `-` |
| `--licenseType` | license type | `-` |
| `--optimizationUsed` | optimization flag | `-` |
| `--runs` | optimizer runs | `-` |
| `--source-code` | source code or --file content | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--yes`

### `etherscan contract verify-proxy`

```text
Usage: etherscan contract verify-proxy <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Notes: Prompts for confirmation unless `--yes` is passed.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--expectedimplementation` | implementation address | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--yes`

### `etherscan contract verify-status`

```text
Usage: etherscan contract verify-status <guid> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `guid` | verification GUID | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--guid` | verification GUID | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

## `etherscan transaction`

```text
Usage: etherscan transaction
```

| Subcommand | Description |
| --- | --- |
| `etherscan transaction receipt-status` | Get transaction receipt status |
| `etherscan transaction status` | Get transaction execution status |

### `etherscan transaction receipt-status`

```text
Usage: etherscan transaction receipt-status <txhash> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `txhash` | transaction hash | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--txhash` | transaction hash | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan transaction status`

```text
Usage: etherscan transaction status <txhash> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `txhash` | transaction hash | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--txhash` | transaction hash | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

## `etherscan block`

```text
Usage: etherscan block
```

| Subcommand | Description |
| --- | --- |
| `etherscan block bytime` | Find block by timestamp |
| `etherscan block countdown` | Get block countdown |
| `etherscan block reward` | Get block reward |
| `etherscan block txcount` | Get block transaction count |

### `etherscan block bytime`

```text
Usage: etherscan block bytime <timestamp> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `timestamp` | unix timestamp | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--closest` | before or after | `-` |
| `--timestamp` | unix timestamp | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan block countdown`

```text
Usage: etherscan block countdown <blockno> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `blockno` | block number | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--blockno` | block number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan block reward`

```text
Usage: etherscan block reward <blockno> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `blockno` | block number | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--blockno` | block number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan block txcount`

```text
Usage: etherscan block txcount <blockno> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `blockno` | block number | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--blockno` | block number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

## `etherscan logs`

```text
Usage: etherscan logs
```

| Subcommand | Description |
| --- | --- |
| `etherscan logs get` | Query event logs |

### `etherscan logs get`

```text
Usage: etherscan logs get [flags]
```

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | contract address | `-` |
| `--from-block` | from block | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |
| `--to-block` | to block | `-` |
| `--topic0` | topic0 | `-` |
| `--topic0-1-opr` | topic operator | `-` |
| `--topic0-2-opr` | topic operator | `-` |
| `--topic0-3-opr` | topic operator | `-` |
| `--topic1` | topic1 | `-` |
| `--topic1-2-opr` | topic operator | `-` |
| `--topic1-3-opr` | topic operator | `-` |
| `--topic2` | topic2 | `-` |
| `--topic2-3-opr` | topic operator | `-` |
| `--topic3` | topic3 | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

## `etherscan gastracker`

```text
Usage: etherscan gastracker
```

| Subcommand | Description |
| --- | --- |
| `etherscan gastracker estimate` | Estimate gas confirmation time |
| `etherscan gastracker oracle` | Get gas oracle |

### `etherscan gastracker estimate`

```text
Usage: etherscan gastracker estimate [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--gasprice` | gas price in wei | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan gastracker oracle`

```text
Usage: etherscan gastracker oracle
```

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

## `etherscan token`

```text
Usage: etherscan token
```

| Subcommand | Description |
| --- | --- |
| `etherscan token info` | Get token metadata |
| `etherscan token tokenholdercount` | Get token holder count |
| `etherscan token tokenholderlist` | List token holders |
| `etherscan token tokenlist` | List tokens |
| `etherscan token topholders` | Get top holders |

### `etherscan token info`

```text
Usage: etherscan token info <contract> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `contractaddress` | token contract | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--contractaddress` | token contract | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan token tokenholdercount`

```text
Usage: etherscan token tokenholdercount <contract> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `contractaddress` | token contract | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--contractaddress` | token contract | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan token tokenholderlist`

```text
Usage: etherscan token tokenholderlist <contract> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `contractaddress` | token contract | Yes |

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--contractaddress` | token contract | `-` |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan token tokenlist`

```text
Usage: etherscan token tokenlist [flags]
```

Notes: Supports `--all` automatic pagination.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--offset` | page size | `-` |
| `--page` | page number | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--all`, `--max-pages`

### `etherscan token topholders`

```text
Usage: etherscan token topholders <contract> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `contractaddress` | token contract | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--contractaddress` | token contract | `-` |
| `--offset` | limit | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

## `etherscan stats`

```text
Usage: etherscan stats
```

| Subcommand | Description |
| --- | --- |
| `etherscan stats chainsize` | Get chain size |
| `etherscan stats dailyavgblocksize` | Get dailyavgblocksize series |
| `etherscan stats dailyavgblocktime` | Get dailyavgblocktime series |
| `etherscan stats dailyavggaslimit` | Get dailyavggaslimit series |
| `etherscan stats dailyavggasprice` | Get dailyavggasprice series |
| `etherscan stats dailyavghashrate` | Get dailyavghashrate series |
| `etherscan stats dailyavgnetdifficulty` | Get dailyavgnetdifficulty series |
| `etherscan stats dailyblkcount` | Get dailyblkcount series |
| `etherscan stats dailyblockrewards` | Get dailyblockrewards series |
| `etherscan stats dailyensregister` | Get dailyensregister series |
| `etherscan stats dailygasused` | Get dailygasused series |
| `etherscan stats dailynetutilization` | Get dailynetutilization series |
| `etherscan stats dailynewaddress` | Get dailynewaddress series |
| `etherscan stats dailytx` | Get dailytx series |
| `etherscan stats dailytxnfee` | Get dailytxnfee series |
| `etherscan stats dailyuncleblkcount` | Get dailyuncleblkcount series |
| `etherscan stats ethdailymarketcap` | Get ethdailymarketcap series |
| `etherscan stats ethdailyprice` | Get ethdailyprice series |
| `etherscan stats ethprice` | Get ETH price |
| `etherscan stats ethsupply` | Get ETH supply |
| `etherscan stats ethsupply2` | Get extended ETH supply |
| `etherscan stats nodecount` | Get node count |
| `etherscan stats nodecounthistory` | Get nodecounthistory series |
| `etherscan stats tokensupply` | Get token supply |
| `etherscan stats tokensupplyhistory` | Get token supply at block |

### `etherscan stats chainsize`

```text
Usage: etherscan stats chainsize [flags]
```

Notes: Ethereum mainnet only.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--clienttype` | client type | `-` |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |
| `--syncmode` | sync mode | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyavgblocksize`

```text
Usage: etherscan stats dailyavgblocksize [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyavgblocktime`

```text
Usage: etherscan stats dailyavgblocktime [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyavggaslimit`

```text
Usage: etherscan stats dailyavggaslimit [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyavggasprice`

```text
Usage: etherscan stats dailyavggasprice [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyavghashrate`

```text
Usage: etherscan stats dailyavghashrate [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyavgnetdifficulty`

```text
Usage: etherscan stats dailyavgnetdifficulty [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyblkcount`

```text
Usage: etherscan stats dailyblkcount [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyblockrewards`

```text
Usage: etherscan stats dailyblockrewards [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyensregister`

```text
Usage: etherscan stats dailyensregister [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailygasused`

```text
Usage: etherscan stats dailygasused [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailynetutilization`

```text
Usage: etherscan stats dailynetutilization [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailynewaddress`

```text
Usage: etherscan stats dailynewaddress [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailytx`

```text
Usage: etherscan stats dailytx [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailytxnfee`

```text
Usage: etherscan stats dailytxnfee [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats dailyuncleblkcount`

```text
Usage: etherscan stats dailyuncleblkcount [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats ethdailymarketcap`

```text
Usage: etherscan stats ethdailymarketcap [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats ethdailyprice`

```text
Usage: etherscan stats ethdailyprice [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats ethprice`

```text
Usage: etherscan stats ethprice
```

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats ethsupply`

```text
Usage: etherscan stats ethsupply
```

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats ethsupply2`

```text
Usage: etherscan stats ethsupply2
```

Notes: Ethereum mainnet only.

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats nodecount`

```text
Usage: etherscan stats nodecount
```

Notes: Ethereum mainnet only.

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats nodecounthistory`

```text
Usage: etherscan stats nodecounthistory [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--enddate` | end date | `-` |
| `--sort` | asc or desc | `-` |
| `--startdate` | start date | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats tokensupply`

```text
Usage: etherscan stats tokensupply <contract> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `contractaddress` | token contract | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--contractaddress` | token contract | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan stats tokensupplyhistory`

```text
Usage: etherscan stats tokensupplyhistory <contract> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `contractaddress` | token contract | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--blockno` | block number | `-` |
| `--contractaddress` | token contract | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

## `etherscan proxy`

```text
Usage: etherscan proxy
```

| Subcommand | Description |
| --- | --- |
| `etherscan proxy eth_blockNumber` | Get latest block number |
| `etherscan proxy eth_call` | Execute eth_call |
| `etherscan proxy eth_estimateGas` | Estimate gas |
| `etherscan proxy eth_gasPrice` | Get gas price |
| `etherscan proxy eth_getBlockByNumber` | Get block by number |
| `etherscan proxy eth_getBlockTransactionCountByNumber` | Get block tx count |
| `etherscan proxy eth_getCode` | Get code |
| `etherscan proxy eth_getStorageAt` | Get storage |
| `etherscan proxy eth_getTransactionByBlockNumberAndIndex` | Get transaction by block/index |
| `etherscan proxy eth_getTransactionByHash` | Get transaction by hash |
| `etherscan proxy eth_getTransactionCount` | Get account nonce |
| `etherscan proxy eth_getTransactionReceipt` | Get receipt |
| `etherscan proxy eth_getUncleByBlockNumberAndIndex` | Get uncle by block/index |
| `etherscan proxy eth_sendRawTransaction` | Broadcast signed transaction |

### `etherscan proxy eth_blockNumber`

```text
Usage: etherscan proxy eth_blockNumber
```

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_call`

```text
Usage: etherscan proxy eth_call [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--data` | call data | `-` |
| `--tag` | block tag | `-` |
| `--to` | contract address | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_estimateGas`

```text
Usage: etherscan proxy eth_estimateGas [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--data` | call data | `-` |
| `--gas` | gas | `-` |
| `--gas-price` | gas price | `-` |
| `--to` | to address | `-` |
| `--value` | value | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_gasPrice`

```text
Usage: etherscan proxy eth_gasPrice
```

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_getBlockByNumber`

```text
Usage: etherscan proxy eth_getBlockByNumber [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--boolean` | include tx objects | `-` |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_getBlockTransactionCountByNumber`

```text
Usage: etherscan proxy eth_getBlockTransactionCountByNumber [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_getCode`

```text
Usage: etherscan proxy eth_getCode <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_getStorageAt`

```text
Usage: etherscan proxy eth_getStorageAt <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--position` | storage position | `-` |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_getTransactionByBlockNumberAndIndex`

```text
Usage: etherscan proxy eth_getTransactionByBlockNumberAndIndex [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--index` | transaction index | `-` |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_getTransactionByHash`

```text
Usage: etherscan proxy eth_getTransactionByHash <txhash> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `txhash` | transaction hash | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--txhash` | transaction hash | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_getTransactionCount`

```text
Usage: etherscan proxy eth_getTransactionCount <address> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `address` | address | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--address` | address | `-` |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_getTransactionReceipt`

```text
Usage: etherscan proxy eth_getTransactionReceipt <txhash> [flags]
```

| Argument | Description | Required |
| --- | --- | --- |
| `txhash` | transaction hash | Yes |

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--txhash` | transaction hash | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_getUncleByBlockNumberAndIndex`

```text
Usage: etherscan proxy eth_getUncleByBlockNumberAndIndex [flags]
```

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--index` | uncle index | `-` |
| `--tag` | block tag | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`

### `etherscan proxy eth_sendRawTransaction`

```text
Usage: etherscan proxy eth_sendRawTransaction [flags]
```

Notes: Prompts for confirmation unless `--yes` is passed.

Options:

| Flag | Description | Default |
| --- | --- | --- |
| `--hex` | signed transaction hex | `-` |

Global flags: `--chain`, `--output`/`--json`/`--csv`, `--api-key`, `--base-url`, `--timeout`, `--rate-limit`, `--yes`

