package chains

import (
	"fmt"
	"sort"
	"strings"
)

type Chain struct {
	ID       string
	Name     string
	Aliases  []string
	Explorer string
	Symbol   string
	Testnet  bool
}

// (https://api.etherscan.io/v2/chainlist). Keep in sync when Etherscan adds chains.
var registry = []Chain{
	{"1", "ethereum", []string{"eth", "mainnet", "ethereum-mainnet"}, "https://etherscan.io", "ETH", false},
	{"11155111", "sepolia", []string{"ethereum-sepolia"}, "https://sepolia.etherscan.io", "ETH", true},
	{"560048", "hoodi", []string{"hoodi-testnet"}, "https://hoodi.etherscan.io", "ETH", true},
	{"56", "bsc", []string{"bnb", "binance", "bnb-smart-chain"}, "https://bscscan.com", "BNB", false},
	{"97", "bsc-testnet", []string{"bnb-testnet"}, "https://testnet.bscscan.com", "BNB", true},
	{"137", "polygon", []string{"matic", "pol"}, "https://polygonscan.com", "POL", false},
	{"80002", "polygon-amoy", []string{"amoy"}, "https://amoy.polygonscan.com", "POL", true},
	{"8453", "base", []string{"base-mainnet"}, "https://basescan.org", "ETH", false},
	{"84532", "base-sepolia", []string{"basesepolia"}, "https://sepolia.basescan.org", "ETH", true},
	{"42161", "arbitrum", []string{"arbitrum-one", "arb"}, "https://arbiscan.io", "ETH", false},
	{"421614", "arbitrum-sepolia", []string{"arb-sepolia"}, "https://sepolia.arbiscan.io", "ETH", true},
	{"59144", "linea", []string{"linea-mainnet"}, "https://lineascan.build", "ETH", false},
	{"59141", "linea-sepolia", []string{"linea-testnet"}, "https://sepolia.lineascan.build", "ETH", true},
	{"81457", "blast", []string{"blast-mainnet"}, "https://blastscan.io", "ETH", false},
	{"168587773", "blast-sepolia", []string{"blast-testnet"}, "https://sepolia.blastscan.io", "ETH", true},
	{"10", "optimism", []string{"op", "op-mainnet"}, "https://optimistic.etherscan.io", "ETH", false},
	{"11155420", "optimism-sepolia", []string{"op-sepolia"}, "https://sepolia-optimism.etherscan.io", "ETH", true},
	{"43114", "avalanche", []string{"avax", "avalanche-c-chain"}, "https://snowscan.xyz", "AVAX", false},
	{"43113", "avalanche-fuji", []string{"fuji"}, "https://testnet.snowscan.xyz", "AVAX", true},
	{"199", "bttc", []string{"bittorrent", "bittorrent-chain"}, "https://bttcscan.com", "BTT", false},
	{"1029", "bttc-testnet", []string{"bittorrent-testnet"}, "https://testnet.bttcscan.com", "BTT", true},
	{"42220", "celo", []string{"celo-mainnet"}, "https://celoscan.io", "CELO", false},
	{"11142220", "celo-sepolia", []string{"celo-testnet"}, "https://sepolia.celoscan.io", "CELO", true},
	{"252", "fraxtal", []string{"frax"}, "https://fraxscan.com", "frxETH", false},
	{"2523", "fraxtal-hoodi", []string{"fraxtal-testnet"}, "https://hoodi.fraxscan.com", "frxETH", true},
	{"100", "gnosis", []string{"xdai"}, "https://gnosisscan.io", "xDAI", false},
	{"5000", "mantle", []string{"mantle-mainnet"}, "https://mantlescan.xyz", "MNT", false},
	{"5003", "mantle-sepolia", []string{"mantle-testnet"}, "https://sepolia.mantlescan.xyz", "MNT", true},
	{"4352", "memecore", nil, "https://memecorescan.io", "", false},
	{"43522", "memecore-testnet", []string{"memecore-insectarium"}, "https://testnet.memecorescan.io", "", true},
	{"1284", "moonbeam", nil, "https://moonbeam.moonscan.io", "GLMR", false},
	{"1285", "moonriver", nil, "https://moonriver.moonscan.io", "MOVR", false},
	{"1287", "moonbase-alpha", []string{"moonbase"}, "https://moonbase.moonscan.io", "DEV", true},
	{"204", "opbnb", nil, "https://opbnb.bscscan.com", "BNB", false},
	{"5611", "opbnb-testnet", nil, "https://opbnb-testnet.bscscan.com", "BNB", true},
	{"167000", "taiko", nil, "https://taikoscan.io", "ETH", false},
	{"167013", "taiko-hoodi", []string{"taiko-testnet"}, "https://hoodi.taikoscan.io", "ETH", true},
	{"50", "xdc", nil, "https://xdcscan.com", "XDC", false},
	{"51", "xdc-apothem", []string{"xdc-testnet", "apothem"}, "https://testnet.xdcscan.com", "XDC", true},
	{"33139", "apechain", []string{"ape"}, "https://apescan.io", "APE", false},
	{"33111", "apechain-curtis", []string{"apechain-testnet", "curtis"}, "https://curtis.apescan.io", "APE", true},
	{"480", "world", []string{"worldchain"}, "https://worldscan.org", "ETH", false},
	{"4801", "world-sepolia", []string{"world-testnet"}, "https://sepolia.worldscan.org", "ETH", true},
	{"146", "sonic", nil, "https://sonicscan.org", "S", false},
	{"14601", "sonic-testnet", nil, "https://testnet.sonicscan.org", "S", true},
	{"130", "unichain", []string{"uni"}, "https://uniscan.xyz", "ETH", false},
	{"1301", "unichain-sepolia", []string{"unichain-testnet"}, "https://sepolia.uniscan.xyz", "ETH", true},
	{"2741", "abstract", nil, "https://abscan.org", "ETH", false},
	{"11124", "abstract-sepolia", []string{"abstract-testnet"}, "https://sepolia.abscan.org", "ETH", true},
	{"80094", "berachain", []string{"bera"}, "https://berascan.com", "BERA", false},
	{"80069", "berachain-bepolia", []string{"berachain-testnet", "bepolia"}, "https://testnet.berascan.com", "BERA", true},
	{"143", "monad", nil, "https://monadscan.com", "MON", false},
	{"10143", "monad-testnet", nil, "https://testnet.monadscan.com", "MON", true},
	{"999", "hyperevm", []string{"hyper"}, "https://hyperevmscan.io", "HYPE", false},
	{"747474", "katana", nil, "https://katanascan.com", "", false},
	{"737373", "katana-bokuto", []string{"katana-testnet", "bokuto"}, "https://bokuto.katanascan.com", "", true},
	{"1329", "sei", nil, "https://seiscan.io", "SEI", false},
	{"1328", "sei-testnet", nil, "https://testnet.seiscan.io", "SEI", true},
	{"988", "stable", nil, "https://stablescan.xyz", "", false},
	{"2201", "stable-testnet", nil, "https://testnet.stablescan.xyz", "", true},
	{"9745", "plasma", nil, "https://plasmascan.to", "", false},
	{"9746", "plasma-testnet", nil, "https://testnet.plasmascan.to", "", true},
	{"4326", "megaeth", []string{"mega"}, "https://mega.etherscan.io", "ETH", false},
	{"6343", "megaeth-testnet", []string{"mega-testnet"}, "https://testnet-mega.etherscan.io", "ETH", true},
}

func Resolve(input string) (Chain, error) {
	if strings.TrimSpace(input) == "" {
		input = "ethereum"
	}
	needle := strings.ToLower(strings.TrimSpace(input))
	for _, chain := range registry {
		if chain.ID == needle || chain.Name == needle {
			return chain, nil
		}
		for _, alias := range chain.Aliases {
			if alias == needle {
				return chain, nil
			}
		}
	}
	return Chain{}, fmt.Errorf("unknown chain %q", input)
}

func All() []Chain {
	out := append([]Chain(nil), registry...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func IsMainnetID(id string) bool {
	return id == "1"
}
