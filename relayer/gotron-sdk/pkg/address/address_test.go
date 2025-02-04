package address

import (
	"bytes"
	"testing"
)

func TestAddress_Scan(t *testing.T) {
	validAddress, err := Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// correct case
	want := validAddress
	a := &Address{}
	src := validAddress.Bytes()
	err = a.Scan(src)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
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
	_, err := StringToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// valid hex
	_, err = StringToAddress("41d4f4f0b3b3d4e3b3b3b3b3b3b3b3b3b3b3b3b3")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// valid evm address
	_, err = StringToAddress("0x41d4f4f0b3b3d4e3b3b3b3b3b3b3b3b3b3b3b3")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// invalid address
	_, err = StringToAddress("invalid address")
	if err == nil {
		t.Errorf("expected an error, but got none")
	}
}
