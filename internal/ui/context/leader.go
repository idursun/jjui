package context

import (
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/key"
)

type LeaderMap = map[string]*Leader

type Leader struct {
	Bind *key.Binding
	Send []string
	Nest LeaderMap
}

func LoadLeader(content string) (LeaderMap, error) {
	type leaderTomlEntry struct {
		Help string
		Send []string
	}
	type leaderToml struct {
		Leader map[string]leaderTomlEntry
	}
	dec := leaderToml{}
	_, err := toml.Decode(content, &dec)
	if err != nil {
		return nil, err
	}
	res := LeaderMap{}
	for name, v := range dec.Leader {
		ks := strings.Split(name, "")
		at := res
		for i, k := range ks {
			m := checkExists(at, k)
			if i == len(ks)-1 {
				if len(v.Send) > 0 {
					m.Send = v.Send
				}
				if len(v.Help) > 0 {
					m.Bind.SetHelp(k, v.Help)
				}
			}
			at = m.Nest
		}
	}
	return res, nil
}

func checkExists(at LeaderMap, k string) *Leader {
	if m, ok := at[k]; ok {
		return m
	} else {
		b := key.NewBinding(
			key.WithKeys(k),
			key.WithHelp(k, ""),
		)
		m = &Leader{
			Bind: &b,
			Send: nil,
			Nest: LeaderMap{},
		}
		at[k] = m
		return m
	}
}
