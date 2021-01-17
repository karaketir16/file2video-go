package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/liyue201/goqr"
	"gocv.io/x/gocv"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"
	//"log"
	"os"
)

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"github.com/cheggaaa/pb/v3"
	"golang.org/x/crypto/blake2b"
	"path/filepath"
)

type MetaData struct {
	Filename         string
	ChunkCount       int
	Filehash         string
	ConverterUrl     string
	ConverterVersion string
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func hashFile(filename string) (hashstr string) {
	hasher, _ := blake2b.New256(nil)

	path := filename
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	if _, err := io.Copy(hasher, f); err != nil {
		log.Fatal(err)
	}

	hash := hasher.Sum(nil)
	hashstr = hex.EncodeToString(hash[:])
	return
}

func recognizeQR(img image.Image) (string, error) {
	qrCodes, err := goqr.Recognize(img)
	if err != nil {
		return "", err
	}

	if len(qrCodes) != 1 {
		return "", errors.New("more then 1")
	}

	return string(qrCodes[0].Payload), nil
}

const WIDTH = 1080
const HEIGHT = 1080

func main() {

	source := flag.String("src", "", "-src source.file")
	destination := flag.String("dst", "", "-dst destination file for F2V, destination folder for V2F")

	convert := flag.Bool("F2V", false, "-F2V for convert file to video")
	reconvert := flag.Bool("V2F", false, "-V2F for convert video to file")

	flag.Parse()

	if *convert == *reconvert || *source == "" || *destination == "" {
		flag.Usage()
		os.Exit(2)
	}
	if *convert {
		convert2Video(*source, *destination)
	} else if *reconvert {
		convert2File(*source, *destination)
	}
}

func convert2File(source, destination string) {
	video, err := gocv.OpenVideoCapture(source)
	checkErr(err)
	defer video.Close()
	img := gocv.NewMat()
	defer img.Close()

	fmt.Printf("Start reading device: %v\n", video)

	checkErr(err)

	firstFrame := true
	var metadata MetaData
	var bar *pb.ProgressBar
	var file *os.File

	for {

		if ok := video.Read(&img); !ok {
			fmt.Printf("Device closed: %v\n", video)
			return
		}
		if img.Empty() {
			continue
		}

		img, err := img.ToImage()
		checkErr(err)
		res, err := recognizeQR(img)
		checkErr(err)
		data, err := base64.StdEncoding.DecodeString(res)
		checkErr(err)

		if firstFrame {
			json.Unmarshal(data, &metadata)
			firstFrame = false
			file, err = os.Create(filepath.Join(destination, metadata.Filename))
			checkErr(err)
			bar = pb.StartNew(metadata.ChunkCount)
		} else {
			_, err = file.Write(data)
			checkErr(err)
			bar.Increment()
		}
	}

	file.Close()

	receivedHash := hashFile(filepath.Join(destination, metadata.Filename))
	metadataHash := metadata.Filehash
	if receivedHash != metadataHash {
		checkErr(errors.New("expected: " + metadataHash + "\n received: " + receivedHash))
	}
	bar.Finish()
}

func convert2Video(source, destination string) {
	path := filepath.Dir(destination)
	stat, err := os.Stat(path)
	checkErr(err)
	if !stat.IsDir() {
		checkErr(errors.New("parent is not directory"))
	}

	videowriter, err := gocv.VideoWriterFile(destination, "H264", 24, WIDTH, HEIGHT, true)
	checkErr(err)

	fileChunks := make(chan []byte, 5)

	count := readCunks(source, 500, fileChunks)

	bar := pb.StartNew(count + 1)

	base64Chunks := make(chan string, 5)
	go func() {
		defer close(base64Chunks)
		for chunk := range fileChunks {
			base64Chunks <- encodeBase64(chunk)
		}
	}()

	images := make(chan image.Image, 5)

	go func() {
		defer close(images)
		for chunk := range base64Chunks {
			images <- encodeQr(chunk)
		}
	}()

	done := make(chan int)
	go func() {
		for image := range images {
			//test, err := recognizeQR(image)
			//checkErr(err)
			//fmt.Println(test)
			img, err := gocv.ImageToMatRGB(image)
			checkErr(err)
			err = videowriter.Write(img)
			checkErr(err)
			bar.Increment()
		}
		videowriter.Close()
		bar.Finish()
		done <- 1
	}()

	<-done
}

func encodeBase64(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded
}

func encodeQr(data string) image.Image {
	// Create the barcode
	qrCode, err := qr.Encode(data, qr.M, qr.Auto)
	checkErr(err)
	// Scale the barcode to 200x200 pixels
	qrCode, err = barcode.Scale(qrCode, WIDTH, HEIGHT)
	checkErr(err)

	return qrCode
}

//https://zetcode.com/golang/readfile/
func readCunks(filename string, chunkSize int, chunkChan chan []byte) (count int) {

	hash := hashFile(filename)

	f, err := os.Open(filename)
	checkErr(err)

	fi, err := f.Stat()
	checkErr(err)

	filesize := fi.Size()

	d := float64(filesize) / float64(chunkSize)

	count = int(math.Ceil(d))

	m := MetaData{
		Filename:         filepath.Base(filename),
		ChunkCount:       count,
		Filehash:         hash,
		ConverterUrl:     "https://github.com/karaketir16/go-File2Video",
		ConverterVersion: "V0.1",
	}

	metadata, err := json.Marshal(m)
	checkErr(err)

	chunkChan <- metadata

	go func() {

		defer f.Close()
		defer close(chunkChan)

		reader := bufio.NewReader(f)

		checker := false
		var err error
		n := 0
		a := 0
		var buf []byte
		for {
			if checker {
				a, err = reader.Read(buf[n:])
				n = n + a
			} else {
				buf = make([]byte, chunkSize)
				n, err = reader.Read(buf)
			}

			if err != nil {
				if err != io.EOF {
					checkErr(err)
				}
				break
			}
			checker = true
			if n < chunkSize {
				continue
			}
			chunkChan <- buf[:n]
			checker = false
		}
		if checker {
			chunkChan <- buf[:n]
		}
	}()
	return
}
