package main

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	//"golang.org/x/sys/unix"
)

var oneZeroBlock4k [4096]byte

// path is just for reporting if intfd > 0 is given.
// Otherwise it is opened and the fd returned.
func insertRange(path string, fd *os.File, offset int64, length int64) (file *os.File, got int64, err error) {
	if fd == nil {
		// docs: "If the file does not exist, and the O_CREATE
		// flag is passed, it is created with mode perm (before umask);
		// the containing directory must exist."
		if fileExists(path) {
			file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				return
			}
		} else {
			// so we can only insert if the file already has
			// a byte in it. Write a zero.
			fd, err = os.Create(path)
			if err != nil {
				return
			}
			// write a whole block so the rest of the
			// file can be block aligned too.
			var nw int
			nw, err = fd.Write(oneZeroBlock4k[:])
			if err != nil {
				return
			}
			if nw != len(oneZeroBlock4k) {
				return nil, 0, fmt.Errorf("could not write one 4096 zero page"+
					" to new path '%v': '%v'", path, err)
			}
		}
	} else {
		file = fd
	}
	got, err = fallocate(file, FALLOC_FL_INSERT_RANGE, offset, length)
	return
}

// do we get an error when our allocation did not succeed?
func TestFallocateInsertRangeTooBig(t *testing.T) {

	path := "out.db"
	// note: file must be writable! so Open() does not suffice!
	//fd, err = os.OpenFile(path, os.O_RDWR, 0666)

	fd, err := os.Create(path)
	panicOn(err)
	defer fd.Close()

	// does file need to be non-empty? yep. hmm... would be nice if that was not the case.
	_, err = fd.Write(oneZeroBlock4k[:])
	panicOn(err)

	offset := int64(0)
	// too biiiig!!!
	length := int64(4096) << 50 // 4 exa bytes // xfs_info says bsize and extsz are 4096
	//length := int64(64 << 20) // 64MB? yep, Apple Filesystem will happily give us this.

	vv("length = '%v'", length)
	var got int64
	got, err = fallocate(fd, FALLOC_FL_INSERT_RANGE, offset, length)
	if err == nil {
		panic(fmt.Sprintf("oh no. asked for 4 exabytes, got back: '%v' without an error??", got))
	}
	if err != ErrFileTooLarge {
		panic(fmt.Sprintf("wanted errFileTooLarge; got error '%v'", err))
	}
	fmt.Printf("asked for length='%v', got='%v'. wrote to path '%v'. all done.\n", length, got, path)
	// succeeds?!? oh noes...
	// asked for length='16777216', got='16777216'. wrote to path 'out.db'. all done.
	// got a 16MB span.
}

/* linux is going to report the extent-added (sparse?) file differently the Darwin.
$ ls -al
-rw-------   1 jaten jaten  65M Mar 18 13:46 out.db
jaten@rog ~/yogadb (master) $ du -sh .
3.2M	.
jaten@rog ~/yogadb (master) $ du -sh out.db
4.0K	out.db
jaten@rog ~/yogadb (master) $ ls -ls out.db
4 -rw------- 1 jaten jaten 67108865 Mar 18 13:46 out.db
jaten@rog ~/yogadb (master) $
*/

// Collapse-range is the opposite of Insert-range.
// These are from https://pkg.go.dev/golang.org/x/sys/unix#section-readme for Linux/amd64:
/*
FALLOC_FL_ALLOCATE_RANGE                    = 0x0
FALLOC_FL_COLLAPSE_RANGE                    = 0x8
FALLOC_FL_INSERT_RANGE                      = 0x20
FALLOC_FL_KEEP_SIZE                         = 0x1
FALLOC_FL_NO_HIDE_STALE                     = 0x4
FALLOC_FL_PUNCH_HOLE                        = 0x2
FALLOC_FL_UNSHARE_RANGE                     = 0x40
FALLOC_FL_ZERO_RANGE                        = 0x10
*/

