package cli

type ParamKind string

const (
	KindString          ParamKind = "string"
	KindAddress         ParamKind = "address"
	KindAddresses       ParamKind = "addresses"
	KindTxHash          ParamKind = "txhash"
	KindUint            ParamKind = "uint"
	KindDate            ParamKind = "date"
	KindSort            ParamKind = "sort"
	KindHex             ParamKind = "hex"
	KindZeroOne         ParamKind = "zero-one"
	KindConstructorArgs ParamKind = "constructor-args"
	KindLicense         ParamKind = "license"
)

type ParamSpec struct {
	Name     string
	Usage    string
	Kind     ParamKind
	Required bool
	Arg      bool
	MaxList  int
}

type EndpointSpec struct {
	Module             string
	Action             string
	Use                string
	Short              string
	Params             []ParamSpec
	Columns            []string
	Paginated          bool
	Post               bool
	Sensitive          bool
	NoRetry            bool
	MainnetOnly        bool
	AcceptsFile        bool
	FixedParams        map[string]string
	AllowedChainIDs    []string
	AllowedCodeFormats []string
	// AdvancedFilter enables the optional from/to/fromto_opr params and their
	// cross-field validation (see validateAdvancedFilter).
	AdvancedFilter bool
	// RequireOneOf lists params of which at least one must be set — the server
	// accepts several alternative filters (e.g. txlist: address XOR from/to)
	// rather than one always-required param.
	RequireOneOf []string
}

