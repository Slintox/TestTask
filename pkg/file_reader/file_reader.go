package file_reader

import (
	"errors"
	"io"
	"os"
)

var (
	ErrInvalidBlockSize = errors.New("invalid batch size")
	ErrInvalidFileName  = errors.New("invalid file name")
)

type FileReader struct {
	label  string
	file   *os.File
	name   string
	size   int64
	offset int64
	eof    bool
	buf    []byte
}

func NewFileReader(fileName string, readBlockSize int) (*FileReader, error) {
	if len(fileName) == 0 {
		return nil, ErrInvalidFileName
	}
	if readBlockSize < 1 {
		return nil, ErrInvalidBlockSize
	}

	file, err := os.OpenFile(fileName, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	var fileStats os.FileInfo
	if fileStats, err = file.Stat(); err != nil {
		return nil, err
	}

	buf := make([]byte, readBlockSize)

	return &FileReader{
		file:   file,
		name:   file.Name(),
		size:   fileStats.Size(),
		offset: 0,
		eof:    false,
		buf:    buf,
	}, nil
}

func (r *FileReader) GetFile() *os.File {
	return r.file
}

func (r *FileReader) Name() string {
	return r.file.Name()
}

func (r *FileReader) Size() int64 {
	return r.size
}

func (r *FileReader) EOF() bool {
	return r.eof
}

func (r *FileReader) ReadBytes() (int, []byte, error) {
	n, err := r.file.ReadAt(r.buf, r.offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, nil, err
	} else if errors.Is(err, io.EOF) && n == 0 {
		r.eof = true
	}

	r.offset += int64(n)
	if r.offset > r.size {
		r.eof = true
	}
	return n, r.buf, err
}

func (r *FileReader) SetOffset(newOffset int64) {
	if newOffset >= 0 {
		r.offset = newOffset
	}
}

func (r *FileReader) Truncate(newSize int64) error {
	if err := r.file.Truncate(newSize); err != nil {
		return err
	}
	r.size = newSize
	return nil
}

func (r *FileReader) Close() error {
	return r.file.Close()
}
