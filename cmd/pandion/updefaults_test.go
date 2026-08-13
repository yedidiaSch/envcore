// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"testing"
	"time"

	"github.com/yedidiaSch/pandion/internal/config"
	"github.com/yedidiaSch/pandion/internal/userconfig"
)

func TestApplyUpDefaults(t *testing.T) {
	const flagTTL = 45 * time.Minute // stand-in for the flag's built-in default
	def := userconfig.Defaults{Region: "nbg1", Size: "cpx21", TTL: "2h"}

	tests := []struct {
		name                 string
		size, region         string
		ttl                  time.Duration
		ttlSet, noTTL        bool
		d                    userconfig.Defaults
		wantSize, wantRegion string
		wantTTL              time.Duration
		wantWarn             bool
	}{
		{
			name: "all-unset takes every config default",
			ttl:  flagTTL, d: def,
			wantSize: "cpx21", wantRegion: "nbg1", wantTTL: 2 * time.Hour,
		},
		{
			name: "explicit flags win over config",
			size: "cx11", region: "fsn1", ttl: 10 * time.Minute, ttlSet: true, d: def,
			wantSize: "cx11", wantRegion: "fsn1", wantTTL: 10 * time.Minute,
		},
		{
			name: "no config leaves flags untouched (auto-select)",
			ttl:  flagTTL, d: userconfig.Defaults{},
			wantSize: "", wantRegion: "", wantTTL: flagTTL,
		},
		{
			name: "no-ttl suppresses the ttl default but not size/region",
			ttl:  flagTTL, noTTL: true, d: def,
			wantSize: "cpx21", wantRegion: "nbg1", wantTTL: flagTTL,
		},
		{
			name: "invalid config ttl warns and keeps the flag ttl",
			ttl:  flagTTL, d: userconfig.Defaults{TTL: "banana"},
			wantSize: "", wantRegion: "", wantTTL: flagTTL, wantWarn: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gs, gr, gt, warn := applyUpDefaults(tc.size, tc.region, tc.ttl, tc.ttlSet, tc.noTTL, tc.d)
			if gs != tc.wantSize || gr != tc.wantRegion || gt != tc.wantTTL {
				t.Errorf("got size=%q region=%q ttl=%v; want size=%q region=%q ttl=%v",
					gs, gr, gt, tc.wantSize, tc.wantRegion, tc.wantTTL)
			}
			if (warn != "") != tc.wantWarn {
				t.Errorf("warn=%q, wantWarn=%v", warn, tc.wantWarn)
			}
		})
	}
}

// TestApplyClusterFlagOverrides is a regression test for P3-03: an explicit
// `up --size` (and --region/--engine, which follow the identical
// resolveKnob/effectiveKnobs pattern per effective.go's doc comment) must win
// over cluster.yaml's defaults.* — flag > env > cluster.yaml > config > default.
// Previously the -f cluster.yaml path never applied these flags at all, so
// cluster.yaml silently won regardless of what was passed on the CLI.
func TestApplyClusterFlagOverrides(t *testing.T) {
	tests := []struct {
		name       string
		cl         config.Cluster
		ov         clusterFlagOverrides
		wantSize   string
		wantRegion string
		wantEngine string
		wantTTL    string
	}{
		{
			name:       "explicit --size overrides cluster.yaml defaults.size",
			cl:         config.Cluster{Defaults: config.NodeCommon{Size: "cpx11"}},
			ov:         clusterFlagOverrides{Size: "cpx31"},
			wantSize:   "cpx31",
			wantEngine: "",
		},
		{
			name:       "explicit --region overrides cluster.yaml provider.region",
			cl:         config.Cluster{Provider: config.Provider{Region: "nbg1"}},
			ov:         clusterFlagOverrides{Region: "fsn1"},
			wantRegion: "fsn1",
		},
		{
			name:       "--region takes only the first comma-separated preference",
			cl:         config.Cluster{Provider: config.Provider{Region: "nbg1"}},
			ov:         clusterFlagOverrides{Region: "fsn1,hel1"},
			wantRegion: "fsn1",
		},
		{
			name:       "explicit --engine overrides cluster.yaml defaults.engine",
			cl:         config.Cluster{Defaults: config.NodeCommon{Engine: "native"}},
			ov:         clusterFlagOverrides{Engine: "docker"},
			wantEngine: "docker",
		},
		{
			name:       "no overrides leaves cluster.yaml untouched",
			cl:         config.Cluster{Defaults: config.NodeCommon{Size: "cpx11", Engine: "native"}, Provider: config.Provider{Region: "nbg1"}},
			ov:         clusterFlagOverrides{},
			wantSize:   "cpx11",
			wantRegion: "nbg1",
			wantEngine: "native",
		},
		{
			name:    "explicit --ttl overrides cluster.yaml defaults.ttl",
			cl:      config.Cluster{Defaults: config.NodeCommon{TTL: "45m"}},
			ov:      clusterFlagOverrides{TTL: "2h00m"},
			wantTTL: "2h00m",
		},
		{
			name:    "--no-ttl disables ttl regardless of cluster.yaml",
			cl:      config.Cluster{Defaults: config.NodeCommon{TTL: "45m"}},
			ov:      clusterFlagOverrides{TTL: "false"},
			wantTTL: "false",
		},
		{
			name:    "no ttl override leaves cluster.yaml's ttl untouched",
			cl:      config.Cluster{Defaults: config.NodeCommon{TTL: "45m"}},
			ov:      clusterFlagOverrides{},
			wantTTL: "45m",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cl := tc.cl
			applyClusterFlagOverrides(&cl, tc.ov)
			if cl.Defaults.Size != tc.wantSize {
				t.Errorf("Defaults.Size = %q, want %q", cl.Defaults.Size, tc.wantSize)
			}
			if cl.Provider.Region != tc.wantRegion {
				t.Errorf("Provider.Region = %q, want %q", cl.Provider.Region, tc.wantRegion)
			}
			if cl.Defaults.Engine != tc.wantEngine {
				t.Errorf("Defaults.Engine = %q, want %q", cl.Defaults.Engine, tc.wantEngine)
			}
			if cl.Defaults.TTL != tc.wantTTL {
				t.Errorf("Defaults.TTL = %q, want %q", cl.Defaults.TTL, tc.wantTTL)
			}
		})
	}
}
