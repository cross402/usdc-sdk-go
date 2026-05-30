package pay

// Asset identifies a payment asset (token) on a chain.
type Asset string

const (
	AssetUSDC  Asset = "usdc"
	AssetUSDT  Asset = "usdt"
	AssetUSDT0 Asset = "usdt0"
)
