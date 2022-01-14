package yagma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// General errors

var (
	HTTPError   = errors.New("failed to send HTTP request")
	JSONError   = errors.New("failed to parse JSON response")
	StatusError = errors.New("unknown status code")
)

type RequestError struct {
	Type    string `json:"error"`
	Message string `json:"errorMessage"`
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func parseRequestError(res *http.Response) (*RequestError, error) {
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}

	parsed := &RequestError{}
	if err := json.Unmarshal(data, parsed); err != nil {
		return nil, fmt.Errorf("%w: %s", JSONError, err)
	}

	return nil, parsed
}

// Profile by username

type Profile struct {
	Name   string
	ID     uuid.UUID
	Legacy bool
	Demo   bool
}

type profileWithCustomUUID struct {
	Name   string
	ID     UUID
	Legacy bool
	Demo   bool
}

var ProfileNotFound = errors.New("there is no user with such username")

func (c *Client) ProfileByUsername(ctx context.Context, username string, timestamp time.Time) (*Profile, error) {
	reqURL := c.urlBase.mojangAPI + "/users/profiles/minecraft/" + url.QueryEscape(username)
	if !timestamp.IsZero() {
		reqURL += "?at=" + strconv.FormatInt(timestamp.UnixMilli(), 10)
	}

	res, err := c.sendHTTPReq(ctx, http.MethodGet, reqURL, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusNoContent:
		return nil, fmt.Errorf("%w: %s", ProfileNotFound, username)
	case http.StatusBadRequest:
		badReqErr, err := parseRequestError(res)
		if err != nil {
			return nil, err
		}
		return nil, badReqErr
	case http.StatusOK:
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", HTTPError, err)
		}

		parsed := &profileWithCustomUUID{}
		if err := json.Unmarshal(data, parsed); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return &Profile{
			Name:   parsed.Name,
			ID:     uuid.UUID(parsed.ID),
			Legacy: parsed.Legacy,
			Demo:   parsed.Demo,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Profile by username (bulk)

func (c *Client) ProfileByUsernameBulk(ctx context.Context, usernames []string) ([]*Profile, error) {
	reqURL := c.urlBase.mojangAPI + "/profiles/minecraft"
	bodyBytes, err := json.Marshal(usernames)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}

	res, err := c.sendHTTPReq(ctx, http.MethodPost, reqURL, nil, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusBadRequest:
		badReqErr, err := parseRequestError(res)
		if err != nil {
			return nil, err
		}
		return nil, badReqErr
	case http.StatusOK:
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", HTTPError, err)
		}

		profiles := make([]*Profile, 0, len(usernames))
		if err := json.Unmarshal(data, &profiles); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return profiles, nil
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}
