//go:build linux

package sparsified

import (
	//"fmt"
	"os"
	//"syscall"

	"golang.org/x/sys/unix"
	// TODO: fiemap
	//"github.com/glycerine/go-fibmap"
)

// used by fileop_test.go, defined here for portability.
// fileop_darwin.go has its own, different, definition.
const FALLOC_FL_PUNCH_HOLE = unix.FALLOC_FL_PUNCH_HOLE

/*
   Deallocating file space
       Specifying the FALLOC_FL_PUNCH_HOLE flag (available since Linux 2.6.38)
       in mode deallocates space (i.e., creates a  hole)  in  the  byte  range
       starting  at offset and continuing for len bytes.  Within the specified
       range, partial filesystem  blocks  are  zeroed,  and  whole  filesystem
       blocks  are removed from the file.  After a successful call, subsequent
       reads from this range will return zeroes.

       The FALLOC_FL_PUNCH_HOLE flag must be ORed with FALLOC_FL_KEEP_SIZE  in
       mode;  in  other words, even when punching off the end of the file, the
       file size (as reported by stat(2)) does not change.

       Not all  filesystems  support  FALLOC_FL_PUNCH_HOLE;  if  a  filesystem
       doesn't  support the operation, an error is returned.  The operation is
       supported on at least the following filesystems:

       *  XFS (since Linux 2.6.38)

       *  ext4 (since Linux 3.0)

       *  Btrfs (since Linux 3.7)

       *  tmpfs(5) (since Linux 3.5)

*/

/*
   Collapsing file space
       Specifying the FALLOC_FL_COLLAPSE_RANGE  flag  (available  since  Linux
       3.15) in mode removes a byte range from a file, without leaving a hole.
       The byte range to be collapsed starts at offset and continues  for  len
       bytes.   At  the  completion of the operation, the contents of the file
       starting at the location offset+len will be appended  at  the  location
       offset, and the file will be len bytes smaller.

       A filesystem may place limitations on the granularity of the operation,
       in order to ensure efficient implementation.  Typically, offset and len
       must  be  a multiple of the filesystem logical block size, which varies
       according to the filesystem type and configuration.   If  a  filesystem
       has such a requirement, fallocate() fails with the error EINVAL if this
       requirement is violated.

       If the region specified by offset plus len reaches or passes the end of
       file,  an  error  is  returned; instead, use ftruncate(2) to truncate a
       file.

       No other flags may be  specified  in  mode  in  conjunction  with  FAL‐
       LOC_FL_COLLAPSE_RANGE.

       As  at  Linux 3.15, FALLOC_FL_COLLAPSE_RANGE is supported by ext4 (only
       for extent-based files) and XFS.
*/
// How do I make an extent-based file on ext4?
/*
A by LLM:

On ext4, files are automatically extent-based by default
since ext4 was introduced - you don't need to do anything
special. The extent feature was one of the major
improvements from ext3 to ext4.

However, if you want to verify that your ext4 filesystem
has extent support enabled:

tune2fs -l /dev/your_device | grep extent

example output, shows extent:

Filesystem features:      has_journal ext_attr resize_inode dir_index filetype needs_recovery extent 64bit flex_bg sparse_super large_file huge_file dir_nlink extra_isize metadata_csum

You should see "extent" in the features list.

Check if a specific file is using extents:

$ filefrag -v your_file

If it shows "ext" in the output, it's using extents.

example output:

$ filefrag -v out.db
Filesystem type is: ef53
File size of out.db is 1 (1 block of 4096 bytes)
 ext:     logical_offset:        physical_offset: length:   expected: flags:
   0:        0..       0:   80912788..  80912788:      1:             last,eof
out.db: 1 extent found
$

For an existing filesystem, extents can be enabled with:

$ tune2fs -O extent /dev/your_device

Important notes:
* All modern ext4 filesystems enable extents by default
* Files created on ext4 automatically use extents unless:
  + The filesystem was upgraded from ext3 without enabling extents
  + The file was created before extents were enabled
  + The filesystem was mounted with noextent option (very rare)

So for your fallocate with FALLOC_FL_COLLAPSE_RANGE operation,
any newly created file on a modern ext4 filesystem will support it by default.
*/

func fallocate(file *os.File, mode uint32, offset int64, length int64) (allocated int64, err error) {

	if mode == FALLOC_FL_PUNCH_HOLE {
		mode |= unix.FALLOC_FL_KEEP_SIZE
	}

	var preSz, postSz int64
	preSz, err = fileSizeFromFile(file)
	panicOn(err)

	intfd := int(file.Fd())

	err = unix.Fallocate(intfd, mode, offset, length)
	vv("unix.Fallocate() returned err = '%v'", err) // "file too large"

	// TODO: TEST this assumption! can we get a less-than full allocation and err==nil?
	// The unix.Fallocate on linux hides the return value from us,
	// (see ~/go/pkg/mod/golang.org/x/sys@v0.31.0/unix/zsyscall_linux_amd64.go ),
	// so we might need to reimplement this ourselves to get at it.
	if err == nil {

		postSz, err = fileSizeFromFile(file)
		panicOn(err)
		vv("preSz = %v; postSz = %v", preSz, postSz)

		allocated = int64(length)
		return
	}
	if err.Error() == "file too large" {
		err = errFileTooLarge
		return
	}
	if allocated < length {
		err = errShortAlloc
	}
	// unknown error, just pass to caller.
	return
}

// before writing new stuff into the extent, we got:
/*
	jaten@rog ~/go/src/github.com/glycerine/yogadb $ xfs_info /mnt/a
	meta-data=/dev/sda               isize=512    agcount=8, agsize=268435455 blks
	         =                       sectsz=512   attr=2, projid32bit=1
	         =                       crc=1        finobt=1 spinodes=0 rmapbt=0
	         =                       reflink=0
	data     =                       bsize=4096   blocks=1953506646, imaxpct=5
	         =                       sunit=0      swidth=0 blks
	naming   =version 2              bsize=4096   ascii-ci=0 ftype=1
	log      =internal               bsize=4096   blocks=521728, version=2
	         =                       sectsz=512   sunit=0 blks, lazy-count=1
	realtime =none                   extsz=4096   blocks=0, rtextents=0

*/
