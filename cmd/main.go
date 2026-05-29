package cmd

import (
	"log"
)

func checkError(err error) {
	if err != nil {
		log.Fatalf("error occured:%v\n", err)
	}
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
