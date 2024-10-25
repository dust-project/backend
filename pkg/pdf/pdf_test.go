package pdf

import (
	"bytes"
	"dust/pkg/ondemand"
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

	buf := &bytes.Buffer{}
	tee := io.TeeReader(reader, buf)

	buffer := make([]byte, 1024)
	for {

		_, err := tee.Read(buffer)
		if err == io.EOF {

			break

		}

	}

	ctn, err := io.ReadAll(buf)
	if err != nil {

		t.Error(err)

	}

	res, err := ondemand.OnDemand(string(ctn))
	if err != nil {

		t.Error(err)

	}

	t.Log(res)
}
