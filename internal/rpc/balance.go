package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sepoliar/internal/model"
)

type BalanceFetcher struct {
	rpcURL       string
	etherscanKey string
}

func New(rpcURL, etherscanKey string) *BalanceFetcher {
	return &BalanceFetcher{rpcURL: rpcURL, etherscanKey: etherscanKey}
}

func (f *BalanceFetcher) GetBalance(ctx context.Context, cfg model.ClaimConfig) (string, error) {
	if cfg.TokenAddress == "" {
		return f.getNativeBalance(ctx, cfg.WalletAddress, cfg.TokenDecimals)
	}
	return f.getERC20Balance(ctx, cfg.WalletAddress, cfg.TokenAddress, cfg.TokenDecimals)
}

func (f *BalanceFetcher) GetLastTxTime(ctx context.Context, address string) (time.Time, error) {
	if f.etherscanKey == "" {
		return time.Time{}, fmt.Errorf("etherscan API key not configured")
	}
	url := fmt.Sprintf(
		"https://api.etherscan.io/v2/api?chainid=11155111&module=account&action=txlist&address=%s&startblock=0&endblock=99999999&page=1&offset=1&sort=desc&apikey=%s",
		address, f.etherscanKey,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return time.Time{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	var apiResp struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return time.Time{}, err
	}
	if apiResp.Status != "1" {
		var msg string
		_ = json.Unmarshal(apiResp.Result, &msg)
		if msg == "" {
			msg = apiResp.Message
		}
		return time.Time{}, fmt.Errorf("etherscan error: %s", msg)
	}
	var txList []struct {
		TimeStamp string `json:"timeStamp"`
	}
	if err := json.Unmarshal(apiResp.Result, &txList); err != nil {
		return time.Time{}, err
	}
	if len(txList) == 0 {
		return time.Time{}, fmt.Errorf("no transactions found for address %s", address)
	}
	ts, err := strconv.ParseInt(txList[0].TimeStamp, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: %w", txList[0].TimeStamp, err)
	}
	return time.Unix(ts, 0), nil
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

	fracStr := fmt.Sprintf("%0*s", decimals, remainder.String())
	if len(fracStr) > 10 {
		fracStr = fracStr[:10]
	}
	fracStr = strings.TrimRight(fracStr, "0")
	if fracStr == "" {
		return whole.String(), nil
	}
	return fmt.Sprintf("%s.%s", whole.String(), fracStr), nil
}
