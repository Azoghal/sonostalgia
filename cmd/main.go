package main

import (
	"log"

	"github.com/azoghal/sonostalgia/src/templater"
)

func main() {
	if err := templater.Run("src", "output"); err != nil {
		log.Fatal(err)
	}
}
