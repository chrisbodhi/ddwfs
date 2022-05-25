package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/timeutil"
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

func Try() {
	fmt.Println("ok, gonna")

	// Create an appropriate file system.
	server, err := FS(timeutil.RealClock())
	if err != nil {
		log.Fatalf("makeFS: %v", err)
	}

	cfg := &fuse.MountConfig{
		ReadOnly: false,
		// DebugLogger: log.New(os.Stderr, "fuse: ", 0),
	}

	owner := os.Getenv("DW_USERNAME")
	if len(owner) == 0 {
		log.Fatalln("Missing DW_USERNAME")
	}

	mntPoint := filepath.Join(".", owner)

	mfs, err := fuse.Mount(mntPoint, server, cfg)
	if err != nil {
		log.Fatalf("Mount: %v", err)
	}

	// Wait for it to be unmounted.
	if err = mfs.Join(context.Background()); err != nil {
		log.Fatalf("Join: %v", err)
	}
}
