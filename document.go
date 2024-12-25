package main

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
)

type Document struct {
	pdf fpdf.Fpdf
}

func NewDocument() *Document {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.SetFillColor(200, 200, 220)

	return &Document{
		pdf: *pdf,
	}
}

func (d *Document) addText(data string, x float64, y float64) {
	d.pdf.Text(x, y, data)
}

func (d *Document) addCircle(number int, x float64, y float64) {
	// draw text (number) in center of the circle
	centerX := x + 20
	centerY := y + 20
	d.pdf.Text(centerX, centerY, fmt.Sprintf("%d", number))

	// draw circle
	d.pdf.SetDrawColor(0, 0, 0)
	d.pdf.SetLineWidth(0.5)
	d.pdf.Circle(centerX, centerY, 20, "D")
}

func (d *Document) addImage(image []byte, x float64, y float64) error {
	width := 40.0
	height := 40.0

	opt := fpdf.ImageOptions{
		ImageType: "PNG",
		ReadDpi:   true,
	}
	info := d.pdf.RegisterImageOptionsReader("image.png", opt, bytes.NewReader(image))
	if info == nil {
		return d.pdf.Error()
	}
	d.pdf.Image("image.png", x, y, width, height, false, "", 0, "")

	// Draw circle around the image
	centerX := x + width/2
	centerY := y + height/2
	radius := width/2 + 5
	d.pdf.Circle(centerX, centerY, radius, "D")

	return nil
}

func (d *Document) bytes() ([]byte, error) {
	var buf bytes.Buffer
	err := d.pdf.Output(&buf)
	if err != nil {
		panic(err)
	}

	// Get the byte array
	return buf.Bytes(), nil
	// var b bytes.Buffer
	// w := bufio.NewWriter(&b)
	// err := d.pdf.Output(w)
	// if err != nil {
	// 	return nil, err
	// }

	// return b.Bytes(), nil
}

func (d *Document) save(path string) error {
	return d.pdf.OutputFileAndClose(path)
}
