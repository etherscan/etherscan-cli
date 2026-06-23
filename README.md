# Etherscan CLI

`etherscan cli` is a cross-platform Go CLI for the public Etherscan V2 API.

Sample commands:

```powershell
etherscan account balance 0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe
etherscan --chain base account balance 0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe
etherscan account txlist 0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe --page 1 --offset 5
etherscan gas oracle
etherscan stats ethprice
etherscan proxy eth_blockNumber
```

```sh
etherscan account txlist 0xSender --to 0xRecipient --fromto-opr and
etherscan account tokentx 0xAddr --from 0xA --to 0xB --fromto-opr or
```

## Login

```sh
etherscan login
etherscan whoami
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


The CLI never accepts private keys or seed phrases.
