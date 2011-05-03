// Public Domain (-) 2010-2011 The Ampify Authors.
// See the Ampify UNLICENSE file for details.

package argo

import (
	"amp/big"
	"bytes"
	"io"
	"os"
	"strings"
)

const (
	magicNumber int64 = 8258175
)

var (
	bigintMagicNumber1, _ = big.NewIntString("8258175")
	bigintMagicNumber2, _ = big.NewIntString("8323072")
	bigint1, _            = big.NewIntString("1")
	bigint253, _          = big.NewIntString("253")
	bigint254, _          = big.NewIntString("254")
	bigint255, _          = big.NewIntString("255")
	zero                  = []byte{'\x01', '\x80', '\x01', '\x01'}
	zeroBase              = []byte{'\x80', '\x01', '\x01'}
)

type ArgoError string

func (err ArgoError) String() string {
	return string(err)
}

type Encoder struct {
	stream io.Writer
}

func (enc *Encoder) WriteSize(value uint64) os.Error {
	data := make([]byte, 0)
	for {
		leftBits := value & 127
		value >>= 7
		if value > 0 {
			leftBits += 128
		}
		data = append(data, byte(leftBits))
		if value == 0 {
			break
		}
	}
	_, err := enc.stream.Write(data)
	return err
}

func (enc *Encoder) WriteStringArray(value []string) (err os.Error) {
	err = enc.WriteSize(uint64(len(value)))
	if err != nil {
		return
	}
	for _, item := range value {
		err = enc.WriteSize(uint64(len(item)))
		if err != nil {
			return
		}
		_, err = enc.stream.Write([]byte(item))
		if err != nil {
			return
		}
	}
	return nil
}

type Decoder struct {
	stream io.Reader
}

func (dec *Decoder) ReadSize() (value uint64, err os.Error) {
	bitShift := uint(0)
	lowByte := uint64(1)
	data := make([]byte, 1)
	for lowByte > 0 {
		n, err := dec.stream.Read(data)
		if n != 1 {
			if err == nil {
				return value, ArgoError("Couldn't read from the data stream.")
			}
			return
		}
		byteValue := uint64(data[0])
		lowByte = byteValue & 128
		value += (byteValue & 127) << bitShift
		bitShift += 7
	}
	return value, nil
}

func (dec *Decoder) ReadStringArray() (value []string, err os.Error) {
	arraySize, err := dec.ReadSize()
	if err != nil {
		return
	}
	var i uint64
	for i < arraySize {
		stringSize, err := dec.ReadSize()
		if err != nil {
			return
		}
		item := make([]byte, stringSize)
		n, err := dec.stream.Read(item)
		if uint64(n) != stringSize {
			if err == nil {
				return value, ArgoError("Couldn't read from the data stream.")
			}
			return
		}
		value = append(value, string(item))
		i++
	}
	return value, nil
}

func pow(x, y int64) (z int64) {
	var i int64
	z = 1
	for i = 0; i < y; i++ {
		z = z * x
	}
	return z
}

func WriteInt(value int64, buffer *bytes.Buffer) {
	if value == 0 {
		buffer.Write(zero)
		return
	}
	if value > 0 {
		if value < magicNumber {
			encoding := []byte{'\x01', '\x80', '\x01', '\x01'}
			div, mod := value/255, value%255
			encoding[3] = byte(mod) + 1
			if div > 0 {
				div, mod = div/255, div%255
				encoding[2] = byte(mod) + 1
				if div > 0 {
					encoding[1] = byte(div) + 128
				}
			}
			buffer.Write(encoding)
		} else {
			value -= magicNumber
			buffer.WriteByte('\x01')
			buffer.WriteByte('\xff')
			lead, left := value/255, value%255
			var n int64 = 1
			for (lead / pow(253, n)) > 0 {
				n += 1
			}
			buffer.WriteByte(byte(n) + 1)
			buffer.WriteByte('\xff')
			leadChars := make([]byte, 0)
			for {
				var mod int64
				if lead == 0 {
					break
				}
				lead, mod = lead/253, lead%253
				leadChars = append(leadChars, byte(mod)+2)
			}
			lenLead := len(leadChars)
			if lenLead > 0 {
				for i := lenLead - 1; i >= 0; i-- {
					buffer.WriteByte(leadChars[i])
				}
			}
			if left > 0 {
				buffer.WriteByte('\x01')
				buffer.WriteByte(byte(left))
			}
		}
	} else {
		value = -value
		if value < magicNumber {
			encoding := []byte{'\x01', '\x7f', '\xfe', '\xfe'}
			div, mod := value/255, value%255
			encoding[3] = 254 - byte(mod)
			if div > 0 {
				div, mod = div/255, div%255
				encoding[2] = 254 - byte(mod)
				if div > 0 {
					encoding[1] = 127 - byte(div)
				}
			}
			buffer.Write(encoding)
		} else {
			value -= magicNumber
			buffer.WriteByte('\x01')
			buffer.WriteByte('\x00')
			lead, left := value/254, value%254
			var n int64 = 1
			for (lead / pow(253, n)) > 0 {
				n += 1
			}
			buffer.WriteByte(254 - byte(n))
			buffer.WriteByte('\x00')
			leadChars := make([]byte, 0)
			for {
				var mod int64
				if lead == 0 {
					break
				}
				lead, mod = lead/253, lead%253
				leadChars = append(leadChars, 253-byte(mod))
			}
			lenLead := len(leadChars)
			if lenLead > 0 {
				for i := lenLead - 1; i >= 0; i-- {
					buffer.WriteByte(leadChars[i])
				}
			}
			if lenLead > 1 {
				buffer.WriteByte('\x00')
			}
			buffer.WriteByte('\xfe')
			if left > 0 {
				buffer.WriteByte(254 - byte(left))
			} else {
				buffer.WriteByte('\xfe')
			}
		}
	}
}

