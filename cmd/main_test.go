package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	"TestTask/pkg/file_reader"

	"github.com/stretchr/testify/assert"
)

const (
	TestFolderPath      = "../data/"
	NamesTestFolderPath = "TestFolders/"
)

func TestGetFileNamesWithMinMaxNameLen(t *testing.T) {
	type TestCase struct {
		Name               string
		TestFolder         string
		AllowNegativeNames bool

		ExpectedMinName string
		ExpectedMaxName string

		MustError     bool
		ExpectedError error
	}

	tcs := []TestCase{
		{
			Name:               "Empty folder",
			TestFolder:         "TC0",
			AllowNegativeNames: true,
			ExpectedMinName:    "",
			ExpectedMaxName:    "",
			MustError:          true,
			ExpectedError:      ErrNoFiles,
		},
		{
			Name:               "Folder with 1 positive file: allow negative true",
			TestFolder:         "TC1_1Positive",
			AllowNegativeNames: true,
			ExpectedMinName:    "",
			ExpectedMaxName:    "",
			MustError:          true,
			ExpectedError:      ErrNotEnoughFiles,
		},
		{
			Name:               "Folder with 1 positive file: allow negative false",
			TestFolder:         "TC1_1Positive",
			AllowNegativeNames: false,
			ExpectedMinName:    "",
			ExpectedMaxName:    "",
			MustError:          true,
			ExpectedError:      ErrNotEnoughFiles,
		},
		{
			Name:               "Folder with 1 negative file: allow negative true",
			TestFolder:         "TC1_1Negative",
			AllowNegativeNames: true,
			ExpectedMinName:    "",
			ExpectedMaxName:    "",
			MustError:          true,
			ExpectedError:      ErrNotEnoughFiles,
		},
		{
			Name:               "Folder with 1 negative file: allow negative false",
			TestFolder:         "TC1_1Negative",
			AllowNegativeNames: false,
			ExpectedMinName:    "",
			ExpectedMaxName:    "",
			MustError:          true,
			ExpectedError:      ErrNoFiles,
		},
		{
			Name:               "Folder with 2 positive files: allow negative true",
			TestFolder:         "TC2_2Positive",
			AllowNegativeNames: true,
			ExpectedMinName:    "5999.log",
			ExpectedMaxName:    "6000.log",
			ExpectedError:      nil,
		},
		{
			Name:               "Folder with 2 positive files: allow negative false",
			TestFolder:         "TC2_2Positive",
			AllowNegativeNames: false,
			ExpectedMinName:    "5999.log",
			ExpectedMaxName:    "6000.log",
			ExpectedError:      nil,
		},
		{
			Name:               "Folder with 2 negative files: allow negative true",
			TestFolder:         "TC2_2Negative",
			AllowNegativeNames: true,
			ExpectedMinName:    "-6000.log",
			ExpectedMaxName:    "-5999.log",
			ExpectedError:      nil,
		},
		{
			Name:               "Folder with 2 negative files: allow negative false",
			TestFolder:         "TC2_2Negative",
			AllowNegativeNames: false,
			ExpectedMinName:    "",
			ExpectedMaxName:    "",
			MustError:          true,
			ExpectedError:      ErrNoFiles,
		},
		{
			Name:               "Folder with 1 positive 1 negative file: allow negative true",
			TestFolder:         "TC2_1Positive1Negative",
			AllowNegativeNames: true,
			ExpectedMinName:    "-6000.log",
			ExpectedMaxName:    "5999.log",
			ExpectedError:      nil,
		},
		{
			Name:               "Folder with 1 positive 1 negative file: allow negative false",
			TestFolder:         "TC2_1Positive1Negative",
			AllowNegativeNames: false,
			ExpectedMinName:    "",
			ExpectedMaxName:    "",
			MustError:          true,
			ExpectedError:      ErrNotEnoughFiles,
		},
		{
			Name:               "Folder with 2 positive 1 negative file: allow negative true",
			TestFolder:         "TC3_2Positive1Negative",
			AllowNegativeNames: true,
			ExpectedMinName:    "-6000.log",
			ExpectedMaxName:    "5999.log",
			ExpectedError:      nil,
		},
		{
			Name:               "Folder with 2 positive 1 negative file: allow negative false",
			TestFolder:         "TC3_2Positive1Negative",
			AllowNegativeNames: false,
			ExpectedMinName:    "5.log",
			ExpectedMaxName:    "5999.log",
			ExpectedError:      nil,
		},
		{
			Name:               "Folder with 1 positive 2 negative file: allow negative true",
			TestFolder:         "TC3_1Positive2Negative",
			AllowNegativeNames: true,
			ExpectedMinName:    "-6000.log",
			ExpectedMaxName:    "5999.log",
			ExpectedError:      nil,
		},
		{
			Name:               "Folder with 1 positive 2 negative file: allow negative false",
			TestFolder:         "TC3_1Positive2Negative",
			AllowNegativeNames: false,
			ExpectedMinName:    "",
			ExpectedMaxName:    "",
			MustError:          true,
			ExpectedError:      ErrNotEnoughFiles,
		},
		{
			Name:               "Long file names: allow negative false: allow negative true",
			TestFolder:         "TC_LongNames",
			AllowNegativeNames: true,
			ExpectedMinName:    "-12345678901011121314151617181920.log",
			ExpectedMaxName:    "12345678901011121314151617181920.log",
			MustError:          true,
			ExpectedError:      nil,
		},
		{
			Name:               "Long file names: allow negative false: allow negative false",
			TestFolder:         "TC_LongNames",
			AllowNegativeNames: false,
			ExpectedMinName:    "12345678901011121314151617181919.log",
			ExpectedMaxName:    "12345678901011121314151617181920.log",
			MustError:          true,
			ExpectedError:      nil,
		},
		{
			Name:               "Same file names: allow negative false",
			TestFolder:         "TC_SameLength",
			AllowNegativeNames: true,
			ExpectedMinName:    "-24.log",
			ExpectedMaxName:    "500.log",
			MustError:          true,
			ExpectedError:      nil,
		},
		{
			Name:               "Same file names: allow negative false",
			TestFolder:         "TC_SameLength",
			AllowNegativeNames: false,
			ExpectedMinName:    "100.log",
			ExpectedMaxName:    "500.log",
			MustError:          true,
			ExpectedError:      nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			min, max, err := GetFileNamesWithMinMaxNameNum(TestFolderPath+NamesTestFolderPath+tc.TestFolder, tc.AllowNegativeNames)
			assert.ErrorIs(t, err, tc.ExpectedError)
			assert.Equal(t, min, tc.ExpectedMinName)
			assert.Equal(t, max, tc.ExpectedMaxName)
		})
	}
}

