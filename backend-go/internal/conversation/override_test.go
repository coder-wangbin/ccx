package conversation

import (
	"testing"
	"time"
)

func TestOverrideManager_SetAndGet(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	seq := []ChannelEntry{
		{ChannelIndex: 1, ChannelName: "backup"},
		{ChannelIndex: 0, ChannelName: "primary"},
	}

	err := om.SetOverride("conv_abc", "chat", "user1", seq, 0)
	if err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}

	result, ok := om.GetOverrideForUser("chat", "user1")
	if !ok {
		t.Fatal("expected override to exist")
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result[0].ChannelIndex != 1 {
		t.Errorf("expected first channel index=1, got %d", result[0].ChannelIndex)
	}
}

func TestOverrideManager_Remove(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq, 0)

	removed := om.RemoveOverride("conv_abc")
	if !removed {
		t.Error("expected RemoveOverride to return true")
	}

	_, ok := om.GetOverrideForUser("chat", "user1")
	if ok {
		t.Error("expected override to be removed")
	}
}

func TestOverrideManager_TTLExpiry(t *testing.T) {
	om := NewOverrideManager(1 * time.Millisecond)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq, 0)

	time.Sleep(5 * time.Millisecond)

	_, ok := om.GetOverrideForUser("chat", "user1")
	if ok {
		t.Error("expected override to be expired")
	}
}

func TestOverrideManager_EmptySequence(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	err := om.SetOverride("conv_abc", "chat", "user1", []ChannelEntry{}, 0)
	if err == nil {
		t.Error("expected error for empty sequence")
	}
}

func TestOverrideManager_GetAllOverrides(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	om.SetOverride("conv_1", "chat", "user1", []ChannelEntry{{ChannelIndex: 0, ChannelName: "a"}}, 0)
	om.SetOverride("conv_2", "messages", "user2", []ChannelEntry{{ChannelIndex: 1, ChannelName: "b"}}, 0)

	all := om.GetAllOverrides()
	if len(all) != 2 {
		t.Errorf("expected 2 overrides, got %d", len(all))
	}
}

func TestOverrideManager_RefreshTTL(t *testing.T) {
	om := NewOverrideManager(100 * time.Millisecond)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq, 0)

	time.Sleep(50 * time.Millisecond)
	om.RefreshTTL("conv_abc")
	time.Sleep(70 * time.Millisecond)

	_, ok := om.GetOverrideForUser("chat", "user1")
	if !ok {
		t.Error("expected override to still be valid after TTL refresh")
	}
}

func TestOverrideManager_SetOverride_CustomDuration(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	err := om.SetOverride("conv_abc", "chat", "user1", seq, 15*time.Minute)
	if err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}

	override, ok := om.GetOverride("conv_abc")
	if !ok {
		t.Fatal("expected override to exist")
	}

	// ExpiresAt 应在 ~15 分钟后，而非系统默认的 30 分钟
	diff := time.Until(override.ExpiresAt)
	if diff > 16*time.Minute || diff < 14*time.Minute {
		t.Errorf("expected ExpiresAt ~15min from now, got %v remaining", diff)
	}
	if override.IsPerpetual {
		t.Error("expected IsPerpetual=false for custom duration")
	}
}

func TestOverrideManager_SetOverride_Perpetual(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	err := om.SetOverride("conv_abc", "chat", "user1", seq, -1)
	if err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}

	override, ok := om.GetOverride("conv_abc")
	if !ok {
		t.Fatal("expected override to exist")
	}
	if !override.IsPerpetual {
		t.Error("expected IsPerpetual=true for perpetual override")
	}
}

func TestOverrideManager_PerpetualNeverExpires(t *testing.T) {
	om := NewOverrideManager(1 * time.Millisecond)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq, -1)

	time.Sleep(10 * time.Millisecond)

	_, ok := om.GetOverride("conv_abc")
	if !ok {
		t.Error("expected perpetual override to still be valid")
	}

	_, ok = om.GetOverrideForUser("chat", "user1")
	if !ok {
		t.Error("expected perpetual override to still be valid via GetUser")
	}
}

func TestOverrideManager_CleanupSkipsPerpetual(t *testing.T) {
	om := NewOverrideManager(1 * time.Millisecond)
	defer om.Stop()

	// 普通 override 和永久 override
	om.SetOverride("conv_normal", "chat", "user1", []ChannelEntry{{ChannelIndex: 0}}, 0)
	om.SetOverride("conv_perpetual", "chat", "user2", []ChannelEntry{{ChannelIndex: 1}}, -1)

	time.Sleep(10 * time.Millisecond)
	om.cleanup()

	// 普通 override 应被清理
	_, ok := om.GetOverride("conv_normal")
	if ok {
		t.Error("expected normal override to be cleaned up")
	}

	// 永久 override 应保留
	_, ok = om.GetOverride("conv_perpetual")
	if !ok {
		t.Error("expected perpetual override to survive cleanup")
	}
}

func TestOverrideManager_RefreshTTL_PerpetualNoOp(t *testing.T) {
	om := NewOverrideManager(100 * time.Millisecond)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq, -1)

	// RefreshTTL 对永久 override 应返回 false
	ok := om.RefreshTTL("conv_abc")
	if ok {
		t.Error("expected RefreshTTL to return false for perpetual override")
	}
}

func TestOverrideManager_RefreshOverrideForUser(t *testing.T) {
	om := NewOverrideManager(100 * time.Millisecond)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq, 0)

	time.Sleep(50 * time.Millisecond)

	// 续期
	ok := om.RefreshOverrideForUser("chat", "user1")
	if !ok {
		t.Fatal("expected RefreshOverrideForUser to return true")
	}

	// 再等 70ms（从设置算起已过 120ms，但续期后应仍在有效期内）
	time.Sleep(70 * time.Millisecond)

	_, ok = om.GetOverrideForUser("chat", "user1")
	if !ok {
		t.Error("expected override to still be valid after RefreshOverrideForUser")
	}
}

