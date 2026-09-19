//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRechargeTierRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		min  float64
		max  float64
		ok   bool
	}{
		{name: "legacy single lower bound", raw: "100", min: 100, max: 0, ok: true},
		{name: "closed range with dash", raw: "100-500", min: 100, max: 500, ok: true},
		{name: "closed range with tilde", raw: "100~500", min: 100, max: 500, ok: true},
		{name: "fullwidth tilde", raw: "100～500", min: 100, max: 500, ok: true},
		{name: "open upper bound", raw: "100-", min: 100, max: 0, ok: true},
		{name: "open lower bound", raw: "-500", min: 0, max: 500, ok: true},
		{name: "whitespace tolerated", raw: " 100 - 500 ", min: 100, max: 500, ok: true},
		{name: "empty rejected", raw: "", ok: false},
		{name: "non numeric rejected", raw: "abc", ok: false},
		{name: "negative rejected", raw: "-10-500", ok: false},
		{name: "inverted range rejected", raw: "500-100", ok: false},
		{name: "no lower and no upper rejected", raw: "-", ok: false},
		{name: "zero width rejected", raw: "0-0", ok: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			min, max, ok := parseRechargeTierRange(tt.raw)
			require.Equal(t, tt.ok, ok, "ok mismatch for %q", tt.raw)
			if !tt.ok {
				require.Zero(t, min)
				require.Zero(t, max)
				return
			}
			require.InDelta(t, tt.min, min, 0.0001)
			require.InDelta(t, tt.max, max, 0.0001)
		})
	}
}

func TestParseRechargeDiscountTiers(t *testing.T) {
	t.Parallel()

	t.Run("legacy threshold format stays compatible", func(t *testing.T) {
		t.Parallel()
		tiers := parseRechargeDiscountTiers("100:2,500:3")
		require.Len(t, tiers, 2)
		require.InDelta(t, 100, tiers[0].Min, 0.0001)
		require.Zero(t, tiers[0].Max)
		require.InDelta(t, 100, tiers[0].Threshold, 0.0001)
		require.InDelta(t, 2, tiers[0].Percent, 0.0001)
		require.InDelta(t, 500, tiers[1].Min, 0.0001)
		require.InDelta(t, 3, tiers[1].Percent, 0.0001)
	})

	t.Run("ranges parsed and sorted by lower bound", func(t *testing.T) {
		t.Parallel()
		tiers := parseRechargeDiscountTiers("500-1000:5,100-500:2")
		require.Len(t, tiers, 2)
		require.InDelta(t, 100, tiers[0].Min, 0.0001)
		require.InDelta(t, 500, tiers[0].Max, 0.0001)
		require.InDelta(t, 2, tiers[0].Percent, 0.0001)
		require.InDelta(t, 500, tiers[1].Min, 0.0001)
		require.InDelta(t, 1000, tiers[1].Max, 0.0001)
		require.InDelta(t, 5, tiers[1].Percent, 0.0001)
	})

	t.Run("same lower bound puts bounded range before unbounded", func(t *testing.T) {
		t.Parallel()
		tiers := parseRechargeDiscountTiers("500:3,500-1000:5")
		require.Len(t, tiers, 2)
		require.InDelta(t, 500, tiers[0].Min, 0.0001)
		require.InDelta(t, 1000, tiers[0].Max, 0.0001)
		require.InDelta(t, 5, tiers[0].Percent, 0.0001)
		require.Zero(t, tiers[1].Max)
		require.InDelta(t, 3, tiers[1].Percent, 0.0001)
	})

	t.Run("equals sign accepted", func(t *testing.T) {
		t.Parallel()
		tiers := parseRechargeDiscountTiers("100-500=2")
		require.Len(t, tiers, 1)
		require.InDelta(t, 2, tiers[0].Percent, 0.0001)
	})

	t.Run("invalid entries dropped", func(t *testing.T) {
		t.Parallel()
		tiers := parseRechargeDiscountTiers("abc:2,100:0,200:150,300-100:4,400:5")
		require.Len(t, tiers, 1)
		require.InDelta(t, 400, tiers[0].Min, 0.0001)
		require.InDelta(t, 5, tiers[0].Percent, 0.0001)
	})

	t.Run("no lower bound without upper bound dropped", func(t *testing.T) {
		t.Parallel()
		tiers := parseRechargeDiscountTiers("0:5,0-0:5")
		require.Empty(t, tiers)
	})

	t.Run("blank returns nil", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, parseRechargeDiscountTiers("   "))
	})
}

