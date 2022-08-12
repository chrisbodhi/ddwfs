package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func Setup() {
	// TODO parse token for this info
	owner := os.Getenv("DW_USERNAME")

	if len(owner) == 0 {
		log.Fatalln("Missing DW_USERNAME")
	}

	// create mnt point if it doesn't exist
	mntPoint := filepath.Join(".", owner)
	err := os.MkdirAll(mntPoint, os.ModePerm)
	if err != nil {
		fmt.Println("Mount point already exists for", owner)
	}
}
