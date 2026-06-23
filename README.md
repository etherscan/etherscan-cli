# etherscan

`etherscan` is a cross-platform Go CLI for the public Etherscan V2 API.

The command tree mirrors the API shape:

```powershell
etherscan account balance 0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe
etherscan --chain base account balance 0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe
etherscan account txlist 0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe --page 1 --offset 5
etherscan account txlist 0xSender --to 0xRecipient --fromto-opr and
etherscan gas oracle
etherscan stats ethprice
etherscan proxy eth_blockNumber
```

## Advanced filters

The transfer-listing commands (`txlist`, `txlistinternal`, `tokentx`, `tokennfttx`,
`token1155tx`) accept optional `--from` / `--to` address filters. When either is set,
`--fromto-opr` is required and must be `and` or `or`; `and` requires both `--from` and
`--to`.

```sh
etherscan account txlist 0xSender --to 0xRecipient --fromto-opr and
etherscan account tokentx 0xAddr --from 0xA --to 0xB --fromto-opr or
```

## Build

Requires Go 1.22+.

```sh
go mod tidy
go build ./cmd/etherscan
```

## Auth

Login stores the API key in the OS keyring where available, falling back to
`~/.etherscan/config.toml` with `0600` permissions.

```sh
etherscan login
etherscan whoami
```

API key precedence:

1. `ETHERSCAN_API_KEY`
2. saved login key from the OS keyring or config fallback

Default settings can be changed with `etherscan config set`:

```sh
etherscan config list
etherscan config set default_chain=base
etherscan config set default_output=json
```

## Output

Tables are the default. Use JSON or CSV for scripts:

```sh
etherscan account balance 0x... --json
etherscan account txlist 0x... --csv
etherscan account txlist 0x... --all --csv
```

Common flags:

| Flag | Purpose |
|---|---|
| `--chain <name-or-id>` | Query another Etherscan-supported chain |
| `--json` | Print raw JSON |
| `--csv` | Print CSV for list-style results |
| `--all` | Auto-paginate list commands |

## Sensitive Commands

Submit-style commands prompt by default:

```sh
etherscan proxy eth_sendRawTransaction --hex 0x...
```

The CLI never accepts private keys or seed phrases.