func WriteNumber(value string, buffer *bytes.Buffer) os.Error {
	if strings.Count(value, ".") > 0 {
		number, ok := big.NewDecimal(value)
		if !ok {
			return ArgoError("Couldn't create a Decimal representation of " + value)
		}
		WriteDecimal(number, buffer)
		return nil
	}
	number, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return ArgoError("Couldn't create an Int representation of " + value)
	}
	WriteBigInt(number, buffer)
	return nil
}

func WriteDecimal(value *big.Decimal, buffer *bytes.Buffer) {
	buffer.WriteByte('\x01')
	left, right := value.Components()
	positive := writeBigInt(left, buffer, bigintMagicNumber1)
	if right != nil {
		if positive {
			buffer.WriteByte('\x00')
		} else {
			buffer.WriteByte('\xff')
		}
		writeBigInt(right, buffer, bigintMagicNumber2)
	}
}

func WriteBigInt(value *big.Int, buffer *bytes.Buffer) {
	buffer.WriteByte('\x01')
	writeBigInt(value, buffer, bigintMagicNumber1)
}

func writeBigInt(value *big.Int, buffer *bytes.Buffer, cutoff *big.Int) (positive bool) {
	if value.IsZero() {
		buffer.Write(zeroBase)
		return !value.RawNeg()
	}
	if !value.RawNeg() {
		positive = true
		if value.Cmp(cutoff) == -1 {
			encoding := []byte{'\x80', '\x01', '\x01'}
			mod := big.NewInt(0)
			div, mod := value.DivMod(value, bigint255, mod)
			encoding[2] = byte(mod.Int64()) + 1
			if div.Sign() == 1 {
				div, mod = div.DivMod(div, bigint255, mod)
				encoding[1] = byte(mod.Int64()) + 1
				if div.Sign() == 1 {
					encoding[0] = byte(div.Int64()) + 128
				}
			}
			buffer.Write(encoding)
		} else {
			value = value.Sub(value, cutoff)
			buffer.WriteByte('\xff')
			left := big.NewInt(0)
			lead, left := value.DivMod(value, bigint255, left)
			var n int64 = 1
			exp := big.NewInt(0)
			div := big.NewInt(0)
			for (div.Div(lead, exp.Exp(big.NewInt(253), big.NewInt(n), nil))).Sign() == 1 {
				n += 1
			}
			buffer.WriteByte(byte(n) + 1)
			buffer.WriteByte('\xff')
			leadChars := make([]byte, 0)
			mod := big.NewInt(0)
			for {
				if lead.IsZero() {
					break
				}
				lead, mod = lead.DivMod(lead, bigint253, mod)
				leadChars = append(leadChars, byte(mod.Int64())+2)
			}
			lenLead := len(leadChars)
			if lenLead > 0 {
				for i := lenLead - 1; i >= 0; i-- {
					buffer.WriteByte(leadChars[i])
				}
			}
			if left.Sign() == 1 {
				buffer.WriteByte('\x01')
				buffer.WriteByte(byte(left.Int64()))
			}
		}
	} else {
		value = value.Neg(value)
		if value.Cmp(cutoff) == -1 {
			encoding := []byte{'\x7f', '\xfe', '\xfe'}
			mod := big.NewInt(0)
			div, mod := value.DivMod(value, bigint255, mod)
			encoding[2] = 254 - byte(mod.Int64())
			if div.Sign() == 1 {
				div, mod = div.DivMod(div, bigint255, mod)
				encoding[1] = 254 - byte(mod.Int64())
				if div.Sign() == 1 {
					encoding[0] = 127 - byte(div.Int64())
				}
			}
			buffer.Write(encoding)
		} else {
			value = value.Sub(value, cutoff)
			buffer.WriteByte('\x00')
			left := big.NewInt(0)
			lead, left := value.DivMod(value, bigint254, left)
			var n int64 = 1
			exp := big.NewInt(0)
			div := big.NewInt(0)
			for (div.Div(lead, exp.Exp(big.NewInt(253), big.NewInt(n), nil))).Sign() == 1 {
				n += 1
			}
			buffer.WriteByte(254 - byte(n))
			buffer.WriteByte('\x00')
			leadChars := make([]byte, 0)
			mod := big.NewInt(0)
			for {
				if lead.IsZero() {
					break
				}
				lead, mod = lead.DivMod(lead, bigint253, mod)
				leadChars = append(leadChars, byte(253-mod.Int64()))
			}
			lenLead := len(leadChars)
			if lenLead > 0 {
				for i := lenLead - 1; i >= 0; i-- {
					buffer.WriteByte(leadChars[i])
				}
			}
			if lenLead > 1 {
				buffer.WriteByte('\x00')
			}
			buffer.WriteByte('\xfe')
			if left.Sign() == 1 {
				buffer.WriteByte('\x01')
				buffer.WriteByte(254 - byte(left.Int64()))
			} else {
				buffer.WriteByte('\xfe')
			}
		}
	}
	return
}