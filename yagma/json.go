package yagma

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UUID is a wrapper around uuid.UUID implementation that provides JSON unmarshalling logic.
type UUID uuid.UUID

func (u *UUID) UnmarshalJSON(p []byte) error {
	v, err := uuid.Parse(strings.Trim(string(p), `"`))
	if err != nil {
		return err
	}

	*u = UUID(v)
	return nil
}

type profileJSONMapping struct {
	ID         UUID               `json:"id"`
	Name       string             `json:"name"`
	Legacy     bool               `json:"legacy"`
	Demo       bool               `json:"demo"`
	Properties []*ProfileProperty `json:"properties"`
}

func (m *profileJSONMapping) Wrap() *Profile {
	return &Profile{
		ID:         uuid.UUID(m.ID),
		Name:       m.Name,
		Legacy:     m.Legacy,
		Demo:       m.Demo,
		Properties: m.Properties,
	}
}

// Profile represents Mojang user profile.
type Profile struct {
	ID         uuid.UUID
	Name       string
	Legacy     bool
	Demo       bool
	Properties []*ProfileProperty
}

// ProfileProperty represents Mojang user profile property.
type ProfileProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (p *ProfileProperty) ProfileTextures() (*ProfileTextures, error) {
	if p.Name != "textures" {
		return nil, fmt.Errorf(`expected property name to be "value", got %#v`, p.Name)
	}

	decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer([]byte(p.Value))))
	if err != nil {
		return nil, err
	}

	m := &profileTexturesJSONMapping{}
	if err = json.Unmarshal(decoded, m); err != nil {
		return nil, err
	}

	return m.Wrap(), nil
}

// SkinModel is an enum type for wrapping Steve/Alex skin model type.
type SkinModel byte

const (
	SkinModelSteve SkinModel = iota
	SkinModelAlex
)

// ProfileTextures represents Minecraft skins associated with Mojang profile along with some metadata.
type ProfileTextures struct {
	Timestamp   time.Time
	ProfileID   uuid.UUID
	ProfileName string
	Skin        string
	SkinModel   SkinModel
	Cape        string
}

// ModelFromUUID determines whether default skin model type for ProfileTextures is Steve or Alex
// based on Profile UUID.
func (t *ProfileTextures) ModelFromUUID() SkinModel {
	u := t.ProfileID
	if (u[3]&0xf)^(u[7]&0xf)^(u[11]&0xf)^(u[15]&0xf) != 0 {
		return SkinModelAlex
	} else {
		return SkinModelSteve
	}
}

type profileTexturesJSONMapping struct {
	ProfileID   uuid.UUID `json:"profileId"`
	ProfileName string    `json:"profileName"`
	Timestamp   int64     `json:"timestamp"`
	Textures    struct {
		Skin struct {
			URL      string `json:"url"`
			Metadata struct {
				Model string `json:"model"`
			} `json:"metadata"`
		} `json:"SKIN"`
		Cape struct {
			URL string `json:"url"`
		} `json:"CAPE"`
	} `json:"textures"`
}

func (m *profileTexturesJSONMapping) Wrap() *ProfileTextures {
	var sm SkinModel
	if m.Textures.Skin.Metadata.Model == "slim" {
		sm = SkinModelAlex
	}

	return &ProfileTextures{
		Timestamp:   time.UnixMilli(m.Timestamp),
		ProfileID:   m.ProfileID,
		ProfileName: m.ProfileName,
		Skin:        m.Textures.Skin.URL,
		SkinModel:   sm,
		Cape:        m.Textures.Cape.URL,
	}
}

// Time is a wrapper around time.Time implementation that provides JSON unmarshalling logic.
type Time time.Time

func (t *Time) UnmarshalJSON(p []byte) error {
	v, err := strconv.ParseInt(string(p), 10, 64)
	if err != nil {
		return err
	}

	*t = Time(time.UnixMilli(v))
	return nil
}

type nameHistoryRecordJSONMapping struct {
	Name        string `json:"name"`
	ChangedToAt Time   `json:"changedToAt"`
}

func (m *nameHistoryRecordJSONMapping) Wrap() *NameHistoryRecord {
	return &NameHistoryRecord{
		Name:      m.Name,
		ChangedAt: time.Time(m.ChangedToAt),
	}
}

type nameHistoryRecordJSONMappingArray []*nameHistoryRecordJSONMapping

func (m nameHistoryRecordJSONMappingArray) Wrap() []*NameHistoryRecord {
	v := make([]*NameHistoryRecord, len(m))
	for i, p := range m {
		v[i] = p.Wrap()
	}
	return v
}

// NameHistoryRecord represents a name change record in Mojang profile name history.
type NameHistoryRecord struct {
	Name      string
	ChangedAt time.Time
}

// MetricKey is an enum type for wrapping Mojang sell statistics metric key type.
type MetricKey string

const (
	MetricMinecraftItemsSold            MetricKey = "item_sold_minecraft"
	MetricMinecraftPrepaidCardsRedeemed           = "prepaid_card_redeemed_minecraft"
	MetricCobaltItemsSold                         = "item_sold_cobalt"
	MetricCobaltPrepaidCardsRedeemed              = "prepaid_card_redeemed_cobalt"
	MetricScrollsItemsSold                        = "item_sold_scrolls"
	MetricDungeonsItemsSold                       = "item_sold_dungeons"
)

// Statistics represents brief statistics for MetricKey.
type Statistics struct {
	Total    int     `json:"total"`
	Last24h  int     `json:"last24h"`
	Velocity float32 `json:"saleVelocityPerSeconds"`
}
