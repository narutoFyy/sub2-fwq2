package service

import "testing"

func TestEffectiveConcurrencyUsesEnabledProxyCapacitySum(t *testing.T) {
	account := &Account{
		Concurrency: 5,
		ProxyBindings: []AccountProxyBinding{
			{ProxyID: 1, Concurrency: 5, Enabled: true},
			{ProxyID: 2, Concurrency: 5, Enabled: true},
			{ProxyID: 3, Concurrency: 5, Enabled: false},
			{ProxyID: 4, Concurrency: 0, Enabled: true},
			{ProxyID: 0, Concurrency: 10, Enabled: true},
		},
	}

	if got := account.EffectiveConcurrency(); got != 10 {
		t.Fatalf("EffectiveConcurrency() = %d, want 10", got)
	}
}

func TestEffectiveConcurrencyKeepsLegacyValueWithoutEnabledProxyCapacity(t *testing.T) {
	account := &Account{
		Concurrency: 7,
		ProxyBindings: []AccountProxyBinding{
			{ProxyID: 1, Concurrency: 5, Enabled: false},
			{ProxyID: 2, Concurrency: 0, Enabled: true},
		},
	}

	if got := account.EffectiveConcurrency(); got != 7 {
		t.Fatalf("EffectiveConcurrency() = %d, want 7", got)
	}
}

func TestEffectiveLoadFactorUsesProxyCapacitySum(t *testing.T) {
	account := &Account{
		Concurrency: 5,
		ProxyBindings: []AccountProxyBinding{
			{ProxyID: 1, Concurrency: 5, Enabled: true},
			{ProxyID: 2, Concurrency: 5, Enabled: true},
		},
	}

	if got := account.EffectiveLoadFactor(); got != 10 {
		t.Fatalf("EffectiveLoadFactor() = %d, want 10", got)
	}
}

func TestEffectiveLoadFactorPreservesExplicitLoadFactor(t *testing.T) {
	loadFactor := 4
	account := &Account{
		Concurrency: 5,
		LoadFactor:  &loadFactor,
		ProxyBindings: []AccountProxyBinding{
			{ProxyID: 1, Concurrency: 5, Enabled: true},
			{ProxyID: 2, Concurrency: 5, Enabled: true},
		},
	}

	if got := account.EffectiveLoadFactor(); got != 4 {
		t.Fatalf("EffectiveLoadFactor() = %d, want explicit 4", got)
	}
}
