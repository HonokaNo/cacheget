package cacheget

import (
	"bytes"
	"compress/flate"
	"encoding/gob"
	"io"
	"net/http"
)

type Cache struct {
	Etag string
	Body []byte
}

var cachemap = make(map[string]*Cache)

// GET request, use etag
// return (data, status code, error)
func CacheGet(url string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}

	cachedata := cachemap[url]
	if cachedata != nil {
		req.Header.Set("if-none-match", cachedata.Etag)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, res.StatusCode, err
	}

	if res.StatusCode == 304 {
		return cachedata.Body, 304, nil
	} else {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return body, res.StatusCode, err
		}
		res.Body.Close()

		if etag := res.Header.Get("etag"); etag != "" {
			cachemap[url] = &Cache{etag, body}
		}
		return body, res.StatusCode, nil
	}
}

// serialize etag cache
func SerializeCache() (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	err := gob.NewEncoder(buf).Encode(&cachemap)
	if err != nil {
		return nil, err
	}

	zbuf := bytes.NewBuffer(nil)
	zw, err := flate.NewWriter(zbuf, flate.BestSpeed)
	if err != nil {
		return nil, err
	}

	if _, err = zw.Write(buf.Bytes()); err != nil {
		return nil, err
	}

	zw.Flush()
	return zbuf, nil
}

// deserialize cache
// fbuf is cache data buffer
func DeserializeCache(fbuf []byte) error {
	zbuf := bytes.NewBuffer(fbuf)
	zr := flate.NewReader(zbuf)
	defer zr.Close()

	err := gob.NewDecoder(zr).Decode(&cachemap)
	if err != nil {
		return err
	}

	return nil
}
