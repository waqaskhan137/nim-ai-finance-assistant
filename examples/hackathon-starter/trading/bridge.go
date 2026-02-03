package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/examples/hackathon-starter/trading/connectors"
)

// ============================================================================
// FUND BRIDGE SERVICE
// ============================================================================

// FundBridge handles fund transfers between Liminal wallet and Binance trading account.
type FundBridge struct {
	LiminalExecutor core.ToolExecutor
	BinanceConnector *connectors.BinanceConnector
	Database        *Database
}

// NewFundBridge creates a new fund bridge service.
func NewFundBridge(liminalExecutor core.ToolExecutor, binanceConnector *connectors.BinanceConnector, db *Database) *FundBridge {
	return &FundBridge{
		LiminalExecutor: liminalExecutor,
		BinanceConnector: binanceConnector,
		Database:        db,
	}
}

// TransferToTrading moves funds from Liminal wallet to Binance for trading.
// PRODUCTION IMPLEMENTATION:
// 1. User withdraws from Liminal to their bank account
// 2. Funds arrive in bank account (1-3 business days)
// 3. User deposits from bank to Binance
// 4. System monitors Binance account for deposit confirmation
//
// For this demo, we'll simulate the transfer and record it.
func (fb *FundBridge) TransferToTrading(ctx context.Context, userID string, amount float64) error {
	log.Printf("🔄 Transferring $%.2f from Liminal to Binance for user %s", amount, userID)

	// Step 1: Check Liminal balance
	balanceResult, err := fb.LiminalExecutor.Execute(ctx, &core.ExecuteRequest{
		UserID: userID,
		Tool:   "get_balance",
		Input:  json.RawMessage(`{}`),
	})
	if err != nil {
		return fmt.Errorf("failed to check Liminal balance: %w", err)
	}

	if !balanceResult.Success {
		return fmt.Errorf("failed to get balance: %s", balanceResult.Error)
	}

	// Parse balance response
	var balanceData map[string]interface{}
	if err := json.Unmarshal(balanceResult.Data, &balanceData); err != nil {
		return fmt.Errorf("failed to parse balance response: %w", err)
	}

	// Extract USD balance (simplified - assumes USD currency)
	balances, ok := balanceData["balances"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid balance response format")
	}

	var usdBalance float64
	for _, bal := range balances {
		balance, ok := bal.(map[string]interface{})
		if !ok {
			continue
		}
		if currency, ok := balance["currency"].(string); ok && currency == "USD" {
			if amount, ok := balance["amount"].(float64); ok {
				usdBalance = amount
				break
			}
		}
	}

	if usdBalance < amount {
		return fmt.Errorf("insufficient Liminal balance: have $%.2f, need $%.2f", usdBalance, amount)
	}

	// PRODUCTION STEPS (currently simulated):
	//
	// Step 2: Initiate Liminal withdrawal to user's bank account
	// if err := fb.withdrawFromLiminal(ctx, userID, amount); err != nil {
	//     return fmt.Errorf("failed to withdraw from Liminal: %w", err)
	// }
	//
	// Step 3: Wait for bank transfer to complete (webhook or polling)
	// if err := fb.waitForBankTransfer(amount); err != nil {
	//     return fmt.Errorf("bank transfer failed: %w", err)
	// }
	//
	// Step 4: Deposit to Binance from bank account
	// if err := fb.depositToBinance(amount); err != nil {
	//     return fmt.Errorf("failed to deposit to Binance: %w", err)
	// }
	//
	// Step 5: Confirm Binance deposit
	// if err := fb.confirmBinanceDeposit(amount); err != nil {
	//     return fmt.Errorf("Binance deposit confirmation failed: %w", err)
	// }

	// DEMO: Simulate the entire process
	log.Printf("💰 [DEMO] Withdrawing $%.2f from Liminal wallet", amount)
	log.Printf("🏦 [DEMO] Bank transfer initiated (would take 1-3 days)")
	log.Printf("📈 [DEMO] Depositing $%.2f to Binance account", amount)
	log.Printf("✅ [DEMO] Binance deposit confirmed")

	// Step 4: Record the allocation in database
	allocation := &TradingAllocation{
		UserID:        userID,
		Amount:        amount,
		Status:        "active",
		AllocatedAt:   time.Now(),
	}

	if fb.Database != nil {
		if err := fb.Database.SaveAllocation(allocation); err != nil {
			log.Printf("⚠️ Failed to save allocation to database: %v", err)
		}
	}

	log.Printf("✅ Successfully allocated $%.2f for trading", amount)
	return nil
}

