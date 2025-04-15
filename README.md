sparsified: sparse file handling in Go
==========

Work-in-progress on sparse file handling.

Currently has Linux and Darwin support. Windows
support is deferred.

Reading/references
------------------

https://github.com/longhorn/sparse-tools/

how darwin copies sparse files... see copyfile_data_sparse() here https://github.com/apple-oss-distributions/copyfile/blob/main/copyfile.c#L2516

https://github.com/frostschutz/go-fibmap

extents: links to more info
-------

https://en.wikipedia.org/wiki/Extent_(file_systems)

https://www.eecs.harvard.edu/~cs161/notes/intro-file-systems.pdf

https://unix.stackexchange.com/questions/223310/what-do-extents-feature-do-in-ext4-filesystem-in-linux

https://stackoverflow.com/questions/3942223/what-is-extentin-fs

https://www.eg.bucknell.edu/~cs315/2013-fall/sec02/notes/25-File-System-Implementation-P3-print.pdf

https://blogs.oracle.com/linux/post/extents-and-extent-allocation-in-ext4

~~~
// How do I make an extent-based file on ext4?

Answer from an LLM:

On ext4, files are automatically extent-based by default
since ext4 was introduced - you don't need to do anything
special. The extent feature was one of the major
improvements from ext3 to ext4.

However, if you want to verify that your ext4 filesystem
has extent support enabled:

~~~
tune2fs -l /dev/your_device | grep extent
~~~

example output, shows extent:

~~~
Filesystem features:      has_journal ext_attr resize_inode dir_index filetype needs_recovery extent 64bit flex_bg sparse_super large_file huge_file dir_nlink extra_isize metadata_csum
~~~

You should see "extent" in the features list.

Check if a specific file is using extents:

~~~
$ filefrag -v your_file
~~~

If it shows "ext" in the output, it's using extents.

example output:
~~~
$ filefrag -v out.db
Filesystem type is: ef53
File size of out.db is 1 (1 block of 4096 bytes)
 ext:     logical_offset:        physical_offset: length:   expected: flags:
   0:        0..       0:   80912788..  80912788:      1:             last,eof
out.db: 1 extent found
$
~~~

For an existing filesystem, extents can be enabled with:

~~~
$ tune2fs -O extent /dev/your_device
~~~

Important notes:

* All modern ext4 filesystems enable extents by default

* Files created on ext4 automatically use extents unless:
  + The filesystem was upgraded from ext3 without enabling extents
  + The file was created before extents were enabled
  + The filesystem was mounted with noextent option (very rare)

So for your fallocate with FALLOC_FL_COLLAPSE_RANGE operation,
any newly created file on a modern ext4 filesystem will support it by default.


APFS (Apple File System, on all iPhones and darwin >= High Sierra, replaced HFS+)
--------

https://umatechnology.org/understanding-apfs-the-new-file-system-from-apple/

https://eclecticlight.co/2024/03/20/apfs-files-and-clones/

XFS
--------

XFS will pre-allocate extent space when it sees append activity...

https://serverfault.com/questions/406069/why-are-my-xfs-filesystems-suddenly-consuming-more-space-and-full-of-sparse-file

"I traced this issue back to a discussion about a commit to the XFS source tree from December 2010. The patch was introduced in Kernel 2.6.38 (and obviously, later backported into some popular Linux distribution kernels).

"The observed fluctuations in disk usage are a result of a new feature; XFS Dynamic Speculative EOF Preallocation.

"This is a move to reduce file fragmentation during streaming writes by speculatively allocating space as file sizes increase. The amount of space preallocated per file is dynamic and is primarily a function of the free space available on the filesystem (to preclude running out of space entirely).

It follows this schedule:
~~~
freespace       max prealloc size
  >5%             full extent (8GB)
  4-5%             2GB (8GB >> 2)
  3-4%             1GB (8GB >> 3)
  2-3%           512MB (8GB >> 4)
  1-2%           256MB (8GB >> 5)
  <1%            128MB (8GB >> 6)
