# Etherscan CLI

A cross-platform Go CLI for the Etherscan V2 API.

`etherscan` wraps the API surface described in the official Etherscan docs, with commands for accounts, tokens, contracts, logs, gas, stats, address name tags, proxy/RPC-style endpoints, API limits, and multi-chain queries.

## Installation

When release archives are published, download the matching build for your platform from [GitHub Releases](https://github.com/etherscan/etherscan-cli/releases).

## Quickstart

```sh
etherscan login
etherscan whoami
etherscan account balance 0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe
etherscan --chain base account txlist 0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe --page 1 --offset 5
etherscan account txlist 0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe --json
```

## Common Workflows

Auth and config:

```sh
etherscan login
etherscan whoami
etherscan logout
etherscan config list
etherscan config set default_chain=base
```

Chains:

```sh
etherscan chains list
etherscan --chain base account balance 0x...
etherscan --chain 8453 gastracker oracle
```

Accounts and tokens:

```sh
etherscan account balance 0x...
etherscan account txlist 0x... --page 1 --offset 10
etherscan account tokentx 0x... --csv
etherscan token info 0x...
etherscan token tokenholderlist 0x... --page 1 --offset 10
```

Contracts and logs:

```sh
etherscan contract getabi 0x...
etherscan contract getsourcecode 0x...
etherscan contract verify 0x... --file Contract.sol --codeformat solidity-single-file --contractname Contract --compilerversion v0.8.24+commit.e11b9ed9
etherscan logs get --address 0x... --from-block 0 --to-block latest
```

Address name tags (Pro Plus):

```sh
etherscan nametag getaddresstag 0x...,0x...
```

API usage:

```sh
etherscan apilimit
```

## Full Documentation

Use CLI help for exact local syntax and flags:

```sh
etherscan --help
etherscan account --help
etherscan account txlist --help
```

CLI groups generally mirror Etherscan API modules, and command names generally mirror endpoint actions where practical.

For API behavior, parameter meanings, supported chains, rate limits, and errors, use the official Etherscan documentation:

- [Etherscan docs](https://docs.etherscan.io/)
- [Etherscan docs index for LLMs](https://docs.etherscan.io/llms.txt)
- [Supported Chains](https://docs.etherscan.io/supported-chains.md)
- [Rate Limits](https://docs.etherscan.io/resources/rate-limits.md)
- [Common Error Messages](https://docs.etherscan.io/resources/common-error-messages.md)

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

Environment variables:

| Variable | Purpose |
| --- | --- |
| `ETHERSCAN_API_KEY` | API key |
| `ETHERSCAN_CHAIN` | Default chain name or chain ID |
| `ETHERSCAN_BASE_URL` | API base URL |

## Safety

The CLI never asks for private keys or seed phrases.

Sensitive submit actions require confirmation unless `--yes` is passed. `proxy eth_sendRawTransaction` broadcasts an already-signed transaction only.
