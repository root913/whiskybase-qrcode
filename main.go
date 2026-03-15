package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"strings"

	"syscall/js"

	"github.com/root913/whiskybase-qrcode/internal/qrcode"
	goqrcode "github.com/skip2/go-qrcode"
)

func hexToColor(h string) color.RGBA {
	h = strings.TrimPrefix(h, "#")
	if len(h) != 6 {
		return color.RGBA{0, 0, 0, 255}
	}
	var r, g, b uint8
	fmt.Sscanf(h[0:2], "%02x", &r)
	fmt.Sscanf(h[2:4], "%02x", &g)
	fmt.Sscanf(h[4:6], "%02x", &b)
	return color.RGBA{r, g, b, 255}
}

func generateQRCode(data string) []byte {
	qrCode, err := goqrcode.Encode(data, goqrcode.Medium, 256)
	if err != nil {
		log.Fatalln(err)
	}
	return qrCode
}

func generatePdf(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.ValueOf("Error: No argument provided")
	}

	jsCodes := args[0]
	if jsCodes.Type() != js.TypeObject || !jsCodes.InstanceOf(js.Global().Get("Array")) {
		return js.ValueOf("Error: Argument must be an array of strings")
	}

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
	}
	fmt.Printf("Generated PDF with %d bytes\n", len(outputBytes))

	size := len(outputBytes)
	result := js.Global().Get("Uint8Array").New(size)
	js.CopyBytesToJS(result, outputBytes)
	return result
}

func generateQrWithLogo(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.ValueOf("Error: no argument provided")
	}

	jsCodes := args[0]
	if jsCodes.Type() != js.TypeObject || !jsCodes.InstanceOf(js.Global().Get("Array")) {
		return js.ValueOf("Error: argument must be an array of strings")
	}

	codes := make([]string, jsCodes.Length())
	for i := 0; i < jsCodes.Length(); i++ {
		codes[i] = jsCodes.Index(i).String()
	}

	opts := qrcode.DefaultOptions()

	// outputFmt is the requested output format: "png", "svg", or "both"
	outputFmt := "png"

	if len(args) >= 2 {
		o := args[1]

		if v := o.Get("fmt"); !v.IsUndefined() {
			outputFmt = v.String()
		}
		if v := o.Get("size"); !v.IsUndefined() {
			opts.Size = v.Int()
		}
		if v := o.Get("logoSize"); !v.IsUndefined() {
			opts.LogoSize = v.Int()
		}
		if v := o.Get("logoRadius"); !v.IsUndefined() {
			opts.LogoRadius = float64(v.Int())
		}
		if v := o.Get("quietZone"); !v.IsUndefined() {
			opts.QuietZone = v.Bool()
		}
		if v := o.Get("logoPad"); !v.IsUndefined() {
			opts.LogoPad = v.Bool()
		}
		if v := o.Get("modulePattern"); !v.IsUndefined() {
			opts.ModulePattern = v.String()
		}
		if v := o.Get("finderPattern"); !v.IsUndefined() {
			opts.FinderPattern = v.String()
		}
		if v := o.Get("colorFg"); !v.IsUndefined() {
			opts.ColorFg = hexToColor(v.String())
		}
		if v := o.Get("colorBg"); !v.IsUndefined() {
			opts.ColorBg = hexToColor(v.String())
		}
		if v := o.Get("ec"); !v.IsUndefined() {
			switch v.String() {
			case "H":
				opts.EC = goqrcode.High
			case "Q":
				opts.EC = goqrcode.Medium
			case "M":
				opts.EC = goqrcode.Low
			}
		}
		if v := o.Get("logoBase64"); !v.IsUndefined() && v.String() != "" {
			logoB64 := v.String()
			if comma := strings.Index(logoB64, ","); comma >= 0 {
				logoB64 = logoB64[comma+1:]
			}
			logoBytes, err := base64.StdEncoding.DecodeString(logoB64)
			if err == nil {
				img, _, err := image.Decode(bytes.NewReader(logoBytes))
				if err == nil {
					opts.Logo = img
					opts.EC = goqrcode.High
				}
			}
		}
	}

	fmt.Printf("outputFmt: %s, codes: %v\n", outputFmt, codes)
	jsArray := js.Global().Get("Array").New(len(codes))

	for i, code := range codes {
		obj := js.Global().Get("Object").New()
		obj.Set("code", code)
		obj.Set("png", "")
		obj.Set("svg", "")
		obj.Set("error", "")

		var lastErr string

		if outputFmt == "png" || outputFmt == "both" {
			img, err := qrcode.GenerateQRWithLogo(code, opts)
			if err != nil {
				lastErr = err.Error()
			} else {
				var buf bytes.Buffer
				if err := png.Encode(&buf, img); err != nil {
					lastErr = err.Error()
				} else {
					obj.Set("png", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(buf.Bytes()))
				}
			}
		}

		if outputFmt == "svg" || outputFmt == "both" {
			fmt.Printf("Generating SVG for code: %s, outputFmt: %s\n", code, outputFmt)
			svgStr, err := qrcode.GenerateQRSVG(code, opts)
			if err != nil {
				fmt.Printf("SVG error: %v\n", err)
				lastErr = err.Error()
			} else {
				fmt.Printf("SVG generated, length: %d\n", len(svgStr))
				obj.Set("svg", svgStr)
			}
		}

		if lastErr != "" {
			obj.Set("error", lastErr)
		}

		jsArray.SetIndex(i, obj)
	}

	return jsArray
}

func main() {
	js.Global().Set("generatePdf", js.FuncOf(generatePdf))
	js.Global().Set("generateQrWithLogo", js.FuncOf(generateQrWithLogo))

	<-make(chan bool)
}
