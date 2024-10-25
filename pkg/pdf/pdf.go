package pdf

import (
	"io"

	"github.com/dslipak/pdf"
)

func ReadPdfDSLIPAK(r io.ReaderAt, size int64) (io.Reader, error) {
	pdfReader, err := pdf.NewReader(r, size)
	if err != nil {
		return nil, err
	}

	return pdfReader.GetPlainText()
}
