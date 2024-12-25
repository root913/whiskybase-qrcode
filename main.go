package main

import (
	"fmt"
	"log"

	"syscall/js"

	"github.com/skip2/go-qrcode"
)

func generateQRCode(data string) []byte {
	qrCode, err := qrcode.Encode(data, qrcode.Medium, 256)
	if err != nil {
		log.Fatalln(err)
	}

	return qrCode
}

func generatePdf(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.ValueOf("Error: No argument provided")
	}

	// Check if the first argument is an array
	jsCodes := args[0]
	if jsCodes.Type() != js.TypeObject || !jsCodes.InstanceOf(js.Global().Get("Array")) {
		return js.ValueOf("Error: Argument must be an array of strings")
	}

	// Convert JavaScript array to Go slice
	codes := make([]string, jsCodes.Length())
	for i := 0; i < jsCodes.Length(); i++ {
		codes[i] = jsCodes.Index(i).String()
	}

	document := NewDocument()

	x := 20.0
	y := 30.0
	for i, code := range codes {
		url := fmt.Sprintf("https://www.whiskybase.com/whiskies/whisky/%s", code)
		document.addCircle(i, x+float64(i*50), y+float64((i/3)*50))

		document.addText(fmt.Sprintf("WB%s", code), x+float64(i*50), 210-60)
		document.addImage(generateQRCode(url), x+float64(i*50), 210-50)
	}

	outputBytes, err := document.bytes()
	if err != nil {
		log.Fatalln(err)
		fmt.Println("Error generating PDF")
	}
	fmt.Printf("Generated PDF with %d bytes\n", len(outputBytes))
	size := len(outputBytes)
	result := js.Global().Get("Uint8Array").New(size)
	js.CopyBytesToJS(result, outputBytes)

	return result
}

func main() {
	js.Global().Set("generatePdf", js.FuncOf(generatePdf))

	<-make(chan bool)
}
