// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	transport "github.com/omec-project/gnbsim/transportcommon"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/util/idgenerator"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v4"
)

// GNodeB holds the context for a gNodeB. It manages the control plane and
// user plane layer of a gNodeB.
type GNodeB struct {
	// TODO IP and port should be the property of transport var
	GnbN2Ip              string                 `yaml:"n2IpAddr"`
	GnbN3Ip              string                 `yaml:"n3IpAddr"`
	GnbName              string                 `yaml:"name"`
	RanId                models.GlobalRanNodeId `yaml:"globalRanId"`
	GnbUes               *GnbUeDao
	GnbPeers             *GnbPeerDao
	RanUeNGAPIDGenerator *idgenerator.IDGenerator
	DlTeidGenerator      *idgenerator.IDGenerator
	Log                  *zap.SugaredLogger

	/*channel to notify all the go routines corresponding to this GNodeB instance to stop*/
	Quit chan int

	/* Default AMF to connect to */
	DefaultAmf *GnbAmf `yaml:"defaultAmf"`

	/* Control Plane transport */
	CpTransport transport.Transport

	/* User Plane transport */
	UpTransport transport.Transport

	SupportedTaList []SupportedTA `yaml:"supportedTaList"`
	GnbN2Port       int           `yaml:"n2Port"`
	GnbN3Port       int           `yaml:"n3Port"`
}

type gNodeBConfig struct {
	GnbN2Ip         string              `yaml:"n2IpAddr"`
	GnbN3Ip         string              `yaml:"n3IpAddr"`
	GnbName         string              `yaml:"name"`
	RanId           globalRanNodeIDYAML `yaml:"globalRanId"`
	DefaultAmf      *GnbAmf             `yaml:"defaultAmf"`
	SupportedTaList []SupportedTA       `yaml:"supportedTaList"`
	GnbN2Port       int                 `yaml:"n2Port"`
	GnbN3Port       int                 `yaml:"n3Port"`
}

type globalRanNodeIDYAML struct {
	N3IwfId *string       `yaml:"n3IwfId"`
	GNbId   *models.GNbId `yaml:"gNbId"`
	NgeNbId *string       `yaml:"ngeNbId"`
	WagfId  *string       `yaml:"wagfId"`
	TngfId  *string       `yaml:"tngfId"`
	Nid     *string       `yaml:"nid"`
	ENbId   *string       `yaml:"eNbId"`
	PlmnId  models.PlmnId `yaml:"plmnId"`
}

func (ranID globalRanNodeIDYAML) toModel() models.GlobalRanNodeId {
	return models.GlobalRanNodeId{
		PlmnId:  ranID.PlmnId,
		N3IwfId: ranID.N3IwfId,
		GNbId:   ranID.GNbId,
		NgeNbId: ranID.NgeNbId,
		WagfId:  ranID.WagfId,
		TngfId:  ranID.TngfId,
		Nid:     ranID.Nid,
		ENbId:   ranID.ENbId,
	}
}

func (gnb *GNodeB) UnmarshalYAML(value *yaml.Node) error {
	var config gNodeBConfig
	if err := value.Decode(&config); err != nil {
		return err
	}

	gnb.GnbN2Ip = config.GnbN2Ip
	gnb.GnbN3Ip = config.GnbN3Ip
	gnb.GnbName = config.GnbName
	gnb.RanId = config.RanId.toModel()
	gnb.DefaultAmf = config.DefaultAmf
	gnb.SupportedTaList = config.SupportedTaList
	gnb.GnbN2Port = config.GnbN2Port
	gnb.GnbN3Port = config.GnbN3Port

	return nil
}

func (gnb *GNodeB) GetDefaultAmf() *GnbAmf {
	return gnb.DefaultAmf
}

func (gnb *GNodeB) AllocateRanUeNgapID() (int64, error) {
	return gnb.RanUeNGAPIDGenerator.Allocate()
}

type SupportedTA struct {
	Tac               string              `yaml:"tac"`
	BroadcastPLMNList []BroadcastPLMNItem `yaml:"broadcastPlmnList"`
}

type BroadcastPLMNItem struct {
	PlmnId              models.PlmnId   `yaml:"plmnId"`
	TaiSliceSupportList []models.Snssai `yaml:"taiSliceSupportList"`
}
