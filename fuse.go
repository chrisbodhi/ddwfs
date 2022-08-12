package main

import (
	"archive/zip"
	"context"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

func NewFS(fs *FS) (fuse.Server, error) {
	return fuseutil.NewFileSystemServer(fs), nil
}

func Mount(path, mnt string) error {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer archive.Close()

	fs := &FS{
		archive: &archive.Reader,
	}

	server, err := NewFS(fs)
	cfg := &fuse.MountConfig{
		ReadOnly: true,
	}
	mfs, err := fuse.Mount(mnt, server, cfg)
	if err != nil {
		log.Fatalf("Mount: %v", err)
	}

	// Wait for it to be unmounted.
	if err = mfs.Join(context.Background()); err != nil {
		log.Fatalf("Join: %v", err)
	}

	return nil
}

type FS struct {
	fuseutil.NotImplementedFileSystem

	archive *zip.Reader
}

// var _ fs.FS = (*FS)(nil)

// func (f *FS) Root() (fs.Node, error) {
func (f *FS) Root() (*Dir, error) {
	n := &Dir{
		archive: f.archive,
	}
	return n, nil
}

type Dir struct {
	archive *zip.Reader
	// nil for the root directory, which has no entry in the zip
	file *zip.File
}

// var _ fs.Node = (*Dir)(nil)

func zipAttr(f *zip.File, a *fuseops.InodeAttributes) {
	a.Size = f.UncompressedSize64
	a.Mode = f.Mode()
	a.Mtime = f.ModTime()
	a.Ctime = f.ModTime()
	a.Crtime = f.ModTime()
}

func (d *Dir) Attr(ctx context.Context, a *fuseops.InodeAttributes) error {
	if d.file == nil {
		// root directory
		a.Mode = os.ModeDir | 0755
		return nil
	}
	zipAttr(d.file, a)
	return nil
}

func (d *Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	path := req.Name
	if d.file != nil {
		path = d.file.Name + path
	}
	for _, f := range d.archive.File {
		switch {
		case f.Name == path:
			child := &File{
				file: f,
			}
			return child, nil
		case f.Name[:len(f.Name)-1] == path && f.Name[len(f.Name)-1] == '/':
			child := &Dir{
				archive: d.archive,
				file:    f,
			}
			return child, nil
		}
	}
	return nil, fuse.ENOENT
}

var _ = fs.HandleReadDirAller(&Dir{})

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	prefix := ""
	if d.file != nil {
		prefix = d.file.Name
	}

	var res []fuse.Dirent
	for _, f := range d.archive.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		name := f.Name[len(prefix):]
		if name == "" {
			// the dir itself, not a child
			continue
		}
		if strings.ContainsRune(name[:len(name)-1], '/') {
			// contains slash in the middle -> is in a deeper subdir
			continue
		}
		var de fuse.Dirent
		if name[len(name)-1] == '/' {
			// directory
			name = name[:len(name)-1]
			de.Type = fuse.DT_Dir
		}
		de.Name = name
		res = append(res, de)
	}
	return res, nil
}

type File struct {
	file *zip.File
}

var _ fs.Node = (*File)(nil)

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	zipAttr(f.file, a)
	return nil
}

var _ = fs.NodeOpener(&File{})

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	r, err := f.file.Open()
	if err != nil {
		return nil, err
	}
	// individual entries inside a zip file are not seekable
	resp.Flags |= fuse.OpenNonSeekable
	return &FileHandle{r: r}, nil
}

type FileHandle struct {
	r io.ReadCloser
}

var _ fs.Handle = (*FileHandle)(nil)

var _ fs.HandleReleaser = (*FileHandle)(nil)

func (fh *FileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return fh.r.Close()
}

var _ = fs.HandleReader(&FileHandle{})

func (fh *FileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// We don't actually enforce Offset to match where previous read
	// ended. Maybe we should, but that would mean'd we need to track
	// it. The kernel *should* do it for us, based on the
	// fuse.OpenNonSeekable flag.
	//
	// One exception to the above is if we fail to fully populate a
	// page cache page; a read into page cache is always page aligned.
	// Make sure we never serve a partial read, to avoid that.
	buf := make([]byte, req.Size)
	n, err := io.ReadFull(fh.r, buf)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	resp.Data = buf[:n]
	return err
}