func endpoints() []EndpointSpec {
	commonList := []ParamSpec{p("startblock", "start block", KindUint), p("endblock", "end block", KindUint), p("page", "page number", KindUint), p("offset", "page size", KindUint), p("sort", "asc or desc (default asc)", KindSort)}
	account := []EndpointSpec{
		{Module: "account", Action: "balance", Use: "balance <address>", Short: "Get native balance", Params: []ParamSpec{argAddress("address"), p("tag", "block tag: latest, pending, earliest, or hex", KindString)}, Columns: []string{"account", "balance"}},
		{Module: "account", Action: "balancemulti", Use: "balancemulti <addr1,addr2,...>", Short: "Get native balances for multiple addresses", Params: []ParamSpec{argAddresses("address", 20), p("tag", "block tag: latest, pending, earliest, or hex", KindString)}, Columns: []string{"account", "balance"}},
		{Module: "account", Action: "txlist", Use: "txlist [address]", Short: "List normal transactions", Params: appendParams([]ParamSpec{optArg("address", "address", KindAddress)}, commonList, filterList()), Columns: txColumns(), Paginated: true, AdvancedFilter: true, RequireOneOf: []string{"address", "from", "to"}},
		{Module: "account", Action: "txlistinternal", Use: "txlistinternal", Short: "List internal transactions", Params: appendParams([]ParamSpec{p("address", "address", KindAddress), p("txhash", "transaction hash", KindTxHash)}, commonList, filterList()), Paginated: true, AdvancedFilter: true},
		{Module: "account", Action: "tokentx", Use: "tokentx [address]", Short: "List ERC-20 transfers", Params: appendParams([]ParamSpec{optArg("address", "address", KindAddress), p("contractaddress", "token contract", KindAddress)}, commonList, filterList()), Columns: tokenColumns(), Paginated: true, AdvancedFilter: true, RequireOneOf: []string{"address", "contractaddress", "from", "to"}},
		{Module: "account", Action: "tokennfttx", Use: "tokennfttx [address]", Short: "List ERC-721 transfers", Params: appendParams([]ParamSpec{optArg("address", "address", KindAddress), p("contractaddress", "token contract", KindAddress)}, commonList, filterList()), Columns: tokenColumns(), Paginated: true, AdvancedFilter: true, RequireOneOf: []string{"address", "contractaddress", "from", "to"}},
		{Module: "account", Action: "token1155tx", Use: "token1155tx [address]", Short: "List ERC-1155 transfers", Params: appendParams([]ParamSpec{optArg("address", "address", KindAddress), p("contractaddress", "token contract", KindAddress)}, commonList, filterList()), Columns: tokenColumns(), Paginated: true, AdvancedFilter: true, RequireOneOf: []string{"address", "contractaddress", "from", "to"}},
		{Module: "account", Action: "getminedblocks", Use: "getminedblocks <address>", Short: "List mined blocks", Params: []ParamSpec{argAddress("address"), p("blocktype", "blocks or uncles (default blocks)", KindString), p("page", "page number", KindUint), p("offset", "page size", KindUint)}, Paginated: true},
		{Module: "account", Action: "balancehistory", Use: "balancehistory <address>", Short: "Get native balance at block", Params: []ParamSpec{argAddress("address"), req("blockno", "block number", KindUint)}},
		{Module: "account", Action: "tokenbalance", Use: "tokenbalance <address>", Short: "Get ERC-20 token balance", Params: []ParamSpec{argAddress("address"), req("contractaddress", "token contract", KindAddress), p("tag", "block tag: latest, pending, earliest, or hex", KindString)}},
		{Module: "account", Action: "tokenbalancehistory", Use: "tokenbalancehistory <address>", Short: "Get token balance at block", Params: []ParamSpec{argAddress("address"), req("contractaddress", "token contract", KindAddress), req("blockno", "block number", KindUint)}},
		{Module: "account", Action: "addresstokenbalance", Use: "addresstokenbalance <address>", Short: "List ERC-20 holdings", Params: []ParamSpec{argAddress("address"), p("page", "page number", KindUint), p("offset", "page size", KindUint)}, Paginated: true},
		{Module: "account", Action: "addresstokennftbalance", Use: "addresstokennftbalance <address>", Short: "List NFT holdings", Params: []ParamSpec{argAddress("address"), p("page", "page number", KindUint), p("offset", "page size", KindUint)}, Paginated: true},
		{Module: "account", Action: "addresstokennftinventory", Use: "addresstokennftinventory <address>", Short: "List NFT inventory", Params: []ParamSpec{argAddress("address"), req("contractaddress", "NFT contract", KindAddress), p("page", "page number", KindUint), p("offset", "page size", KindUint)}, Paginated: true},
		{Module: "account", Action: "getdeposittxs", Use: "getdeposittxs <address>", Short: "List L2 deposits", Params: []ParamSpec{argAddress("address"), p("page", "page number", KindUint), p("offset", "page size", KindUint), p("sort", "asc or desc (default asc)", KindSort)}, Paginated: true},
		{Module: "account", Action: "getwithdrawaltxs", Use: "getwithdrawaltxs <address>", Short: "List L2 withdrawals", Params: []ParamSpec{argAddress("address"), p("page", "page number", KindUint), p("offset", "page size", KindUint), p("sort", "asc or desc (default asc)", KindSort)}, Paginated: true},
		{Module: "account", Action: "txsBeaconWithdrawal", Use: "txsBeaconWithdrawal <address>", Short: "List beacon withdrawals", Params: append([]ParamSpec{argAddress("address")}, commonList...), Paginated: true, MainnetOnly: true},
		{Module: "account", Action: "fundedby", Use: "fundedby <address>", Short: "Get likely funder", Params: []ParamSpec{argAddress("address")}},
		{Module: "account", Action: "txnbridge", Use: "txnbridge <address>", Short: "List bridge transactions", Params: []ParamSpec{argAddress("address"), p("page", "page number", KindUint), p("offset", "page size", KindUint)}, Paginated: true},
	}
	contract := []EndpointSpec{
		{Module: "contract", Action: "getabi", Use: "getabi <address>", Short: "Get contract ABI", Params: []ParamSpec{argAddress("address")}},
		{Module: "contract", Action: "getsourcecode", Use: "getsourcecode <address>", Short: "Get contract source metadata", Params: []ParamSpec{argAddress("address")}},
		{Module: "contract", Action: "getcontractcreation", Use: "getcontractcreation <addr1,...>", Short: "Get contract creation data", Params: []ParamSpec{argAddresses("contractaddresses", 5)}},
		{Module: "contract", Action: "verifysourcecode", Use: "verify <address>", Short: "Submit Solidity source verification", Params: []ParamSpec{argAddress("contractaddress"), req("sourceCode", "source code or --file content", KindString), req("codeformat", "source code format", KindString), req("contractname", "contract name", KindString), req("compilerversion", "compiler version", KindString), p("optimizationUsed", "optimization flag: 0 or 1", KindZeroOne), p("runs", "optimizer runs", KindUint), p("constructorArguments", "ABI-encoded constructor arguments", KindConstructorArgs), p("evmVersion", "EVM version", KindString), p("licenseType", "license type (1-14)", KindLicense)}, Post: true, Sensitive: true, NoRetry: true, AcceptsFile: true, AllowedCodeFormats: []string{"solidity-single-file", "solidity-standard-json-input", "vyper-json", "stylus"}},
		{Module: "contract", Action: "verifysourcecode", Use: "verify-zksync <address>", Short: "Submit Abstract zkSync-stack source verification", Params: []ParamSpec{argAddress("contractaddress"), req("sourceCode", "source code or --file content", KindString), req("codeformat", "solidity-single-file or solidity-standard-json-input", KindString), req("contractname", "contract name", KindString), req("compilerversion", "compiler version", KindString), req("zksolcVersion", "zkSolc compiler version", KindString), p("optimizationUsed", "optimization flag: 0 or 1", KindZeroOne), p("constructorArguments", "ABI-encoded constructor arguments", KindConstructorArgs)}, Post: true, Sensitive: true, NoRetry: true, AcceptsFile: true, AllowedChainIDs: []string{"2741", "11124"}, AllowedCodeFormats: []string{"solidity-single-file", "solidity-standard-json-input"}},
		{Module: "contract", Action: "verifysourcecode", Use: "verify-vyper <address>", Short: "Submit Vyper source verification", Params: []ParamSpec{argAddress("contractaddress"), req("sourceCode", "source code or --file content", KindString), req("contractname", "contract name", KindString), req("compilerversion", "compiler version", KindString), req("optimizationUsed", "optimization flag: 0 or 1", KindZeroOne), p("constructorArguments", "ABI-encoded constructor arguments", KindConstructorArgs)}, Post: true, Sensitive: true, NoRetry: true, AcceptsFile: true, FixedParams: map[string]string{"codeformat": "vyper-json"}, AllowedCodeFormats: []string{"vyper-json"}},
		{Module: "contract", Action: "verifysourcecode", Use: "verify-stylus <address>", Short: "Submit Stylus source verification", Params: []ParamSpec{argAddress("contractaddress"), req("sourceCode", "public Git repository URL", KindString), req("contractname", "contract name", KindString), req("compilerversion", "Stylus compiler version", KindString), p("licenseType", "license type (1-14)", KindLicense)}, Post: true, Sensitive: true, NoRetry: true, FixedParams: map[string]string{"codeformat": "stylus"}, AllowedChainIDs: []string{"42161", "421614"}, AllowedCodeFormats: []string{"stylus"}},
		{Module: "contract", Action: "verifyproxycontract", Use: "verify-proxy <address>", Short: "Submit proxy verification", Params: []ParamSpec{argAddress("address"), p("expectedimplementation", "implementation address", KindAddress)}, Post: true, Sensitive: true, NoRetry: true},
		{Module: "contract", Action: "checkverifystatus", Use: "verify-status <guid>", Short: "Check verification status", Params: []ParamSpec{arg("guid", "verification GUID", KindString)}},
		{Module: "contract", Action: "checkproxyverification", Use: "check-proxy <guid>", Short: "Check proxy verification", Params: []ParamSpec{arg("guid", "verification GUID", KindString)}},
	}
	transaction := []EndpointSpec{
		{Module: "transaction", Action: "getstatus", Use: "status <txhash>", Short: "Get transaction execution status", Params: []ParamSpec{arg("txhash", "transaction hash", KindTxHash)}},
		{Module: "transaction", Action: "gettxreceiptstatus", Use: "receipt-status <txhash>", Short: "Get transaction receipt status", Params: []ParamSpec{arg("txhash", "transaction hash", KindTxHash)}},
	}
	block := []EndpointSpec{
		{Module: "block", Action: "getblockreward", Use: "reward <blockno>", Short: "Get block reward", Params: []ParamSpec{arg("blockno", "block number", KindUint)}},
		{Module: "block", Action: "getblockcountdown", Use: "countdown <blockno>", Short: "Get block countdown", Params: []ParamSpec{arg("blockno", "block number", KindUint)}},
		{Module: "block", Action: "getblocktxnscount", Use: "txcount <blockno>", Short: "Get block transaction count", Params: []ParamSpec{arg("blockno", "block number", KindUint)}},
		{Module: "block", Action: "getblocknobytime", Use: "bytime <timestamp>", Short: "Find block by timestamp", Params: []ParamSpec{arg("timestamp", "unix timestamp", KindUint), p("closest", "before or after (default before)", KindString)}},
	}
	logs := []EndpointSpec{{Module: "logs", Action: "getLogs", Use: "get", Short: "Query event logs", Params: []ParamSpec{p("fromBlock", "from block", KindString), p("toBlock", "to block", KindString), p("address", "contract address", KindAddress), p("topic0", "topic0", KindHex), p("topic1", "topic1", KindHex), p("topic2", "topic2", KindHex), p("topic3", "topic3", KindHex), p("topic0_1_opr", "topic operator: and | or (default and)", KindString), p("topic1_2_opr", "topic operator: and | or (default and)", KindString), p("topic2_3_opr", "topic operator: and | or (default and)", KindString), p("topic0_2_opr", "topic operator: and | or (default and)", KindString), p("topic0_3_opr", "topic operator: and | or (default and)", KindString), p("topic1_3_opr", "topic operator: and | or (default and)", KindString), p("page", "page number", KindUint), p("offset", "page size", KindUint)}, Paginated: true}}
	gas := []EndpointSpec{{Module: "gastracker", Action: "gasoracle", Use: "oracle", Short: "Get gas oracle"}, {Module: "gastracker", Action: "gasestimate", Use: "estimate", Short: "Estimate gas confirmation time", Params: []ParamSpec{p("gasprice", "gas price in wei (default 2000000000)", KindUint)}}}
	token := []EndpointSpec{
		{Module: "token", Action: "tokeninfo", Use: "info <contract>", Short: "Get token metadata", Params: []ParamSpec{arg("contractaddress", "token contract", KindAddress)}},
		{Module: "token", Action: "tokenholderlist", Use: "tokenholderlist <contract>", Short: "List token holders", Params: []ParamSpec{arg("contractaddress", "token contract", KindAddress), p("page", "page number", KindUint), p("offset", "page size", KindUint)}, Paginated: true},
		{Module: "token", Action: "tokenholdercount", Use: "tokenholdercount <contract>", Short: "Get token holder count", Params: []ParamSpec{arg("contractaddress", "token contract", KindAddress)}},
		{Module: "token", Action: "topholders", Use: "topholders <contract>", Short: "Get top holders", Params: []ParamSpec{arg("contractaddress", "token contract", KindAddress), p("offset", "limit", KindUint)}},
	}
	nametag := []EndpointSpec{
		{Module: "nametag", Action: "getaddresstag", Use: "getaddresstag <addr1,addr2,...>", Short: "Get address name tags and metadata (Pro Plus)", Params: []ParamSpec{argAddresses("address", 100)}, Columns: []string{"address", "nametag", "labels", "reputation"}},
	}
	stats := statsEndpoints()
	proxy := proxyEndpoints()
	out := append(account, contract...)
	out = append(out, transaction...)
	out = append(out, block...)
	out = append(out, logs...)
	out = append(out, stats...)
	out = append(out, token...)
	out = append(out, gas...)
	out = append(out, nametag...)
	out = append(out, proxy...)
	out = append(out, EndpointSpec{Module: "getapilimit", Action: "getapilimit", Use: "apilimit", Short: "Show API credit usage", Columns: []string{"creditsUsed", "creditsAvailable", "creditLimit", "limitInterval", "intervalExpiryTimespan"}})
	return out
}

