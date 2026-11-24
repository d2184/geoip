package mihomo

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"geoip/lib"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/oschwald/maxminddb-golang/v2/mmdbdata"
)

func buildMetaDB(t *testing.T, records map[string]mmdbtype.DataType) []byte {
	t.Helper()
	db, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            "Meta-geoip0",
		DisableIPv4Aliasing:     true,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for prefix, record := range records {
		_, network, err := net.ParseCIDR(prefix)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Insert(network, record); err != nil {
			t.Fatalf("Insert(%s): %v", prefix, err)
		}
	}
	var buf bytes.Buffer
	if _, err := db.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestMetaDBIn_Input(t *testing.T) {
	data := buildMetaDB(t, map[string]mmdbtype.DataType{
		"192.0.2.0/25":   mmdbtype.String(" cN "),
		"192.0.2.128/25": mmdbtype.Slice{mmdbtype.String("cn"), mmdbtype.String(" US "), mmdbtype.String(" ")},
		"2001:db8::/32":  mmdbtype.Slice{mmdbtype.String("cn"), mmdbtype.String("us")},
	})
	path := filepath.Join(t.TempDir(), "input.metadb")
	writeFile(t, path, data)

	t.Run("local file with scalar and array records", func(t *testing.T) {
		in, err := NewMetaDBIn(lib.ActionAdd, WithMetaDBURI(path))
		if err != nil {
			t.Fatal(err)
		}
		container, err := in.Input(lib.NewContainer())
		if err != nil {
			t.Fatal(err)
		}
		if got := container.Len(); got != 2 {
			t.Fatalf("got %d entries, want 2", got)
		}
		assertCIDRs(t, entryCIDRs(t, container, "CN"), []string{"192.0.2.0/24", "2001:db8::/32"})
		assertCIDRs(t, entryCIDRs(t, container, "US"), []string{"192.0.2.128/25", "2001:db8::/32"})
	})

	t.Run("remote file with category and address family filters", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(data)
		}))
		t.Cleanup(server.Close)

		in, err := NewMetaDBIn(lib.ActionAdd,
			WithMetaDBURI(server.URL),
			WithMetaDBInWantedList([]string{" cn "}),
			WithMetaDBInOnlyIPType(lib.IPv6),
		)
		if err != nil {
			t.Fatal(err)
		}
		container, err := in.Input(lib.NewContainer())
		if err != nil {
			t.Fatal(err)
		}
		if got := container.Len(); got != 1 {
			t.Fatalf("got %d entries, want 1", got)
		}
		assertCIDRs(t, entryCIDRs(t, container, "CN"), []string{"2001:db8::/32"})
	})

	t.Run("remove selected prefixes", func(t *testing.T) {
		in, err := NewMetaDBIn(lib.ActionRemove,
			WithMetaDBURI(path),
			WithMetaDBInWantedList([]string{"us"}),
		)
		if err != nil {
			t.Fatal(err)
		}
		container := newTestContainer(t, map[string][]string{
			"US": {"192.0.2.0/24", "2001:db8::/32"},
			"CN": {"192.0.2.0/24"},
		})
		got, err := in.Input(container)
		if err != nil {
			t.Fatal(err)
		}
		assertCIDRs(t, entryCIDRs(t, got, "US"), []string{"192.0.2.0/25"})
		assertCIDRs(t, entryCIDRs(t, got, "CN"), []string{"192.0.2.0/24"})
	})
}

func TestMetaDBIn_InputErrors(t *testing.T) {
	badRecord := buildMetaDB(t, map[string]mmdbtype.DataType{"192.0.2.0/24": mmdbtype.Uint32(42)})
	valid := buildMetaDB(t, map[string]mmdbtype.DataType{"192.0.2.0/24": mmdbtype.String("cn")})
	tests := []struct {
		name      string
		data      []byte
		wanted    []string
		wantErr   error
		invalidDB bool
		message   string
	}{
		{name: "missing file", wantErr: os.ErrNotExist},
		{name: "invalid database", data: []byte("not an MMDB"), invalidDB: true},
		{name: "unsupported record", data: badRecord, message: "decode metadb record for network 192.0.2.0/24"},
		{name: "no matching categories", data: valid, wanted: []string{"US"}, message: "no entry is generated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.metadb")
			if tt.data != nil {
				writeFile(t, path, tt.data)
			}
			in, err := NewMetaDBIn(lib.ActionAdd, WithMetaDBURI(path), WithMetaDBInWantedList(tt.wanted))
			if err != nil {
				t.Fatal(err)
			}
			_, err = in.Input(lib.NewContainer())
			if err == nil || (tt.wantErr != nil && !errors.Is(err, tt.wantErr)) || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("Input() error = %v, want %v with message containing %q", err, tt.wantErr, tt.message)
			}
			if tt.invalidDB {
				if _, ok := errors.AsType[mmdbdata.InvalidDatabaseError](err); !ok {
					t.Fatalf("Input() error = %v, want InvalidDatabaseError", err)
				}
			}
		})
	}
}
