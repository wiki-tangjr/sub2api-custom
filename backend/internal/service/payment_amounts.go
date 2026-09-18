package service

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const defaultBalanceRechargeMultiplier = 1.0

func normalizeBalanceRechargeMultiplier(multiplier float64) float64 {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier <= 0 {
		return defaultBalanceRechargeMultiplier
	}
	return multiplier
}

// normalizeSubscriptionUSDToCNYRate 将非法值归一为 0（换算关闭）。
// 与余额倍率不同，0 是合法状态：表示订阅保持 price 直付的存量行为。
func normalizeSubscriptionUSDToCNYRate(rate float64) float64 {
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 {
		return 0
	}
	return rate
}

func calculateCreditedBalance(paymentAmount, multiplier float64) float64 {
	return decimal.NewFromFloat(paymentAmount).
		Mul(decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(multiplier))).
		Round(2).
		InexactFloat64()
}

func calculateGatewayRefundAmount(orderAmount, payAmount, refundAmount float64, currency string) float64 {
	if orderAmount <= 0 || payAmount <= 0 || refundAmount <= 0 {
		return 0
	}
	fractionDigits := int32(payment.CurrencyMaxFractionDigits(currency))
	if math.Abs(refundAmount-orderAmount) <= paymentAmountToleranceForCurrency(currency) {
		return decimal.NewFromFloat(payAmount).Round(fractionDigits).InexactFloat64()
	}
	return decimal.NewFromFloat(payAmount).
		Mul(decimal.NewFromFloat(refundAmount)).
		Div(decimal.NewFromFloat(orderAmount)).
		Round(fractionDigits).
		InexactFloat64()
}

// --- Customization (#16): quick recharge amounts and tiered recharge discount ---

// parseRechargeQuickAmounts 解析后台配置的快捷充值按钮金额。
// 支持英文/中文逗号、分号、空格与换行分隔；非法值直接丢弃。
// 返回空切片表示"未配置"，此时前端沿用历史默认按钮，页面零变化。
func parseRechargeQuickAmounts(raw string) []float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	replacer := strings.NewReplacer("\uff0c", ",", "\uff1b", ",", ";", ",", "\n", ",", "\t", ",", " ", ",")
	fields := strings.Split(replacer.Replace(raw), ",")
	out := make([]float64, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		v, err := strconv.ParseFloat(f, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			continue
		}
		// 四舍五入到 2 位小数，避免浮点尾差写回设置。
		v = decimal.NewFromFloat(v).Round(2).InexactFloat64()
		if v <= 0 {
			continue
		}
		key := strconv.FormatFloat(v, 'f', -1, 64)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
		if len(out) >= maxRechargeQuickAmounts {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.Float64s(out)
	return out
}

// FormatRechargeQuickAmounts 供 admin handler 回读设置使用，输出规范化文本。
func FormatRechargeQuickAmounts(vals []float64) string {
	return formatRechargeQuickAmounts(vals)
}

// formatRechargeQuickAmounts 将数组序列化为设置值。
func formatRechargeQuickAmounts(vals []float64) string {
	if len(vals) == 0 {
		return ""
	}
	parts := make([]string, 0, len(vals))
	for _, v := range vals {
		parts = append(parts, strconv.FormatFloat(v, 'f', -1, 64))
	}
	return strings.Join(parts, ",")
}

// parseRechargeDiscountTiers 解析 `threshold:percent` 形式的阶梯优惠配置。
// 例：`100:2,500:3,1000:5` 表示充值满 100 减 2%、满 500 减 3%、满 1000 减 5%。
// 百分比必须满足 0 < percent < 100，否则丢弃；按 threshold 升序返回。
func parseRechargeDiscountTiers(raw string) []RechargeDiscountTier {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	replacer := strings.NewReplacer("\uff0c", ",", "\uff1b", ",", ";", ",", "\n", ",", "\t", ",", " ", ",")
	fields := strings.Split(replacer.Replace(raw), ",")
	byThreshold := make(map[float64]float64, len(fields))
	order := make([]float64, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		sep := ":"
		if !strings.Contains(f, sep) {
			sep = "="
		}
		parts := strings.SplitN(f, sep, 2)
		if len(parts) != 2 {
			continue
		}
		threshold, errT := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		percent, errP := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if errT != nil || errP != nil {
			continue
		}
		if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold <= 0 {
			continue
		}
		if math.IsNaN(percent) || math.IsInf(percent, 0) || percent <= 0 || percent >= 100 {
			continue
		}
		threshold = decimal.NewFromFloat(threshold).Round(2).InexactFloat64()
		percent = decimal.NewFromFloat(percent).Round(4).InexactFloat64()
		if threshold <= 0 || percent <= 0 || percent >= 100 {
			continue
		}
		if _, ok := byThreshold[threshold]; !ok {
			order = append(order, threshold)
		}
		byThreshold[threshold] = percent
	}
	if len(order) == 0 {
		return nil
	}
	sort.Float64s(order)
	if len(order) > maxRechargeDiscountTiers {
		order = order[:maxRechargeDiscountTiers]
	}
	out := make([]RechargeDiscountTier, 0, len(order))
	for _, t := range order {
		out = append(out, RechargeDiscountTier{Threshold: t, Percent: byThreshold[t]})
	}
	return out
}

// FormatRechargeDiscountTiers 供 admin handler 回读设置使用，输出规范化文本。
func FormatRechargeDiscountTiers(tiers []RechargeDiscountTier) string {
	return formatRechargeDiscountTiers(tiers)
}

// formatRechargeDiscountTiers 将阶梯优惠序列化为设置值。
func formatRechargeDiscountTiers(tiers []RechargeDiscountTier) string {
	if len(tiers) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tiers))
	for _, t := range tiers {
		parts = append(parts, strconv.FormatFloat(t.Threshold, 'f', -1, 64)+":"+strconv.FormatFloat(t.Percent, 'f', -1, 64))
	}
	return strings.Join(parts, ",")
}

// resolveRechargeDiscountPercent 返回命中阶梯的优惠百分比；无命中返回 0。
// 阶梯按阈值升序，取"阈值不超过充值金额"的最后一条（即最高档）。
func resolveRechargeDiscountPercent(amount float64, tiers []RechargeDiscountTier) float64 {
	if len(tiers) == 0 || math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		return 0
	}
	best := 0.0
	for _, tier := range tiers {
		if tier.Threshold > amount {
			break
		}
		if tier.Percent > 0 && tier.Percent < 100 {
			best = tier.Percent
		}
	}
	return best
}

// applyRechargeDiscount 按阶梯优惠折算实付金额。percent 为 0 时原样返回，
// 保证未配置优惠时与历史行为完全一致。
func applyRechargeDiscount(payAmount float64, percent float64, currency string) float64 {
	if percent <= 0 || percent >= 100 || payAmount <= 0 {
		return payAmount
	}
	fractionDigits := int32(payment.CurrencyMaxFractionDigits(currency))
	// 与既有手续费逻辑一致：向上取整到币种最小支付单位，避免支付网关拒单。
	return decimal.NewFromFloat(payAmount).
		Mul(decimal.NewFromInt(100).Sub(decimal.NewFromFloat(percent)).Div(decimal.NewFromInt(100))).
		RoundUp(fractionDigits).
		InexactFloat64()
}
