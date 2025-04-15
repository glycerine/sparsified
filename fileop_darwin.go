//go:build darwin

package main

/*
#include <sys/fcntl.h> // for fpunchhole_t
*/
import "C"

import (
	"fmt"
	"os"
	//"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const FALLOC_FL_PUNCH_HOLE = 99 // F_PUNCHHOLE = 99 // from sys/fcntl.h:319

// =========================================
// =========================================
// collapse-range
// =========================================
// =========================================

// maybe punch-hole?
// #define F_PUNCHHOLE     99              /* Deallocate a range of the file */
// /* fpunchhole_t used by F_PUNCHHOLE */
//typedef struct fpunchhole {
//	unsigned int fp_flags; /* unused */
//	unsigned int reserved; /* (to maintain 8-byte alignment) */
//	off_t fp_offset; /* IN: start of the region */
//	off_t fp_length; /* IN: size of the region */
//} fpunchhole_t;

// bsm/audit_fcntl.h
// 118:#define BSM_F_PUNCHHOLE         362     /* Darwin-specific. */

// https://freebsd-arch.freebsd.narkive.com/JKZnzc4p/hole-punching-trim-etc
// Hole-punching has been discussed on these lists before[1]. It basically
// means to turn a dense file into a sparse file by deallocating storage for
// some of the blocks in the middle. There's no standard API for it. Linux
// uses fallocate(2); Solaris and OSX add a new opcode to fcntl(2).

// The *linux* docs for fallocate(1) say
//        -p, --punch-hole
// Deallocates space (i.e., creates a hole) in the byte range
// starting at offset and continuing for length bytes. Within the
// specified range, partial filesystem blocks are zeroed, and
// whole filesystem blocks are removed from the file. After a
// successful call, subsequent reads from this range will return
// zeroes. This option may not be specified at the same time as
// the --zero-range option. Also, when using this option,
//            --keep-size is implied.

// might need to write a tomb-stone and then
// jump past or back-track up the btree, if collapse-range
// is not available.

// Circa 2017-2018 APFS came out.
//
// https://eclecticlight.co/2017/01/27/apple-file-system-what-it-means-what-it-brings/
//
// "APFS can also clone files and directories almost instantly. The majority of apps now support keeping previous revisions of documents as you work on them. Currently, this is done by saving each as a version. When APFS manages versions, only the changed data in each version have to be saved to storage, which uses far less space, and works much faster."
//
// "APFS also supports ‘sparse’ files, which work a bit like highly efficient compression, but without the overhead. For example, a huge file containing very little data only occupies the space required to store those data, not the entire file size."
//
// "Time Machine is currently an amazing trick accomplished despite HFS+. APFS includes a feature to create a read-only copy of a file system, a snapshot, which will form the basis of Time Machine’s backups in the future. These should significantly reduce the time taken to make backups and to restore from them, and should increase their reliability even when working with massive backups accumulated over time."
//
// "Another weakness in HFS+ which you will be familiar with is the time that it takes to work out the number of files and space taken by a folder or volume. APFS is designed to make those figures available almost instantly."
//
// https://developer.apple.com/documentation/foundation/file_system/about_apple_file_system
// Apple File System replaces HFS Plus as the default file system for iOS 10.3 and later, and for macOS High Sierra and later. Apple File System offers improved file system fundamentals as well as several new features, including cloning, snapshots, space sharing, fast directory sizing, atomic safe-save, and sparse files.
//
// says sparse files are supported.
// https://developer.apple.com/go/?id=apfs-file-format-spec
//
//
// Clones Reduce the Cost of Copying
// A clone is a copy of a file or directory that occupies no additional space on disk. Clones let you make fast, power-efficient file copies on the same volume. The copyItem(at:to:) and copyItem(atPath:toPath:) methods of FileManager automatically create a clone for Apple File System volumes, as shown in the listing below.

// Object maps
// An object map uses a B-tree to store a mapping from virtual object identifiers and transaction identifiers to the physical
// addresses where those objects are stored. The keys in the B-tree are instances of omap_key_t and the values are
// instances of paddr_t.
//
//
// It has B-trees in its interface.
//
//
//
// It has a Space Manager.
// The space manager allocates and frees blocks where objects and file data can be stored. Thereʼs exactly one instance
// of this structure in a container.
//
// There is a Reaper
// The reaper is a mechanism that allows large objects to be deleted over a period spanning multiple transactions. Thereʼs
// exactly one instance of this structure in a container.

//Block and Container Sizes
//Constants used when choosing the size of a block or container.
//	The block size for a container is defined by the nx_block_size field of nx_superblock_t.
//	#define NX_MINIMUM_BLOCK_SIZE 4096
//#define NX_DEFAULT_BLOCK_SIZE 4096
//#define NX_MAXIMUM_BLOCK_SIZE 65536

// about collapse-range
/*
Q: On Linux on XFS or ext4, the
FALLOC_FL_COLLAPSE_RANGE argument to fallocate can be
used to splice out and completely remove a block-aligned
range of bytes from a file. What is the equivalent on darwin/MacOS?

LLM Answer:

On macOS/Darwin, there isn't a direct equivalent to Linux's FALLOC_FL_COLLAPSE_RANGE.

The closest operations available are:
  1. Using fcntl(2) with F_PREALLOCATE/F_SETSIZE - but this doesn't do range removal
  2. Manual data copying - which is less efficient than Linux's atomic operation

To remove a range from a file on macOS, you need to manually:
  1. Read the data after the range
  2. Seek to the start of the range
  3. Write the data
  4. Truncate the file

Here's how it would look in Go:

func collapseRange(f *os.File, offset, length int64) error {
    // Get file size
    fi, err := f.Stat()
    if err != nil {
        return err
    }

    // Read all data after the range we want to remove
    afterData := make([]byte, fi.Size()-offset-length)
    if _, err := f.ReadAt(afterData, offset+length); err != nil {
        return err
    }

    // Write data back at the collapse point
    if _, err := f.WriteAt(afterData, offset); err != nil {
        return err
    }

    // Truncate file to new size
    return f.Truncate(fi.Size() - length)
}

This is not as efficient as Linux's FALLOC_FL_COLLAPSE_RANGE because:
 + It's not atomic
 + Requires additional memory for buffering
 + Requires actual data copying
 + Doesn't handle sparse files efficiently
 + The lack of this feature is due to differences in how
   APFS and HFS+ handle file allocation compared to Linux
   filesystems like XFS and ext4.
*/

// =========================================
// =========================================
// insert-range
// =========================================
// =========================================

// https://stackoverflow.com/questions/11497567/fallocate-command-equivalent-in-os-x
//
// fallocate() doesn't exist on OSX. You can "fake" it though;
// Mozilla fakes it in their FileUtils class. See this...
//
// https://stackoverflow.com/questions/29682880/preallocate-storage-with-fcntl-doesnt-work-as-expected
//
// It appears that the posix_fallocate equivalent [on OS X] is to the fnctl followed by a truncate() call (which actually forces data to be written to the file)
//
// https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/fcntl.2.html
// command is F_PREALLOCATE
//
// The F_PREALLOCATE command operates on the following structure:
//
//	typedef struct fstore {
//
// u_int32_t fst_flags;      /* IN: flags word */
// int       fst_posmode;    /* IN: indicates offset field */
// off_t     fst_offset;     /* IN: start of the region */
// off_t     fst_length;     /* IN: size of the region */
// off_t     fst_bytesalloc; /* OUT: number of bytes allocated */
//
//	       } fstore_t;
//
//	The flags (fst_flags) for the F_PREALLOCATE command are as follows:
//
// F_ALLOCATECONTIG   Allocate contiguous space.  0x2
//
// F_ALLOCATEALL      Allocate all requested space or no space at all. 0x4.
//
// Note F_ALLOCATEPERSIST is 0x8.
// from
// /Applications/Xcode.app/Contents/Developer/Platforms/
// MacOSX.platform/Developer/SDKs/MacOSX.sdk/usr/include/sys/fcntl.h
//
// sys/fcntl.h:376:#define F_ALLOCATEPERSIST 0x00000008    /* do not free space upon close(2) */
//
//	 F_ALLOCATEPERSIST  Allocate space that is not freed when close(2)
//		is called. (Note that the file system may ignore
//		this request.)
const F_ALLOCATEPERSIST = 0x8

//
// The position modes (fst_posmode) for the F_PREALLOCATE command indicate
// how to use the offset field.  The modes are as follows:
//
// F_PEOFPOSMODE   Allocate from the physical end of file.
//
// F_VOLPOSMODE    Allocate from the volume offset.

// darwin man page fcntl(2) says:
//     F_PREALLOCATE          Preallocate file storage space.  Note: upon
//                            success, the space that is allocated can be the
//                            size requested, larger than the size requested, or
//                            (if the F_ALLOCATEALL flag is not provided)
//                            smaller than the space requested.

var fcntl64Syscall uintptr = unix.SYS_FCNTL

// The fallocate Linux syscall does not exist on Darwin/MacOS,
// so our fallocate(FALLOC_FL_INSERT_RANGE) needs to be emulated here.
// We try to emulate it with a fcntl(2) call, which
// should be (hopefully!) similar, according to the
// StackOverflow suggestion above/what Mozilla does to "fake it".
func fallocate(fd *os.File, mode uint32, off int64, length int64) (allocated int64, err error) {

	if mode == FALLOC_FL_INSERT_RANGE {

		// fstore_t store = {F_ALLOCATECONTIG, F_PEOFPOSMODE, 0, aLength};
		// Try to get a continous chunk of disk space
		store := &unix.Fstore_t{
			Flags:   unix.F_ALLOCATEALL | unix.F_ALLOCATEALL | F_ALLOCATEPERSIST,
			Posmode: unix.F_PEOFPOSMODE,
			Offset:  off,
			Length:  length,
		}

		// nice: avoids needing unsafe locally, and cgo, so build is fast.
		err = unix.FcntlFstore(fd.Fd(), int(unix.F_PREALLOCATE), store)

		allocated = int64(store.Bytesalloc)
		// 0x1b == 27
		// 0x1b on too big => 'file too large'.
		vv("err = '%v'/'%#v'; allocated='%v'; length=%v", err, err, allocated, length)
		if err != nil {
			if err.Error() == "file too large" {
				err = ErrFileTooLarge
				return
			}
		}
		if allocated < length {
			err = ErrShortAlloc
		}
		return
	} else if mode == FALLOC_FL_PUNCH_HOLE {

		punch := &C.fpunchhole_t{
			fp_offset: C.off_t(off),
			fp_length: C.off_t(length),
		}
		_ = punch
		// use darwin_F_PUNCHHOLE = 99.
		// #define F_PUNCHHOLE     99              /* Deallocate a range of the file */
		// /* fpunchhole_t used by F_PUNCHHOLE */
		//typedef struct fpunchhole {
		//	unsigned int fp_flags; /* unused */
		//	unsigned int reserved; /* (to maintain 8-byte alignment) */
		//	off_t fp_offset; /* IN: start of the region */
		//	off_t fp_length; /* IN: size of the region */
		//} fpunchhole_t;

		//err = unix.FcntlFstore(fd.Fd(), int(darwin_F_PUNCHHOLE), punch)

		// gives cgo: ./fileop_darwin.go:276:12: unexpected type: ...
		// because C decl is int     fcntl(int, int, ...) __DARWIN_ALIAS_C(fcntl);
		// in sys/fcntl.h:603
		// full path: /Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/usr/include/sys/fcntl.h
		//_, err = C.fcntl(int(fd), uintptr(darwin_F_PUNCHHOLE), int(uintptr(unsafe.Pointer(punch))))

		// this seems to work! see du -h on the file,
		// and use Stat().Sys().(*syscall.Stat_t).Blocks to get actual blocks in use.
		_, _, errno := unix.Syscall(fcntl64Syscall,
			uintptr(fd.Fd()), uintptr(darwin_F_PUNCHHOLE), uintptr(unsafe.Pointer(punch)))
		if errno != 0 {
			vv("errno = '%v'", errno)
			if errno.Error() == "result too large" {
				err = ErrFileTooLarge
				return
			}
			err = errno
			return
		}
	} else {
		panic(fmt.Sprintf("unhandled fallocate mode '%v'", int(mode)))
	}
	return

	// from the Mozilla code example linked above:
	//
	// fstore_t store = {F_ALLOCATECONTIG, F_PEOFPOSMODE, 0, aLength};
	// Try to get a continous chunk of disk space
	// int ret = fcntl(fd, syscall.F_PREALLOCATE, &store);
	// if(-1 == ret){
	//  	// OK, perhaps we are too fragmented, allocate non-contiguous
	//  	store.fst_flags = F_ALLOCATEALL;
	//  	ret = fcntl(fd, F_PREALLOCATE, &store);
	//  	if (-1 == ret)
	//  	return false;
	//  }
	//  return 0 == ftruncate(fd, aLength);
	//
}

/*
./yogadb

yoga.go:21 2025-03-18 00:54:30.574 -0500 CDT got intfd = '3'

fileop_darwin.go:99 2025-03-18 00:54:30.574 -0500 CDT errno = '0x0'; valptr = '0x0'; allocated='4096'
wrote 'out.db'. all done.

jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $ stat out.db

## does the 4096 mean unused?
16777220 40314569 -rw------- 1 jaten staff 0 1 "Mar 18 06:00:08 2025" "Mar 18 06:06:39 2025" "Mar 18 06:06:39 2025" "Mar 18 05:54:30 2025" 4096 16 0 out.db

jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $ ls -al out.db

-rw-------  1 jaten  staff  1 Mar 18 06:06 out.db
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $ ls -ls out.db

16 -rw-------  1 jaten  staff  1 Mar 18 06:06 out.db
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $


$ stat -r out.db

16777220 40314569 0100600 1 501 20 0 1 1742277608 1742277999 1742277999 1742277270 4096 16 0 out.db
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $ stat -s out.db

st_dev=16777220 st_ino=40314569 st_mode=0100600 st_nlink=1 st_uid=501 st_gid=20 st_rdev=0 st_size=1 st_atime=1742277608 st_mtime=1742277999 st_ctime=1742277999 st_birthtime=1742277270 st_blksize=4096 st_blocks=16 st_flags=0

$ stat -x out.db
stat -x out.db
  File: "out.db"
  Size: 1            FileType: Regular File
  Mode: (0600/-rw-------)         Uid: (  501/   jaten)  Gid: (   20/   staff)
Device: 1,4   Inode: 40314569    Links: 1
Access: Tue Mar 18 06:00:08 2025
Modify: Tue Mar 18 06:06:39 2025
Change: Tue Mar 18 06:06:39 2025
 Birth: Tue Mar 18 05:54:30 2025

$ ls -lh out.db

-rw-------  1 jaten  staff     1B Mar 18 06:06 out.db
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $ du out.db

16	out.db
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $ du -h out.db

8.0K	out.db
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $

AH HAH! there it is! 1 byte plus 4096 more means we are using two blocks on APFS.

## vet seems okay with the cgo passing of pointer above.
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $ go vet   -cgocall .
go vet   -cgocall .
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $


## cgo checking did not have a problem...
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $ GOEXPERIMENT=cgocheck2  go build
GOEXPERIMENT=cgocheck2  go build
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $ ./yogadb
./yogadb

yoga.go:21 2025-03-18 01:29:05.911 -0500 CDT got intfd = '3'

fileop_darwin.go:70 2025-03-18 01:29:05.911 -0500 CDT errno = '0x0'; valptr = '0x0'; allocated='4096'
wrote 'out.db'. all done.
jaten@Js-MacBook-Pro ~/go/src/github.com/glycerine/yogadb (master) $
*/
/*
Porting tips for Darwin, how to use CGO safely:

Ian Lance Taylor
2:21 PM (4 hours ago)
to Jason E. Aten, golang-nuts
On Tue, Mar 18, 2025 at 12:08 AM Jason E. Aten <j.e....@gmail.com> wrote:
>
> I'm porting some C code that does manual filesystem "extent"
> file space management from Linux to Darwin (and into Go).
>
> The Linux fallocate() call and its FALLOC_FL_INSERT_RANGE
> operation are not directly available on Darwin, but suggestions from StackOverflow
> indicate that fcntl(2) with F_PREALLOCATE might be a close substitute.
>
> It seems to be working. Full code here in this playground, but I have a couple of questions--
>
> https://go.dev/play/p/Z5JB__fUBpI
>
> Can I get away without the runtime.Pinner? (Am I still safe from 'store' being pre-maturely garbage collected without the pin, since Go memory has been converted to an uintptr before calling unix.Syscall())

You don't need the runtime.Pinner but you do need to do the conversion
to uintptr in the call to unix.Syscall. That is

unix.Syscall(fcntl64Syscall, uintptr(fd),
uintptr(syscall.F_PREALLOCATE), uintptr(unsafe.Pointer(ustore)))

This uses rule 4 at https://pkg.go.dev/unsafe#Pointer. The memory will
be pinned for the duration of the system call.

> Is there a more appropriate way to call fcntl() (on Darwin)?

Darwin doesn't particularly like calling unix.Syscall directly,
although it does work. They prefer that you call the C function. You
could do that via cgo by calling C.fcntl.

Or in this case it seems simpler to just use the unix.FcntlFstore
function which already exists and does the right thing.

me:
I filed https://github.com/golang/go/issues/72923

Ian:
(there is a hidden drop down!)

It's definitely awkward for packages like x/sys/unix that provide a different API depending on the target. If you look over on the right and scroll down a bit, you'll see a "Rendered for" drop down selection. If you select darwin/amd64, you'll be redirected to https://pkg.go.dev/golang.org/x/sys/unix?GOOS=darwin, which does show the function: https://pkg.go.dev/golang.org/x/sys/unix?GOOS=darwin#FcntlFstore . It would be nice if this could be clearer.

Ian
*/

/*
APFS provides F_GETPATH and FSCTL_GETPATH to get path information.
Most importantly, APFS supports two key operations for finding file extents:
FSCTL_FIOSEEKHOLE: Finds the next hole after a given offset
FSCTL_FIOSEEKDATA: Finds the next data region after a given offset
*/
