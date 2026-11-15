package service

// SelectGroupForModel 在一把 Key 绑定的多个分组里，为本次请求的模型选一个分组。
//
// candidates 的第一个是主分组，其余是附加分组，顺序即优先级。规则：
//  1. 只有一个可用分组，直接用它。
//  2. 无法从模型名判断平台（别名、空模型等）时，回到主分组，行为与单分组 Key 一致。
//  3. 在「平台等于模型所属平台」的分组里取排在最前、且模型白名单放行的那个；
//     都被白名单挡住时取第一个，由白名单中间件按该分组的口径拒绝。
//  4. 没有同平台分组时，再找 composite 分组（它自己按模型挑平台）。
//  5. 都没有，返回 nil，调用方应当报「这把 Key 没有可用于该模型的分组」。
func SelectGroupForModel(candidates []*Group, model string) *Group {
	usable := make([]*Group, 0, len(candidates))
	for _, g := range candidates {
		if g != nil && g.IsActive() {
			usable = append(usable, g)
		}
	}
	if len(usable) == 0 {
		return nil
	}
	if len(usable) == 1 {
		return usable[0]
	}

	platform, ok := DetectModelPlatform(model)
	if !ok {
		// 平台判断不了：优先让显式放行该模型的分组接手，否则主分组。
		if model != "" {
			for _, g := range usable {
				if g.ModelAllowlistEnabled() && g.ModelAllowlist.Allows(model) {
					return g
				}
			}
		}
		return usable[0]
	}

	if g := firstGroupAllowing(usable, model, func(g *Group) bool { return g.Platform == platform }); g != nil {
		return g
	}
	if g := firstGroupAllowing(usable, model, func(g *Group) bool { return g.Platform == PlatformComposite }); g != nil {
		return g
	}
	return nil
}

func firstGroupAllowing(groups []*Group, model string, match func(*Group) bool) *Group {
	var first *Group
	for _, g := range groups {
		if !match(g) {
			continue
		}
		if first == nil {
			first = g
		}
		if !g.ModelAllowlistEnabled() || g.ModelAllowlist.Allows(model) {
			return g
		}
	}
	return first
}
