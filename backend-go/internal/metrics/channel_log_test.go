package metrics

import (
	"testing"
	"time"
)

func TestChannelLogStoreRecordAndGet(t *testing.T) {
	store := NewChannelLogStore()
	now := time.Now()

	store.Record("key-a", &ChannelLog{RequestID: "r1", Model: "model-a", Timestamp: now.Add(-3 * time.Second)})
	store.Record("key-a", &ChannelLog{RequestID: "r2", Model: "model-b", Timestamp: now.Add(-1 * time.Second)})
	store.Record("key-b", &ChannelLog{RequestID: "r3", Model: "model-c", Timestamp: now})

	// key-a 桶应有 2 条，按时间倒序
	logsA := store.Get("key-a")
	if len(logsA) != 2 {
		t.Fatalf("key-a logs = %d, want 2", len(logsA))
	}
	if logsA[0].RequestID != "r2" || logsA[1].RequestID != "r1" {
		t.Fatalf("key-a order = [%s %s], want [r2 r1]", logsA[0].RequestID, logsA[1].RequestID)
	}

	// key-b 桶应有 1 条
	logsB := store.Get("key-b")
	if len(logsB) != 1 || logsB[0].RequestID != "r3" {
		t.Fatalf("key-b = %v, want [r3]", logsB)
	}

	// 空桶返回 nil
	if got := store.Get("key-nonexistent"); got != nil {
		t.Fatalf("nonexistent key = %v, want nil", got)
	}
}

func TestChannelLogStoreGetMerged(t *testing.T) {
	store := NewChannelLogStore()
	now := time.Now()

	// key-a: 2 条
	store.Record("key-a", &ChannelLog{RequestID: "r1", Timestamp: now.Add(-2 * time.Second)})
	store.Record("key-a", &ChannelLog{RequestID: "r2", Timestamp: now.Add(-1 * time.Second)})

	// key-b: 1 条（更早）
	store.Record("key-b", &ChannelLog{RequestID: "r3", Timestamp: now.Add(-3 * time.Second)})

	// 合并 key-a + key-b，按时间倒序
	logs := store.GetMerged([]string{"key-a", "key-b"})
	if len(logs) != 3 {
		t.Fatalf("merged count = %d, want 3", len(logs))
	}
	if logs[0].RequestID != "r2" || logs[1].RequestID != "r1" || logs[2].RequestID != "r3" {
		t.Fatalf("merged order = [%s %s %s], want [r2 r1 r3]", logs[0].RequestID, logs[1].RequestID, logs[2].RequestID)
	}

	// 重复 key 去重
	logs2 := store.GetMerged([]string{"key-a", "key-a", "key-b"})
	if len(logs2) != 3 {
		t.Fatalf("merged dedup count = %d, want 3", len(logs2))
	}

	// 空列表
	if got := store.GetMerged(nil); got != nil {
		t.Fatalf("nil keys = %v, want nil", got)
	}
}

func TestChannelLogStoreGetMergedRespectLimit(t *testing.T) {
	store := NewChannelLogStore()
	now := time.Now()

	// 向 key-a 写入 30 条，key-b 写入 30 条
	for i := 0; i < 30; i++ {
		store.Record("key-a", &ChannelLog{RequestID: "ra" + string(rune('0'+i)), Timestamp: now.Add(-time.Duration(i) * time.Second)})
		store.Record("key-b", &ChannelLog{RequestID: "rb" + string(rune('0'+i)), Timestamp: now.Add(-time.Duration(i) * time.Second)})
	}

	logs := store.GetMerged([]string{"key-a", "key-b"})
	if len(logs) > maxChannelLogs {
		t.Fatalf("merged count = %d, want <= %d", len(logs), maxChannelLogs)
	}
}

func TestChannelLogStoreRemove(t *testing.T) {
	store := NewChannelLogStore()

	store.Record("key-a", &ChannelLog{RequestID: "r1"})
	store.Record("key-b", &ChannelLog{RequestID: "r2"})
	store.Record("key-c", &ChannelLog{RequestID: "r3"})

	// 记录在途请求
	store.Record("key-a", &ChannelLog{RequestID: "r-pending", Status: StatusPending})

	store.Remove([]string{"key-a", "key-c"})

	if got := store.Get("key-a"); got != nil {
		t.Fatalf("key-a should be removed, got %v", got)
	}
	if got := store.Get("key-c"); got != nil {
		t.Fatalf("key-c should be removed, got %v", got)
	}
	// key-b 不受影响
	if got := store.Get("key-b"); len(got) != 1 {
		t.Fatalf("key-b should be untouched, got %v", got)
	}

	// 在途请求 r-pending 应该也被清理
	status, _ := store.Update("key-a", "r-pending", func(log *ChannelLog) {
		log.Status = StatusCompleted
	})
	if status != UpdateMissingDeleted {
		t.Fatalf("r-pending after Remove should be UpdateMissingDeleted, got %v", status)
	}
}