func TestOverrideManager_RefreshOverrideForUser_PerpetualNoOp(t *testing.T) {
	om := NewOverrideManager(100 * time.Millisecond)
	defer om.Stop()

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq, -1)

	ok := om.RefreshOverrideForUser("chat", "user1")
	if ok {
		t.Error("expected RefreshOverrideForUser to return false for perpetual override")
	}
}

func TestOverrideManager_RefreshPreservesCustomDuration(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute) // 系统默认 30 分钟
	defer om.Stop()

	// 用户选择 1 小时有效期
	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq, 1*time.Hour)

	// 续期应使用 1 小时，而非系统默认 30 分钟
	ok := om.RefreshOverrideForUser("chat", "user1")
	if !ok {
		t.Fatal("expected RefreshOverrideForUser to return true")
	}

	override, ok := om.GetOverride("conv_abc")
	if !ok {
		t.Fatal("expected override to exist")
	}

	// ExpiresAt 应在 ~1 小时后（而非 30 分钟）
	diff := time.Until(override.ExpiresAt)
	if diff < 50*time.Minute || diff > 65*time.Minute {
		t.Errorf("expected ExpiresAt ~1h after refresh (preserving custom duration), got %v remaining", diff)
	}
}

func TestOverrideManager_SetDefaultTTL(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	// 修改默认 TTL 为 1 小时
	om.SetDefaultTTL(1 * time.Hour)

	seq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	om.SetOverride("conv_abc", "chat", "user1", seq, 0)

	override, ok := om.GetOverride("conv_abc")
	if !ok {
		t.Fatal("expected override to exist")
	}

	// ExpiresAt 应在 ~1 小时后
	diff := time.Until(override.ExpiresAt)
	if diff > 61*time.Minute || diff < 59*time.Minute {
		t.Errorf("expected ExpiresAt ~1h from now after SetDefaultTTL, got %v remaining", diff)
	}
}

func TestOverrideManager_SubagentOverrideDoesNotAffectMainSequence(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	subagentSeq := []ChannelEntry{
		{ChannelIndex: 2, ChannelName: "subagent"},
		{ChannelIndex: 0, ChannelName: "primary"},
	}

	if err := om.SetSubagentOverride("conv_abc", "chat", "user1", subagentSeq, nil, 0); err != nil {
		t.Fatalf("SetSubagentOverride failed: %v", err)
	}

	if _, ok := om.GetOverrideForUser("chat", "user1"); ok {
		t.Fatal("expected main conversation to ignore subagent-only override")
	}

	if _, ok := om.GetOverrideForUserWithRole("chat", "user1", "main"); ok {
		t.Fatal("expected main role to ignore subagent-only override")
	}

	result, ok := om.GetOverrideForUserWithRole("chat", "user1", "subagent")
	if !ok {
		t.Fatal("expected subagent override to exist")
	}
	if len(result) != 2 || result[0].ChannelIndex != 2 {
		t.Fatalf("unexpected subagent override sequence: %#v", result)
	}

	override, ok := om.GetOverride("conv_abc")
	if !ok {
		t.Fatal("expected override snapshot to exist for UI")
	}
	if override.HasMainSequence {
		t.Fatal("expected subagent-only override to report hasMainSequence=false")
	}
}

func TestOverrideManager_ClearSubagentOnlyOverrideRemovesSnapshot(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	if err := om.SetSubagentOverride("conv_abc", "chat", "user1", []ChannelEntry{{ChannelIndex: 2}}, nil, 0); err != nil {
		t.Fatalf("SetSubagentOverride failed: %v", err)
	}

	if !om.ClearSubagentOverride("conv_abc") {
		t.Fatal("expected ClearSubagentOverride to return true")
	}

	if _, ok := om.GetOverride("conv_abc"); ok {
		t.Fatal("expected subagent-only override snapshot to be removed")
	}
	if _, ok := om.GetOverrideForUserWithRole("chat", "user1", "subagent"); ok {
		t.Fatal("expected subagent routing override to be removed")
	}
}

func TestOverrideManager_ClearSubagentKeepsMainOverride(t *testing.T) {
	om := NewOverrideManager(30 * time.Minute)
	defer om.Stop()

	mainSeq := []ChannelEntry{{ChannelIndex: 0, ChannelName: "primary"}}
	subagentSeq := []ChannelEntry{{ChannelIndex: 2, ChannelName: "subagent"}}

	if err := om.SetOverride("conv_abc", "chat", "user1", mainSeq, 0); err != nil {
		t.Fatalf("SetOverride failed: %v", err)
	}
	if err := om.SetSubagentOverride("conv_abc", "chat", "user1", subagentSeq, mainSeq, 0); err != nil {
		t.Fatalf("SetSubagentOverride failed: %v", err)
	}

	if !om.ClearSubagentOverride("conv_abc") {
		t.Fatal("expected ClearSubagentOverride to return true")
	}

	result, ok := om.GetOverrideForUser("chat", "user1")
	if !ok {
		t.Fatal("expected main override to remain")
	}
	if len(result) != 1 || result[0].ChannelIndex != 0 {
		t.Fatalf("unexpected main override sequence: %#v", result)
	}

	override, ok := om.GetOverride("conv_abc")
	if !ok {
		t.Fatal("expected main override snapshot to remain")
	}
	if !override.HasMainSequence {
		t.Fatal("expected main override to keep hasMainSequence=true")
	}
	if len(override.SubagentSequence) != 0 {
		t.Fatalf("expected subagent sequence to be cleared, got %#v", override.SubagentSequence)
	}
}
