package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"runtime"
	"sync"
)

const partSize = 16 * 1024 * 1024

type work struct {
	io.Reader
	*uint32
}

func main() {
	var wg sync.WaitGroup
	works := make(chan work)
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buffer := make([]byte, partSize)
			for work := range works {
				n, _ := io.ReadFull(work.Reader, buffer)
				*work.uint32 = crc32.ChecksumIEEE(buffer[:n])
			}
		}()
	}
	size, err := os.Stdin.Seek(0, io.SeekEnd)
	if err != nil {
		panic(err)
	}
	checksums := make([]uint32, (size+partSize-1)/partSize)
	for i := range checksums {
		works <- work{io.NewSectionReader(os.Stdin, int64(i*partSize), partSize), &checksums[i]}
		fmt.Fprint(os.Stderr, ".")
	}
	close(works)
	wg.Wait()
	fmt.Fprintln(os.Stderr, "")
	var buffer bytes.Buffer
	binary.Write(&buffer, binary.LittleEndian, checksums)
	checksum := crc32.ChecksumIEEE(buffer.Bytes())
	binary.Write(os.Stdout, binary.LittleEndian, checksum)
	binary.Write(os.Stdout, binary.LittleEndian, checksums)
}
