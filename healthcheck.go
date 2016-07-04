package main

import (
	"fmt"
	"log"
	"net/http"
)

func listenforchecks() {
	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", globalFlags.Port), nil))
}
