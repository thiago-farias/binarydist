package main

import (
	"os"
	"github.com/thiago-farias/binarydist"
)

func main() {
	old, err := os.Open("old.tar")
	if err != nil {
		panic(err)
	}
	new, err := os.Create("recover.tar")
	if err != nil {
		panic(err)
	}
	patch, err := os.Open("patch")
	if err != nil {
		panic(err)
	}
	err = binarydist.Patch(old, new, patch)
	if err != nil {
		panic(err)
	}
	
	old.Close()
	new.Close()
	patch.Close()
}