package qrwork

import (
	"bytes"
	"fmt"

	go_qr "github.com/piglig/go-qr"
)

// Generate QR svg file and return string value of <svg> code
func GenerateTOTPSvgQrHTML(totpURL []byte) (string, error) {
	var (
		buf    bytes.Buffer
		result string
	)

	// Encode & Generate QR
	errCorLvl := go_qr.Low
	qr, err := go_qr.EncodeText(string(totpURL), errCorLvl)
	if err != nil {
		return result, err
	}
	config := go_qr.NewQrCodeImgConfig(10, 4)

	// write svg code to buffer
	err = qr.WriteAsSVG(config, &buf, "#FFFFFF", "#000000")
	if err != nil {
		return result, err
	}

	// read from buffer
	data := make([]byte, buf.Len())
	buf.Read(data)

	// set result and check if empty
	result = string(data)
	if len(result) == 0 {
		return result, fmt.Errorf("empty result of generating svg")
	}

	return result, nil
}