func statsEndpoints() []EndpointSpec {
	series := []string{"ethdailyprice", "dailytx", "dailynewaddress", "dailyavgblocksize", "dailyavgblocktime", "dailyavggasprice", "dailyavggaslimit", "dailygasused", "dailyblockrewards", "dailyblkcount", "dailytxnfee", "dailynetutilization", "dailyuncleblkcount", "dailyavghashrate", "dailyavgnetdifficulty", "dailyensregister", "nodecounthistory"}
	out := []EndpointSpec{
		{Module: "stats", Action: "ethsupply", Use: "ethsupply", Short: "Get ETH supply"},
		{Module: "stats", Action: "ethsupply2", Use: "ethsupply2", Short: "Get extended ETH supply", MainnetOnly: true},
		{Module: "stats", Action: "ethprice", Use: "ethprice", Short: "Get ETH price"},
		{Module: "stats", Action: "chainsize", Use: "chainsize", Short: "Get chain size", Params: []ParamSpec{p("startdate", "start date (yyyy-MM-dd)", KindDate), p("enddate", "end date (yyyy-MM-dd)", KindDate), p("clienttype", "client type: geth or parity", KindString), p("syncmode", "sync mode: default or archive", KindString), p("sort", "asc or desc (default asc)", KindSort)}, MainnetOnly: true},
		{Module: "stats", Action: "nodecount", Use: "nodecount", Short: "Get node count", MainnetOnly: true},
		{Module: "stats", Action: "tokensupply", Use: "tokensupply <contract>", Short: "Get token supply", Params: []ParamSpec{arg("contractaddress", "token contract", KindAddress)}},
		{Module: "stats", Action: "tokensupplyhistory", Use: "tokensupplyhistory <contract>", Short: "Get token supply at block", Params: []ParamSpec{arg("contractaddress", "token contract", KindAddress), req("blockno", "block number", KindUint)}},
	}
	for _, action := range series {
		out = append(out, EndpointSpec{Module: "stats", Action: action, Use: action, Short: "Get " + action + " series", Params: []ParamSpec{p("startdate", "start date (yyyy-MM-dd)", KindDate), p("enddate", "end date (yyyy-MM-dd)", KindDate), p("sort", "asc or desc (default asc)", KindSort)}})
	}
	return out
}

