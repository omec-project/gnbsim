package ngap

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/free5gc/aper"
	"github.com/free5gc/ngap/ngapType"
)

// TS 38.412
const PPID uint32 = 0x3c000000

// Decoder is to decode raw data to NGAP pdu pointer with PER Aligned
func Decoder(b []byte) (pdu *ngapType.NGAPPDU, err error) {
	pdu = &ngapType.NGAPPDU{}

	err = aper.UnmarshalWithParams(b, pdu, "valueExt,valueLB:0,valueUB:2")
	return
}

// Encoder is to NGAP pdu to raw data with PER Aligned
func Encoder(pdu ngapType.NGAPPDU) ([]byte, error) {
	return aper.MarshalWithParams(pdu, "valueExt,valueLB:0,valueUB:2")
}

func PrintResult(v reflect.Value, layer int) string {

	fieldType := v.Type()
	if v.Kind() == reflect.Ptr {
		return "&" + PrintResult(v.Elem(), layer)
	}
	switch fieldType {
	case aper.OctetStringType:
		return fmt.Sprintf("OctetString (0x%x)[%d]\n", v.Bytes(), len(v.Bytes()))

	case aper.BitStringType:
		return fmt.Sprintf("BitString (%0.8b)[%d]\n", v.Field(0).Bytes(), v.Field(1).Uint())

	case aper.EnumeratedType:
		return fmt.Sprintf("Enumerated(%d)\n", v.Uint())
	}

	var s string
	switch v.Kind() {
	case reflect.Struct:
		structType := fieldType
		s += "{\n"
		end := strings.Repeat(" ", layer) + "}\n"
		layer += 2
		space := strings.Repeat(" ", layer)
		if structType.Field(0).Name == "Present" {
			present := v.Field(0).Int()
			s += (space + fmt.Sprintf("Present: %d\n", present))
			s += (space + fmt.Sprintf("%s : ", structType.Field(int(present)).Name))
			s += PrintResult(v.Field(int(present)), layer)
			s += end
			return s
		}
		for i := 0; i < v.NumField(); i++ {
			// optional
			if v.Field(i).Type().Kind() == reflect.Ptr && v.Field(i).IsNil() {
				continue
			}

			s += (space + fmt.Sprintf("%s : ", structType.Field(i).Name))
			s += PrintResult(v.Field(i), layer)
		}
		s += end
	case reflect.Slice:
		s += fmt.Sprintf("[%d]{\n", v.Len())
		end := strings.Repeat(" ", layer) + "}\n"
		layer += 2
		space := strings.Repeat(" ", layer)
		for i := 0; i < v.Len(); i++ {
			s += space
			s += PrintResult(v.Index(i), layer)
			s += (space + ",\n")
		}
		s += end
	case reflect.String:
		s = fmt.Sprintf("PrintableString(\"%s\")\n", v.String())
	case reflect.Int32, reflect.Int64:
		s = fmt.Sprintf("INTEGER(%d)\n", v.Int())
	default:
		fmt.Printf("Type: %s does not handle", v.Type())

	}
	return s
}
