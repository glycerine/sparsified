package main

/*
import (
	"os"
)

const (
	SEEK_DATA = 3
	SEEK_HOLE = 4
)

func getpos(fd *os.File) int {

	return fd.Lseek(0, SEEK_CUR)
//  off_t res = lseek(fd, 0, SEEK_CUR);
//  return(res);
}

void print_results(char* file_name, off_t data, off_t sparse)
{
  off_t total = data + sparse;
  int length = (int) log10((long  double) total / 1024) + 2;

  printf("%s:\n", file_name);
  printf("Data:  % *ld kB % 12ld KiB\n", length, data / 1000, data / 1024);
  printf("Holes: % *ld kB % 12ld KiB\n", length, sparse / 1000, sparse / 1024);
  printf("Total: % *ld kB % 12ld KiB\n", length, total / 1000, total / 1024);

  return;
}

void scan_file(char* file_name, int fd, off_t len)
{
  off_t data_s = 0;
  off_t hole_s = 0;
  off_t start  = 0;
  off_t end    = 0;
  bool hole    = false;

  lseek(fd, 0, SEEK_SET);
  // First, determine if we start in a data or hole region,
  // and quit early if there are no holes.
  end = lseek(fd, 0, SEEK_HOLE);

  if (end == len)
  {
    print_results(file_name, len, hole_s);
    return;
  }

  if (start == end)
  {
    hole = true;
    end = lseek(fd, 0, SEEK_DATA);
  }

  while (end <= len && end != -1)
  {
    if (hole)
    {
      hole_s += end - start;
      start = end;
      end = lseek(fd, start, SEEK_HOLE);
      hole = false;
    }
    else
    {
      data_s += end - start;
      start = end;
      end = lseek(fd, start, SEEK_DATA);
      hole = true;
    }
  }

  print_results(file_name, data_s, hole_s);
  return;
}

int main(int argc, char **argv)
{
  int argc_i = 1;

  if (argc < 2)
  {
    fprintf(stderr, "Usage: sparsehole FILE...\n");
    exit(EINVAL);
  }

  while (argc > argc_i)
  {
    struct stat *finfo = malloc(sizeof(struct stat));
    int fd = open(argv[argc_i], O_RDONLY);

    if (fd == -1)
    {
      fprintf(stderr, "Can't open %s\n", argv[argc_i]);
      return(EINVAL);
    }

    fstat(fd, finfo);
    scan_file(argv[argc_i], fd, finfo->st_size);

    free(finfo);
    close(fd);

    argc_i += 1;
  }

  return(0);
}
*/