~~~

"Performance on XFS volumes with this feature is drastically improved. I'm seeing consistent < 1% fragmentation on volumes that previously displayed up to 50% fragmentation. Write performance is up globally!

"Oh, wonderful find. This was using 750GB on 35GB of files. After xfs_fsr it's back down to about 35GB. I'll have to keep an eye on that

https://web.archive.org/web/20171010174337/http://oss.sgi.com/archives/xfs/2010-12/msg00328.html

https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=055388a3188f56676c21e92962fc366ac8b5cb72



a bunch of random notes on sparse file handling
-----------------------------------------------

https://stackoverflow.com/questions/43035271/sparse-files-are-huge-with-io-copy

Q: can I piggy back on my zero RLE compression to write that back as a sparse file thing?

https://github.com/golang/go/issues/13548

good discussion of sparse files for go archive/tar

Change https://golang.org/cl/56771 mentions this issue: archive/tar: refactor Reader support for sparse files

Change https://golang.org/cl/57212 mentions this issue: archive/tar: implement Writer support for sparse files

** [ ] add sparse file hole detection to rsync / bytes.Zeros RLE detection to
make it more efficient.

if the OS/filesystem gives SEEK_HOLE and SEEK_DATA support

"In Windows, you can use os.File.Fd() to access the underlying HANDLE, with which you can call DeviceIOControl with control code FSCTL_QUERY_ALLOCATED_RANGES to access the hole list (see this example).
https://www.codeproject.com/Articles/53000/Managing-Sparse-Files-on-Windows

"Currently released versions of macOS (or rather HFS+) doesn't support sparse files. The new APFS filesystem supports them, but the documentation is rather sparse at the moment, given that macOS with APFS is still in beta (this is the only APFS-related API list I found, and it touches several features but not sparse files).

"I did some quick test on both beta e non beta version of macOS, and it looks like APFS allows to create sparse file just like Linux, by simply seeking; for instance, I did dd if=/dev/zero of=file.img bs=1 count=0 seek=512000000 to create a file of apparent size of 512 MB that occupies zero bytes (verified with du file.img). Also, the man page of lseek includes SEEK_HOLE and SEEK_DATA, though I haven't directly tested them, but they're described as working exactly as they work in Linux and Solaris. So it looks like that macOS support will be achieved with the same code that will be used on Linux.

"You seem to want to avoid OS-specific code in Reader / Writer. I'm afraid that's not fully possible because on Windows you need to create holes through a specific API; seeking by itself does not create holes, just zeros. So Reader.WriteTo will have to call OS-specific code, when Windows support is added.

Change https://golang.org/cl/60871 mentions this issue: archive/tar: add Header.DetectSparseHoles

Change https://golang.org/cl/60872 mentions this issue: archive/tar: add Reader.WriteTo and Writer.ReadFrom

Came across this issue looking for sparse-file support in Golang. API looks good to me and certainly fits my usecase :). Is there no sysSparsePunch needed for unix?

On Unix OSes that support sparse files, seeking past EOF and writing or resizing the file to be larger automatically produces a sparse file.

Change https://golang.org/cl/78030 mentions this issue: archive/tar: partially revert sparse file support

good pointers in https://github.com/lxc/incus/issues/662

archive/tar: re-add sparse file support golang/go#22735
https://github.com/golang/go/issues/22735

archive/tar: add support for writing tar containing sparse files golang/go#13548
https://github.com/golang/go/issues/13548

lxc publish expands sparse files canonical/lxd#4239
https://github.com/canonical/lxd/issues/4239

rsc design discussion
https://github.com/golang/go/issues/22735

AFAICT, on Windows, you can't create sparse zero areas by seeking, as the MSDN documentation clearly states:

https://msdn.microsoft.com/it-it/library/windows/desktop/aa365566%28v=vs.85%29.aspx
https://blogs.msdn.microsoft.com/oldnewthing/20110922-00/?p=9573/

