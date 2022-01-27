package yagma

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

// JSON-unmarshallable UUID implementation

type UUID uuid.UUID

func (u *UUID) UnmarshalJSON(p []byte) error {
	v, err := uuid.Parse(strings.Trim(string(p), `"`))
	if err != nil {
		return err
	}

	*u = UUID(v)
	return nil
}

// Profile

type profileJSONMapping struct {
	ID         UUID        `json:"id"`
	Name       string      `json:"name"`
	Legacy     bool        `json:"legacy"`
	Demo       bool        `json:"demo"`
	Properties []*Property `json:"properties"`
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

type Profile struct {
	ID         uuid.UUID
	Name       string
	Legacy     bool
	Demo       bool
	Properties []*Property
}

// Profile property

type Property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (p *Property) ProfileTextures() (*ProfileTextures, error) {
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

// Skin model

type SkinModel byte

const (
	SkinModelSteve SkinModel = iota
	SkinModelAlex
)

type ProfileTextures struct {
	Timestamp   time.Time
	ProfileID   uuid.UUID
	ProfileName string
	Skin        string
	SkinModel   SkinModel
	Cape        string
}

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