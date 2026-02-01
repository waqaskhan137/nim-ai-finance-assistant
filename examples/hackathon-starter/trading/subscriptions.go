// Subscription Manager - Detects and analyzes recurring payments
package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/tools"
)

// ============================================================================
// SUBSCRIPTION DATA STRUCTURES
// ============================================================================

// Subscription represents a detected recurring payment
type Subscription struct {
	ID                string                `json:"id"`
	Name              string                `json:"name"`
	Recipient         string                `json:"recipient"`
	Category          string                `json:"category"`
	Amount            float64               `json:"amount"`
	Frequency         string                `json:"frequency"`
	LastCharged       string                `json:"last_charged"`
	NextCharge        string                `json:"next_charge"`
	Status            string                `json:"status"`
	ConfidenceScore   float64               `json:"confidence_score"`
	PaymentHistory    []SubscriptionPayment `json:"payment_history"`
	TotalSpentAllTime float64               `json:"total_spent_all_time"`
	MonthsActive      int                   `json:"months_active"`
	PriceChange       *PriceChangeInfo      `json:"price_change,omitempty"`
	UsageInsight      string                `json:"usage_insight"`
	Icon              string                `json:"icon"`
	Color             string                `json:"color"`
}

// SubscriptionPayment represents a single subscription payment
type SubscriptionPayment struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

// PriceChangeInfo tracks subscription price changes
type PriceChangeInfo struct {
	OldPrice      float64 `json:"old_price"`
	NewPrice      float64 `json:"new_price"`
	ChangeDate    string  `json:"change_date"`
	ChangePercent float64 `json:"change_percent"`
}

// SubscriptionAnalysis is the full response
type SubscriptionAnalysis struct {
	Subscriptions          []Subscription               `json:"subscriptions"`
	TotalMonthly           float64                      `json:"total_monthly"`
	TotalYearly            float64                      `json:"total_yearly"`
	SubscriptionCount      int                          `json:"subscription_count"`
	CategoryBreakdown      []SubscriptionCategoryCost   `json:"category_breakdown"`
	PotentialSavings       float64                      `json:"potential_savings"`
	ForgottenSubscriptions []Subscription               `json:"forgotten_subscriptions"`
	PriceIncreases         []Subscription               `json:"price_increases"`
	DuplicateServices      []DuplicateServiceGroup      `json:"duplicate_services"`
	Recommendations        []SubscriptionRecommendation `json:"recommendations"`
	MonthlyCostTrend       []SubscriptionMonthCost      `json:"monthly_cost_trend"`
	HealthScore            int                          `json:"health_score"`
	HealthGrade            string                       `json:"health_grade"`
	SummaryInsight         string                       `json:"summary_insight"`
}

// SubscriptionCategoryCost shows spending by category
type SubscriptionCategoryCost struct {
	Category     string  `json:"category"`
	MonthlyTotal float64 `json:"monthly_total"`
	Count        int     `json:"count"`
	Percentage   float64 `json:"percentage"`
	Icon         string  `json:"icon"`
	Color        string  `json:"color"`
}

// SubscriptionMonthCost for trend tracking
type SubscriptionMonthCost struct {
	Month string  `json:"month"`
	Total float64 `json:"total"`
	Count int     `json:"count"`
}

// DuplicateServiceGroup represents services that overlap
type DuplicateServiceGroup struct {
	Category      string         `json:"category"`
	Services      []Subscription `json:"services"`
	TotalMonthly  float64        `json:"total_monthly"`
	SuggestedKeep string         `json:"suggested_keep"`
	PotentialSave float64        `json:"potential_save"`
}

