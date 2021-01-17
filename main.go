package main

import (
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
	"io/ioutil"
	"math"
	//"log"
	"os"
)

import (
	"bufio"
	"encoding/base64"
)
import "github.com/icza/mjpeg"

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
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

	//converttoQR()

	video, err := gocv.OpenVideoCapture("outVideo/opencvout.mp4")
	checkErr(err)
	defer video.Close()
	img := gocv.NewMat()
	defer img.Close()
	i := 0
	fmt.Printf("Start reading device: %v\n", video)

	file, err := os.Create("test/outfile.jpg")
	checkErr(err)

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
		_, err = file.Write(data)
		checkErr(err)
		i++
		fmt.Println(i)
	}

	file.Close()
}

func converttoQR() {
	videowriter, err := gocv.VideoWriterFile("outVideo/opencvout.mp4", "H264", 24, WIDTH, HEIGHT, true)
	checkErr(err)

	outputLOG := make(chan string, 10000)
	go func() {
		for {
			fmt.Print(<-outputLOG)
		}
	}()

	fileChunks := make(chan []byte, 1000)

	count := readCunks("test/test.jpg", 500, fileChunks)

	base64Chunks := make(chan string, 1000)
	go func() {
		defer close(base64Chunks)
		for chunk := range fileChunks {
			base64Chunks <- encodeBase64(chunk)
		}
	}()

	images := make(chan image.Image, 1000)

	go func() {
		defer close(images)
		for chunk := range base64Chunks {
			images <- encodeQr(chunk)
		}
	}()

	done := make(chan int)
	go func() {
		i := 0
		for image := range images {
			test, err := recognizeQR(image)
			checkErr(err)
			fmt.Println(test)
			img, err := gocv.ImageToMatRGB(image)
			checkErr(err)
			err = videowriter.Write(img)
			checkErr(err)
			fmt.Printf("written %d, tot: %d --- ", i, count)
			fmt.Printf("fileChunks: %d, base64Chunks: %d, images %d\n", len(fileChunks), len(base64Chunks), len(images))
			i++
		}
		videowriter.Close()
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

func encodeVideo(tot int) {
	// Video size: 1080x1080 pixels, FPS: 24
	aw, err := mjpeg.New("outVideo/out.avi", 1080, 1080, 24)
	checkErr(err)

	// Create a movie from images: 1.jpg, 2.jpg, ..., 10.jpg
	for i := 0; i < tot; i++ {
		fmt.Printf("Converting video: %d\n", i)
		data, err := ioutil.ReadFile(fmt.Sprintf("outQR/%010d.jpg", i))
		checkErr(err)
		checkErr(aw.AddFrame(data))
	}

	checkErr(aw.Close())
}

// https://stackoverflow.com/a/61469854/8328237
//func Chunks(s string, chunkSize int) []string {
//	if chunkSize >= len(s) {
//		return []string{s}
//	}
//	var chunks []string
//	chunk := make([]rune, chunkSize)
//	len := 0
//	for _, r := range s {
//		chunk[len] = r
//		len++
//		if len == chunkSize {
//			chunks = append(chunks, string(chunk))
//			len = 0
//		}
//	}
//	if len > 0 {
//		chunks = append(chunks, string(chunk[:len]))
//	}
//	return chunks
//}

//https://zetcode.com/golang/readfile/
func readCunks(filename string, chunkSize int, chunkChan chan []byte) (count int) {
	f, err := os.Open(filename)
	checkErr(err)

	fi, err := f.Stat()
	checkErr(err)

	filesize := fi.Size()

	d := float64(filesize) / float64(chunkSize)

	count = int(math.Ceil(d))

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
