package main

import (
	"net/http"
	"log"
	"fmt"
	"context"
	"time"
	"os"
	"html/template"
)

func main() {
	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	server := &http.Server{
		Addr: ":9000",
		Handler: tracing(nextRequestID)(logging(logger)(setupRoutes())),
		ReadTimeout: 10000,
		WriteTimeout: 10000,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(server.ListenAndServe())
}

type key int

const (
	requestIDKey key = 0
)

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, World!")
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func setupRoutes() *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("/", indexHandler)
	router.HandleFunc("/hello", helloHandler)
	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	const inlineTemplate = `
		<h1>Ecommerce</h1>
	`

	data := struct {
		Name string
		Links map[string]string
		IP string
	}{
		Name: "John",
		IP: r.RemoteAddr,
		Links: map[string]string{
			"Home": "/",
			"Hello": "/hello",
		},
	}

	tmpl, err := template.New("index").Funcs(template.FuncMap{
		"CurrentTime": func() string { return time.Now().Format(time.RFC3339) },
		"SayHi": func(name string) string { return fmt.Sprintf("Hi %s!", name) },
	}).Parse(inlineTemplate)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		fmt.Println(err)
	}
}