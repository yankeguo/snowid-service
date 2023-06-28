package ufx

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// extractStringSlice extracts string slice into map with optional prefix
func extractStringSlice(out map[string]any, pfx string, key string, vals []string) {
	var v any
	if len(vals) == 1 {
		v = vals[0]
	} else {
		v = vals
	}
	out[key] = v
	out[key+"_array"] = vals
	if pfx != "" {
		out[pfx+key] = v
		out[pfx+key+"_array"] = vals
	}
}

// extractRequest extracts request data into map
func extractRequest(out map[string]any, fOut map[string][]*multipart.FileHeader, req *http.Request) (err error) {
	// host is not included in headers by default
	extractStringSlice(out, "header_", "host", []string{req.Host})

	// header
	for key, vals := range req.Header {
		key = strings.ToLower(strings.ReplaceAll(key, "-", "_"))
		extractStringSlice(out, "header_", key, vals)
	}

	// query
	for key, vals := range req.URL.Query() {
		extractStringSlice(out, "query_", key, vals)
	}

	// body
	var buf []byte

	contentType, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type"))

	if contentType != ContentTypeMultipart {
		if buf, err = io.ReadAll(req.Body); err != nil {
			return
		}

		if len(buf) == 0 {
			return
		}
	}

	switch contentType {
	case ContentTypeTextPlain:
		out["body"] = string(buf)
	case ContentTypeApplicationJSON:
		var data map[string]any
		if err = json.Unmarshal(buf, &data); err != nil {
			return
		}
		for key, val := range data {
			out[key] = val
		}
	case ContentTypeFormURLEncoded:
		var query url.Values
		if query, err = url.ParseQuery(string(buf)); err != nil {
			return
		}
		for key, vals := range query {
			extractStringSlice(out, "form_", key, vals)
		}
	case ContentTypeMultipart:
		if err = req.ParseMultipartForm(1024 * 1024 * 10); err != nil {
			return
		}
		for key, vals := range req.MultipartForm.Value {
			extractStringSlice(out, "form_", key, vals)
		}
		for key, file := range req.MultipartForm.File {
			fOut[key] = file
		}
	default:
		out["body"] = buf
		return
	}

	return
}