func generateNewLogData(size int) []byte {
	newFileValue := make([]byte, size)

	for i := 0; i < len(newFileValue); i++ {
		if i != 0 && i%80 == 0 {
			newFileValue[i] = '\n'
		} else if i != 0 && i%79 == 0 && (i+1)%80 == 0 {
			newFileValue[i] = '\r'
		} else {
			newFileValue[i] = 'a'
		}

		if i == len(newFileValue)-3 {
			newFileValue[i] = 'e'
		} else if i == len(newFileValue)-2 {
			newFileValue[i] = 'n'
		} else if i == len(newFileValue)-1 {
			newFileValue[i] = 'd'
		}
	}

	return newFileValue
}

func generateNewLogData2(size int) []byte {
	newFileValue := make([]byte, size)

	for i := 0; i < len(newFileValue); i++ {
		if i != 0 && i%80 == 0 {
			newFileValue[i] = '\n'
		} else if i != 0 && i%79 == 0 && (i+1)%80 == 0 {
			newFileValue[i] = '\r'
		} else {
			newFileValue[i] = 'b'
		}

		if i == len(newFileValue)-3 {
			newFileValue[i] = '3'
		} else if i == len(newFileValue)-2 {
			newFileValue[i] = 'n'
		} else if i == len(newFileValue)-1 {
			newFileValue[i] = '6'
		}
	}

	return newFileValue
}

func TestByteRecordingToFile(t *testing.T) {
	testFileName := TestFolderPath + "testCaseBRTF.log"
	outFileName := TestFolderPath + "outCaseBRTF.log"

	_ = os.Truncate(testFileName, 0)
	_ = os.Truncate(outFileName, 0)

	newTestFile, err := os.OpenFile(testFileName, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}

	newFileValue := generateNewLogData(32*1024 + 77)

	_, err = newTestFile.Write(newFileValue)
	if err != nil {
		t.Fatal(err)
	}

	_ = newTestFile.Close()

	startReader, err := file_reader.NewFileReader(testFileName, 64)
	if err != nil {
		t.Fatal(err)
	}

	startSize := startReader.Size()

	outFile, err := os.OpenFile(outFileName, os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err)
	}
	_ = outFile.Truncate(0)

	bytes := make(chan byte, 1)
	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go ByteRecordingToFile(outFile, bytes, errCh, wg)

	for !startReader.EOF() {
		n, batch, err := startReader.ReadBytes()
		if errors.Is(err, io.EOF) {
			if n == 0 {
				break
			}
		} else if err != nil {
			t.Fatal(err)
		}

		if n < len(batch) {
			batch = batch[:n]
		}

		select {
		case err = <-errCh:
			t.Fatal(err)
		default:
		}

		for i, c := range batch {
			_ = i
			bytes <- c
		}
	}

	close(bytes)
	wg.Wait()

	outReader, err := file_reader.NewFileReader(outFileName, 64)
	if err != nil {
		t.Fatal(err)
	}

	if outReader.Size() != startReader.Size() || outReader.Size() != startSize {
		fmt.Println("Несовпадение размеров файла")
		fmt.Println("Изначальный файл:", startReader.Size())
		fmt.Println("Конечный файл:", outReader.Size())
	}

	startReader.SetOffset(0)

	allEOF := false
	for !allEOF {
		allEOF = false

		n, sourceText, err := startReader.ReadBytes()
		if errors.Is(err, io.EOF) {
			allEOF = true
		} else if err != nil {
			t.Fatal(err)
		}

		m, destText, err := outReader.ReadBytes()
		if errors.Is(err, io.EOF) {
			allEOF = allEOF == true
		} else if err != nil {
			t.Fatal(err)
		}

		if n != m {
			t.Fatal("n != m:", n, "!=", m)
		}

		assert.Equal(t, sourceText, destText)
	}

	fmt.Println("End of test")

	_ = os.Remove(testFileName)
	_ = os.Remove(outFileName)
}