func proxyEndpoints() []EndpointSpec {
	return []EndpointSpec{
		{Module: "proxy", Action: "eth_blockNumber", Use: "eth_blockNumber", Short: "Get latest block number"},
		{Module: "proxy", Action: "eth_getBlockByNumber", Use: "eth_getBlockByNumber", Short: "Get block by number", Params: []ParamSpec{p("tag", "block tag: latest, pending, earliest, or hex", KindString), p("boolean", "include tx objects: true or false", KindString)}},
		{Module: "proxy", Action: "eth_getTransactionByHash", Use: "eth_getTransactionByHash <txhash>", Short: "Get transaction by hash", Params: []ParamSpec{arg("txhash", "transaction hash", KindTxHash)}},
		{Module: "proxy", Action: "eth_getTransactionByBlockNumberAndIndex", Use: "eth_getTransactionByBlockNumberAndIndex", Short: "Get transaction by block/index", Params: []ParamSpec{req("tag", "block tag: latest, pending, earliest, or hex", KindString), req("index", "transaction index", KindHex)}},
		{Module: "proxy", Action: "eth_getTransactionCount", Use: "eth_getTransactionCount <address>", Short: "Get account nonce", Params: []ParamSpec{argAddress("address"), p("tag", "block tag: latest, pending, earliest, or hex", KindString)}},
		{Module: "proxy", Action: "eth_getBlockTransactionCountByNumber", Use: "eth_getBlockTransactionCountByNumber", Short: "Get block tx count", Params: []ParamSpec{req("tag", "block tag: latest, pending, earliest, or hex", KindString)}},
		{Module: "proxy", Action: "eth_getUncleByBlockNumberAndIndex", Use: "eth_getUncleByBlockNumberAndIndex", Short: "Get uncle by block/index", Params: []ParamSpec{req("tag", "block tag: latest, pending, earliest, or hex", KindString), req("index", "uncle index", KindHex)}},
		{Module: "proxy", Action: "eth_sendRawTransaction", Use: "eth_sendRawTransaction", Short: "Broadcast signed transaction", Params: []ParamSpec{req("hex", "signed transaction hex", KindHex)}, Sensitive: true, NoRetry: true},
		{Module: "proxy", Action: "eth_call", Use: "eth_call", Short: "Execute eth_call", Params: []ParamSpec{req("to", "contract address", KindAddress), p("data", "call data", KindHex), p("tag", "block tag: latest, pending, earliest, or hex", KindString)}},
		{Module: "proxy", Action: "eth_estimateGas", Use: "eth_estimateGas", Short: "Estimate gas", Params: []ParamSpec{p("to", "to address", KindAddress), p("data", "call data", KindHex), p("value", "value", KindHex), p("gas", "gas", KindHex), p("gasPrice", "gas price", KindHex)}},
		{Module: "proxy", Action: "eth_getTransactionReceipt", Use: "eth_getTransactionReceipt <txhash>", Short: "Get receipt", Params: []ParamSpec{arg("txhash", "transaction hash", KindTxHash)}},
		{Module: "proxy", Action: "eth_getCode", Use: "eth_getCode <address>", Short: "Get code", Params: []ParamSpec{argAddress("address"), p("tag", "block tag: latest, pending, earliest, or hex", KindString)}},
		{Module: "proxy", Action: "eth_getStorageAt", Use: "eth_getStorageAt <address>", Short: "Get storage", Params: []ParamSpec{argAddress("address"), req("position", "storage position", KindHex), p("tag", "block tag: latest, pending, earliest, or hex", KindString)}},
		{Module: "proxy", Action: "eth_gasPrice", Use: "eth_gasPrice", Short: "Get gas price"},
	}
}

