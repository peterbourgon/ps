package pshttp

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strings"
)

func respondJSON(w http.ResponseWriter, code int, response any) error {
	body, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		body = []byte(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, err = w.Write(body)
	return err
}

func requestExplicitlyAccepts(r *http.Request, acceptable ...string) bool {
	accept := parseAcceptMediaTypes(r)
	for _, want := range acceptable {
		if _, ok := accept[want]; ok {
			return true
		}
	}
	return false
}

func parseAcceptMediaTypes(r *http.Request) map[string]map[string]string {
	mediaTypes := map[string]map[string]string{} // type: params
	for _, a := range strings.Split(r.Header.Get("accept"), ",") {
		mediaType, params, err := mime.ParseMediaType(a)
		if err != nil {
			continue
		}
		mediaTypes[mediaType] = params
	}
	return mediaTypes
}

func parseDefault[T any](s string, parse func(string) (T, error), def T) T {
	if v, err := parse(s); err == nil {
		return v
	}
	return def
}
