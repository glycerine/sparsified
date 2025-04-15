package sparsified

import (
	"fmt"
	"os"
	"syscall"
	//"golang.org/x/sys/unix"
	// TODO: fiemap
	//"github.com/glycerine/go-fibmap"
)

// fortunately, this is the same number on linux and darwin.
// const FALLOC_FL_INSERT_RANGE = unix.FALLOC_FL_INSERT_RANGE
const FALLOC_FL_INSERT_RANGE = 32

const linux_FALLOC_FL_COLLAPSE_RANGE = 8
const linux_FALLOC_FL_PUNCH_HOLE = 2 // linux
const darwin_F_PUNCHHOLE = 99        // from sys/fcntl.h:319

var ErrShortAlloc = fmt.Errorf("smaller extent than requested was allocated.")

// allocated probably zero in this case, especially since
// we asked for "all-or-nothing"
var ErrFileTooLarge = fmt.Errorf("extent requested was too large.")

// if file already exists we return nil, error.
// otherwise fd refers to an apparent nblock * 4KB but actual 0 byte file.
func CreateSparseFile(path string, nblock int) (fd *os.File, err error) {

	if nblock < 1 {
		return nil, fmt.Errorf("nblock must be >= 1; not %v", nblock)
	}

	if fileExists(path) {
		//fd, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
		//	if err != nil {
		//		return
		//	}
		return nil, fmt.Errorf("error: file exists '%v'", path)
	}

	// so we can only insert if the file already has
	// a byte in it. Write a zero.
	fd, err = os.Create(path)
	if err != nil {
		return
	}
	// write a whole block so the rest of the
	// file can be block aligned too.
	var nw int
	for i := range nblock {
		nw, err = fd.Write(oneZeroBlock4k[:])
		if err != nil || nw != len(oneZeroBlock4k) {
			fd.Close()
			return nil, fmt.Errorf("could not write %v 4096 zero page"+
				" (i=%v) to new path '%v': '%v'", nblock, i, path, err)
		}
	}

	// punch out all of the first of those blocks
	var offset int64
	length := int64(4096)
	var got int64
	got, err = fallocate(fd, FALLOC_FL_PUNCH_HOLE, offset, length)
	panicOn(err)
	_ = got
	return
}

func IsSparseFile(fd *os.File) (isSparse bool, err error) {

	fi, err := fd.Stat()
	if err != nil {
		return false, err
	}

	stat := fi.Sys().(*syscall.Stat_t)
	apparent := stat.Size
	actual := stat.Blocks * 512 // lies: int64(stat.Blksize), says 4096.
	vv("fi = '%#v'; apparent = '%v'; actual = '%v'", fi, apparent, actual)

	// are there are other ways to be sparse?
	// Just having multiple extents does not matter.
	// Checking the inodes for the sparseness flags seems
	// too much, and I'm not sure they would tell us
	// if the file is "still" sparse. This seems
	// reasonable for now; an "operational definition".

	return actual < apparent, nil
}