The blog post hints that you can set the file sparse and create immediately a full-size sparse span, so that later writers+seeks would basically fragment it but leaving sparse areas under the seeks. I have no clue if there is an impact on performance with this approach, and also it doesn't really belong to a os.File.SetSparse API I would say.

Note that this was discussed at length in #13548, where also your proposal of lazy header writing was analyzed and discarded.

------------

https://github.com/seaweedfs/seaweedfs/blob/1d89d20798f0b7289882a50dd164a449bba408b4/weed/storage/backend/volume_create_linux.go#L19

weed/storage/backend/volume_create_linux.go


sparse-tools for longhorn storage synchronization.
https://github.com/longhorn/sparse-tools

sparse storage slab synchronization, etc.

https://github.com/longhorn/sparse-tools/

great example
https://github.com/svenwiltink/sparsecat

## make a 1MB sparse file
$ truncate -s1M image.raw

$ dd if=/dev/urandom bs=1K count=1 conv=notrunc seek=30 of=image.raw
1+0 records in
1+0 records out
1024 bytes transferred in 0.000490 secs (2089796 bytes/sec)
$ ls -lsk image.raw 
1024 -rw-------  1 jaten  staff  1048576 Apr 14 22:36 image.raw
$ 
ls -lsk image.raw 
1024 -rw-------  1 jaten  staff  1048576 Apr 14 22:36 image.raw
jaten@jbook ~/yogadb (master) $
-------

https://pkg.go.dev/github.com/frostschutz/go-fibmap

https://stackoverflow.com/questions/38669605/how-to-use-ioctl-with-fs-ioc-fiemap

https://github.com/coreutils/coreutils/blob/df88fce71651afb2c3456967a142db0ae4bf9906/src/extent-scan.c#L112

"Note fiemap is not recommended as you have to be sure to pass FIEMAP_FLAG_SYNC which has side effects. The lseek(), SEEK_DATA and SEEK_HOLE interface is the recommended one, though note that will, depending on file system, represent unwritten extents (allocated zeros) as holes."

Thanks for the suggestion. We did try SEEK_DATA and SEEK_HOLE with lseek, but it looks like it is supported only from a higher linux kernel version for xfs file system than the one we are on. So, we had to resort to the ioctl way. I am kind of new to this low level programming, could you advice on what could be the side effects with the FIEMAP_FLAG_SYNC flag? –
Aila
 CommentedJul 31, 2016 at 1:26
 
"syncing can have large performance implications and should be avoided where possible"

https://github.com/torvalds/linux/blob/master/include/uapi/linux/fiemap.h

struct fiemap - file extent mappings

https://lwn.net/Articles/260795/
By Jonathan Corbet
December 3, 2007
"Sparse files have an apparent size which is larger than the amount of storage actually allocated to them.  The usual way to create such a file is to seek past its end and write some new data; Unix-derived systems will traditionally not allocate disk blocks for the portion of the file past the previous end which was skipped over. The result is a "hole," a piece of the file which logically exists, but which is not represented on disk. A read operation on a hole succeeds, with the returned data being all zeroes. Relatively smart file archival and backup utilities will recognize holes in files; these holes are not stored in the resulting archive and will not be filled if the file is restored from that archive."

Even so, this patch looks relatively unlikely to make it into the mainline. The API is unpopular, being seen as ugly and as a change in the semantics of the lseek() call. But, more to the point, it may be interesting to learn much more about the representation of a file than just where the holes are. And, as it turns out, there is already a proposed ioctl() command which can provide all of that information. That interface is the FIEMAP ioctl() specified by Andreas Dilger back in October."

FIEMAP appeared in Linux kernel 2.6.28, released on 25 December, 2008.
SEEK_HOLE and SEEK_DATA appeared in Linux kernel 3.1, although ext4 support for these was only added in Linux 3.8.


fibmap

https://stackoverflow.com/questions/2894824/linux-how-do-i-know-the-block-map-of-the-given-file-and-or-the-free-space-map

