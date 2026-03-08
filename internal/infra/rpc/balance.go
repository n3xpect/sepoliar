package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"sepoliar/internal/domain"
)

type BalanceFetcher struct {
	rpcURL string
}

func New(rpcURL string) *BalanceFetcher {
	return &BalanceFetcher{rpcURL: rpcURL}
}

func (f *BalanceFetcher) GetBalance(ctx context.Context, cfg domain.ClaimConfig) (string, error) {
	if cfg.TokenAddress == "" {
		return f.getNativeBalance(ctx, cfg.WalletAddress, cfg.TokenDecimals)
	}
	return f.getERC20Balance(ctx, cfg.WalletAddress, cfg.TokenAddress, cfg.TokenDecimals)
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	Result string `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (f *BalanceFetcher) call(ctx context.Context, req rpcRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, f.rpcURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return "", err
	}
	if rpcResp.Error != nil {
		return "", fmt.Errorf("rpc error: %s", rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}
func (f *BalanceFetcher) getNativeBalance(ctx context.Context, address string, decimals int) (string, error) {
	result, err := f.call(ctx, rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBalance",
		Params:  []interface{}{address, "latest"},
		ID:      1,
	})
	if err != nil {
		return "", err
	}
	return hexToDecimal(result, decimals)
}
func (f *BalanceFetcher) getERC20Balance(ctx context.Context, walletAddress, tokenAddress string, decimals int) (string, error) {
	// balanceOf(address) = 0x70a08231 + 32-byte padded address
	addr := strings.ToLower(strings.TrimPrefix(walletAddress, "0x"))
	data := "0x70a08231" + strings.Repeat("0", 64-len(addr)) + addr

	result, err := f.call(ctx, rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_call",
		Params: []interface{}{
			map[string]string{"to": tokenAddress, "data": data},
			"latest",
		},
		ID: 1,
	})
	if err != nil {
		return "", err
	}
	return hexToDecimal(result, decimals)
}
func hexToDecimal(hexStr string, decimals int) (string, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")
	if hexStr == "" || hexStr == "0" {
		return "0", nil
	}

	val := new(big.Int)
	if _, ok := val.SetString(hexStr, 16); !ok {
		return "", fmt.Errorf("invalid hex value: %s", hexStr)
	}

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Div(val, divisor)
	remainder := new(big.Int).Mod(val, divisor)

	// Küsurat kısmını decimals uzunluğunda sıfır dolgulu string'e çevir
	fracStr := fmt.Sprintf("%0*s", decimals, remainder.String())
	// Anlamlı rakam sayısını 10 ile sınırla, sondaki sıfırları temizle
	if len(fracStr) > 10 {
		fracStr = fracStr[:10]
	}
	fracStr = strings.TrimRight(fracStr, "0")
	if fracStr == "" {
		return whole.String(), nil
	}
	return fmt.Sprintf("%s.%s", whole.String(), fracStr), nil
}
