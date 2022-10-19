package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type DumpFile struct {
	filePath string
}

func NewDumpFile(dir, fileName string) *DumpFile {
	return &DumpFile{
		filePath: filepath.Join(dir, fileName),
	}
}

func (f *DumpFile) Save(ctx context.Context, numbers []int64) error {
	sort.Slice(numbers, func(i, j int) bool { return numbers[i] < numbers[j] })

	newFilePath, err := f.createNewDump(ctx, numbers)
	if err != nil {
		return fmt.Errorf("create new dump: %w", err)
	}

	if err = os.Rename(newFilePath, f.filePath); err != nil {
		os.Remove(newFilePath)
		return fmt.Errorf("change old dump to new dump: %w", err)
	}

	return nil
}

func (f *DumpFile) createNewDump(ctx context.Context, numbers []int64) (newFilePath string, err error) {
	newFilePath = fmt.Sprintf("%s.new", f.filePath)

	newFile, err := os.Create(newFilePath)
	if err != nil {
		err = fmt.Errorf("create new file: %w", err)
		return
	}
	defer func() {
		if err == nil {
			return
		}
		os.Remove(newFilePath)
	}()
	defer newFile.Close()

	w := bufio.NewWriter(newFile)

	buf := make([]byte, 10)
	for _, number := range numbers {
		binary.PutVarint(buf, number)

		if _, err = w.Write(buf); err != nil {
			err = fmt.Errorf("write number to file: %w", err)
			return
		}
	}

	if err = w.Flush(); err != nil {
		err = fmt.Errorf("flush byffer to file: %w", err)
		return
	}

	return
}
