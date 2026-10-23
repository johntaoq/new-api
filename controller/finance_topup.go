package controller

import (
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
)

// An explicit currency follows the production wallet contract. Requests from
// older upstream clients retain their existing amount/display semantics.
func calculateFinanceTopup(amount int64, currency, group string) (model.TopUp, error) {
	order := model.TopUp{InputAmount: float64(amount)}
	rate := operation_setting.USDExchangeRate
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) || amount <= 0 {
		return order, common.NewMessage("Invalid top-up amount")
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	var quota int
	var err error
	if currency == "" {
		if amount < getMinTopup() {
			return order, common.NewMessage("Top-up amount cannot be less than {{min}}", map[string]any{"min": getMinTopup()})
		}
		quota, err = validateTopUpQuota(amount)
		order.Money = getPayMoney(amount, group)
		order.Currency = "USD"
	} else {
		if currency != "CNY" && currency != "USD" {
			return order, common.NewMessage("Unsupported top-up currency")
		}
		baseUSD := decimal.NewFromInt(amount)
		cny := baseUSD.Mul(decimal.NewFromFloat(rate))
		if currency == "CNY" {
			cny = decimal.NewFromInt(amount)
			baseUSD = cny.Div(decimal.NewFromFloat(rate))
		}
		if cny.LessThan(decimal.NewFromInt(int64(max(10, operation_setting.MinTopUp)))) {
			return order, common.NewMessage("Top-up amount cannot be less than {{min}} CNY", map[string]any{"min": max(10, operation_setting.MinTopUp)})
		}
		quota, err = common.WalletQuotaFromDecimalStrict(baseUSD.Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Floor())
		ratio := common.GetTopupGroupRatio(group)
		if ratio == 0 {
			ratio = 1
		}
		discount := operation_setting.GetPaymentSetting().AmountDiscount[int(amount)]
		if discount == 0 {
			discount = 1
		}
		if ratio <= 0 || discount <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) ||
			math.IsNaN(discount) || math.IsInf(discount, 0) {
			return order, common.NewMessage("Invalid financial amount")
		}
		order.Money = cny.Mul(decimal.NewFromFloat(ratio)).Mul(decimal.NewFromFloat(discount)).Round(2).InexactFloat64()
		order.Currency = currency
	}
	if err != nil {
		return order, err
	}
	if quota <= 0 {
		return order, common.NewMessage("Invalid top-up amount")
	}
	if math.IsNaN(order.Money) || math.IsInf(order.Money, 0) || order.Money < 0.01 {
		return order, common.NewMessage("Payment amount is too low")
	}
	order.Quota = int64(quota)
	order.Amount = decimal.NewFromInt(int64(quota)).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	order.ExchangeRate = rate
	order.RevenueUSD = decimal.NewFromFloat(order.Money).Div(decimal.NewFromFloat(rate)).Round(6).InexactFloat64()
	return order, nil
}
