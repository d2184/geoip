package mihomo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"geoip/lib"

	"github.com/oschwald/maxminddb-golang/v2"
)

const (
	TypeMetaDBIn = "metadb"
	DescMetaDBIn = "Convert MetaDB (Meta-geoip0) metadb database to other formats"
)

var (
	defaultMetaDBInputFile = filepath.Join(".", "metadb", "geoip.metadb")
)

func init() {
	lib.RegisterInputConfigCreator(TypeMetaDBIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newMetaDBIn(action, data)
	})
	lib.RegisterInputConverter(TypeMetaDBIn, &MetaDBIn{
		Description: DescMetaDBIn,
	})
}

func newMetaDBIn(action lib.Action, data json.RawMessage) (*MetaDBIn, error) {
	var tmp struct {
		URI        string     `json:"uri"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.URI == "" {
		tmp.URI = defaultMetaDBInputFile
	}

	// Filter want list
	wantList := make(map[string]bool)
	for _, want := range tmp.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList[want] = true
		}
	}

	return &MetaDBIn{
		Type:        TypeMetaDBIn,
		Action:      action,
		Description: DescMetaDBIn,
		URI:         tmp.URI,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,
	}, nil
}

type MetaDBIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func (m *MetaDBIn) GetType() string {
	return m.Type
}

func (m *MetaDBIn) GetAction() lib.Action {
	return m.Action
}

func (m *MetaDBIn) GetDescription() string {
	return m.Description
}

func (m *MetaDBIn) Input(container lib.Container) (lib.Container, error) {
	var content []byte
	var err error
	switch {
	case strings.HasPrefix(strings.ToLower(m.URI), "http://"), strings.HasPrefix(strings.ToLower(m.URI), "https://"):
		content, err = lib.GetRemoteURLContent(m.URI)
	default:
		content, err = os.ReadFile(m.URI)
	}
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*lib.Entry, 300)
	err = m.generateEntries(content, entries)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", m.Type, m.Action)
	}

	ignoreIPType := lib.GetIgnoreIPType(m.OnlyIPType)

	for _, entry := range entries {
		switch m.Action {
		case lib.ActionAdd:
			if err := container.Add(entry, ignoreIPType); err != nil {
				return nil, err
			}
		case lib.ActionRemove:
			if err := container.Remove(entry, lib.CaseRemovePrefix, ignoreIPType); err != nil {
				return nil, err
			}
		default:
			return nil, lib.ErrUnknownAction
		}
	}

	return container, nil
}

func (m *MetaDBIn) generateEntries(content []byte, entries map[string]*lib.Entry) error {
	db, err := maxminddb.OpenBytes(content)
	if err != nil {
		return err
	}
	defer db.Close()

	for network := range db.Networks() {
		// MetaDB record is either a single string or an array of strings
		var names []string

		// Try decoding as array first
		var arr []string
		if err := network.Decode(&arr); err == nil {
			for _, item := range arr {
				if s := strings.ToUpper(strings.TrimSpace(item)); s != "" {
					names = append(names, s)
				}
			}
		} else {
			arrErr := err
			// Fallback: try decoding as a single string
			var s string
			if err := network.Decode(&s); err != nil {
				return fmt.Errorf("decode metadb record for network %s: as array: %v; as string: %w", network.Prefix(), arrErr, err)
			}
			if s = strings.ToUpper(strings.TrimSpace(s)); s != "" {
				names = append(names, s)
			}
		}

		if len(names) == 0 || !network.Found() {
			continue
		}

		for _, name := range names {
			if len(m.Want) > 0 && !m.Want[name] {
				continue
			}

			entry, found := entries[name]
			if !found {
				entry = lib.NewEntry(name)
			}

			if err := entry.AddPrefix(network.Prefix()); err != nil {
				return err
			}

			entries[name] = entry
		}
	}

	return nil
}
