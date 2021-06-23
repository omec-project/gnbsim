package nasConvert

import (
	"github.com/free5gc/nas/logger"
	"github.com/free5gc/nas/nasMessage"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

type ProtocolOrContainerUnit struct {
	ProtocolOrContainerID uint16
	LengthOfContents      uint8
	Contents              []byte
}

type ProtocolConfigurationOptions struct {
	ProtocolOrContainerList []*ProtocolOrContainerUnit
}

type PCOReadingState int

const (
	ReadingID PCOReadingState = iota
	ReadingLength
	ReadingContent
)

func NewProtocolOrContainerUnit() (pcu *ProtocolOrContainerUnit) {
	pcu = &ProtocolOrContainerUnit{
		ProtocolOrContainerID: 0,
		LengthOfContents:      0,
		Contents:              []byte{},
	}
	return
}

func NewProtocolConfigurationOptions() (pco *ProtocolConfigurationOptions) {

	pco = &ProtocolConfigurationOptions{
		ProtocolOrContainerList: make([]*ProtocolOrContainerUnit, 0),
	}

	return
}

func (protocolConfigurationOptions *ProtocolConfigurationOptions) Marshal() []byte {

	var metaInfo uint8
	var extension uint8 = 1
	var spare uint8 = 0
	var configurationProtocol uint8 = 0
	buffer := new(bytes.Buffer)

	metaInfo = (extension << 7) | (spare << 6) | (configurationProtocol)
	if err := binary.Write(buffer, binary.BigEndian, &metaInfo); err != nil {
		logger.ConvertLog.Warnf("Write metaInfo failed: %+v", err)
	}

	for _, containerUnit := range protocolConfigurationOptions.ProtocolOrContainerList {

		if err := binary.Write(buffer, binary.BigEndian, &containerUnit.ProtocolOrContainerID); err != nil {
			logger.ConvertLog.Warnf("Write protocolOrContainerID failed: %+v", err)
		}
		if err := binary.Write(buffer, binary.BigEndian, &containerUnit.LengthOfContents); err != nil {
			logger.ConvertLog.Warnf("Write length of contents failed: %+v", err)
		}
		if err := binary.Write(buffer, binary.BigEndian, &containerUnit.Contents); err != nil {
			logger.ConvertLog.Warnf("Write contents failed: %+v", err)
		}
	}

	return buffer.Bytes()
}

func (protocolConfigurationOptions *ProtocolConfigurationOptions) UnMarshal(data []byte) error {
	logger.ConvertLog.Traceln("In ProtocolConfigurationOptions UnMarshal")

	var Buf uint8
	numOfBytes := len(data)
	byteReader := bytes.NewReader(data)
	if err := binary.Read(byteReader, binary.BigEndian, &Buf); err != nil {
		return err
	}

	numOfBytes = numOfBytes - 1
	readingState := ReadingID
	var curContainer *ProtocolOrContainerUnit

	for numOfBytes > 0 {

		switch readingState {
		case ReadingID:
			curContainer = NewProtocolOrContainerUnit()
			if err := binary.Read(byteReader, binary.BigEndian, &curContainer.ProtocolOrContainerID); err != nil {
				return err
			}
			logger.ConvertLog.Traceln("Reading ID: ", strconv.Itoa(int(curContainer.ProtocolOrContainerID)))
			readingState = ReadingLength
			numOfBytes = numOfBytes - 2
		case ReadingLength:
			if err := binary.Read(byteReader, binary.BigEndian, &curContainer.LengthOfContents); err != nil {
				return err
			}
			logger.ConvertLog.Traceln("Reading Length: ", strconv.Itoa(int(curContainer.LengthOfContents)))
			readingState = ReadingContent
			numOfBytes = numOfBytes - 1
			if curContainer.LengthOfContents == 0 {
				protocolConfigurationOptions.ProtocolOrContainerList = append(
					protocolConfigurationOptions.ProtocolOrContainerList, curContainer)
				logger.ConvertLog.Traceln("For loop ProtocolOrContainerList: ",
					protocolConfigurationOptions.ProtocolOrContainerList)
			}
		case ReadingContent:
			if curContainer.LengthOfContents > 0 {
				curContainer.Contents = make([]uint8, curContainer.LengthOfContents)
				if err := binary.Read(byteReader, binary.BigEndian, curContainer.Contents); err != nil {
					return err
				}
				protocolConfigurationOptions.ProtocolOrContainerList = append(
					protocolConfigurationOptions.ProtocolOrContainerList, curContainer)
				logger.ConvertLog.Traceln("For loop ProtocolOrContainerList: ",
					protocolConfigurationOptions.ProtocolOrContainerList)
			}
			numOfBytes = numOfBytes - int(curContainer.LengthOfContents)
			readingState = ReadingID
		}
	}

	logger.ConvertLog.Infoln("ProtocolOrContainerList: ", protocolConfigurationOptions.ProtocolOrContainerList)
	return nil
}

func (protocolConfigurationOptions *ProtocolConfigurationOptions) AddDNSServerIPv4AddressRequest() {
	protocolOrContainerUnit := NewProtocolOrContainerUnit()

	protocolOrContainerUnit.ProtocolOrContainerID = nasMessage.DNSServerIPv4AddressRequestUL
	protocolOrContainerUnit.LengthOfContents = 0

	protocolConfigurationOptions.ProtocolOrContainerList = append(protocolConfigurationOptions.ProtocolOrContainerList,
		protocolOrContainerUnit)
}

func (protocolConfigurationOptions *ProtocolConfigurationOptions) AddDNSServerIPv6AddressRequest() {
	protocolOrContainerUnit := NewProtocolOrContainerUnit()

	protocolOrContainerUnit.ProtocolOrContainerID = nasMessage.DNSServerIPv6AddressRequestUL
	protocolOrContainerUnit.LengthOfContents = 0

	protocolConfigurationOptions.ProtocolOrContainerList = append(protocolConfigurationOptions.ProtocolOrContainerList,
		protocolOrContainerUnit)
}

func (protocolConfigurationOptions *ProtocolConfigurationOptions) AddIPAddressAllocationViaNASSignallingUL() {
	protocolOrContainerUnit := NewProtocolOrContainerUnit()

	protocolOrContainerUnit.ProtocolOrContainerID = nasMessage.IPAddressAllocationViaNASSignallingUL
	protocolOrContainerUnit.LengthOfContents = 0

	protocolConfigurationOptions.ProtocolOrContainerList = append(protocolConfigurationOptions.ProtocolOrContainerList,
		protocolOrContainerUnit)
}

func (protocolConfigurationOptions *ProtocolConfigurationOptions) AddDNSServerIPv4Address(dnsIP net.IP) (err error) {

	if dnsIP.To4() == nil {
		err = fmt.Errorf("The DNS IP should be IPv4 in AddDNSServerIPv4Address!")
		return
	}
	dnsIP = dnsIP.To4()

	if len(dnsIP) != net.IPv4len {
		err = fmt.Errorf("The length of DNS IPv4 is wrong!")
		return
	}

	logger.ConvertLog.Traceln("In AddDNSServerIPv4Address")
	protocolOrContainerUnit := NewProtocolOrContainerUnit()

	protocolOrContainerUnit.ProtocolOrContainerID = nasMessage.DNSServerIPv4AddressDL
	protocolOrContainerUnit.LengthOfContents = uint8(net.IPv4len)
	logger.ConvertLog.Traceln("LengthOfContents: ", protocolOrContainerUnit.LengthOfContents)
	protocolOrContainerUnit.Contents = append(protocolOrContainerUnit.Contents, dnsIP.To4()...)
	logger.ConvertLog.Traceln("Contents: ", protocolOrContainerUnit.Contents)

	protocolConfigurationOptions.ProtocolOrContainerList = append(protocolConfigurationOptions.ProtocolOrContainerList,
		protocolOrContainerUnit)
	return
}

func (protocolConfigurationOptions *ProtocolConfigurationOptions) AddPCSCFIPv4Address(pcscfIP net.IP) (err error) {
	if pcscfIP.To4() == nil {
		err = fmt.Errorf("The P-CSCF IP should be IPv4!")
		return
	}
	pcscfIP = pcscfIP.To4()

	if len(pcscfIP) != net.IPv4len {
		err = fmt.Errorf("The length of P-CSCF IP IPv4 is wrong!")
		return
	}

	logger.ConvertLog.Traceln("In AddDNSServerIPv4Address")
	protocolOrContainerUnit := NewProtocolOrContainerUnit()
	protocolOrContainerUnit.ProtocolOrContainerID = nasMessage.PCSCFIPv4AddressDL
	protocolOrContainerUnit.LengthOfContents = uint8(net.IPv4len)
	logger.ConvertLog.Traceln("LengthOfContents: ", protocolOrContainerUnit.LengthOfContents)
	protocolOrContainerUnit.Contents = append(protocolOrContainerUnit.Contents, pcscfIP.To4()...)
	logger.ConvertLog.Traceln("Contents: ", protocolOrContainerUnit.Contents)

	protocolConfigurationOptions.ProtocolOrContainerList = append(protocolConfigurationOptions.ProtocolOrContainerList,
		protocolOrContainerUnit)
	return
}

func (protocolConfigurationOptions *ProtocolConfigurationOptions) AddDNSServerIPv6Address(dnsIP net.IP) (err error) {

	if dnsIP.To16() == nil {
		err = fmt.Errorf("The DNS IP should be IPv6 in AddDNSServerIPv6Address!")
		return
	}

	if len(dnsIP) != net.IPv6len {
		err = fmt.Errorf("The length of DNS IPv6 is wrong!")
		return
	}

	protocolOrContainerUnit := NewProtocolOrContainerUnit()

	protocolOrContainerUnit.ProtocolOrContainerID = nasMessage.DNSServerIPv6AddressDL
	protocolOrContainerUnit.LengthOfContents = uint8(net.IPv6len)
	protocolOrContainerUnit.Contents = append(protocolOrContainerUnit.Contents, dnsIP.To16()...)

	protocolConfigurationOptions.ProtocolOrContainerList = append(protocolConfigurationOptions.ProtocolOrContainerList,
		protocolOrContainerUnit)
	return
}

func (protocolConfigurationOptions *ProtocolConfigurationOptions) AddIPv4LinkMTU(mtu uint16) (err error) {
	logger.ConvertLog.Traceln("In AddIPv4LinkMTU")
	protocolOrContainerUnit := NewProtocolOrContainerUnit()

	protocolOrContainerUnit.ProtocolOrContainerID = nasMessage.IPv4LinkMTUDL
	protocolOrContainerUnit.LengthOfContents = 2
	logger.ConvertLog.Traceln("LengthOfContents: ", protocolOrContainerUnit.LengthOfContents)
	protocolOrContainerUnit.Contents =
		append(protocolOrContainerUnit.Contents, []byte{uint8(mtu >> 8), uint8(mtu & 0xff)}...)
	logger.ConvertLog.Traceln("Contents: ", protocolOrContainerUnit.Contents)

	protocolConfigurationOptions.ProtocolOrContainerList =
		append(protocolConfigurationOptions.ProtocolOrContainerList, protocolOrContainerUnit)
	return
}
