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

// parseRechargeTierRange 解析档位的金额区间，支持以下写法：
//
//	`100`      -> [100, ∞)      满 100
//	`100-500`  -> [100, 500]    100 到 500
//	`100~500`  -> [100, 500]    （也接受全角 ～、—、–）
//	`100-`     -> [100, ∞)
//	`-500`     -> [0, 500]
//
// Customization (#20): 优惠档位由单一门槛升级为区间，便于后台配置
// "多少钱到多少钱优惠多少"。返回 ok=false 表示该写法非法。
func parseRechargeTierRange(raw string) (float64, float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, 0, false
	}
	for _, sep := range []string{"~", "\uff5e", "\u2014", "\u2013"} {
		if i := strings.Index(raw, sep); i >= 0 {
			return finishRechargeTierRange(raw[:i], raw[i+len(sep):])
		}
	}
	if i := strings.Index(raw, "-"); i >= 0 {
		return finishRechargeTierRange(raw[:i], raw[i+1:])
	}
	return finishRechargeTierRange(raw, "")
}

func finishRechargeTierRange(minText, maxText string) (float64, float64, bool) {
	minText = strings.TrimSpace(minText)
	maxText = strings.TrimSpace(maxText)
	minVal := 0.0
	if minText != "" {
		v, err := strconv.ParseFloat(minText, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return 0, 0, false
		}
		minVal = v
	} else if maxText == "" {
		return 0, 0, false
	}
	maxVal := 0.0
	if maxText != "" {
		v, err := strconv.ParseFloat(maxText, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return 0, 0, false
		}
		maxVal = v
	}
	minVal = decimal.NewFromFloat(minVal).Round(2).InexactFloat64()
	if maxVal > 0 {
		maxVal = decimal.NewFromFloat(maxVal).Round(2).InexactFloat64()
		if maxVal < minVal {
			return 0, 0, false
		}
	}
	// 上下限都没有的档位（如 "0:5" / "0-0:5"）对任何金额都成立，
	// 会让优惠变成无条件生效；这里直接判为非法配置。
	if minVal <= 0 && maxVal <= 0 {
		return 0, 0, false
	}
	return minVal, maxVal, true
}

// parseRechargeDiscountTiers 解析充值优惠档位配置，支持"满减"与"区间"两种写法：
//
//	`100:2,500:3`        -> 满 100 减 2%，满 500 减 3%（沿用 #16 旧格式）
//	`100-500:2,500-:5`   -> 100~500 减 2%，满 500 减 5%（#20 区间格式）
//
// 百分比必须满足 0 < percent < 100，否则丢弃；结果按区间下限升序返回。
func parseRechargeDiscountTiers(raw string) []RechargeDiscountTier {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	replacer := strings.NewReplacer("\uff0c", ",", "\uff1b", ",", ";", ",", "\n", ",", "\t", ",", " ", ",")
	fields := strings.Split(replacer.Replace(raw), ",")
	type tierKey struct {
		min, max float64
	}
	byKey := make(map[tierKey]float64, len(fields))
	keys := make([]tierKey, 0, len(fields))
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
		minVal, maxVal, ok := parseRechargeTierRange(parts[0])
		if !ok {
			continue
		}
		percent, errP := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if errP != nil || math.IsNaN(percent) || math.IsInf(percent, 0) || percent <= 0 || percent >= 100 {
			continue
		}
		percent = decimal.NewFromFloat(percent).Round(4).InexactFloat64()
		if percent <= 0 || percent >= 100 {
			continue
		}
		k := tierKey{minVal, maxVal}
		if _, seen := byKey[k]; !seen {
			keys = append(keys, k)
		}
		byKey[k] = percent
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].min != keys[j].min {
			return keys[i].min < keys[j].min
		}
		// 上限 0（无上限）排在后面，保证"窄区间优先"的直觉顺序。
		if keys[i].max == 0 {
			return false
		}
		if keys[j].max == 0 {
			return true
		}
		return keys[i].max < keys[j].max
	})
	if len(keys) > maxRechargeDiscountTiers {
		keys = keys[:maxRechargeDiscountTiers]
	}
	out := make([]RechargeDiscountTier, 0, len(keys))
	for _, k := range keys {
		out = append(out, RechargeDiscountTier{
			Min:       k.min,
			Max:       k.max,
			Percent:   byKey[k],
			Threshold: k.min,
		})
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
		lower := tierLowerBound(t)
		rangeText := strconv.FormatFloat(lower, 'f', -1, 64)
		// Customization (#20): 只在上限显式存在时才写区间，旧的"满 X"配置原样保留。
		if t.Max > 0 {
			rangeText += "-" + strconv.FormatFloat(t.Max, 'f', -1, 64)
		}
		parts = append(parts, rangeText+":"+strconv.FormatFloat(t.Percent, 'f', -1, 64))
	}
	return strings.Join(parts, ",")
}

// resolveRechargeDiscountPercent 返回命中档位的优惠百分比；无命中返回 0。
//
// Customization (#20): 改为区间匹配 [Min, Max]（Max=0 表示无上限）。
// 若多个区间同时覆盖该金额，取"下限最高、其次上限最窄"的那条，
// 让管理员后续补充的精细化档位可以覆盖旧的宽泛档位。
// 判定与调用顺序无关，且与前端预览逻辑保持一致。
func resolveRechargeDiscountPercent(amount float64, tiers []RechargeDiscountTier) float64 {
	if len(tiers) == 0 || math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		return 0
	}
	bestPercent := 0.0
	bestLower := -1.0
	bestUpper := -1.0
	matched := false
	for _, tier := range tiers {
		if tier.Percent <= 0 || tier.Percent >= 100 {
			continue
		}
		lower := tierLowerBound(tier)
		if amount < lower {
			continue
		}
		if tier.Max > 0 && amount > tier.Max {
			continue
		}
		narrower := false
		switch {
		case !matched:
			narrower = true
		case lower > bestLower:
			narrower = true
		case lower == bestLower:
			// 同下限时优先更窄的上限（无上限视为最宽）。
			if tier.Max > 0 && (bestUpper <= 0 || tier.Max < bestUpper) {
				narrower = true
			}
		}
		if narrower {
			matched = true
			bestPercent = tier.Percent
			bestLower = lower
			bestUpper = tier.Max
		}
	}
	if !matched {
		return 0
	}
	return bestPercent
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
