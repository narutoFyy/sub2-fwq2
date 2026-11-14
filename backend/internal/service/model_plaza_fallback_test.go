//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubPlazaModelLister struct {
	byGroup map[int64][]string
	calls   map[int64]int
}

func (l *stubPlazaModelLister) ListGroupModelIDs(_ context.Context, g *Group) []string {
	if l.calls == nil {
		l.calls = map[int64]int{}
	}
	l.calls[g.ID]++
	return l.byGroup[g.ID]
}

func TestListGroupsWithModelFallback_DerivesModelsForGroupsWithoutChannels(t *testing.T) {
	channels := []Channel{{
		ID: 1, Name: "ch", Status: StatusActive, GroupIDs: []int64{20},
		ModelPricing: []ChannelModelPricing{{Platform: PlatformOpenAI, Models: []string{"gpt-5.4"}, BillingMode: BillingModeToken}},
	}}
	groups := []Group{
		{ID: 10, Name: "derived", Platform: PlatformOpenAI, RateMultiplier: 0.25, LongContextPricingEnabled: true},
		{ID: 20, Name: "channel", Platform: PlatformOpenAI, RateMultiplier: 1, LongContextPricingEnabled: true},
		{ID: 30, Name: "composite", Platform: PlatformComposite, RateMultiplier: 1},
		{ID: 40, Name: "no-accounts", Platform: PlatformOpenAI, RateMultiplier: 1},
		{ID: 50, Name: "unpriced-only", Platform: PlatformOpenAI, RateMultiplier: 1},
	}
	svc := newPlazaServiceWithBilling(channels, groups,
		map[int64]string{10: PlatformOpenAI, 20: PlatformOpenAI, 30: PlatformComposite, 40: PlatformOpenAI, 50: PlatformOpenAI},
		newStubPricingServiceFromJSON(t, openAILadderCatalogJSON))
	lister := &stubPlazaModelLister{byGroup: map[int64][]string{
		10: {"gpt-5.4", " gpt-5.4 ", "gpt-*", "", "totally-unknown-model"},
		30: {"gpt-5.4"},
		50: {"totally-unknown-model"},
	}}

	out, err := svc.ListGroupsWithModelFallback(context.Background(), lister)
	require.NoError(t, err)

	names := make([]string, 0, len(out))
	for _, g := range out {
		names = append(names, g.Name)
	}
	require.Equal(t, []string{"derived", "channel"}, names, "按倍率升序；无账号/全无定价/Composite 分组不展示")

	derived := out[0]
	require.Len(t, derived.Models, 1, "去重、跳过通配符与空串、去掉无定价模型")
	m := derived.Models[0]
	require.Equal(t, "gpt-5.4", m.Name)
	require.Equal(t, PlatformOpenAI, m.Platform)
	require.NotNil(t, m.Pricing)
	require.InDelta(t, 2.5e-6, *m.Pricing.InputPrice, 1e-15, "展示定价走计费阶梯，倍率由前端按分组乘")
	require.NotNil(t, m.OfficialPricing)

	require.Zero(t, lister.calls[20], "已有渠道模型的分组不走推导")
	require.Zero(t, lister.calls[30], "Composite 分组不走推导")
}

func TestListGroups_WithoutListerKeepsChannelOnlyBehaviour(t *testing.T) {
	groups := []Group{{ID: 10, Name: "derived", Platform: PlatformOpenAI, RateMultiplier: 1}}
	svc := newPlazaServiceWithBilling(nil, groups, map[int64]string{10: PlatformOpenAI},
		newStubPricingServiceFromJSON(t, openAILadderCatalogJSON))
	out, err := svc.ListGroups(context.Background())
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestListGroupsFiltered_SkipsHiddenGroupsWithoutQueryingThem(t *testing.T) {
	channels := []Channel{{
		ID: 1, Name: "ch", Status: StatusActive, GroupIDs: []int64{20, 30},
		ModelPricing: []ChannelModelPricing{{Platform: PlatformOpenAI, Models: []string{"gpt-5.4"}, BillingMode: BillingModeToken}},
	}}
	groups := []Group{
		{ID: 10, Name: "derived-visible", Platform: PlatformOpenAI, RateMultiplier: 0.25, LongContextPricingEnabled: true},
		{ID: 15, Name: "derived-hidden", Platform: PlatformOpenAI, RateMultiplier: 0.5, LongContextPricingEnabled: true},
		{ID: 20, Name: "channel-hidden", Platform: PlatformOpenAI, RateMultiplier: 1, LongContextPricingEnabled: true},
		{ID: 30, Name: "channel-visible", Platform: PlatformOpenAI, RateMultiplier: 2, LongContextPricingEnabled: true},
	}
	svc := newPlazaServiceWithBilling(channels, groups,
		map[int64]string{10: PlatformOpenAI, 15: PlatformOpenAI, 20: PlatformOpenAI, 30: PlatformOpenAI},
		newStubPricingServiceFromJSON(t, openAILadderCatalogJSON))
	lister := &stubPlazaModelLister{byGroup: map[int64][]string{
		10: {"gpt-5.4"},
		15: {"gpt-5.4"},
	}}

	out, err := svc.ListGroupsFiltered(context.Background(), lister, map[int64]struct{}{15: {}, 20: {}})
	require.NoError(t, err)

	names := make([]string, 0, len(out))
	for _, g := range out {
		names = append(names, g.Name)
	}
	require.Equal(t, []string{"derived-visible", "channel-visible"}, names, "被隐藏的分组（无论来自渠道还是账号推导）都不展示")
	require.Zero(t, lister.calls[15], "被隐藏的分组不应再触发账号模型推导")

	// 不传 hidden 时行为与原来一致。
	all, err := svc.ListGroupsFiltered(context.Background(), &stubPlazaModelLister{byGroup: lister.byGroup}, nil)
	require.NoError(t, err)
	require.Len(t, all, 4)
}
