//go:build windows

package sparsified

/*
(Windows not implemented yet)

notes:

"There’s some extra work required to use sparse files on Windows. You must first create an empty file, set the FSCTL_SET_SPARSE attribute on it, and close the file pointer. You then need to re-open a file pointer to the file before you can write any data to it sparsely.

It’s not a hugely complicated process, but the added complexity is enough to cause many programs to not bother supporting sparse files on Windows. Notably, closing file handlers is slow on Windows because it triggers indexing and virus scans

On FreeBSD, Linux, MacOS, and Solaris; you can check on a file’s sparseness using the ls -lsk test.file command and some arithmetic. The commands return a couple of columns; the first contains the sector allocation count (the on-disk storage size in blocks) and the sixth column returns the apparent file size in bytes. Take the first number and multiply it by 1024. The file is sparse (or compressed by the file system!) if the resulting number is lower than the sector allocation count.

me on darwin:
$ ls -lsk test003.sparse

sector allocation count       apparent file size in bytes
|                             |
v                             v
0 -rw-------  1 jaten  staff  4096 Apr 14 21:42 test003.sparse

what ls -k is doing here:
     -k      This has the same effect as setting environment variable
             BLOCKSIZE to 1024, except that it also nullifies any -h options
             to its left.


The above method is easy to memorize and is portable across all file systems and operating system (except Windows). On Windows, you can check whether a file is sparse using the fsutil sparse queryflag test.file command.

You can also get these numbers everywhere (except Windows) with the du (disk usage) command. However, its arguments and capabilities are different on each operating system, so it’s more difficult to memorize.

Recent versions of FreeBSD, Linux, MacOS, and Solaris include an API that can detect sparse “holes” in files. This is the SEEK_HOLE extension to the lseek function, detailed in its man page. https://www.man7.org/linux/man-pages/man2/lseek.2.html

I built a small program using the above API. My sparseseek program scans files and lists how much of it is stored as spares/holes and how much is data. (I initially named it sparsehole and didn’t notice the problem before reading it aloud.)
https://codeberg.org/da/sparseseek/src/branch/main/sparseseek.c


 -- https://www.ctrl.blog/entry/sparse-files.html
*/
