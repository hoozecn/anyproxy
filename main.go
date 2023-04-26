package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
)

var urlPattern = regexp.MustCompile(`^/(?P<schema>https?):(//)?(?P<host>[^/]+)(?P<path>.*)$`)

func main() {
	e := echo.New()
	e.Any("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matchedGroups := urlPattern.FindStringSubmatch(r.URL.Path)
		if len(matchedGroups) <= 0 {
			w.WriteHeader(400)
			w.Write([]byte("Bad request " + r.URL.Path))
			return
		}

		log.Printf("proxying %s", r.URL.String())
		paramsMap := make(map[string]string)
		for i, name := range urlPattern.SubexpNames() {
			if i > 0 && i < len(matchedGroups) && len(name) > 0 {
				paramsMap[name] = matchedGroups[i]
			}
		}

		nr := r.Clone(context.Background())
		nr.RequestURI = ""
		nr.Header.Add("X-Forwared-Host", r.URL.Host)
		nr.URL.Path = paramsMap["path"]
		nr.URL.Host = paramsMap["host"]
		nr.URL.Scheme = paramsMap["schema"]
		nr.Host = paramsMap["host"]

		resp, err := http.DefaultClient.Do(nr)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("bad upstream " + err.Error()))
			return
		}

		w.WriteHeader(resp.StatusCode)
		if _, err = io.Copy(w, resp.Body); err != nil {
			log.Printf("failed to copy response body: %s, %v", r.RequestURI, err)
		}
		resp.Body.Close()
	})))

	addr := ":31000"
	log.Printf("server start to listen on %s", addr)
	http.ListenAndServe(addr, e)
}
