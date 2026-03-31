package config

const sepoliaRPCURL = "https://ethereum-sepolia-rpc.publicnode.com"

type RPCConfig struct {
	SepoliaRPCURL   string
	EtherscanAPIKey string
}

func loadRPCConfig(etherscanAPIKey string) RPCConfig {
	return RPCConfig{
		SepoliaRPCURL:   sepoliaRPCURL,
		EtherscanAPIKey: etherscanAPIKey,
	}
}
