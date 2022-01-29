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
	"strings"
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

var ProfileNotFound = errors.New("user not found")

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
	case http.StatusOK:
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", HTTPError, err)
		}

		m := &profileJSONMapping{}
		if err := json.Unmarshal(data, m); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return m.Wrap(), nil
	case http.StatusNoContent:
		return nil, fmt.Errorf("%w: %s", ProfileNotFound, username)
	case http.StatusBadRequest:
		badReqErr, err := parseRequestError(res)
		if err != nil {
			return nil, err
		}
		return nil, badReqErr
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
	case http.StatusBadRequest:
		badReqErr, err := parseRequestError(res)
		if err != nil {
			return nil, err
		}
		return nil, badReqErr
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Name history by UUID

func (c *Client) NameHistoryByUUID(ctx context.Context, uuid uuid.UUID) ([]*NameHistoryRecord, error) {
	reqURL := c.urlBase.mojangAPI + "/user/profiles/" + uuid.String() + "/names"

	res, err := c.sendHTTPReq(ctx, http.MethodGet, reqURL, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", HTTPError, err)
		}

		records := make(nameHistoryRecordJSONMappingArray, 0, 8)
		if err := json.Unmarshal(data, &records); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return records.Wrap(), nil
	case http.StatusNoContent:
		return nil, fmt.Errorf("%w: %s", ProfileNotFound, uuid)
	case http.StatusBadRequest:
		badReqErr, err := parseRequestError(res)
		if err != nil {
			return nil, err
		}
		return nil, badReqErr
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Profile with skin/cape by UUID

func (c *Client) ProfileByUUID(ctx context.Context, uuid uuid.UUID) (*Profile, error) {
	reqURL := c.urlBase.sessionServer + "/session/minecraft/profile/" + strings.Replace(uuid.String(), "-", "", -1)

	res, err := c.sendHTTPReq(ctx, http.MethodGet, reqURL, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", HTTPError, err)
		}

		m := &profileJSONMapping{}
		if err := json.Unmarshal(data, m); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return m.Wrap(), nil
	case http.StatusNoContent:
		return nil, ProfileNotFound
	case http.StatusBadRequest:
		badReqErr, err := parseRequestError(res)
		if err != nil {
			return nil, err
		}
		return nil, badReqErr
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}
