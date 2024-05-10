package main

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/nfnt/resize"
)

func decodeFromImage(img image.Image, gridSize int) []byte {
	// Resize the image to the required gridSize using the resize library
	smallImg := resize.Resize(uint(gridSize), uint(gridSize), img, resize.NearestNeighbor)

	// Convert the resized image to grayscale and decode the binary data
	grayImg := image.NewGray(image.Rect(0, 0, gridSize, gridSize))
	data := make([]byte, (gridSize*gridSize)/8)
	bitIndex := 0
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			px := smallImg.At(x, y)
			r, g, b, _ := px.RGBA()
			lum := uint8((r + g + b) / 3 >> 8) // convert to 0-255 range
			if lum > 128 {
				grayImg.SetGray(x, y, color.Gray{255})
				setBit(data, bitIndex)
			}
			bitIndex++
		}
	}
	return data
}

func createCustomCode(data []byte, gridSize int) image.Image {
	grid := image.NewGray(image.Rect(0, 0, gridSize, gridSize))
	bitLength := len(data) * 8
	bitIndex := 0
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			if bitIndex >= bitLength {
				return grid
			}
			if getBit(data, bitIndex) == 1 {
				grid.SetGray(x, y, color.Gray{255})
			}
			bitIndex++
		}
	}
	return grid
}

func encodeToImage(data []byte, gridSize, resolution int) image.Image {
	grid := createCustomCode(data, gridSize)
	// Use resize library to scale the grayscale image
	scaled := resize.Resize(uint(resolution), uint(resolution), grid, resize.NearestNeighbor)
	return scaled
}

func MyExample() {
	testString := "Hello, World!"
	data := []byte(testString)

	println("Data len: ", len(data))

	// Encode to image
	img := encodeToImage(data, 20, 100) // Use a smaller grid and lower resolution for visualization

	// Save image to file
	f, _ := os.Create("output.png")
	defer f.Close()
	png.Encode(f, img)

	// Decode from image
	decodedData := decodeFromImage(img, 20)
	decodedString := string(decodedData)

	// Display results
	println("Decoded String:", decodedString)
}

func getBit(data []byte, bitIndex int) int {
	byteIndex := bitIndex / 8
	bitPosition := bitIndex % 8
	return int((data[byteIndex] >> bitPosition) & 1)
}

func setBit(data []byte, bitIndex int) {
	byteIndex := bitIndex / 8
	bitPosition := bitIndex % 8
	data[byteIndex] |= (1 << bitPosition)
}
