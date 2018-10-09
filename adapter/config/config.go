package config

import "github.com/republicprotocol/swapperd/foundation"

// Config is the the global config object
type Config struct {
	Version             string             `json:"version"`
	SupportedCurrencies []foundation.Token `json:"supportedCurrencies"`
	AuthorizedAddresses []string           `json:"authorizedAddresses"`
	HomeDir             string             `json:"homeDir"`
	Ethereum            EthereumNetwork    `json:"ethereum"`
	Bitcoin             BitcoinNetwork     `json:"bitcoin"`
	RenEx               RenExNetwork       `json:"renex"`
}

// EthereumNetwork is the ethereum specific config object
type EthereumNetwork struct {
	Swapper string `json:"swapper"`
	Network string `json:"network"`
	URL     string `json:"url"`
}

// BitcoinNetwork is the bitcoin specific config object
type BitcoinNetwork struct {
	Network string `json:"network"`
	URL     string `json:"url"`
}

// RenExNetwork is the renex specific config object
type RenExNetwork struct {
	Network    string `json:"network"`
	Ingress    string `json:"ingress"`
	Settlement string `json:"settlement"`
	Orderbook  string `json:"orderbook"`
}