func TestChannelLogStoreRenameChannel(t *testing.T) {
	store := NewChannelLogStore()
	store.Record("key-a", &ChannelLog{RequestID: "old", ChannelName: "OldName"})
	store.Record("key-a", &ChannelLog{RequestID: "other", ChannelName: "OtherName"})

	store.RenameChannel("OldName", "NewName")
	logs := store.Get("key-a")
	if len(logs) != 2 {
		t.Fatalf("logs count = %d, want 2", len(logs))
	}
	if logs[1].ChannelName != "NewName" {
		t.Fatalf("renamed log channel = %q, want NewName", logs[1].ChannelName)
	}
	if logs[0].ChannelName != "OtherName" {
		t.Fatalf("other log channel = %q, want OtherName", logs[0].ChannelName)
	}
}

func TestChannelLogStoreUpdateUsesRequestLocationsKey(t *testing.T) {
	store := NewChannelLogStore()
	store.Record("key-a", &ChannelLog{RequestID: "r1", Status: StatusPending})

	// 通过 requestLocations 定位到 key-a，即使传入不同的 metricsKey
	status, actualKey := store.Update("key-bogus", "r1", func(log *ChannelLog) {
		log.Status = StatusConnecting
	})
	if status != UpdateFound {
		t.Fatalf("status = %v, want UpdateFound", status)
	}
	if actualKey != "key-a" {
		t.Fatalf("actualKey = %s, want key-a", actualKey)
	}
}

func TestChannelLogStoreUpdateTerminalRemovesTracking(t *testing.T) {
	store := NewChannelLogStore()
	store.Record("key-a", &ChannelLog{RequestID: "r1", Status: StatusPending})

	status, _ := store.Update("key-a", "r1", func(log *ChannelLog) {
		log.Status = StatusCompleted
	})
	if status != UpdateFound {
		t.Fatalf("status = %v, want UpdateFound", status)
	}

	// 完成后 requestLocations 应清除
	status2, _ := store.Update("key-a", "r1", func(log *ChannelLog) {
		log.Status = StatusFailed
	})
	if status2 != UpdateMissingDeleted {
		t.Fatalf("second update status = %v, want UpdateMissingDeleted", status2)
	}
}

func TestChannelLogStoreEvictedUpdate(t *testing.T) {
	store := NewChannelLogStore()
	store.Record("key-a", &ChannelLog{RequestID: "r1", Status: StatusPending})

	// 模拟环形缓冲淘汰（清空桶但保留 requestLocations）
	store.mu.Lock()
	store.logs["key-a"] = []*ChannelLog{}
	store.mu.Unlock()

	status, actualKey := store.Update("key-a", "r1", func(log *ChannelLog) {
		log.Status = StatusCompleted
	})
	if status != UpdateMissingEvicted {
		t.Fatalf("status = %v, want UpdateMissingEvicted", status)
	}
	if actualKey != "key-a" {
		t.Fatalf("actualKey = %s, want key-a", actualKey)
	}
}

func TestChannelLogStoreClearAll(t *testing.T) {
	store := NewChannelLogStore()
	store.Record("key-a", &ChannelLog{RequestID: "r1"})
	store.Record("key-b", &ChannelLog{RequestID: "r2"})

	store.ClearAll()

	if got := store.Get("key-a"); got != nil {
		t.Fatalf("after ClearAll key-a = %v, want nil", got)
	}
	if got := store.Get("key-b"); got != nil {
		t.Fatalf("after ClearAll key-b = %v, want nil", got)
	}
}

func TestChannelLogStoreRecordSkipsEmptyKey(t *testing.T) {
	store := NewChannelLogStore()
	store.Record("", &ChannelLog{RequestID: "r1"})

	if got := store.Get(""); got != nil {
		t.Fatalf("empty key should not have logs, got %v", got)
	}
}

func TestChannelLogStoreMetricsKeySnapshot(t *testing.T) {
	store := NewChannelLogStore()
	logEntry := &ChannelLog{RequestID: "r1", Status: StatusPending}
	store.Record("key-a", logEntry)

	// Record 应设置 log.MetricsKey
	if logEntry.MetricsKey != "key-a" {
		t.Fatalf("MetricsKey = %s, want key-a", logEntry.MetricsKey)
	}

	// Get 返回的深拷贝也应有 MetricsKey
	logs := store.Get("key-a")
	if len(logs) != 1 || logs[0].MetricsKey != "key-a" {
		t.Fatalf("Get MetricsKey = %s, want key-a", logs[0].MetricsKey)
	}
}

func TestChannelLogStoreRemoveSkipsInflightAtOtherKeys(t *testing.T) {
	store := NewChannelLogStore()
	store.Record("key-a", &ChannelLog{RequestID: "r-pending", Status: StatusPending})
	store.Record("key-b", &ChannelLog{RequestID: "r-pending2", Status: StatusPending})

	store.Remove([]string{"key-a"})

	// key-a 的在途请求被清理
	status1, _ := store.Update("key-a", "r-pending", func(log *ChannelLog) {
		log.Status = StatusCompleted
	})
	if status1 != UpdateMissingDeleted {
		t.Fatalf("r-pending after Remove key-a should be UpdateMissingDeleted, got %v", status1)
	}

	// key-b 的在途请求不受影响
	status2, actualKey := store.Update("key-b", "r-pending2", func(log *ChannelLog) {
		log.Status = StatusConnecting
	})
	if status2 != UpdateFound {
		t.Fatalf("r-pending2 status = %v, want UpdateFound", status2)
	}
	if actualKey != "key-b" {
		t.Fatalf("r-pending2 actualKey = %s, want key-b", actualKey)
	}
}
