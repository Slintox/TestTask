package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"TestTask/internal/config"
	"TestTask/pkg/file_reader"
)

var (
	ErrNoFiles        = errors.New("there are no files that fit the conditions ([-][0-9]*.log or [0-9]*.log)")
	ErrNotEnoughFiles = errors.New("there are not enough files (at least 2) that match the conditions")
)

// Реализовано чтение и запись по одному символу, однако такой подход крайне медленный.
// Поэтому реализованы также буферизованные версии функций записи и чтения (используются для отладки).
// Для чтения\записи по одному символу можно выставить значения readBlockSize \ writeBlockSize как единицу.
func main() {
	var configPath string
	var allowNegativeNames bool
	var readBlockSize, writeBlockSize int

	flag.StringVar(&configPath, "config-path", "configs/config.yml", "Path to the config file")
	flag.BoolVar(&allowNegativeNames, "neg", false, "Allow reading negative names")
	flag.IntVar(&readBlockSize, "rbs", 1, "The number of bytes read at a time")
	flag.IntVar(&writeBlockSize, "wbs", 1, "The number of bytes written at a time")
	flag.Parse()

	start := time.Now()

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		fmt.Printf("cannot read config file: %s\n", err)
		return
	}

	var minName, maxName string

	minName, maxName, err = GetFileNamesWithMinMaxNameNum(cfg.PathToFiles, allowNegativeNames)
	if err != nil {
		fmt.Printf("GetFileNamesWithMinMaxNameNum: %s\n", err)
		return
	}

	fmt.Printf("File with min value: [%s], File with max value: [%s].\n", minName, maxName)

	err = SwapTwoFiles(cfg.PathToFiles, minName, maxName, readBlockSize, writeBlockSize)
	if err != nil {
		fmt.Printf("Processing error: %s\n", err)
		return
	}
	fmt.Printf("The files was successfully swapped.\nExec time: %s\n", time.Now().Sub(start))
}

