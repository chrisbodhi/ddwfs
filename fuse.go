package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/timeutil"
)

func FS(clock timeutil.Clock) (fuse.Server, error) {
	fs := &TBD{
		Clock: clock,
	}

	return fuseutil.NewFileSystemServer(fs), nil
}

type TBD struct {
	fuseutil.NotImplementedFileSystem

	mu          sync.Mutex
	Clock       timeutil.Clock
	createTime  time.Time
	nextHandle  fuseops.HandleID
	fileHandles map[fuseops.HandleID]string
}

const (
	rootInode fuseops.InodeID = fuseops.RootInodeID + iota
	helloInode
	dirInode
	worldInode
)

type inodeInfo struct {
	attributes fuseops.InodeAttributes

	// File or directory?
	dir bool

	// For directories, children.
	children []fuseutil.Dirent
}

var zipName = "spacefem-gestation-data"

func MakeDirent(dir string, offset int) fuseutil.Dirent {
	return fuseutil.Dirent{
		Offset: fuseops.DirOffset(offset),
		Inode:  MakeDirNode(),
		Name:   dir,
		Type:   fuseutil.DT_Directory,
	}
}

func MakeDirNode() inodeInfo {
	return inodeInfo{
		attributes: fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0555 | os.ModeDir,
		},
		dir:      true,
		children: []fuseutil.Dirent{},
	}
}

var gInodeInfo = map[fuseops.InodeID]inodeInfo{
	// root
	rootInode: {
		attributes: fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0555 | os.ModeDir,
		},
		dir: true,
		children: []fuseutil.Dirent{
			{
				Offset: 1,
				Inode:  helloInode,
				Name:   "hello",
				Type:   fuseutil.DT_File,
			},
			{
				Offset: 2,
				Inode:  dirInode,
				Name:   "dir",
				Type:   fuseutil.DT_Directory,
			},
		},
	},
	helloInode: {
		attributes: fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0444,
			// Leave size blank, i.e. 0, for dynamic files
			// Size:  uint64(len("Hello, world!")),
		},
	},
	dirInode: {
		attributes: fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0555 | os.ModeDir,
		},
		dir: true,
		children: []fuseutil.Dirent{
			{
				Offset: 1,
				Inode:  worldInode,
				Name:   "world",
				Type:   fuseutil.DT_File,
			},
		},
	},
	// world
	worldInode: {
		attributes: fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0444,
			Size:  uint64(len("Hello, world!")),
		},
	},
}

func findChildInode(
	name string,
	children []fuseutil.Dirent) (fuseops.InodeID, error) {
	for _, e := range children {
		if e.Name == name {
			return e.Inode, nil
		}
	}

	return 0, fuse.ENOENT
}

func (fs *TBD) findUnusedHandle() fuseops.HandleID {
	handle := fs.nextHandle
	for _, exists := fs.fileHandles[handle]; exists; _, exists = fs.fileHandles[handle] {
		handle++
	}
	fs.nextHandle = handle + 1
	return handle
}

func (fs *TBD) patchAttributes(
	attr *fuseops.InodeAttributes) {
	now := fs.Clock.Now()
	attr.Atime = now
	attr.Mtime = now
	attr.Crtime = now
}

func (fs *TBD) StatFS(
	ctx context.Context,
	op *fuseops.StatFSOp) error {
	return nil
}

func (fs *TBD) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) error {
	// Find the info for the parent.
	parentInfo, ok := gInodeInfo[op.Parent]
	if !ok {
		return fuse.ENOENT
	}

	// Find the child within the parent.
	childInode, err := findChildInode(op.Name, parentInfo.children)
	if err != nil {
		return err
	}

	// Copy over information.
	op.Entry.Child = childInode
	op.Entry.Attributes = gInodeInfo[childInode].attributes

	// Patch attributes.
	fs.patchAttributes(&op.Entry.Attributes)

	return nil
}

func (fs *TBD) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) error {
	// Find the info for this inode.
	info, ok := gInodeInfo[op.Inode]
	if !ok {
		return fuse.ENOENT
	}

	// Copy over its attributes.
	op.Attributes = info.attributes

	// Patch attributes.
	fs.patchAttributes(&op.Attributes)

	return nil
}

func (fs *TBD) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) error {
	// Allow opening any directory.
	return nil
}

func (fs *TBD) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) error {
	// Find the info for this inode.
	info, ok := gInodeInfo[op.Inode]
	if !ok {
		return fuse.ENOENT
	}

	if !info.dir {
		return fuse.EIO
	}

	entries := info.children

	// Grab the range of interest.
	if op.Offset > fuseops.DirOffset(len(entries)) {
		return fuse.EIO
	}

	entries = entries[op.Offset:]

	// Resume at the specified offset into the array.
	for _, e := range entries {
		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], e)
		if n == 0 {
			break
		}

		op.BytesRead += n
	}

	return nil
}

func (fs *TBD) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	var contents string
	// Update file contents on (and only on) open.
	switch op.Inode {
	case ageInode:
		now := fs.Clock.Now()
		ageInSeconds := int(now.Sub(fs.createTime).Seconds())
		contents = fmt.Sprintf("This filesystem is %d seconds old.", ageInSeconds)
	case weekdayInode:
		contents = fmt.Sprintf("Today is %s.", fs.Clock.Now().Weekday())
	default:
		return fuse.EINVAL
	}
	handle := fs.findUnusedHandle()
	fs.fileHandles[handle] = contents
	op.UseDirectIO = true
	op.Handle = handle
	return nil
}

func (fs *TBD) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) error {
	// Let io.ReaderAt deal with the semantics.
	reader := strings.NewReader("Hello, world!")

	var err error
	op.BytesRead, err = reader.ReadAt(op.Dst, op.Offset)

	// Special case: FUSE doesn't expect us to return io.EOF.
	if err == io.EOF {
		return nil
	}

	return err
}
