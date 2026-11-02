// Package catalog gathers bounded foreground evidence. Public model metadata is
// not evidence of a gateway's capacity, protocol support or account entitlement.
package catalog

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"
)

// Error contains only a fixed classification, never a URL, response body,
// transport error, credential or provider-supplied message.
type Error struct{ Kind string }

func (e *Error) Error() string {
	switch e.Kind {
	case "unauthorized":
		return "catalog authentication failed (401); check the selected credential"
	case "forbidden":
		return "catalog access refused (403); check route permissions"
	case "rate_limit":
		return "catalog rate limited (429); retry later explicitly"
	case "tls":
		return "catalog TLS verification failed; check the endpoint certificate"
	case "canceled":
		return "catalog request canceled or timed out"
	case "redirect":
		return "catalog redirect refused; use the direct trusted endpoint"
	case "oversize":
		return "catalog response exceeded its size limit"
	case "json":
		return "catalog response is not valid unambiguous JSON"
	case "endpoint":
		return "catalog requires a valid HTTPS endpoint (HTTP only on loopback) without credentials, query or fragment"
	case "credential":
		return "catalog credential is invalid"
	case "inventory":
		return "catalog inventory requires unique exact model IDs without whitespace or control characters (maximum 256 bytes)"
	case "metadata":
		return "public model metadata has an invalid structure"
	case "preset":
		return "confirmed catalog requires an exact eligible Codex preset, semantic version and confirmed tiers"
	case "native":
		return "confirmed catalog requires one complete version-matched native model entry with valid tier metadata"
	case "http":
		return "catalog server returned an unexpected HTTP status"
	default:
		return "catalog request failed; check route availability"
	}
}
func boundedClient(client *http.Client) *http.Client {
	out := http.Client{}
	if client != nil {
		out = *client
	}
	out.Timeout = 10 * time.Second
	out.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	out.Jar = nil
	return &out
}
func get(ctx context.Context, url, key string, limit int64, client *http.Client) ([]byte, error) {
	// Deadline also covers injected transports which correctly honor contexts.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &Error{Kind: "endpoint"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "oneline-ccc-catalog")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	response, err := boundedClient(client).Do(req)
	if err != nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		kind := "network"
		var unknown x509.UnknownAuthorityError
		var hostname x509.HostnameError
		var invalid x509.CertificateInvalidError
		var record tls.RecordHeaderError
		var verify *tls.CertificateVerificationError
		if errors.As(err, &unknown) || errors.As(err, &hostname) || errors.As(err, &invalid) || errors.As(err, &record) || errors.As(err, &verify) {
			kind = "tls"
		}
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			kind = "canceled"
		}
		return nil, &Error{Kind: kind}
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, &Error{Kind: "unauthorized"}
	case http.StatusForbidden:
		return nil, &Error{Kind: "forbidden"}
	case http.StatusTooManyRequests:
		return nil, &Error{Kind: "rate_limit"}
	default:
		if response.StatusCode >= 300 && response.StatusCode < 400 {
			return nil, &Error{Kind: "redirect"}
		}
		return nil, &Error{Kind: "http"}
	}
	if response.ContentLength > limit {
		return nil, &Error{Kind: "oversize"}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if int64(len(data)) > limit {
		return nil, &Error{Kind: "oversize"}
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, &Error{Kind: "canceled"}
		}
		return nil, &Error{Kind: "network"}
	}
	if !utf8.Valid(data) || !validJSON(data) {
		return nil, &Error{Kind: "json"}
	}
	return data, nil
}

// encoding/json accepts duplicate object fields; reject those and excessive
// nesting before decoding into maps, where duplicates would silently disappear.
func validJSON(data []byte) bool {
	if !validUnicodeEscapes(data) {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if !jsonValue(decoder, 0) {
		return false
	}
	_, err := decoder.Token()
	return err == io.EOF
}
func jsonValue(d *json.Decoder, depth int) bool {
	if depth > 100 {
		return false
	}
	token, err := d.Token()
	if err != nil {
		return false
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return true
	}
	switch delimiter {
	case '{':
		keys := map[string]bool{}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return false
			}
			key, ok := token.(string)
			if !ok || keys[key] {
				return false
			}
			keys[key] = true
			if !jsonValue(d, depth+1) {
				return false
			}
		}
		token, err = d.Token()
		return err == nil && token == json.Delim('}')
	case '[':
		for d.More() {
			if !jsonValue(d, depth+1) {
				return false
			}
		}
		token, err = d.Token()
		return err == nil && token == json.Delim(']')
	default:
		return false
	}
}

// encoding/json replaces unpaired UTF-16 escapes with U+FFFD. Refuse them
// rather than silently changing a remote model ID or native metadata string.
func validUnicodeEscapes(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] != '\\' {
			continue
		}
		if i+1 >= len(data) {
			return false
		}
		if data[i+1] != 'u' {
			i++
			continue
		}
		if i+6 > len(data) {
			return false
		}
		scalar, err := strconv.ParseUint(string(data[i+2:i+6]), 16, 16)
		if err != nil {
			return false
		}
		i += 5
		if scalar >= 0xDC00 && scalar <= 0xDFFF {
			return false
		}
		if scalar >= 0xD800 && scalar <= 0xDBFF {
			if i+7 > len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
			if err != nil || low < 0xDC00 || low > 0xDFFF {
				return false
			}
			i += 6
		}
	}
	return true
}
