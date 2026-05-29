//go:generate go run i18n/gen/main.go

package main

import (
	"app/cmd"
	"log"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
