package config

type WalletConfig struct {
	ETH string
}

func loadWalletConfig() WalletConfig {
	return WalletConfig{
		ETH: getEnvRequired("WALLET_ADDRESS_ETH"),
	}
}