func TestByteRecordingToFileBuffered(t *testing.T) {
	testFileName := TestFolderPath + "testCaseBRTFB.log"
	outFileName := TestFolderPath + "outCaseBRTFB.log"

	_ = os.Truncate(testFileName, 0)
	_ = os.Truncate(outFileName, 0)

	newTestFile, err := os.OpenFile(testFileName, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}

	newFileValue := generateNewLogData(1 * 64)

	_, err = newTestFile.Write(newFileValue)
	if err != nil {
		t.Fatal(err)
	}

	_ = newTestFile.Close()

	startReader, err := file_reader.NewFileReader(testFileName, 64)
	if err != nil {
		t.Fatal(err)
	}

	startSize := startReader.Size()

	outFile, err := os.OpenFile(outFileName, os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err)
	}
	_ = outFile.Truncate(0)

	bytes := make(chan byte, 1)
	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go ByteRecordingToFileBuffered(outFile, bytes, 32, errCh, wg)

	for !startReader.EOF() {
		n, batch, err := startReader.ReadBytes()
		if errors.Is(err, io.EOF) {
			if n == 0 {
				break
			}
		} else if err != nil {
			t.Fatal(err)
		}

		if n < len(batch) {
			batch = batch[:n]
		}

		select {
		case err = <-errCh:
			t.Fatal(err)
		default:
		}

		for i, c := range batch {
			_ = i
			bytes <- c
		}
	}

	close(bytes)
	wg.Wait()

	outReader, err := file_reader.NewFileReader(outFileName, 64)
	if err != nil {
		t.Fatal(err)
	}

	if outReader.Size() != startReader.Size() || outReader.Size() != startSize {
		fmt.Println("Несовпадение размеров файла")
		fmt.Println("Изначальный файл:", startReader.Size())
		fmt.Println("Конечный файл:", outReader.Size())
	}

	startReader.SetOffset(0)

	allEOF := false
	for !allEOF {
		allEOF = false

		n, sourceText, err := startReader.ReadBytes()
		if errors.Is(err, io.EOF) {
			allEOF = true
		} else if err != nil {
			t.Fatal(err)
		}

		m, destText, err := outReader.ReadBytes()
		if errors.Is(err, io.EOF) {
			allEOF = allEOF == true
		} else if err != nil {
			t.Fatal(err)
		}

		if n != m {
			t.Error("n != m:", n, "!=", m)
		}

		assert.Equal(t, sourceText, destText)
	}

	fmt.Println("End of test")

	_ = os.Remove(testFileName)
	_ = os.Remove(outFileName)
}

func TestSwapTwoFiles(t *testing.T) {
	firstFileName := TestFolderPath + "TestSwapTwoFiles1.log"
	secondFileName := TestFolderPath + "TestSwapTwoFiles2.log"

	var err error

	_ = os.Truncate(firstFileName, 0)
	_ = os.Truncate(secondFileName, 0)

	firstFileData := generateNewLogData(32*1024 + 77)
	if err = os.WriteFile(firstFileName, firstFileData, 0600); err != nil {
		t.Fatal(err)
	}

	secondFileData := generateNewLogData2(36*1024 + 77)
	if err = os.WriteFile(secondFileName, secondFileData, 0600); err != nil {
		t.Fatal(err)
	}

	err = SwapTwoFiles("", firstFileName, secondFileName, 64, 32)
	if err != nil {
		t.Fatal(err)
	}

	var firstOutData, secondOutData []byte
	if firstOutData, err = os.ReadFile(firstFileName); err != nil {
		t.Fatal(err)
	}
	if secondOutData, err = os.ReadFile(secondFileName); err != nil {
		t.Fatal(err)
	}

	fmt.Println("Размеры файлов")
	fmt.Println("Первый начальный файл:", len(firstFileData))
	fmt.Println("Первый конечный файл: ", len(firstOutData))
	fmt.Println("Второй начальный файл:", len(secondFileData))
	fmt.Println("Второй конечный файл: ", len(secondOutData))

	assert.Equal(t, firstFileData, secondOutData)
	assert.Equal(t, secondFileData, firstOutData)

	_ = os.Remove(firstFileName)
	_ = os.Remove(secondFileName)
}

// Used for profiling
func BenchmarkSwapTwoFiles(b *testing.B) {
	readBlockSize := 4 * 1024
	writeBlockSize := 4 * 1024

	err := SwapTwoFiles(TestFolderPath, "202209161152.log", "202209152012010000002.log", readBlockSize, writeBlockSize)
	if err != nil {
		b.Fatal("error:", err)
	}
}
