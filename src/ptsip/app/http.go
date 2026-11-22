// Package app implements the PTSIP decision control plane in Go.
package app

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

const requestLimit = 1 << 20

type Object = map[string]any

type ControlPlane interface {
	Gate(context.Context, Object) (Object, error)
	Decision(context.Context, Object) (Object, error)
	Resolve(context.Context, Object) (Object, error)
	Application(context.Context, Object) (Object, error)
}

type Webhook interface {
	HandleWebhook(context.Context, string, Object) (Object, error)
}

type InputError struct{ Message string }

func (e InputError) Error() string { return e.Message }

func badInput(message string) error { return InputError{Message: message} }

// NewHandler binds the existing v1 HTTP routes to a decision backend.
func NewHandler(service ControlPlane, webhook Webhook, token, secret, version string) (http.Handler, error) {
	if service == nil || webhook == nil || token == "" || secret == "" || version == "" {
		return nil, fmt.Errorf("control plane, webhook handler, token, secret and version are required")
	}
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, status int, payload Object) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}
	dispatch := func(w http.ResponseWriter, payload Object, err error) {
		if err != nil {
			var input InputError
			if errors.As(err, &input) {
				write(w, http.StatusBadRequest, Object{"error": input.Message})
			} else {
				write(w, http.StatusInternalServerError, Object{"error": "control plane operation failed"})
			}
			return
		}
		write(w, http.StatusOK, payload)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, Object{"status": "ok", "service": "ptsip-control-plane", "version": version})
	})
	mux.HandleFunc("POST /github/webhook", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, requestLimit))
		if err != nil {
			write(w, http.StatusBadRequest, Object{"error": "cannot read webhook body"})
			return
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(data)
		signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(signature), []byte(r.Header.Get("X-Hub-Signature-256"))) {
			write(w, http.StatusUnauthorized, Object{"error": "invalid webhook signature"})
			return
		}
		payload, err := decodeObject(data)
		if err != nil {
			write(w, http.StatusBadRequest, Object{"error": "webhook payload must be a JSON object with unique keys"})
			return
		}
		result, err := webhook.HandleWebhook(r.Context(), r.Header.Get("X-GitHub-Event"), payload)
		dispatch(w, result, err)
	})
	for route, run := range map[string]func(context.Context, Object) (Object, error){
		"/v1/gate": service.Gate, "/v1/decision": service.Decision,
		"/v1/resolve": service.Resolve, "/v1/application": service.Application,
	} {
		mux.HandleFunc("POST "+route, func(w http.ResponseWriter, r *http.Request) {
			if !hmac.Equal([]byte("Bearer "+token), []byte(r.Header.Get("Authorization"))) {
				write(w, http.StatusUnauthorized, Object{"error": "unauthorized"})
				return
			}
			data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, requestLimit))
			if err != nil {
				write(w, http.StatusBadRequest, Object{"error": "cannot read request body"})
				return
			}
			payload, err := decodeObject(data)
			if err != nil {
				write(w, http.StatusBadRequest, Object{"error": "request must be a JSON object with unique keys"})
				return
			}
			result, err := run(r.Context(), payload)
			dispatch(w, result, err)
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusNotFound, Object{"error": "not found"})
	})
	return mux, nil
}

func decodeObject(data []byte) (Object, error) {
	value, err := jsondoc.Decode(data)
	if err != nil {
		return nil, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected a JSON object")
	}
	return object, nil
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" ||
		u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || token == "" {
		return nil, fmt.Errorf("a valid control plane URL and token are required")
	}
	return &Client{
		baseURL: strings.TrimRight(u.String(), "/"), token: token,
		http: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}},
	}, nil
}

func (c *Client) Gate(ctx context.Context, payload Object) (Object, error) {
	return c.post(ctx, "/v1/gate", payload)
}

func (c *Client) Decision(ctx context.Context, payload Object) (Object, error) {
	return c.post(ctx, "/v1/decision", payload)
}

func (c *Client) Resolve(ctx context.Context, payload Object) (Object, error) {
	return c.post(ctx, "/v1/resolve", payload)
}

func (c *Client) Application(ctx context.Context, payload Object) (Object, error) {
	return c.post(ctx, "/v1/application", payload)
}

func (c *Client) post(ctx context.Context, route string, payload Object) (Object, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+route, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, requestLimit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > requestLimit {
		return nil, fmt.Errorf("control plane response exceeds size limit")
	}
	result, err := decodeObject(data)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := result["error"].(string)
		return nil, fmt.Errorf("control plane HTTP %d: %s", response.StatusCode, message)
	}
	return result, err
}
