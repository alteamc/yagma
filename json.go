package main

import (
	"strings"

	"github.com/google/uuid"
)

type UUID uuid.UUID

func (u *UUID) UnmarshalJSON(p []byte) error {
	v, err := uuid.Parse(strings.Trim(string(p), `"`))
	if err != nil {
		return err
	}

	*u = UUID(v)
	return nil
}