// SubscriptionRecommendation is an actionable suggestion
type SubscriptionRecommendation struct {
	Type        string  `json:"type"`
	Priority    string  `json:"priority"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Savings     float64 `json:"savings,omitempty"`
	SubID       string  `json:"subscription_id,omitempty"`
}

// ============================================================================
// SUBSCRIPTION DETECTION ALGORITHM
// ============================================================================

// Known subscription services for better detection
var knownSubscriptions = map[string]struct {
	Name     string
	Category string
	Icon     string
	Color    string
}{
	"@netflix":           {"Netflix", "entertainment", "🎬", "#E50914"},
	"@spotify":           {"Spotify", "entertainment", "🎵", "#1DB954"},
	"@hbo_max":           {"HBO Max", "entertainment", "🎭", "#B026FF"},
	"@disney_plus":       {"Disney+", "entertainment", "🏰", "#113CCF"},
	"@apple_music":       {"Apple Music", "entertainment", "🎧", "#FA243C"},
	"@youtube_premium":   {"YouTube Premium", "entertainment", "📺", "#FF0000"},
	"@amazon_prime":      {"Amazon Prime", "shopping", "📦", "#FF9900"},
	"@gym_membership":    {"Gym Membership", "health", "💪", "#4CAF50"},
	"@planet_fitness":    {"Planet Fitness", "health", "🏋️", "#5C2D91"},
	"@peloton":           {"Peloton", "health", "🚴", "#DF1C2F"},
	"@adobe":             {"Adobe Creative Cloud", "software", "🎨", "#FF0000"},
	"@microsoft365":      {"Microsoft 365", "software", "💼", "#0078D4"},
	"@dropbox":           {"Dropbox", "software", "📁", "#0061FF"},
	"@icloud":            {"iCloud+", "software", "☁️", "#3693F3"},
	"@google_one":        {"Google One", "software", "🗄️", "#4285F4"},
	"@notion":            {"Notion", "software", "📝", "#000000"},
	"@slack":             {"Slack", "software", "💬", "#4A154B"},
	"@nytimes":           {"NY Times", "news", "📰", "#000000"},
	"@wsj":               {"Wall Street Journal", "news", "📊", "#0080C6"},
	"@medium":            {"Medium", "news", "📖", "#000000"},
	"@att":               {"AT&T", "utilities", "📱", "#00A8E0"},
	"@verizon":           {"Verizon", "utilities", "📶", "#CD040B"},
	"@tmobile":           {"T-Mobile", "utilities", "📲", "#E20074"},
	"@electric_company":  {"Electric Company", "utilities", "⚡", "#FFB700"},
	"@water_utility":     {"Water Utility", "utilities", "💧", "#00B4D8"},
	"@internet_provider": {"Internet Provider", "utilities", "🌐", "#6C63FF"},
	"@insurance_co":      {"Insurance", "insurance", "🛡️", "#2E7D32"},
	"@car_insurance":     {"Car Insurance", "insurance", "🚗", "#1565C0"},
	"@health_insurance":  {"Health Insurance", "insurance", "🏥", "#C62828"},
	"@landlord":          {"Rent", "housing", "🏠", "#795548"},
	"@mortgage_bank":     {"Mortgage", "housing", "🏦", "#455A64"},
}

// Category configuration
var subscriptionCategories = map[string]struct {
	Icon  string
	Color string
}{
	"entertainment": {"🎬", "#E50914"},
	"health":        {"💪", "#4CAF50"},
	"software":      {"💻", "#0078D4"},
	"news":          {"📰", "#000000"},
	"utilities":     {"📱", "#00A8E0"},
	"insurance":     {"🛡️", "#2E7D32"},
	"housing":       {"🏠", "#795548"},
	"shopping":      {"🛍️", "#FF9900"},
	"other":         {"📦", "#9E9E9E"},
}

// detectSubscriptions analyzes transactions to find recurring payments
func detectSubscriptions(transactions []Transaction) SubscriptionAnalysis {
	// Group transactions by recipient
	recipientTx := make(map[string][]Transaction)
	for _, tx := range transactions {
		if tx.Type == "send" && tx.Recipient != "" {
			recipientTx[tx.Recipient] = append(recipientTx[tx.Recipient], tx)
		}
	}

	var subscriptions []Subscription
	var forgotten []Subscription
	var priceIncreases []Subscription

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	ninetyDaysAgo := now.AddDate(0, 0, -90)

	for recipient, txs := range recipientTx {
		// Need at least 2 transactions to detect a pattern
		if len(txs) < 2 {
			continue
		}

		// Sort by date
		sort.Slice(txs, func(i, j int) bool {
			return txs[i].CreatedAt.Before(txs[j].CreatedAt)
		})

		// Analyze payment pattern
		sub := analyzePaymentPattern(recipient, txs, now)
		if sub == nil {
			continue
		}

		subscriptions = append(subscriptions, *sub)

		// Check if forgotten (no payment in 30+ days but was regular)
		lastPayment := txs[len(txs)-1].CreatedAt
		if lastPayment.Before(thirtyDaysAgo) && sub.ConfidenceScore > 0.6 {
			forgotten = append(forgotten, *sub)
		}

		// Check for price increases
		if sub.PriceChange != nil && sub.PriceChange.ChangePercent > 5 {
			priceIncreases = append(priceIncreases, *sub)
		}
	}

	// Sort subscriptions by amount (highest first)
	sort.Slice(subscriptions, func(i, j int) bool {
		return subscriptions[i].Amount > subscriptions[j].Amount
	})

	// Calculate totals
	totalMonthly := 0.0
	for _, sub := range subscriptions {
		totalMonthly += normalizeToMonthly(sub.Amount, sub.Frequency)
	}

	// Category breakdown
	categoryBreakdown := calculateCategoryBreakdown(subscriptions)

	// Find duplicate services
	duplicates := findDuplicateServices(subscriptions)

	// Calculate potential savings
	potentialSavings := calculatePotentialSavings(forgotten, duplicates)

	// Generate recommendations
	recommendations := generateSubscriptionRecommendations(subscriptions, forgotten, priceIncreases, duplicates)

	// Monthly cost trend
	costTrend := calculateMonthlyCostTrend(transactions, ninetyDaysAgo, now)

	// Calculate health score
	healthScore, healthGrade := calculateSubscriptionHealthScore(subscriptions, totalMonthly, len(forgotten), len(duplicates))

	// Generate summary insight
	summaryInsight := generateSubscriptionSummary(subscriptions, totalMonthly, potentialSavings, len(forgotten))

	return SubscriptionAnalysis{
		Subscriptions:          subscriptions,
		TotalMonthly:           totalMonthly,
		TotalYearly:            totalMonthly * 12,
		SubscriptionCount:      len(subscriptions),
		CategoryBreakdown:      categoryBreakdown,
		PotentialSavings:       potentialSavings,
		ForgottenSubscriptions: forgotten,
		PriceIncreases:         priceIncreases,
		DuplicateServices:      duplicates,
		Recommendations:        recommendations,
		MonthlyCostTrend:       costTrend,
		HealthScore:            healthScore,
		HealthGrade:            healthGrade,
		SummaryInsight:         summaryInsight,
	}
}

// analyzePaymentPattern determines if transactions represent a subscription
func analyzePaymentPattern(recipient string, txs []Transaction, now time.Time) *Subscription {
	if len(txs) < 2 {
		return nil
	}

	// Calculate intervals between payments
	var intervals []int
	for i := 1; i < len(txs); i++ {
		days := int(txs[i].CreatedAt.Sub(txs[i-1].CreatedAt).Hours() / 24)
		intervals = append(intervals, days)
	}

	// Calculate average interval
	avgInterval := 0.0
	for _, interval := range intervals {
		avgInterval += float64(interval)
	}
	avgInterval /= float64(len(intervals))

	// Determine frequency
	frequency := determineFrequency(avgInterval)
	if frequency == "" {
		return nil // Not a recognizable subscription pattern
	}

	// Calculate amount consistency
	amounts := make([]float64, len(txs))
	totalAmount := 0.0
	for i, tx := range txs {
		amounts[i] = tx.Amount
		totalAmount += tx.Amount
	}
	avgAmount := totalAmount / float64(len(txs))
	amountVariance := calculateVariance(amounts, avgAmount)
	amountConsistency := 1.0 - math.Min(amountVariance/avgAmount, 1.0)

	// Calculate interval consistency
	floatIntervals := make([]float64, len(intervals))
	for i, interval := range intervals {
		floatIntervals[i] = float64(interval)
	}
	intervalVariance := calculateVariance(floatIntervals, avgInterval)
	expectedInterval := getExpectedInterval(frequency)
	intervalConsistency := 1.0 - math.Min(intervalVariance/expectedInterval, 1.0)

	// Calculate confidence score
	confidence := (amountConsistency*0.4 + intervalConsistency*0.4 + math.Min(float64(len(txs))/5, 1.0)*0.2)

	// Must have reasonable confidence
	if confidence < 0.4 {
		return nil
	}

	// Get subscription info
	name, category, icon, color := getSubscriptionInfo(recipient)

	// Build payment history
	var paymentHistory []SubscriptionPayment
	for _, tx := range txs {
		paymentHistory = append(paymentHistory, SubscriptionPayment{
			Date:   tx.CreatedAt.Format("2006-01-02"),
			Amount: tx.Amount,
		})
	}

	// Check for price changes
	var priceChange *PriceChangeInfo
	if len(txs) >= 3 {
		recentAmount := txs[len(txs)-1].Amount
		olderAvg := 0.0
		for i := 0; i < len(txs)-1; i++ {
			olderAvg += txs[i].Amount
		}
		olderAvg /= float64(len(txs) - 1)

		changePercent := ((recentAmount - olderAvg) / olderAvg) * 100
		if math.Abs(changePercent) > 5 {
			priceChange = &PriceChangeInfo{
				OldPrice:      olderAvg,
				NewPrice:      recentAmount,
				ChangeDate:    txs[len(txs)-1].CreatedAt.Format("2006-01-02"),
				ChangePercent: changePercent,
			}
		}
	}

	// Calculate months active
	firstPayment := txs[0].CreatedAt
	monthsActive := int(now.Sub(firstPayment).Hours() / 24 / 30)
	if monthsActive < 1 {
		monthsActive = 1
	}

	// Predict next charge
	lastPayment := txs[len(txs)-1].CreatedAt
	nextCharge := predictNextCharge(lastPayment, frequency)

	// Determine status
	status := "active"
	if now.Sub(lastPayment).Hours()/24 > getExpectedInterval(frequency)*1.5 {
		status = "possibly_cancelled"
	}

	// Generate usage insight
	usageInsight := generateUsageInsight(name, category, avgAmount, monthsActive, len(txs))

	return &Subscription{
		ID:                fmt.Sprintf("sub_%s", strings.TrimPrefix(recipient, "@")),
		Name:              name,
		Recipient:         recipient,
		Category:          category,
		Amount:            avgAmount,
		Frequency:         frequency,
		LastCharged:       lastPayment.Format("2006-01-02"),
		NextCharge:        nextCharge,
		Status:            status,
		ConfidenceScore:   confidence,
		PaymentHistory:    paymentHistory,
		TotalSpentAllTime: totalAmount,
		MonthsActive:      monthsActive,
		PriceChange:       priceChange,
		UsageInsight:      usageInsight,
		Icon:              icon,
		Color:             color,
	}
}

// Helper functions

func determineFrequency(avgDays float64) string {
	switch {
	case avgDays >= 5 && avgDays <= 9:
		return "weekly"
	case avgDays >= 12 && avgDays <= 18:
		return "biweekly"
	case avgDays >= 25 && avgDays <= 35:
		return "monthly"
	case avgDays >= 85 && avgDays <= 100:
		return "quarterly"
	case avgDays >= 350 && avgDays <= 380:
		return "yearly"
	default:
		return ""
	}
}

func getExpectedInterval(frequency string) float64 {
	switch frequency {
	case "weekly":
		return 7
	case "biweekly":
		return 14
	case "monthly":
		return 30
	case "quarterly":
		return 90
	case "yearly":
		return 365
	default:
		return 30
	}
}

func calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)))
}

func getSubscriptionInfo(recipient string) (name, category, icon, color string) {
	if info, ok := knownSubscriptions[recipient]; ok {
		return info.Name, info.Category, info.Icon, info.Color
	}

	// Generate name from recipient
	name = strings.TrimPrefix(recipient, "@")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.Title(name)

	// Default category based on patterns
	recipientLower := strings.ToLower(recipient)
	switch {
	case strings.Contains(recipientLower, "gym") || strings.Contains(recipientLower, "fitness"):
		category = "health"
	case strings.Contains(recipientLower, "netflix") || strings.Contains(recipientLower, "spotify") || strings.Contains(recipientLower, "hbo"):
		category = "entertainment"
	case strings.Contains(recipientLower, "insurance"):
		category = "insurance"
	case strings.Contains(recipientLower, "electric") || strings.Contains(recipientLower, "water") || strings.Contains(recipientLower, "phone"):
		category = "utilities"
	default:
		category = "other"
	}

	catInfo := subscriptionCategories[category]
	return name, category, catInfo.Icon, catInfo.Color
}

func normalizeToMonthly(amount float64, frequency string) float64 {
	switch frequency {
	case "weekly":
		return amount * 4.33
	case "biweekly":
		return amount * 2.17
	case "monthly":
		return amount
	case "quarterly":
		return amount / 3
	case "yearly":
		return amount / 12
	default:
		return amount
	}
}

func predictNextCharge(lastPayment time.Time, frequency string) string {
	days := int(getExpectedInterval(frequency))
	nextDate := lastPayment.AddDate(0, 0, days)
	return nextDate.Format("2006-01-02")
}

func generateUsageInsight(name, category string, amount float64, months, payments int) string {
	avgPerMonth := float64(payments) / float64(months)

	switch category {
	case "entertainment":
		if avgPerMonth >= 1 {
			return fmt.Sprintf("You've been subscribed to %s for %d months. Consider if you're getting enough value.", name, months)
		}
		return fmt.Sprintf("Your %s subscription has been active for %d months.", name, months)
	case "health":
		return fmt.Sprintf("Great investment in your health! %d months of %s.", months, name)
	case "software":
		return fmt.Sprintf("You've used %s for %d months - total invested: $%.2f", name, months, amount*float64(months))
	default:
		return fmt.Sprintf("Active for %d months with %d payments.", months, payments)
	}
}

func calculateCategoryBreakdown(subscriptions []Subscription) []SubscriptionCategoryCost {
	categoryTotals := make(map[string]struct {
		total float64
		count int
	})

	totalMonthly := 0.0
	for _, sub := range subscriptions {
		monthly := normalizeToMonthly(sub.Amount, sub.Frequency)
		totalMonthly += monthly
		cat := categoryTotals[sub.Category]
		cat.total += monthly
		cat.count++
		categoryTotals[sub.Category] = cat
	}

	var breakdown []SubscriptionCategoryCost
	for category, data := range categoryTotals {
		catInfo := subscriptionCategories[category]
		percentage := 0.0
		if totalMonthly > 0 {
			percentage = (data.total / totalMonthly) * 100
		}
		breakdown = append(breakdown, SubscriptionCategoryCost{
			Category:     category,
			MonthlyTotal: data.total,
			Count:        data.count,
			Percentage:   percentage,
			Icon:         catInfo.Icon,
			Color:        catInfo.Color,
		})
	}

	// Sort by total descending
	sort.Slice(breakdown, func(i, j int) bool {
		return breakdown[i].MonthlyTotal > breakdown[j].MonthlyTotal
	})

	return breakdown
}

func findDuplicateServices(subscriptions []Subscription) []DuplicateServiceGroup {
	// Group by category for entertainment/streaming
	categoryGroups := make(map[string][]Subscription)
	for _, sub := range subscriptions {
		if sub.Category == "entertainment" || sub.Category == "news" {
			categoryGroups[sub.Category] = append(categoryGroups[sub.Category], sub)
		}
	}

	var duplicates []DuplicateServiceGroup
	for category, subs := range categoryGroups {
		if len(subs) >= 2 {
			totalMonthly := 0.0
			for _, sub := range subs {
				totalMonthly += normalizeToMonthly(sub.Amount, sub.Frequency)
			}

			// Find cheapest to suggest keeping
			sort.Slice(subs, func(i, j int) bool {
				return normalizeToMonthly(subs[i].Amount, subs[i].Frequency) <
					normalizeToMonthly(subs[j].Amount, subs[j].Frequency)
			})

			cheapestMonthly := normalizeToMonthly(subs[0].Amount, subs[0].Frequency)
			potentialSave := totalMonthly - cheapestMonthly

			duplicates = append(duplicates, DuplicateServiceGroup{
				Category:      category,
				Services:      subs,
				TotalMonthly:  totalMonthly,
				SuggestedKeep: subs[0].Name,
				PotentialSave: potentialSave,
			})
		}
	}

	return duplicates
}

func calculatePotentialSavings(forgotten []Subscription, duplicates []DuplicateServiceGroup) float64 {
	savings := 0.0

	// Savings from forgotten subscriptions
	for _, sub := range forgotten {
		savings += normalizeToMonthly(sub.Amount, sub.Frequency)
	}

	// Savings from consolidating duplicates
	for _, dup := range duplicates {
		savings += dup.PotentialSave
	}

	return savings
}

func generateSubscriptionRecommendations(subscriptions, forgotten, priceIncreases []Subscription, duplicates []DuplicateServiceGroup) []SubscriptionRecommendation {
	var recommendations []SubscriptionRecommendation

	// Forgotten subscription recommendations
	for _, sub := range forgotten {
		monthly := normalizeToMonthly(sub.Amount, sub.Frequency)
		recommendations = append(recommendations, SubscriptionRecommendation{
			Type:        "cancel",
			Priority:    "high",
			Title:       fmt.Sprintf("Cancel unused %s?", sub.Name),
			Description: fmt.Sprintf("No charges since %s. You could save $%.2f/month.", sub.LastCharged, monthly),
			Savings:     monthly,
			SubID:       sub.ID,
		})
	}

	// Price increase recommendations
	for _, sub := range priceIncreases {
		if sub.PriceChange != nil && sub.PriceChange.ChangePercent > 0 {
			recommendations = append(recommendations, SubscriptionRecommendation{
				Type:        "review",
				Priority:    "medium",
				Title:       fmt.Sprintf("%s price increased %.0f%%", sub.Name, sub.PriceChange.ChangePercent),
				Description: fmt.Sprintf("Was $%.2f, now $%.2f. Consider downgrading or finding alternatives.", sub.PriceChange.OldPrice, sub.PriceChange.NewPrice),
				SubID:       sub.ID,
			})
		}
	}

	// Duplicate service recommendations
	for _, dup := range duplicates {
		names := make([]string, len(dup.Services))
		for i, svc := range dup.Services {
			names[i] = svc.Name
		}
		recommendations = append(recommendations, SubscriptionRecommendation{
			Type:        "consolidate",
			Priority:    "medium",
			Title:       fmt.Sprintf("Consolidate %s services", dup.Category),
			Description: fmt.Sprintf("You have %d %s subscriptions (%s). Consider keeping just %s.", len(dup.Services), dup.Category, strings.Join(names, ", "), dup.SuggestedKeep),
			Savings:     dup.PotentialSave,
		})
	}

	// Sort by priority and savings
	sort.Slice(recommendations, func(i, j int) bool {
		if recommendations[i].Priority != recommendations[j].Priority {
			priorityOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
			return priorityOrder[recommendations[i].Priority] < priorityOrder[recommendations[j].Priority]
		}
		return recommendations[i].Savings > recommendations[j].Savings
	})

	return recommendations
}

func calculateMonthlyCostTrend(transactions []Transaction, start, end time.Time) []SubscriptionMonthCost {
	// Group subscription transactions by month
	monthlyTotals := make(map[string]struct {
		total float64
		count int
	})

	for _, tx := range transactions {
		if tx.Type != "send" {
			continue
		}
		if tx.CreatedAt.Before(start) || tx.CreatedAt.After(end) {
			continue
		}
		monthKey := tx.CreatedAt.Format("2006-01")
		data := monthlyTotals[monthKey]
		data.total += tx.Amount
		data.count++
		monthlyTotals[monthKey] = data
	}

	// Convert to slice and sort
	var trend []SubscriptionMonthCost
	for month, data := range monthlyTotals {
		trend = append(trend, SubscriptionMonthCost{
			Month: month,
			Total: data.total,
			Count: data.count,
		})
	}

	sort.Slice(trend, func(i, j int) bool {
		return trend[i].Month < trend[j].Month
	})

	return trend
}

func calculateSubscriptionHealthScore(subscriptions []Subscription, totalMonthly float64, forgottenCount, duplicateCount int) (int, string) {
	score := 100

	// Deduct for forgotten subscriptions
	score -= forgottenCount * 15

	// Deduct for duplicate services
	score -= duplicateCount * 10

	// Deduct for high subscription spending (>$200/month)
	if totalMonthly > 200 {
		score -= int((totalMonthly - 200) / 50 * 5)
	}

	// Deduct for price increases not addressed
	for _, sub := range subscriptions {
		if sub.PriceChange != nil && sub.PriceChange.ChangePercent > 10 {
			score -= 5
		}
	}

	// Clamp score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// Determine grade
	grade := "F"
	switch {
	case score >= 90:
		grade = "A"
	case score >= 80:
		grade = "B"
	case score >= 70:
		grade = "C"
	case score >= 60:
		grade = "D"
	}

	return score, grade
}

func generateSubscriptionSummary(subscriptions []Subscription, totalMonthly, potentialSavings float64, forgottenCount int) string {
	if len(subscriptions) == 0 {
		return "No recurring subscriptions detected in your transactions."
	}

	summary := fmt.Sprintf("You have %d active subscriptions totaling $%.2f/month ($%.2f/year).", len(subscriptions), totalMonthly, totalMonthly*12)

	if potentialSavings > 0 {
		summary += fmt.Sprintf(" You could save up to $%.2f/month by reviewing your subscriptions.", potentialSavings)
	}

	if forgottenCount > 0 {
		summary += fmt.Sprintf(" %d subscription(s) appear to be unused.", forgottenCount)
	}

	return summary
}

// ============================================================================
// SUBSCRIPTION TOOL
// ============================================================================

// CreateDetectSubscriptionsTool creates the subscription detection tool
func CreateDetectSubscriptionsTool(executor core.ToolExecutor) core.Tool {
	return tools.New("detect_subscriptions").
		Description("Analyze transactions to detect recurring subscriptions and memberships. Identifies forgotten subscriptions, price increases, duplicate services, and provides recommendations to save money.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"days": tools.IntegerProperty("Number of days of transaction history to analyze (default: 90, max: 365)"),
		})).
		Handler(func(ctx context.Context, toolParams *core.ToolParams) (*core.ToolResult, error) {
			var params struct {
				Days int `json:"days"`
			}
			if err := json.Unmarshal(toolParams.Input, &params); err != nil {
				params.Days = 90
			}

			if params.Days <= 0 {
				params.Days = 90
			}
			if params.Days > 365 {
				params.Days = 365
			}

			// Fetch transactions
			var transactions []Transaction
			if DemoMode {
				transactions = GenerateMockTransactions(params.Days)
			} else {
				txReq, _ := json.Marshal(map[string]interface{}{"limit": 500})
				txResp, err := executor.Execute(ctx, &core.ExecuteRequest{
					UserID:    toolParams.UserID,
					Tool:      "get_transactions",
					Input:     txReq,
					RequestID: toolParams.RequestID,
				})
				if err != nil {
					return &core.ToolResult{Success: false, Error: err.Error()}, nil
				}
				if txResp != nil && txResp.Success {
					var txData struct{ Transactions []Transaction }
					json.Unmarshal(txResp.Data, &txData)
					transactions = txData.Transactions
				}
			}

			// Detect subscriptions
			analysis := detectSubscriptions(transactions)

			return &core.ToolResult{
				Success: true,
				Data:    analysis,
			}, nil
		}).
		Build()
}

// GenerateSubscriptionAnalysisForAPI generates subscription analysis for the HTTP API
func GenerateSubscriptionAnalysisForAPI(ctx context.Context, executor core.ToolExecutor) SubscriptionAnalysis {
	var transactions []Transaction

	if DemoMode {
		transactions = GenerateMockTransactions(90)
	} else {
		txReq, _ := json.Marshal(map[string]interface{}{"limit": 500})
		txResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
			UserID:    "user",
			Tool:      "get_transactions",
			Input:     txReq,
			RequestID: "subscriptions-api",
		})
		if txResp != nil && txResp.Success {
			var txData struct{ Transactions []Transaction }
			json.Unmarshal(txResp.Data, &txData)
			transactions = txData.Transactions
		}
	}

	return detectSubscriptions(transactions)
}
