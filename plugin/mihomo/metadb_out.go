package mihomo

import (
	"cmp"
	"encoding/json"
	"log"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Loyalsoldier/geoip/lib"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/inserter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/sagernet/sing/common"
)

const (
	TypeMetaDBOut = "metadb"
	DescMetaDBOut = "Convert data to MetaDB (Meta-geoip0) metadb format"
)

var (
	defaultMetaDBOutputName = "geoip.metadb"
	defaultMetaDBOutputDir  = filepath.Join(".", "output", "metadb")
)

func init() {
	lib.RegisterOutputConfigCreator(TypeMetaDBOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newMetaDBOut(action, data)
	})
	lib.RegisterOutputConverter(TypeMetaDBOut, &MetaDBOut{
		Description: DescMetaDBOut,
	})
}

func newMetaDBOut(action lib.Action, data json.RawMessage) (*MetaDBOut, error) {
	var tmp struct {
		OutputName string     `json:"outputName"`
		OutputDir  string     `json:"outputDir"`
		Want       []string   `json:"wantedList"`
		Exclude    []string   `json:"excludedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.OutputName == "" {
		tmp.OutputName = defaultMetaDBOutputName
	}

	if tmp.OutputDir == "" {
		tmp.OutputDir = defaultMetaDBOutputDir
	}

	return &MetaDBOut{
		Type:        TypeMetaDBOut,
		Action:      action,
		Description: DescMetaDBOut,
		OutputName:  tmp.OutputName,
		OutputDir:   tmp.OutputDir,
		Want:        tmp.Want,
		Exclude:     tmp.Exclude,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type MetaDBOut struct {
	Type        string
	Action      lib.Action
	Description string
	OutputName  string
	OutputDir   string
	Want        []string
	Exclude     []string
	OnlyIPType  lib.IPType
}

func (m *MetaDBOut) GetType() string {
	return m.Type
}

func (m *MetaDBOut) GetAction() lib.Action {
	return m.Action
}

func (m *MetaDBOut) GetDescription() string {
	return m.Description
}

func (m *MetaDBOut) Output(container lib.Container) error {
	names := m.filterEntries(container)
	if len(names) == 0 {
		return nil
	}

	return m.marshalData(container, names)
}

func (m *MetaDBOut) filterEntries(container lib.Container) []string {
	excludeMap := make(map[string]bool)
	for _, exclude := range m.Exclude {
		if exclude = strings.ToUpper(strings.TrimSpace(exclude)); exclude != "" {
			excludeMap[exclude] = true
		}
	}

	wantList := make([]string, 0, len(m.Want))
	for _, want := range m.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" && !excludeMap[want] {
			wantList = append(wantList, want)
		}
	}

	if len(wantList) > 0 {
		// Sort the list
		slices.Sort(wantList)
		return wantList
	}

	list := make([]string, 0, 300)
	for entry := range container.Loop() {
		name := entry.GetName()
		if excludeMap[name] {
			continue
		}
		list = append(list, name)
	}

	// Sort the list
	slices.Sort(list)

	return list
}

func (m *MetaDBOut) marshalData(container lib.Container, names []string) error {
	codeMap := make(map[netip.Prefix][]string)
	included := make([]netip.Prefix, 0)

	for _, name := range names {
		entry, found := container.GetEntry(name)
		if !found {
			log.Printf("❌ entry %s not found\n", name)
			continue
		}

		code := strings.ToLower(name)
		prefixes, err := entry.MarshalPrefix(lib.GetIgnoreIPType(m.OnlyIPType))
		if err != nil {
			continue
		}

		for _, prefix := range prefixes {
			p := netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits())
			included = append(included, p)
			codeMap[p] = append(codeMap[p], code)
		}
	}

	if len(included) == 0 {
		return nil
	}

	included = common.Uniq(included)

	slices.SortFunc(included, func(a, b netip.Prefix) int {
		return cmp.Compare(a.Bits(), b.Bits())
	})

	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType:            "Meta-geoip0",
		IPVersion:               6,
		RecordSize:              24,
		Inserter:                inserter.ReplaceWith,
		DisableIPv4Aliasing:     true,
		IncludeReservedNetworks: true,
	})
	if err != nil {
		return err
	}

	for _, prefix := range included {
		ipNet := &net.IPNet{
			IP:   prefix.Addr().AsSlice(),
			Mask: net.CIDRMask(prefix.Bits(), prefix.Addr().BitLen()),
		}

		codes := common.Uniq(codeMap[prefix])

		_, record := writer.Get(ipNet.IP)

		record = m.mergeRecord(record, codes)
		if err := writer.Insert(ipNet, record); err != nil {
			return err
		}
	}

	return m.writeFile(writer)
}

func (m *MetaDBOut) mergeRecord(existing mmdbtype.DataType, newCodes []string) mmdbtype.DataType {
	if len(newCodes) == 0 {
		return existing
	}

	switch r := existing.(type) {
	case nil:
		if len(newCodes) == 1 {
			return mmdbtype.String(newCodes[0])
		}
		slice := make([]mmdbtype.DataType, len(newCodes))
		for i, c := range newCodes {
			slice[i] = mmdbtype.String(c)
		}
		return mmdbtype.Slice(slice)

	case mmdbtype.String:
		all := append([]string{string(r)}, newCodes...)
		all = common.Uniq(all)
		if len(all) == 1 {
			return mmdbtype.String(all[0])
		}
		slice := make([]mmdbtype.DataType, len(all))
		for i, c := range all {
			slice[i] = mmdbtype.String(c)
		}
		return mmdbtype.Slice(slice)

	case mmdbtype.Slice:
		all := make([]string, 0, len(r)+len(newCodes))
		for _, item := range r {
			if s, ok := item.(mmdbtype.String); ok {
				all = append(all, string(s))
			}
		}
		all = append(all, newCodes...)
		all = common.Uniq(all)
		slice := make([]mmdbtype.DataType, len(all))
		for i, c := range all {
			slice[i] = mmdbtype.String(c)
		}
		return mmdbtype.Slice(slice)

	default:
		panic("bad record type")
	}
}

func (m *MetaDBOut) writeFile(writer *mmdbwriter.Tree) error {
	if err := os.MkdirAll(m.OutputDir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(m.OutputDir, m.OutputName), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := writer.WriteTo(f); err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", m.Type, m.OutputName, m.OutputDir)

	return nil
}
