package chains

import (
	"fmt"
	"strings"
)

type Chain struct {
	ID          string
	Name        string // stable CLI slug
	DisplayName string // official Etherscan supported-chains name
	Aliases     []string
	Explorer    string
	Symbol      string
	Testnet     bool
	FreeTier    bool
}

// (https://api.etherscan.io/v2/chainlist). Keep in sync when Etherscan adds chains.
var registry = []Chain{
	{"1", "ethereum", "Ethereum Mainnet", []string{"eth", "mainnet", "ethereum-mainnet"}, "https://etherscan.io", "ETH", false, true},
	{"11155111", "sepolia", "Sepolia Testnet", []string{"ethereum-sepolia"}, "https://sepolia.etherscan.io", "ETH", true, true},
	{"560048", "hoodi", "Hoodi Testnet", []string{"hoodi-testnet"}, "https://hoodi.etherscan.io", "ETH", true, true},
	{"56", "bsc", "BNB Smart Chain Mainnet", []string{"bnb", "binance", "bnb-smart-chain"}, "https://bscscan.com", "BNB", false, false},
	{"97", "bsc-testnet", "BNB Smart Chain Testnet", []string{"bnb-testnet"}, "https://testnet.bscscan.com", "BNB", true, false},
	{"137", "polygon", "Polygon Mainnet", []string{"matic", "pol"}, "https://polygonscan.com", "POL", false, true},
	{"80002", "polygon-amoy", "Polygon Amoy Testnet", []string{"amoy"}, "https://amoy.polygonscan.com", "POL", true, true},
	{"8453", "base", "Base Mainnet", []string{"base-mainnet"}, "https://basescan.org", "ETH", false, false},
	{"84532", "base-sepolia", "Base Sepolia Testnet", []string{"basesepolia"}, "https://sepolia.basescan.org", "ETH", true, false},
	{"42161", "arbitrum", "Arbitrum One Mainnet", []string{"arbitrum-one", "arb"}, "https://arbiscan.io", "ETH", false, true},
	{"421614", "arbitrum-sepolia", "Arbitrum Sepolia Testnet", []string{"arb-sepolia"}, "https://sepolia.arbiscan.io", "ETH", true, true},
	{"59144", "linea", "Linea Mainnet", []string{"linea-mainnet"}, "https://lineascan.build", "ETH", false, true},
	{"59141", "linea-sepolia", "Linea Sepolia Testnet", []string{"linea-testnet"}, "https://sepolia.lineascan.build", "ETH", true, true},
	{"81457", "blast", "Blast Mainnet", []string{"blast-mainnet"}, "https://blastscan.io", "ETH", false, true},
	{"168587773", "blast-sepolia", "Blast Sepolia Testnet", []string{"blast-testnet"}, "https://sepolia.blastscan.io", "ETH", true, true},
	{"10", "optimism", "OP Mainnet", []string{"op", "op-mainnet"}, "https://optimistic.etherscan.io", "ETH", false, false},
	{"11155420", "optimism-sepolia", "OP Sepolia Testnet", []string{"op-sepolia"}, "https://sepolia-optimism.etherscan.io", "ETH", true, false},
	{"43114", "avalanche", "Avalanche C-Chain", []string{"avax", "avalanche-c-chain"}, "https://snowscan.xyz", "AVAX", false, false},
	{"43113", "avalanche-fuji", "Avalanche Fuji Testnet", []string{"fuji"}, "https://testnet.snowscan.xyz", "AVAX", true, false},
	{"199", "bttc", "BitTorrent Chain Mainnet", []string{"bittorrent", "bittorrent-chain"}, "https://bttcscan.com", "BTT", false, true},
	{"1029", "bttc-testnet", "BitTorrent Chain Testnet", []string{"bittorrent-testnet"}, "https://testnet.bttcscan.com", "BTT", true, true},
	{"42220", "celo", "Celo Mainnet", []string{"celo-mainnet"}, "https://celoscan.io", "CELO", false, true},
	{"11142220", "celo-sepolia", "Celo Sepolia Testnet", []string{"celo-testnet"}, "https://sepolia.celoscan.io", "CELO", true, true},
	{"252", "fraxtal", "Fraxtal Mainnet", []string{"frax"}, "https://fraxscan.com", "frxETH", false, true},
	{"2523", "fraxtal-hoodi", "Fraxtal Hoodi Testnet", []string{"fraxtal-testnet"}, "https://hoodi.fraxscan.com", "frxETH", true, true},
	{"100", "gnosis", "Gnosis", []string{"xdai"}, "https://gnosisscan.io", "xDAI", false, true},
	{"5000", "mantle", "Mantle Mainnet", []string{"mantle-mainnet"}, "https://mantlescan.xyz", "MNT", false, true},
	{"5003", "mantle-sepolia", "Mantle Sepolia Testnet", []string{"mantle-testnet"}, "https://sepolia.mantlescan.xyz", "MNT", true, true},
	{"4352", "memecore", "Memecore Mainnet", nil, "https://memecorescan.io", "", false, true},
	{"43522", "memecore-testnet", "Memecore Insectarium Testnet", []string{"memecore-insectarium"}, "https://testnet.memecorescan.io", "", true, true},
	{"1284", "moonbeam", "Moonbeam Mainnet", nil, "https://moonbeam.moonscan.io", "GLMR", false, true},
	{"1285", "moonriver", "Moonriver Mainnet", nil, "https://moonriver.moonscan.io", "MOVR", false, true},
	{"1287", "moonbase-alpha", "Moonbase Alpha Testnet", []string{"moonbase"}, "https://moonbase.moonscan.io", "DEV", true, true},
	{"204", "opbnb", "opBNB Mainnet", nil, "https://opbnb.bscscan.com", "BNB", false, true},
	{"5611", "opbnb-testnet", "opBNB Testnet", nil, "https://opbnb-testnet.bscscan.com", "BNB", true, true},
	{"167000", "taiko", "Taiko Mainnet", nil, "https://taikoscan.io", "ETH", false, true},
	{"167013", "taiko-hoodi", "Taiko Hoodi", []string{"taiko-testnet"}, "https://hoodi.taikoscan.io", "ETH", true, true},
	{"50", "xdc", "XDC Mainnet", nil, "https://xdcscan.com", "XDC", false, true},
	{"51", "xdc-apothem", "XDC Apothem Testnet", []string{"xdc-testnet", "apothem"}, "https://testnet.xdcscan.com", "XDC", true, true},
	{"33139", "apechain", "ApeChain Mainnet", []string{"ape"}, "https://apescan.io", "APE", false, true},
	{"33111", "apechain-curtis", "ApeChain Curtis Testnet", []string{"apechain-testnet", "curtis"}, "https://curtis.apescan.io", "APE", true, true},
	{"480", "world", "World Mainnet", []string{"worldchain"}, "https://worldscan.org", "ETH", false, true},
	{"4801", "world-sepolia", "World Sepolia Testnet", []string{"world-testnet"}, "https://sepolia.worldscan.org", "ETH", true, true},
	{"146", "sonic", "Sonic Mainnet", nil, "https://sonicscan.org", "S", false, true},
	{"14601", "sonic-testnet", "Sonic Testnet", nil, "https://testnet.sonicscan.org", "S", true, true},
	{"130", "unichain", "Unichain Mainnet", []string{"uni"}, "https://uniscan.xyz", "ETH", false, true},
	{"1301", "unichain-sepolia", "Unichain Sepolia Testnet", []string{"unichain-testnet"}, "https://sepolia.uniscan.xyz", "ETH", true, true},
	{"2741", "abstract", "Abstract Mainnet", nil, "https://abscan.org", "ETH", false, true},
	{"11124", "abstract-sepolia", "Abstract Sepolia Testnet", []string{"abstract-testnet"}, "https://sepolia.abscan.org", "ETH", true, true},
	{"80094", "berachain", "Berachain Mainnet", []string{"bera"}, "https://berascan.com", "BERA", false, true},
	{"80069", "berachain-bepolia", "Berachain Bepolia Testnet", []string{"berachain-testnet", "bepolia"}, "https://testnet.berascan.com", "BERA", true, true},
	{"143", "monad", "Monad Mainnet", nil, "https://monadscan.com", "MON", false, true},
	{"10143", "monad-testnet", "Monad Testnet", nil, "https://testnet.monadscan.com", "MON", true, true},
	{"999", "hyperevm", "HyperEVM Mainnet", []string{"hyper"}, "https://hyperevmscan.io", "HYPE", false, true},
	{"747474", "katana", "Katana Mainnet", nil, "https://katanascan.com", "", false, true},
	{"737373", "katana-bokuto", "Katana Bokuto", []string{"katana-testnet", "bokuto"}, "https://bokuto.katanascan.com", "", true, true},
	{"1329", "sei", "Sei Mainnet", nil, "https://seiscan.io", "SEI", false, true},
	{"1328", "sei-testnet", "Sei Testnet", nil, "https://testnet.seiscan.io", "SEI", true, true},
	{"988", "stable", "Stable Mainnet", nil, "https://stablescan.xyz", "", false, true},
	{"2201", "stable-testnet", "Stable Testnet", nil, "https://testnet.stablescan.xyz", "", true, true},
	{"9745", "plasma", "Plasma Mainnet", nil, "https://plasmascan.to", "", false, true},
	{"9746", "plasma-testnet", "Plasma Testnet", nil, "https://testnet.plasmascan.to", "", true, true},
	{"4326", "megaeth", "MegaETH Mainnet", []string{"mega"}, "https://mega.etherscan.io", "ETH", false, true},
	{"6343", "megaeth-testnet", "MegaETH Testnet", []string{"mega-testnet"}, "https://testnet-mega.etherscan.io", "ETH", true, true},
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
	return append([]Chain(nil), registry...)
}

func IsMainnetID(id string) bool {
	return id == "1"
}
