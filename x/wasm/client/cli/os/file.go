package os

// This is same file with https://github.com/Finschia/finschia-sdk/blob/main/internal/os/file.go
// If the file.go of Finschia/finschia-sdk move to other directory not internal, please use finschia-sdk's file.go

import (
	"fmt"
	"io"
	"os"
)

// ReadFileWithSizeLimit expanded os.ReadFile for checking the file size before reading it
func ReadFileWithSizeLimit(name string, sizeLimit int64) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			fmt.Printf("Cannot close the file: %s\n", name)
		}
	}()

	var size int
	if info, err := f.Stat(); err == nil {
		size64 := info.Size()
		// Check the file size
		if size64 > sizeLimit {
			return nil, fmt.Errorf("the file is too large: %s, size limit over > %d", name, sizeLimit)
		}
		if int64(int(size64)) == size64 {
			size = int(size64)
		}
	}
	size++ // one byte for final read at EOF

	// If a file claims a small size, read at least 512 bytes.
	// In particular, files in Linux's /proc claim size 0 but
	// then do not work right if read in small pieces,
	// so an initial read of 1 byte would not work correctly.
	if size < 512 {
		size = 512
	}

	data := make([]byte, 0, size)
	for {
		if len(data) >= cap(data) {
			d := data[:cap(data)]
			d = append(d, 0)
			data = d[:len(data)]
		}
		n, err := f.Read(data[len(data):cap(data)])
		data = data[:len(data)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return data, err
		}
	}
}
