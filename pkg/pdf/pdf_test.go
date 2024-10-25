package pdf

import (
	"io"
	"os"
	"testing"
)

func TestReadPdf(t *testing.T) {

	file, err := os.Open("sample.pdf")
	if err != nil {

		t.Fatalf("Could not open sample pdf file")

	}

	stat, err := file.Stat()
	if err != nil {

		t.Fatalf("Could not stat sample pdf file")

	}

	reader, err := ReadPdfDSLIPAK(file, stat.Size())
	if err != nil {

		t.Fatalf("Could not read pdf file")

	}

	tee := io.TeeReader(reader, os.Stdout)

	buffer := make([]byte, 1024)
	for {

		_, err := tee.Read(buffer)
		if err == io.EOF {

			break

		}

	}

}