https://serverfault.com/questions/29886/how-do-i-list-a-files-data-blocks-on-linux/29918#29918

A simple way to get the list of blocks (without having to read from the partition like in the debugfs answers) is to use the FIBMAP ioctl. I do not know of any command to do so, but it is very simple to write one; a quick Google search gave me an example of FIBMAP use, which does exactly what you want. One advantage is that it will work on any filesystem which supports the bmap operation, not just ext3.

A newer (and more efficient) alternative is the FIEMAP ioctl, which can also return detailed information about extents (useful for ext4).

https://github.com/Thomas-Tsai/partclone/issues/174

Partclone provides utilities to backup a partition smartly and it is designed for higher compatibility of the file system by using existing library.

Partclone now supports ext2, ext3, ext4, hfs+, reiserfs, reiser4, btrfs, vmfs3, vmfs5, xfs, jfs, ufs, ntfs, fat(12/16/32), exfat...

windows NTFS!
http://partclone.org/

FSCTL_FIOSEEKHOLE

https://eclecticlight.co/2024/06/08/apfs-how-sparse-files-work/

To achieve this, APFS does very little indeed. The file’s inode contains the INODE_IS_SPARSE flag, and in its extended-field the number of sparse bytes in the data stream, INO_EXT_TYPE_SPARSE_BYTES, is given as an unsigned 64-bit integer.

The trick is accomplished in the file’s extent map, which gives the offset in the file’s data in bytes, against the physical block address that the extent starts at. To return to the example 10 GB sparse file, its inode has the INODE_IS_SPARSE flag set, its extended-field gives the number of sparse bytes in the file, and its file extent map gives the physical block address for the non-null data at the offset at the end of the file. There’s no need for any additional metadata.

Tools
Sparsity creates test sparse files and can discover which files in any given folder are in sparse format;
Precize provides full information about files, including whether they are sparse or clone files.

https://eclecticlight.co/taccy-signet-precize-alifix-utiutility-alisma/
https://eclecticlight.co/taccy-signet-precize-alifix-utiutility-alisma/

You can create sparse file with truncate(1).
truncate -s +1000M sparsefile


Howard, turns out it’s actually not hard to deallocate unused blocks from a file and thus make it a sparse file.

I found a little utility written in C for Windows and Linux that takes an SQLite file and deallocates unused pages in the database. I was able to make this work on the Mac using:

fcntl(fd, F_PUNCHHOLE, &punchhole)

See https://github.com/iljitschvanbeijnum/sqlite_sparse/blob/master/sqlite_sparse.c

https://eclecticlight.co/2024/06/08/apfs-how-sparse-files-work/

If you dig in the macOS source, you’ll find the implementation of `copyfile(3)`. And there’s a very interesting function called `copyfile_data_sparse` that is great as an example of handling sparse files:

using `lseek(…, SEEK_HOLE)` to locate existing holes in files
using `lseek(…, SEEK_DATA)` to locate non-hole data in files
using `fcntl(…, F_PUNCHHOLE, …)` to punch new holes in files
There’s also the kernel implementation of `F_PUNCHHOLE` in vnode-land, but that’s way over my head.

https://github.com/apple-oss-distributions/copyfile/blob/ed3f0a8bf8b6bac6838c92c297afcc826fec75f4/copyfile.c#L2191

https://github.com/apple-oss-distributions/copyfile/blob/ed3f0a8bf8b6bac6838c92c297afcc826fec75f4/copyfile.c#L2191

https://github.com/apple-oss-distributions/copyfile/raw/ed3f0a8bf8b6bac6838c92c297afcc826fec75f4/copyfile.c

https://github.com/apple-oss-distributions/copyfile/raw/refs/heads/main/copyfile.c

and... how darwin copies sparse files... see copyfile_data_sparse() here https://github.com/apple-oss-distributions/copyfile/blob/main/copyfile.c#L2516

https://github.com/longhorn/sparse-tools/blob/master/sparse/fiemap.go
