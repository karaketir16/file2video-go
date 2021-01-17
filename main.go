package main

import (
	"bytes"
	"fmt"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/liyue201/goqr"
	"gocv.io/x/gocv"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
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

func recognizeFile(path string) {
	fmt.Printf("recognize file: %v\n", path)
	imgdata, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	img, _, err := image.Decode(bytes.NewReader(imgdata))
	if err != nil {
		fmt.Printf("image.Decode error: %v\n", err)
		return
	}
	qrCodes, err := goqr.Recognize(img)
	if err != nil {
		fmt.Printf("Recognize failed: %v\n", err)
		return
	}
	for _, qrCode := range qrCodes {
		fmt.Printf("qrCode text: %s\n", qrCode.Payload)
	}
	return
}

const COL = 1080
const ROW = 1080

var videowriter *gocv.VideoWriter

func main() {
	writer, err := gocv.VideoWriterFile("outVideo/opencvout.mp4", "H264", 24, COL, ROW, true)
	checkErr(err)
	videowriter = writer

	chunks := encodeBase64("test/test.mp4")
	tot := len(chunks)
	for i, chunk := range chunks {
		encodeQr(chunk, fmt.Sprintf("outQR/%010d.jpg", i))
		fmt.Printf("total: %d, now:%d\n\r", tot, i)
	}
	//encodeVideo(tot)
	videowriter.Close()
}

func encodeBase64(filename string) []string {
	// Open file on disk.
	f, _ := os.Open(filename)

	// Read entire JPG into byte slice.
	reader := bufio.NewReader(f)
	content, _ := ioutil.ReadAll(reader)

	// Encode as base64.
	encoded := base64.StdEncoding.EncodeToString(content)
	return Chunks(encoded, 1000)
}

func encodeQr(data, filename string) {
	// Create the barcode
	qrCode, err := qr.Encode(data, qr.M, qr.Auto)
	checkErr(err)
	// Scale the barcode to 200x200 pixels
	qrCode, err = barcode.Scale(qrCode, 1080, 1080)
	checkErr(err)

	// create the output file
	//fileJpeg, err := os.Create(filename)
	//checkErr(err)
	//defer fileJpeg.Close()
	//
	//// encode the barcode as png
	//
	//err = jpeg.Encode(fileJpeg, qrCode,nil)
	//checkErr(err)

	img, err := gocv.ImageToMatRGB(qrCode)
	checkErr(err)
	if img.Empty() {
		fmt.Printf("Error reading image from: %v\n", filename)
		return
	}

	videowriter.Write(img)

	//recognizeFile(filename)
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
func Chunks(s string, chunkSize int) []string {
	if chunkSize >= len(s) {
		return []string{s}
	}
	var chunks []string
	chunk := make([]rune, chunkSize)
	len := 0
	for _, r := range s {
		chunk[len] = r
		len++
		if len == chunkSize {
			chunks = append(chunks, string(chunk))
			len = 0
		}
	}
	if len > 0 {
		chunks = append(chunks, string(chunk[:len]))
	}
	return chunks
}
