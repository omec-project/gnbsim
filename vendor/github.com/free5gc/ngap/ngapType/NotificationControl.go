package ngapType

import "github.com/free5gc/aper"

// Need to import "github.com/free5gc/aper" if it uses "aper"

const (
	NotificationControlPresentNotificationRequested aper.Enumerated = 0
)

type NotificationControl struct {
	Value aper.Enumerated `aper:"valueExt,valueLB:0,valueUB:0"`
}
