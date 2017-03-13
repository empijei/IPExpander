package parsers

import (
	"fmt"
	"testing"
)

func comparer(actual [4][2]byte, expected []byte) bool {
	for i, val := range expected {
		if val != actual[i/2][i%2] {
			return false
		}
	}
	return true
}

func arrayToString(in [4][2]byte) (out string) {
	for i := 0; i < 8; i++ {
		out += fmt.Sprintf("%d ", in[i/2][i%2])
	}
	return
}
func sliceToString(in []byte) (out string) {
	for _, val := range in {
		out += fmt.Sprintf("%d ", val)
	}
	return
}

var testIPS = []struct {
	input  string
	evalue []byte
	eerr   bool
}{
	{
		"10.0.0.1-2",
		[]byte{10, 10, 0, 0, 0, 0, 1, 2},
		false,
	},
	{
		"10.0.1-2.1",
		[]byte{10, 10, 0, 0, 1, 2, 1, 1},
		false,
	},
	{
		"10-11.0.1-2.1",
		[]byte{10, 11, 0, 0, 1, 2, 1, 1},
		false,
	},
	{
		"-1.0.-.1",
		[]byte{0, 1, 0, 0, 0, 255, 1, 1},
		false,
	},
	{
		"10.0.1-2.254-",
		[]byte{10, 10, 0, 0, 1, 2, 254, 255},
		false,
	},
	{
		"10.0.1-2.255-",
		[]byte{10, 10, 0, 0, 1, 2, 255, 255},
		false,
	},
	{
		"10.0.1",
		[]byte{},
		true,
	},
	{
		"10.0.1-2.255.30",
		[]byte{},
		true,
	},
	{
		"10.0.1-2.255-f",
		[]byte{},
		true,
	},
	{
		"this is not an IP",
		[]byte{},
		true,
	},
	{
		"--.0.0.1",
		[]byte{},
		true,
	},
	{
		"-...1",
		[]byte{},
		true,
	},
	{
		"10.0.0.270",
		[]byte{},
		true,
	},
	{
		"10.0.0.27000000000000000000000000000000000000000",
		[]byte{},
		true,
	},
	{
		"10.0.0.0.",
		[]byte{},
		true,
	},
	{
		"10-1f.0.0.0.",
		[]byte{},
		true,
	},
	{
		"1f.0.0.0.",
		[]byte{},
		true,
	},
}

func TestParse(t *testing.T) {
	for _, tc := range testIPS {
		input := Input{tc.input, 0, 0}
		actual, err := parse(input)
		if tc.eerr && err == nil {
			t.Errorf("Was expecting error but did not get it. (%s)", tc.input)
			continue
		}
		if err != nil && !tc.eerr {
			t.Errorf("Got error %s but did not expect it. (%s)", err, tc.input)
			continue
		}
		if !comparer(actual, tc.evalue) {
			t.Errorf("\nExpected %s \nObtained %s", sliceToString(tc.evalue), arrayToString(actual))
		}
	}
}