// Collapse-Range was added in 2014 to Linux.
// https://lwn.net/Articles/587819/
// (about FALLOC_FL_COLLAPSE_RANGE)
/*
From: Namjae Jeon <namjae.jeon@samsung.com>

This patch series is in response of the following post:
http://lwn.net/Articles/556136/
"ext4: introduce two new ioctls"

Dave Chinner [XFS developer lead] suggested that truncate_block_range
(which was one of the ioctls name) should be an fallocate operation
and not any fs specific ioctl, hence we add this functionality to fallocate.

This patch series introduces new flag FALLOC_FL_COLLAPSE_RANGE for fallocate
and implements it for XFS and Ext4.

The semantics of this flag are following:
1) It collapses the range lying between offset and length by removing any data
   blocks which are present in this range and than updates all the logical
   offsets of extents beyond "offset + len" to nullify the hole created by
   removing blocks. In short, it does not leave a hole.
2) It should be used exclusively. No other fallocate flag in combination.
3) Offset and length supplied to fallocate should be fs block size aligned
   in case of xfs and ext4.
4) Collaspe range does not work beyond i_size.

This new functionality of collapsing range could be used by media editing tools
which does non linear editing to quickly purge and edit parts of a media file.
This will immensely improve the performance of these operations.
The limitation of fs block size aligned offsets can be easily handled
by media codecs which are encapsulated in a conatiner as they have to
just change the offset to next keyframe value to match the proper alignment.

*/

// on darwin, unix.FALLOC_FL_COLLAPSE_RANGE is not defined.
// on linux, unix.FALLOC_FL_COLLAPSE_RANGE = 8
//
// Of note, ZFS does not support this. Open issue for it:
// "Support FALLOC_FL_COLLAPSE_RANGE" Issue #15178
// https://github.com/openzfs/zfs/issues/15178

func TestCollapseRange(t *testing.T) {

	path := "out.db"
	// note: file must be writable! so Open() does not suffice!
	//fd, err = os.OpenFile(path, os.O_RDWR, 0666)

	fd, err := os.Create(path)
	panicOn(err)
	defer fd.Close()

	// does file need to be non-empty? yep. hmm... would be nice if that was not the case.
	var nw int
	nw, err = fd.Write(oneZeroBlock4k[:])
	if err != nil {
		return
	}
	if nw != len(oneZeroBlock4k) {
		panic(fmt.Errorf("could not write one 4096 zero page"+
			" to new path '%v': '%v'", path, err))
	}

	var sz, postsz int64
	sz, err = fileSizeFromFile(fd)
	panicOn(err)
	vv("starting file sz is '%v'", sz)

	//panicOn(err)
	intfd := int(fd.Fd())
	vv("got intfd = '%v'", intfd)

	offset := int64(0)
	// too biiiig!!!
	//length := int64(4096) << 50 // 4 exa bytes // xfs_info says bsize and extsz are 4096
	//length := int64(64 << 20) // 64MB? yep, Apple Filesystem will happily give us this.
	length := int64(4096)

	vv("length = '%v'", length)
	// unix.FALLOC_FL_COLLAPSE_RANGE = 8
	//vv("unix.FALLOC_FL_COLLAPSE_RANGE = %v", unix.FALLOC_FL_COLLAPSE_RANGE)
	var got int64
	//got, err = fallocate(fd, FALLOC_FL_COLLAPSE_RANGE, offset, length)
	got, err = fallocate(fd, FALLOC_FL_PUNCH_HOLE, offset, length)
	vv("fallocate(FALLOC_FL_PUNCH_HOLE) err was '%v'; got = '%v'", err, got)
	if err == nil {
		//panic(fmt.Sprintf("oh no. asked for 4 exabytes, got back: '%v' without an error??", got))
	}
	if err == ErrShortAlloc {

	}
	vv("got: '%v'; err = '%v'", got, err) // "smaller extent than requested was allocated."
	if err != ErrFileTooLarge {
		//panic(fmt.Sprintf("wanted errFileTooLarge; got error '%v'", err))
	}
	//fmt.Printf("asked for length='%v', got='%v'. wrote to path '%v'. all done.\n", length, got, path)

	fi, err := fd.Stat()
	panicOn(err)
	vv("fi = '%#v'", fi)
	postsz = fi.Sys().(*syscall.Stat_t).Blocks * 512
	vv("ending file postsz is '%v'", postsz)
	// same size still, but blocks were deallocated.
	//$ stat out.db
	//  File: out.db
	//  Size: 4096      	Blocks: 0          IO Block: 4096   regular file

	if postsz != 0 {
		panic("why not 0 size file after hole punching?")
	}
}

// test if file is sparse at all.
// and test if we can make a sparse file for sure.
func Test003_sparse_file_creation_and_detection(t *testing.T) {

	path := "test003.sparse"
	os.Remove(path) // don't panic, it might not exist

	nblock := 6
	fd, err := CreateSparseFile(path, nblock)
	panicOn(err)
	isSparse, err := IsSparseFile(fd)
	panicOn(err)
	if !isSparse {
		t.Fatalf("problem: wanted sparse file '%v' but it was not.", path)
	}
}

// if file is sparse, can we locate the holes in it efficiently?
