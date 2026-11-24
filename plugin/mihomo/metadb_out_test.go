package mihomo

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"geoip/lib"

	"github.com/maxmind/mmdbwriter/mmdbtype"
	mihomoMMDB "github.com/metacubex/mihomo/component/mmdb"
)

func TestMetaDBOut_Output(t *testing.T) {
	container := newTestContainer(t, map[string][]string{
		"CN": {"192.0.2.0/24", "2001:db8::/32"},
		"DE": {"192.0.2.192/26", "2001:db8:1:8000::/49"},
		"JP": {"192.0.2.128/25", "2001:db8:1::/48"},
		"US": {"192.0.2.128/25", "2001:db8:1::/48"},
	})
	tests := []struct {
		name string
		opts []lib.OutputOption
		want map[string][]string
	}{
		{
			name: "overlapping IPv4 and IPv6 categories",
			want: map[string][]string{
				"192.0.2.1":          {"cn"},
				"192.0.2.129":        {"cn", "jp", "us"},
				"192.0.2.193":        {"cn", "de", "jp", "us"},
				"2001:db8:2::1":      {"cn"},
				"2001:db8:1::1":      {"cn", "jp", "us"},
				"2001:db8:1:8000::1": {"cn", "de", "jp", "us"},
				"198.51.100.1":       nil,
			},
		},
		{
			name: "wanted categories, exclusions and IPv4 only",
			opts: []lib.OutputOption{
				WithMetaDBOutWantedList([]string{" cn ", "jP", "elsewhere"}),
				WithMetaDBOutExcludedList([]string{" jp "}),
				WithMetaDBOutOnlyIPType(lib.IPv4),
			},
			want: map[string][]string{
				"192.0.2.193":   {"cn"},
				"2001:db8:1::1": nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out, err := NewMetaDBOut(lib.ActionOutput, append([]lib.OutputOption{WithMetaDBOutputDir(dir)}, tt.opts...)...)
			if err != nil {
				t.Fatal(err)
			}
			if err := out.Output(container); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "geoip.metadb")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !mihomoMMDB.Verify(path) {
				t.Fatal("Mihomo could not verify the generated database")
			}
			// Mihomo stores the reader in a process-wide singleton.
			mihomoMMDB.ReloadIP()
			t.Cleanup(mihomoMMDB.ReloadIP)
			mihomoMMDB.LoadFromBytes(data)
			reader := mihomoMMDB.IPInstance()
			if got := reader.Metadata.DatabaseType; got != "Meta-geoip0" {
				t.Errorf("database type = %q, want Meta-geoip0", got)
			}
			for addr, want := range tt.want {
				got := reader.LookupCode(net.ParseIP(addr))
				slices.Sort(got)
				if !slices.Equal(got, want) {
					t.Errorf("Mihomo LookupCode(%s) = %v, want %v", addr, got, want)
				}
			}
		})
	}
}

func TestMetaDBOut_EmptyContainer(t *testing.T) {
	dir := t.TempDir()
	out, err := NewMetaDBOut(lib.ActionOutput, WithMetaDBOutputDir(dir))
	if err != nil {
		t.Fatal(err)
	}
	if err := out.Output(lib.NewContainer()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "geoip.metadb")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty output: Stat() error = %v, want %v", err, os.ErrNotExist)
	}
}

func TestMetaDBRoundTripViaConfig(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.metadb")
	outputPath := filepath.Join(dir, "roundtrip.metadb")
	writeFile(t, inputPath, buildMetaDB(t, map[string]mmdbtype.DataType{
		"192.0.2.0/24":  mmdbtype.Slice{mmdbtype.String("cn"), mmdbtype.String("us")},
		"2001:db8::/32": mmdbtype.String("jp"),
	}))

	config, err := json.Marshal(map[string][]map[string]any{
		"input":  {{"type": "metadb", "action": "add", "args": map[string]any{"uri": inputPath}}},
		"output": {{"type": "metadb", "args": map[string]any{"outputDir": dir, "outputName": filepath.Base(outputPath)}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	instance, err := lib.NewInstance()
	if err != nil {
		t.Fatal(err)
	}
	if err := instance.InitConfigFromBytes(config); err != nil {
		t.Fatal(err)
	}
	if err := instance.Run(); err != nil {
		t.Fatal(err)
	}

	in, err := NewMetaDBIn(lib.ActionAdd, WithMetaDBURI(outputPath))
	if err != nil {
		t.Fatal(err)
	}
	container, err := in.Input(lib.NewContainer())
	if err != nil {
		t.Fatal(err)
	}
	if got := container.Len(); got != 3 {
		t.Fatalf("got %d entries, want 3", got)
	}
	assertCIDRs(t, entryCIDRs(t, container, "CN"), []string{"192.0.2.0/24"})
	assertCIDRs(t, entryCIDRs(t, container, "US"), []string{"192.0.2.0/24"})
	assertCIDRs(t, entryCIDRs(t, container, "JP"), []string{"2001:db8::/32"})
}
