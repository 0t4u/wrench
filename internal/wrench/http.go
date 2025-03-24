package wrench

import (
	"fmt"
	"log"
	"net/http"
)

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func (b *Bot) httpStart() error {
	if b.Config.Address == "" {
		b.logf("http: disabled (Config.Address not configured)")
		return nil
	}

	handler := func(prefix string, handle handlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := handle(w, r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "error: %s", err.Error())
				b.logf("http: %s: %v", prefix, err)
			}
		})
	}

	var mux http.Handler

	b.logf("http: zig mirror mode enabled")
	mux = b.httpMuxPkgProxy(handler)

	b.logf("http: listening on %v - %v", b.Config.Address, b.Config.ExternalURL)
	go func() {
		err := http.ListenAndServe(b.Config.Address, mux)
		if err != nil {
			log.Fatal("ListenAndServe(addr):", err)
		}
	}()
	return nil
}
