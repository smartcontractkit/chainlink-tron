package address

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddress_Scan(t *testing.T) {
	validAddress, err := Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	if err != nil {
		t.Errorf("unexpected error: %w", err)
	}

	// correct case
	want := validAddress
	a := &Address{}
	src := validAddress.Bytes()
	err = a.Scan(src)
	if err != nil {
		t.Errorf("unexpected error: %w", err)
	}
	if !bytes.Equal(a.Bytes(), want.Bytes()) {
		t.Errorf("got %v, want %v", *a, want)
	}

	// invalid type of src
	a = &Address{}
	err = a.Scan("not a byte slice")
	if err == nil {
		t.Errorf("expected an error, but got none")
	}

	// invalid length of src
	a = &Address{}
	src = make([]byte, 4)
	err = a.Scan(src)
	if err == nil {
		t.Errorf("expected an error, but got none")
	}
	src = make([]byte, 22) // Створюємо байтовий масив з неправильною довжиною
	err = a.Scan(src)
	if err == nil {
		t.Errorf("expected an error, but got none")
	}
}

func TestStringToAddress(t *testing.T) {
	// valid base58
	a, err := StringToAddress("T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb")
	assert.NoError(t, err)
	assert.Equal(t, "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb", a.String())

	// valid hex
	a, err = StringToAddress("410000000000000000000000000000000000000000")
	assert.NoError(t, err)
	assert.Equal(t, "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb", a.String())

	// valid evm address
	a, err = StringToAddress("0x0000000000000000000000000000000000000000")
	assert.NoError(t, err)
	assert.Equal(t, "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb", a.String())

	// invalid address
	_, err = StringToAddress("0x41d4f4f0b3b3d4e3b3b3b3b3b3b3b3b3b3b3")
	assert.Error(t, err)
}
