package config

const sepoliaRPCURL = "https://ethereum-sepolia-rpc.publicnode.com"

type RPCConfig struct {
	SepoliaRPCURL string
}

func loadRPCConfig() RPCConfig {
	return RPCConfig{
		SepoliaRPCURL: sepoliaRPCURL,
	}
}