func TestFormatRechargeDiscountTiers(t *testing.T) {
	t.Parallel()

	t.Run("legacy tiers keep legacy text", func(t *testing.T) {
		t.Parallel()
		got := formatRechargeDiscountTiers([]RechargeDiscountTier{
			{Min: 100, Percent: 2, Threshold: 100},
			{Min: 500, Percent: 3, Threshold: 500},
		})
		require.Equal(t, "100:2,500:3", got)
	})

	t.Run("bounded tiers serialise with range", func(t *testing.T) {
		t.Parallel()
		got := formatRechargeDiscountTiers([]RechargeDiscountTier{
			{Min: 100, Max: 500, Percent: 2, Threshold: 100},
		})
		require.Equal(t, "100-500:2", got)
	})

	t.Run("threshold alias still serialises", func(t *testing.T) {
		t.Parallel()
		got := formatRechargeDiscountTiers([]RechargeDiscountTier{{Threshold: 200, Percent: 4}})
		require.Equal(t, "200:4", got)
	})

	t.Run("round trip keeps ranges", func(t *testing.T) {
		t.Parallel()
		// "500-"（无上限）与 "500"（满 500）语义相同，规范化后统一输出为 "500"。
		raw := "100-500:2,500-:5,1000:7"
		got := formatRechargeDiscountTiers(parseRechargeDiscountTiers(raw))
		require.Equal(t, "100-500:2,500:5,1000:7", got)
	})
}

func TestResolveRechargeDiscountPercent(t *testing.T) {
	t.Parallel()

	tiers := []RechargeDiscountTier{
		{Min: 100, Percent: 2, Threshold: 100},
		{Min: 100, Max: 500, Percent: 5, Threshold: 100},
		{Min: 500, Percent: 8, Threshold: 500},
		{Min: 1000, Max: 2000, Percent: 12, Threshold: 1000},
	}

	tests := []struct {
		name   string
		amount float64
		want   float64
	}{
		{name: "below all tiers", amount: 99.99, want: 0},
		{name: "exact lower bound hits narrow range", amount: 100, want: 5},
		{name: "inside narrow range", amount: 300, want: 5},
		{name: "upper bound still hits narrow range", amount: 500, want: 8},
		{name: "above bounded tiers uses open ended", amount: 999, want: 8},
		{name: "exact next range start", amount: 1000, want: 12},
		{name: "inside next range", amount: 1500, want: 12},
		{name: "past next range end falls back", amount: 5000, want: 8},
		{name: "zero amount", amount: 0, want: 0},
		{name: "negative amount", amount: -5, want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.InDelta(t, tt.want, resolveRechargeDiscountPercent(tt.amount, tiers), 0.0001)
		})
	}

	t.Run("invalid percent ignored", func(t *testing.T) {
		t.Parallel()
		bad := []RechargeDiscountTier{
			{Min: 1, Percent: 0, Threshold: 1},
			{Min: 2, Percent: 100, Threshold: 2},
			{Min: 3, Percent: -1, Threshold: 3},
		}
		require.Zero(t, resolveRechargeDiscountPercent(100, bad))
	})

	t.Run("threshold only legacy data still works", func(t *testing.T) {
		t.Parallel()
		legacy := []RechargeDiscountTier{{Threshold: 100, Percent: 3}}
		require.InDelta(t, 3, resolveRechargeDiscountPercent(150, legacy), 0.0001)
		require.Zero(t, resolveRechargeDiscountPercent(50, legacy))
	})

	t.Run("order independent", func(t *testing.T) {
		t.Parallel()
		reversed := make([]RechargeDiscountTier, len(tiers))
		for i, tier := range tiers {
			reversed[len(tiers)-1-i] = tier
		}
		require.InDelta(t,
			resolveRechargeDiscountPercent(300, tiers),
			resolveRechargeDiscountPercent(300, reversed), 0.0001)
	})
}
