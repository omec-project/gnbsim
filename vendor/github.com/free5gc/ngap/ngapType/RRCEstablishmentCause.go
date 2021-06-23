package ngapType

import "github.com/free5gc/aper"

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	RRCEstablishmentCausePresentEmergency          aper.Enumerated = 0
	RRCEstablishmentCausePresentHighPriorityAccess aper.Enumerated = 1
	RRCEstablishmentCausePresentMtAccess           aper.Enumerated = 2
	RRCEstablishmentCausePresentMoSignalling       aper.Enumerated = 3
	RRCEstablishmentCausePresentMoData             aper.Enumerated = 4
	RRCEstablishmentCausePresentMoVoiceCall        aper.Enumerated = 5
	RRCEstablishmentCausePresentMoVideoCall        aper.Enumerated = 6
	RRCEstablishmentCausePresentMoSMS              aper.Enumerated = 7
	RRCEstablishmentCausePresentMpsPriorityAccess  aper.Enumerated = 8
	RRCEstablishmentCausePresentMcsPriorityAccess  aper.Enumerated = 9
	RRCEstablishmentCausePresentNotAvailable       aper.Enumerated = 10
)

type RRCEstablishmentCause struct {
	Value aper.Enumerated `aper:"valueExt,valueLB:0,valueUB:10"`
}
