package yagma

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

// Utility methods

func parseRequestError(res *http.Response) error {
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("%w: %s", HTTPError, err)
	}

	reqErr := &RequestError{}
	if err = json.Unmarshal(data, reqErr); err != nil {
		return fmt.Errorf("%w: %s", JSONError, err)
	}

	return reqErr
}

func readBody(res *http.Response) ([]byte, error) {
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	return data, nil
}

// Profile by username

var ProfileNotFound = errors.New("user not found")

func (c *Client) ProfileByUsername(ctx context.Context, username string, timestamp time.Time) (*Profile, error) {
	reqURL := c.urlBase.mojangAPI + "/users/profiles/minecraft/" + username
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
		data, err := readBody(res)
		if err != nil {
			return nil, err
		}

		m := &profileJSONMapping{}
		if err = json.Unmarshal(data, m); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return m.Wrap(), nil
	case http.StatusNoContent:
		return nil, fmt.Errorf("%w: %s", ProfileNotFound, username)
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
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
		data, err := readBody(res)
		if err != nil {
			return nil, err
		}

		profiles := make([]*Profile, 0, len(usernames))
		if err = json.Unmarshal(data, &profiles); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return profiles, nil
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
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
		data, err := readBody(res)
		if err != nil {
			return nil, err
		}

		records := make(nameHistoryRecordJSONMappingArray, 0, 8)
		if err = json.Unmarshal(data, &records); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return records.Wrap(), nil
	case http.StatusNoContent:
		return nil, fmt.Errorf("%w: %s", ProfileNotFound, uuid)
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
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
		data, err := readBody(res)
		if err != nil {
			return nil, err
		}

		m := &profileJSONMapping{}
		if err = json.Unmarshal(data, m); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return m.Wrap(), nil
	case http.StatusNoContent:
		return nil, ProfileNotFound
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Blocked servers

func (c *Client) BlockedServerHashes(ctx context.Context) ([]string, error) {
	reqURL := c.urlBase.sessionServer + "/blockedservers"

	res, err := c.sendHTTPReq(ctx, http.MethodGet, reqURL, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		ch := make(chan string)

		go func() {
			s := bufio.NewScanner(res.Body)
			for s.Scan() {
				ch <- s.Text()
			}

			close(ch)
		}()

		hash := make([]string, 0, 512)
		for {
			select {
			case h, ok := <-ch:
				if ok {
					hash = append(hash, h)
				} else {
					return hash, nil
				}
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}

// Statistics

func (c *Client) Statistics(ctx context.Context, keys []MetricKey) (*Statistics, error) {
	reqURL := c.urlBase.mojangAPI + "/orders/statistics"

	header := make(http.Header)
	header.Add("Content-Type", "application/json")

	bodyBytes, err := json.Marshal(map[string][]MetricKey{"metricKeys": keys})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}

	res, err := c.sendHTTPReq(ctx, http.MethodPost, reqURL, header, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", HTTPError, err)
	}
	defer func() { _ = res.Body.Close() }()

	switch res.StatusCode {
	case http.StatusOK:
		data, err := readBody(res)
		if err != nil {
			return nil, err
		}

		s := &Statistics{}
		if err = json.Unmarshal(data, s); err != nil {
			return nil, fmt.Errorf("%w: %s", JSONError, err)
		}

		return s, nil
	case http.StatusBadRequest:
		return nil, parseRequestError(res)
	default:
		return nil, fmt.Errorf("%w: %s", StatusError, res.Status)
	}
}