func p(name, usage string, kind ParamKind) ParamSpec {
	return ParamSpec{Name: name, Usage: usage, Kind: kind}
}
func req(name, usage string, kind ParamKind) ParamSpec {
	return ParamSpec{Name: name, Usage: usage, Kind: kind, Required: true}
}
func arg(name, usage string, kind ParamKind) ParamSpec {
	return ParamSpec{Name: name, Usage: usage, Kind: kind, Required: true, Arg: true}
}
func argAddress(name string) ParamSpec { return arg(name, "address", KindAddress) }

// optArg is a positional param that may be omitted (used with RequireOneOf when
// the server accepts alternative filters instead).
func optArg(name, usage string, kind ParamKind) ParamSpec {
	return ParamSpec{Name: name, Usage: usage, Kind: kind, Arg: true}
}

// filterList returns the optional "advanced filter" params shared by the account
// transfer-listing actions. In the API these are not separate endpoints, just
// extra optional params on txlist/txlistinternal/tokentx/tokennfttx/token1155tx.
func filterList() []ParamSpec {
	return []ParamSpec{
		p("from", "filter by sender address", KindAddress),
		p("to", "filter by recipient address", KindAddress),
		p("fromto_opr", "combine from/to filters: and | or (required with from/to)", KindString),
	}
}

// appendParams concatenates parameter groups into a fresh slice so the shared
// commonList/filterList backing arrays are never mutated across endpoints.
func appendParams(groups ...[]ParamSpec) []ParamSpec {
	var out []ParamSpec
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}
func argAddresses(name string, max int) ParamSpec {
	return ParamSpec{Name: name, Usage: "comma-separated addresses", Kind: KindAddresses, Required: true, Arg: true, MaxList: max}
}
func txColumns() []string {
	return []string{"blockNumber", "timeStamp", "hash", "from", "to", "value", "gas", "gasPrice", "isError"}
}
func tokenColumns() []string {
	return []string{"blockNumber", "timeStamp", "hash", "from", "to", "contractAddress", "tokenName", "tokenSymbol", "value"}
}
