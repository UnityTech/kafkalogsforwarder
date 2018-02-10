package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

var starttime = time.Now()

func listenforchecks() {
	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		if globalFlags.Lifetime > 0 {
			if time.Since(starttime) > globalFlags.Lifetime {
				http.Error(w, "Service lifetime exceeded", http.StatusServiceUnavailable)
				return
			}
		}
		fmt.Fprint(w, "OK")
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", globalFlags.Port), nil))
}
