// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/omec-project/aper"
	transport "github.com/omec-project/gnbsim/transportcommon"
	"github.com/omec-project/ngap/ngapConvert"
	"github.com/omec-project/ngap/ngapType"

	"github.com/omec-project/idgenerator"
	"github.com/omec-project/openapi/models"
	"github.com/sirupsen/logrus"
)

const MAX_CGI_BIT_LENGTH int32 = 36

// GNodeB holds the context for a gNodeB. It manages the control plane and
// user plane layer of a gNodeB.
type GNodeB struct {
	//TODO IP and port should be the property of transport var
	GnbN2Ip              string                 `yaml:"n2IpAddr"`
	GnbN2Port            int                    `yaml:"n2Port"`
	GnbN3Ip              string                 `yaml:"n3IpAddr"`
	GnbN3Port            int                    `yaml:"n3Port"`
	GnbName              string                 `yaml:"name"`
	RanId                models.GlobalRanNodeId `yaml:"globalRanId"`
	SupportedTaList      []SupportedTA          `yaml:"supportedTaList"`
	GnbUes               *GnbUeDao
	GnbPeers             *GnbPeerDao
	RanUeNGAPIDGenerator *idgenerator.IDGenerator
	DlTeidGenerator      *idgenerator.IDGenerator

	/*channel to notify all the go routines corresponding to this GNodeB instance to stop*/
	Quit chan int

	/* Default AMF to connect to */
	DefaultAmf *GnbAmf `yaml:"defaultAmf"`

	/* Control Plane transport */
	CpTransport transport.Transport

	/* User Plane transport */
	UpTransport transport.Transport

	/* logger */
	Log *logrus.Entry
}

func (gnb *GNodeB) GetDefaultAmf() *GnbAmf {
	return gnb.DefaultAmf
}

func (gnb *GNodeB) AllocateRanUeNgapID() (int64, error) {
	return gnb.RanUeNGAPIDGenerator.Allocate()
}

func (gnb *GNodeB) GetUserLocation(selectedPlmn *models.PlmnId) (*ngapType.UserLocationInformationNR, error) {
	ueLocInfo := new(ngapType.UserLocationInformationNR)
	ueLocInfo.NRCGI.PLMNIdentity = ngapConvert.PlmnIdToNgap(*gnb.RanId.PlmnId)

	gnbId := gnb.RanId.GNbId

	// CGI is formed by appending cell ID (4 - 14 bits) to gNB ID (22 - 32) bits
	// We currently have gNB ID configured, so applying left shift operation
	// according to its length. (The leftmost bits of the NR Cell Identity IE
	// correspond to the gNB ID - TS 38.413)
	gnbIdUint, err := strconv.ParseUint(gnbId.GNBValue, 16, int(gnbId.BitLength))
	if err != nil {
		return nil, fmt.Errorf("invalid gnbid configured:%v", err)
	}
	gnbIdUint = gnbIdUint << (MAX_CGI_BIT_LENGTH - gnbId.BitLength)
	// Cell ID
	gnbIdUint |= 0x01

	cgi := fmt.Sprintf("%x", gnbIdUint)

	// 36 bits sums up to odd byte count, so prepending 0 to make byte
	// count as 40
	cgi = "0" + cgi

	bs, err := hex.DecodeString(cgi)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string: %v", err)
	}

	ueLocInfo.NRCGI.NRCellIdentity.Value = aper.BitString{
		Bytes:     bs,
		BitLength: 36,
	}

	tai := gnb.GetTaiFromSelectedPLMN(selectedPlmn)
	if tai == nil {
		return nil, fmt.Errorf("plmn configured for ue is not supported by gnb. check supported ta list")
	}
	ueLocInfo.TAI = ngapConvert.TaiToNgap(*tai)
	return ueLocInfo, nil
}

// GetTaiFromSelectedPLMN fetches the TAI corresponding to the selectedPlmn
// from the Supported TA list in gNB
func (gnb *GNodeB) GetTaiFromSelectedPLMN(selectedPlmn *models.PlmnId) *models.Tai {
	for _, ta := range gnb.SupportedTaList {
		for _, plmn := range ta.BroadcastPLMNList {
			if plmn.PlmnId == *selectedPlmn {
				tai := &models.Tai{
					PlmnId: selectedPlmn,
					Tac:    ta.Tac,
				}
				return tai
			}
		}
	}
	return nil
}

type SupportedTA struct {
	Tac               string              `yaml:"tac"`
	BroadcastPLMNList []BroadcastPLMNItem `yaml:"broadcastPlmnList"`
}

type BroadcastPLMNItem struct {
	PlmnId              models.PlmnId   `yaml:"plmnId"`
	TaiSliceSupportList []models.Snssai `yaml:"taiSliceSupportList"`
}
