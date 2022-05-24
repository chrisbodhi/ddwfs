package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/samples/hellofs"
	"github.com/jacobsa/timeutil"
)

func Try() {
	fmt.Println("ok, gonna")

	// Create an appropriate file system.
	server, err := hellofs.NewHelloFS(timeutil.RealClock())
	if err != nil {
		log.Fatalf("makeFS: %v", err)
	}

	cfg := &fuse.MountConfig{
		ReadOnly: false,
		// DebugLogger: log.New(os.Stderr, "fuse: ", 0),
	}

	mfs, err := fuse.Mount("./mnt", server, cfg)
	if err != nil {
		log.Fatalf("Mount: %v", err)
	}

	// Wait for it to be unmounted.
	if err = mfs.Join(context.Background()); err != nil {
		log.Fatalf("Join: %v", err)
	}
}
