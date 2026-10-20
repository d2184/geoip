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
		return NewMetaDBInFromBytes(action, data)
	})
	lib.RegisterInputConverter(TypeMetaDBIn, &MetaDBIn{
		Description: DescMetaDBIn,
	})
}

type MetaDBIn struct {
	Type        string
	Action      lib.Action
	Description string
	URI         string
	Want        map[string]bool
	OnlyIPType  lib.IPType
}

func NewMetaDBIn(action lib.Action, opts ...lib.InputOption) (lib.InputConverter, error) {
	m := &MetaDBIn{
		Type:        TypeMetaDBIn,
		Action:      action,
		Description: DescMetaDBIn,
		URI:         defaultMetaDBInputFile,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	if m.URI == "" {
		return nil, fmt.Errorf("❌ [type %s | action %s] uri must be specified", m.Type, m.Action)
	}

	return m, nil
}

func WithMetaDBURI(uri string) lib.InputOption {
	uri = strings.TrimSpace(uri)
	return func(m lib.InputConverter) {
		inputURI := uri
		if inputURI == "" {
			inputURI = defaultMetaDBInputFile
		}

		m.(*MetaDBIn).URI = inputURI
	}
}

func WithMetaDBInWantedList(lists []string) lib.InputOption {
	return func(m lib.InputConverter) {
		wantList := make(map[string]bool)
		for _, want := range lists {
			if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
				wantList[want] = true
			}
		}

		m.(*MetaDBIn).Want = wantList
	}
}

func WithMetaDBInOnlyIPType(onlyIPType lib.IPType) lib.InputOption {
	return func(m lib.InputConverter) {
		m.(*MetaDBIn).OnlyIPType = onlyIPType
	}
}

func NewMetaDBInFromBytes(action lib.Action, data []byte) (lib.InputConverter, error) {
	var tmp struct {
		URI        string     `json:"uri"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", TypeMetaDBIn, action, err)
		}
	}

	return NewMetaDBIn(
		action,
		WithMetaDBURI(tmp.URI),
		WithMetaDBInWantedList(tmp.Want),
		WithMetaDBInOnlyIPType(tmp.OnlyIPType),
	)
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
		return nil, fmt.Errorf("❌ [type %s | action %s] %w", m.Type, m.Action, err)
	}

	entries := make(map[string]*lib.Entry)
	err = m.generateEntries(content, entries)
	if err != nil {
		return nil, fmt.Errorf("❌ [type %s | action %s] %w", m.Type, m.Action, err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("❌ [type %s | action %s] no entry is generated", m.Type, m.Action)
	}

	ignoreIPType := lib.GetIgnoreIPType(m.OnlyIPType)

	for _, entry := range entries {
		switch m.Action {
		case lib.ActionAdd:
			if err := container.Add(entry, ignoreIPType); err != nil {
				return nil, fmt.Errorf("❌ [type %s | action %s] %w", m.Type, m.Action, err)
			}
		case lib.ActionRemove:
			if err := container.Remove(entry, lib.CaseRemovePrefix, ignoreIPType); err != nil {
				return nil, fmt.Errorf("❌ [type %s | action %s] %w", m.Type, m.Action, err)
			}
		default:
			return nil, fmt.Errorf("❌ [type %s | action %s] %w", m.Type, m.Action, lib.ErrUnknownAction)
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