func GetFileNamesWithMinMaxNameNum(filesPath string, allowNegativeNames bool) (string, string, error) {
	f, err := os.Open(filesPath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	fileInfo, err := f.Readdir(-1)
	if err != nil {
		return "", "", err
	}

	var count int
	var minName, maxName string

	var numsReg *regexp.Regexp
	if allowNegativeNames {
		// If we accept extreme conditions, including negative numbers in the name
		numsReg = regexp.MustCompile("^-?[0-9]+.log$")
	} else {
		// If the condition is: all names are not negative
		numsReg = regexp.MustCompile("^[0-9]+.log$")
	}

	for _, file := range fileInfo {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if !numsReg.MatchString(fileName) {
			continue
		}

		if minName == "" || maxName == "" {
			minName = fileName
			maxName = fileName
		}

		if allowNegativeNames {
			// If we accept extreme conditions, including negative numbers in the name

			// Compare min name
			switch {
			case fileName[0] == '-' && minName[0] == '-':
				if len(fileName) >= len(minName) && fileName > minName {
					minName = fileName
				}
			case fileName[0] == '-':
				minName = fileName
			case minName[0] == '-':
			default:
				if len(fileName) < len(minName) || (len(fileName) == len(minName) && fileName < minName) {
					minName = fileName
				}
			}

			// Compare max name
			switch {
			case fileName[0] == '-' && maxName[0] == '-':
				if len(fileName) <= len(maxName) && fileName < maxName {
					maxName = fileName
				}
			case fileName[0] == '-':
			case maxName[0] == '-':
				maxName = fileName
			default:
				if len(fileName) > len(maxName) || (len(fileName) == len(maxName) && fileName > maxName) {
					maxName = fileName
				}
			}
		} else {
			// If the condition is: all names are not negative
			if len(fileName) < len(minName) || (len(fileName) == len(minName) && fileName < minName) {
				minName = fileName
			} else if len(fileName) > len(maxName) || (len(fileName) == len(maxName) && fileName > maxName) {
				maxName = fileName
			}
		}

		count++
	}

	if minName == "" || maxName == "" {
		return "", "", ErrNoFiles
	} else if minName == maxName {
		return "", "", ErrNotEnoughFiles
	}

	return minName, maxName, nil
}

// ByteRecordingToFileBuffered writes bytes received from chan to a file.
// An optimized variant of the ByteRecordingToFile.
// Reduces the number of file accesses (1.7s vs 1m 20s for files 16MB and 16 MB).
// Buffers input.
func ByteRecordingToFileBuffered(dstFile *os.File, bytesToWrite <-chan byte, writeBlockSize int, errCh chan error, wg *sync.WaitGroup) {
	var chIndex int64
	buf := make([]byte, writeBlockSize)
	currSymbol := 0

	for ch := range bytesToWrite {
		// Checking errors from the second goroutine
		select {
		case err := <-errCh:
			errCh <- err
			wg.Done()
			return
		default:
		}

		//if currSymbol < writeBlockSize {
		buf[currSymbol] = ch
		currSymbol++
		if currSymbol == writeBlockSize {
			if _, err := dstFile.WriteAt(buf, chIndex); err != nil {
				errCh <- err
				wg.Done()
				return
			}
			chIndex += int64(writeBlockSize)
			currSymbol = 0
		}
	}

	if currSymbol > 0 && currSymbol < writeBlockSize {
		shortBuf := buf[:currSymbol]
		_, _ = dstFile.WriteAt(shortBuf, chIndex)
		chIndex += int64(currSymbol)
	}

	wg.Done()
}

// ByteRecordingToFile writes bytes received from chan to a file.
func ByteRecordingToFile(dstFile *os.File, bytesToWrite <-chan byte, errCh chan error, wg *sync.WaitGroup) {
	var chIndex int64
	buf := make([]byte, 1)

	for ch := range bytesToWrite {
		select {
		case err := <-errCh:
			errCh <- err
			wg.Done()
			return
		default:
		}

		buf[0] = ch
		if _, err := dstFile.WriteAt(buf, chIndex); err != nil {
			errCh <- err
			break
		}

		chIndex++
	}

	wg.Done()
}

func SwapTwoFiles(path, firstName, secondName string, readBlockSize int, writeBlockSize int) error {
	var firstFileReader, secondFileReader *file_reader.FileReader
	var err error

	firstFileReader, err = file_reader.NewFileReader(path+firstName, readBlockSize)
	if err != nil {
		return err
	}
	defer firstFileReader.Close()

	secondFileReader, err = file_reader.NewFileReader(path+secondName, readBlockSize)
	if err != nil {
		return err
	}
	defer secondFileReader.Close()

	recordWg := &sync.WaitGroup{}
	errCh := make(chan error, 1)
	symbolsFromFirstFile, symbolsFromSecondFile := make(chan byte), make(chan byte)

	// Start recording processes

	recordWg.Add(1)
	//go ByteRecordingToFile(firstFileReader.GetFile(), symbolsFromSecondFile, errCh, recordWg)
	go ByteRecordingToFileBuffered(firstFileReader.GetFile(), symbolsFromSecondFile, writeBlockSize, errCh, recordWg)

	recordWg.Add(1)
	//go ByteRecordingToFile(secondFileReader.GetFile(), symbolsFromFirstFile, errCh, recordWg)
	go ByteRecordingToFileBuffered(secondFileReader.GetFile(), symbolsFromFirstFile, writeBlockSize, errCh, recordWg)

	var firstL, secondL int
	var firstText, secondText []byte

runtimeError:
	for !firstFileReader.EOF() || !secondFileReader.EOF() {
		if !firstFileReader.EOF() {
			firstL, firstText, err = firstFileReader.ReadBytes()
			if firstFileReader.EOF() {
				close(symbolsFromFirstFile)
			}
		}

		if !secondFileReader.EOF() {
			secondL, secondText, err = secondFileReader.ReadBytes()
			if secondFileReader.EOF() {
				close(symbolsFromSecondFile)
			}
		}

		sendBytesWg := &sync.WaitGroup{}

		if !firstFileReader.EOF() {
			sendBytesWg.Add(1)
			go func(err *error) {
				for i := 0; i < firstL; i++ {
					select {
					case *err = <-errCh:
						errCh <- *err
						sendBytesWg.Done()
						return
					default:
						symbolsFromFirstFile <- firstText[i]
					}
				}
				sendBytesWg.Done()
			}(&err)
		}

		if !secondFileReader.EOF() {
			for i := 0; i < secondL; i++ {
				select {
				case err = <-errCh:
					errCh <- err
					break runtimeError
				default:
					symbolsFromSecondFile <- secondText[i]
				}
			}
		}
		sendBytesWg.Wait()

		// Getting an error if it exists
		select {
		case err = <-errCh:
			break runtimeError
		default:
		}
	}

	recordWg.Wait()

	// Truncate the remaining part
	_ = os.Truncate(firstFileReader.GetFile().Name(), secondFileReader.Size())
	_ = os.Truncate(secondFileReader.GetFile().Name(), firstFileReader.Size())

	// Getting an error if it exists
	select {
	case err = <-errCh:
		if err != nil {
			return err
		}
	default:
	}
	return nil
}

//// If we accept extreme conditions, including negative numbers in the name
//bothNegativeWithMin := fileName[0] == '-' && minName[0] == '-'
//bothNegativeWithMax := fileName[0] == '-' && maxName[0] == '-'
//
//if bothNegativeWithMin && len(fileName) >= len(minName) && fileName > minName {
//	minName = fileName
//
//} else if !bothNegativeWithMin &&
//	(len(fileName) < len(minName) || (len(fileName) == len(minName) && fileName < minName)) {
//
//	if fileName[0] == '-' {
//		minName = fileName
//	}
//}
//
//if bothNegativeWithMax && len(fileName) <= len(maxName) && fileName < maxName {
//	maxName = fileName
//} else if !bothNegativeWithMax &&
//	(len(fileName) > len(maxName) || (len(fileName) == len(maxName) && fileName > maxName)) {
//
//	if maxName[0] == '-' {
//		maxName = fileName
//	}
//}

//var firstText, secondText []byte
//var firstL, secondL int // Number of characters read
//go func() {
//errExit:
//	for !firstFileReader.EOF() {
//		firstL, firstText, err = firstFileReader.ReadBytes()
//
//		for i := 0; i < firstL; i++ {
//			select {
//			case err = <-errCh:
//				errCh <- err
//				close(symbolsFromFirstFile)
//				readWg.Done()
//				break errExit
//			default:
//				symbolsFromFirstFile <- firstText[i]
//			}
//		}
//
//		select {
//		case _ = <-syncStepDone:
//			continue
//		default:
//			syncStepDone <- struct{}{}
//		}
//	}
//	close(symbolsFromFirstFile)
//	readWg.Done()
//}()
//
//readWg.Add(1)
//go func() {
//errExit:
//	for !secondFileReader.EOF() {
//		if !secondFileReader.EOF() {
//			secondL, secondText, err = secondFileReader.ReadBytes()
//		}
//
//		for i := 0; i < secondL; i++ {
//			select {
//			case err = <-errCh:
//				errCh <- err
//				close(symbolsFromSecondFile)
//				readWg.Done()
//				break errExit
//			default:
//				symbolsFromSecondFile <- secondText[i]
//			}
//		}
//
//		select {
//		case _ = <-syncStepDone:
//			continue
//		default:
//			syncStepDone <- struct{}{}
//		}
//	}
//	close(symbolsFromSecondFile)
//	readWg.Done()
//}()
//
//readWg.Wait()

//runtimeError:
//	for !firstFileReader.EOF() || !secondFileReader.EOF() {
//		if !firstFileReader.EOF() {
//			firstL, firstText, err = firstFileReader.ReadBytes()
//		}
//
//		if !secondFileReader.EOF() {
//			secondL, secondText, err = secondFileReader.ReadBytes()
//		}
//
//		lWg := &sync.WaitGroup{}
//
//		if !firstFileReader.EOF() {
//			lWg.Add(1)
//			go func(err *error) {
//				for i := 0; i < firstL; i++ {
//					select {
//					case *err = <-errCh:
//						errCh <- *err
//						lWg.Done()
//						return
//					default:
//						symbolsFromFirstFile <- firstText[i]
//					}
//				}
//				lWg.Done()
//			}(&err)
//			if err != nil {
//				break runtimeError
//			}
//		}
//
//		if !secondFileReader.EOF() {
//			for i := 0; i < secondL; i++ {
//				select {
//				case err = <-errCh:
//					errCh <- err
//					break runtimeError
//				default:
//					symbolsFromSecondFile <- secondText[i]
//				}
//			}
//		}
//		lWg.Wait()
//	}
//
//	// After closing the channels, the files are truncated
//	close(symbolsFromFirstFile)
//	close(symbolsFromSecondFile)
//
//	// Waiting for the end of the recording
//	recordWg.Wait()

//func ParallelSwapping(reader *file_reader.FileReader, dstChan chan<- byte, errCh chan error, syncStepDone chan struct{}, wg *sync.WaitGroup) {
//	// If another goroutine is working, it needs to synchronize them
//	// If another goroutine has terminated, then synchronization is no longer required.
//	runSync := true
//
//	if reader.Name() == "TestSwapTwoFiles1.log" {
//		reader.SetLabel("-1")
//	} else {
//		reader.SetLabel("--2")
//	}
//
//errExit:
//	for !reader.EOF() {
//		// Catching error from another goroutine
//		if runSync {
//			select {
//			case err := <-errCh:
//				errCh <- err
//				break errExit
//			default:
//			}
//		}
//
//		blockLen, blockText, err := reader.ReadBytes()
//		if err != nil && !errors.Is(err, io.EOF) {
//			errCh <- err
//			break errExit
//		} else if err != nil {
//			break errExit
//		}
//
//		for i := 0; i < blockLen; i++ {
//			dstChan <- blockText[i]
//		}
//
//		// If the second goroutine has completed the work
//		if runSync {
//			select {
//
//			// Catching error from another goroutine
//			case err = <-errCh:
//				errCh <- err
//				break errExit
//
//			// If another goroutine is already waiting
//			case _, ok := <-syncStepDone:
//				fmt.Println(reader.Label(), ": Done step. Another wait, process")
//				if !ok {
//					fmt.Println(reader.Label(), ": Disable sync")
//					runSync = false
//				}
//			// Send signal and waiting another goroutine
//			default:
//				fmt.Println(reader.Label(), ": Done step. Sync chan empty. Waiting")
//				syncStepDone <- struct{}{}
//				fmt.Println(reader.Label(), ": Done step. Process")
//			}
//		}
//	}
//
//	// Disabling synchronization for another subroutine. Executed once!
//	select {
//	// If opened
//	case _, ok := <-syncStepDone:
//		fmt.Println(time.Now(), reader.Label(), ": Exit loop. Channel sync ok: [", ok, "]. Not closed")
//	default:
//		fmt.Println(time.Now(), reader.Label(), ": Exit loop. Closing sync chanel")
//		close(syncStepDone)
//	}
//
//	fmt.Println(time.Now(), reader.Label(), ": Exit loop. Closing dst chanel")
//	close(dstChan)
//	wg.Done()
//}
