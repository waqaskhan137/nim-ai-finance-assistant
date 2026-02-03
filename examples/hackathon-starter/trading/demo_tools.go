package trading

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/tools"
)

// DemoLiminalTools provides demo implementations for Liminal read tools.
func DemoLiminalTools() []core.Tool {
	return []core.Tool{
		createDemoGetBalanceTool(),
		createDemoGetSavingsBalanceTool(),
		createDemoGetVaultRatesTool(),
		createDemoGetTransactionsTool(),
		createDemoGetProfileTool(),
	}
}

func createDemoGetBalanceTool() core.Tool {
	return tools.New("get_balance").
		Description("Get the user's wallet balance (demo data).").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"currency": tools.StringProperty("Optional: filter by currency (e.g., 'USD', 'EUR', 'LIL')"),
		})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Currency string `json:"currency"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			demoData, err := LoadDemoDataFromFile()
			if err != nil {
				return nil, fmt.Errorf("failed to load demo wallet data: %w", err)
			}

			balance := map[string]interface{}{
				"currency": demoData.Wallet.Currency,
				"amount":   demoData.Wallet.Balance,
			}

			balances := []interface{}{balance}
			if params.Currency != "" && params.Currency != demoData.Wallet.Currency {
				balances = []interface{}{}
			}

			return map[string]interface{}{
				"balances": balances,
				"user_id":  demoData.Wallet.UserID,
				"username": demoData.Wallet.Username,
			}, nil
		}).
		Build()
}

func createDemoGetSavingsBalanceTool() core.Tool {
	return tools.New("get_savings_balance").
		Description("Get the user's savings positions and current APY (demo data).").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"vault": tools.StringProperty("Optional: filter by vault name"),
		})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Vault string `json:"vault"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			demoData, err := LoadDemoDataFromFile()
			if err != nil {
				return nil, fmt.Errorf("failed to load demo savings data: %w", err)
			}

			positions := make([]interface{}, 0, len(demoData.Savings.Positions))
			total := 0.0
			for _, pos := range demoData.Savings.Positions {
				if params.Vault != "" && params.Vault != pos.VaultName {
					continue
				}
				positions = append(positions, map[string]interface{}{
					"vault_id":     pos.VaultID,
					"vault":        pos.VaultName,
					"amount":       pos.Amount,
					"apy":          pos.APY,
					"currency":     pos.Currency,
					"deposited_at": pos.DepositedAt,
				})
				total += pos.Amount
			}

			return map[string]interface{}{
				"positions":   positions,
				"total_value": total,
			}, nil
		}).
		Build()
}

func createDemoGetVaultRatesTool() core.Tool {
	return tools.New("get_vault_rates").
		Description("Get current APY rates for available savings vaults (demo data).").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			demoData, err := LoadDemoDataFromFile()
			if err != nil {
				return nil, fmt.Errorf("failed to load demo vault rates: %w", err)
			}

			vaults := make([]interface{}, 0, len(demoData.VaultRates))
			for _, vault := range demoData.VaultRates {
				vaults = append(vaults, map[string]interface{}{
					"vault_id":    vault.VaultID,
					"name":        vault.Name,
					"apy":         vault.APY,
					"risk":        vault.Risk,
					"min_deposit": vault.MinDeposit,
				})
			}

			return map[string]interface{}{
				"vaults": vaults,
			}, nil
		}).
		Build()
}

func createDemoGetTransactionsTool() core.Tool {
	return tools.New("get_transactions").
		Description("Get the user's recent transaction history (demo data).").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"limit": tools.IntegerProperty("Number of transactions to return (default: 10)"),
			"type":  tools.StringEnumProperty("Filter by transaction type", "send", "receive", "deposit", "withdraw"),
		})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Limit int    `json:"limit"`
				Type  string `json:"type"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			demoData, err := LoadDemoDataFromFile()
			if err != nil {
				return nil, fmt.Errorf("failed to load demo transactions: %w", err)
			}

			limit := params.Limit
			if limit <= 0 {
				limit = 10
			}

			filtered := make([]map[string]interface{}, 0, limit)
			for _, tx := range demoData.Transactions {
				if params.Type != "" && params.Type != tx.Type {
					continue
				}
				filtered = append(filtered, map[string]interface{}{
					"id":         tx.ID,
					"type":       tx.Type,
					"amount":     tx.Amount,
					"currency":   tx.Currency,
					"recipient":  tx.Recipient,
					"sender":     tx.Sender,
					"note":       tx.Note,
					"created_at": tx.CreatedAt,
					"category":   tx.Category,
				})
				if len(filtered) >= limit {
					break
				}
			}

			return map[string]interface{}{
				"transactions": filtered,
			}, nil
		}).
		Build()
}

func createDemoGetProfileTool() core.Tool {
	return tools.New("get_profile").
		Description("Get the user's profile information (demo data).").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			demoData, err := LoadDemoDataFromFile()
			if err != nil {
				return nil, fmt.Errorf("failed to load demo profile: %w", err)
			}

			return map[string]interface{}{
				"user_id":    demoData.Profile.UserID,
				"username":   demoData.Profile.Username,
				"email":      demoData.Profile.Email,
				"name":       demoData.Profile.Name,
				"created_at": demoData.Profile.CreatedAt,
			}, nil
		}).
		Build()
}
