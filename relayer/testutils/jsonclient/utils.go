package jsonclient

import (
	"io"
	"os"
)

// readTestdata reads json responses from a subdirectory
func readTestdata(filename string) ([]byte, error) {
	pathtofile := "./testdata/" + filename
	fd, err := os.Open(pathtofile)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(fd)
	return data, err

}
