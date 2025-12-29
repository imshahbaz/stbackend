package middleware

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/go-resty/resty/v2"
)

func DecompressMiddleware(c *resty.Client, resp *resty.Response) error {
	encoding := resp.Header().Get("Content-Encoding")
	if encoding == "" {
		return nil
	}

	var reader io.ReadCloser
	var err error

	switch encoding {
	case "br":
		reader = io.NopCloser(brotli.NewReader(bytes.NewReader(resp.Body())))
	case "gzip":
		reader, err = gzip.NewReader(bytes.NewReader(resp.Body()))
		if err != nil {
			return err
		}
		defer reader.Close()
	default:
		return nil
	}

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	resp.SetBody(decompressed)
	return nil
}
