package main

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v2.8.9", "v2.9.0", -1},
		{"v2.9.0", "v2.8.9", 1},
		{"v2.8.9", "v2.8.9", 0},
		{"2.8.9", "v2.8.9", 0},
		{"v2.8", "v2.8.0", 0},
		{"v2.10.0", "v2.9.99", 1},
		{"v2.8.9-rc1", "v2.8.9", 0}, // 数值部分相同，预发布后缀忽略
		{"", "v0.0.1", -1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"/"+tt.b, func(t *testing.T) {
			if got := compareVersions(tt.a, tt.b); got != tt.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestPrereleasePattern(t *testing.T) {
	cases := map[string]bool{
		"v2.9.0":          false,
		"v2.9.0-rc1":      true,
		"v2.9.0-beta.2":   true,
		"v2.9.0-alpha":    true,
		"v2.9.0-NIGHTLY":  true,
		"v2.9.0-pre.1":    true,
		"v2.9.0-canary.1": true,
		"v2.9.0-stable":   false,
	}
	for tag, want := range cases {
		got := prereleasePattern.MatchString(tag)
		if got != want {
			t.Fatalf("prereleasePattern.MatchString(%q) = %v, want %v", tag, got, want)
		}
	}
}