// TransferFromTrading moves funds from Binance back to Liminal wallet.
// This handles profit withdrawals and closing trading positions.
func (fb *FundBridge) TransferFromTrading(ctx context.Context, userID string, amount float64) error {
	log.Printf("🔄 Transferring $%.2f from Binance to Liminal for user %s", amount, userID)

	// Step 1: Check Binance balance
	if fb.BinanceConnector == nil {
		return fmt.Errorf("Binance connector not available")
	}

	balances, err := fb.BinanceConnector.GetBalances()
	if err != nil {
		return fmt.Errorf("failed to check Binance balance: %w", err)
	}

	// Find USDT balance
	var usdtBalance float64
	for _, balance := range balances {
		if balance.Asset == "USDT" {
			usdtBalance = balance.Free
			break
		}
	}

	if usdtBalance < amount {
		return fmt.Errorf("insufficient Binance USDT balance: have $%.2f, need $%.2f", usdtBalance, amount)
	}

	// Step 2: Withdraw from Binance to bank account
	// In production, this would initiate a withdrawal to a bank account
	log.Printf("🏦 Withdrawing $%.2f from Binance", amount)

	// Step 3: Deposit to Liminal wallet
	// In production, this would involve bank transfer to Liminal
	log.Printf("💰 Depositing $%.2f to Liminal wallet", amount)

	// Step 4: Update allocation status
	if fb.Database != nil {
		if err := fb.Database.CloseAllocation(userID, amount); err != nil {
			log.Printf("⚠️ Failed to update allocation status: %v", err)
		}
	}

	log.Printf("✅ Successfully transferred $%.2f back to Liminal", amount)
	return nil
}

// GetAllocationStatus returns the current trading allocation for a user.
func (fb *FundBridge) GetAllocationStatus(userID string) (*TradingAllocation, error) {
	if fb.Database == nil {
		return nil, fmt.Errorf("database not available")
	}

	return fb.Database.GetAllocation(userID)
}

// CloseAllocation marks a trading allocation as closed.
func (fb *FundBridge) CloseAllocation(userID string, finalValue float64) error {
	if fb.Database == nil {
		return fmt.Errorf("database not available")
	}

	return fb.Database.CloseAllocation(userID, finalValue)
}

// ============================================================================
// PRODUCTION IMPLEMENTATION METHODS
// ============================================================================

// withdrawFromLiminal initiates a withdrawal from Liminal to user's bank account.
// NOTE: This would require a new Liminal API endpoint for external withdrawals.
func (fb *FundBridge) withdrawFromLiminal(ctx context.Context, userID string, amount float64) error {
	// This would call a hypothetical Liminal API endpoint:
	// POST /api/withdraw
	// {
	//   "user_id": userID,
	//   "amount": amount,
	//   "currency": "USD",
	//   "destination": "bank_account"  // or specific bank account ID
	// }

	// For now, this is simulated
	log.Printf("💰 [PRODUCTION] Would call Liminal withdrawal API: $%.2f to bank account", amount)
	return nil
}

// waitForBankTransfer waits for the bank transfer to complete.
// In production, this could use webhooks or polling.
func (fb *FundBridge) waitForBankTransfer(expectedAmount float64) error {
	// Options:
	// 1. Webhook: Liminal calls our endpoint when transfer completes
	// 2. Polling: Check bank account balance periodically
	// 3. Manual confirmation: User confirms in UI

	log.Printf("🏦 [PRODUCTION] Would wait for bank transfer confirmation: $%.2f", expectedAmount)
	log.Printf("⏳ [PRODUCTION] Bank transfers typically take 1-3 business days")
	return nil
}

// depositToBinance initiates a deposit from bank account to Binance.
// This would typically be done via ACH/wire transfer to Binance's bank details.
func (fb *FundBridge) depositToBinance(amount float64) error {
	if fb.BinanceConnector == nil {
		return fmt.Errorf("Binance connector not available")
	}

	// In production, this would:
	// 1. Get user's bank account details
	// 2. Initiate ACH/wire transfer to Binance's bank account
	// 3. Binance provides specific account details for deposits

	log.Printf("📈 [PRODUCTION] Would initiate bank transfer to Binance: $%.2f", amount)
	log.Printf("🏦 [PRODUCTION] Transfer destination: Binance USD deposit account")
	return nil
}

// confirmBinanceDeposit waits for Binance to confirm the deposit.
func (fb *FundBridge) confirmBinanceDeposit(expectedAmount float64) error {
	if fb.BinanceConnector == nil {
		return fmt.Errorf("Binance connector not available")
	}

	// In production, this would:
	// 1. Poll Binance API for deposit confirmations
	// 2. Check for deposits matching the expected amount and timestamp
	// 3. Confirm the deposit is credited to the trading account

	log.Printf("✅ [PRODUCTION] Would confirm Binance deposit: $%.2f", expectedAmount)
	log.Printf("🔍 [PRODUCTION] Checking Binance account for deposit confirmation")
	return nil
}