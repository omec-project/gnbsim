package logger_util

type Logger struct {
	AMF   *LogSetting `yaml:"AMF"`
	AUSF  *LogSetting `yaml:"AUSF"`
	N3IWF *LogSetting `yaml:"N3IWF"`
	NRF   *LogSetting `yaml:"NRF"`
	NSSF  *LogSetting `yaml:"NSSF"`
	PCF   *LogSetting `yaml:"PCF"`
	SMF   *LogSetting `yaml:"SMF"`
	UDM   *LogSetting `yaml:"UDM"`
	UDR   *LogSetting `yaml:"UDR"`
	NEF   *LogSetting `yaml:"NEF"`
	WEBUI *LogSetting `yaml:"WEBUI"`

	Aper                         *LogSetting `yaml:"Aper"`
	CommonConsumerTest           *LogSetting `yaml:"CommonConsumerTest"`
	FSM                          *LogSetting `yaml:"FSM"`
	MongoDBLibrary               *LogSetting `yaml:"MongoDBLibrary"`
	NAS                          *LogSetting `yaml:"NAS"`
	NGAP                         *LogSetting `yaml:"NGAP"`
	OpenApi                      *LogSetting `yaml:"OpenApi"`
	NamfCommunication            *LogSetting `yaml:"NamfCommunication"`
	NamfEventExposure            *LogSetting `yaml:"NamfEventExposure"`
	NnssfNSSAIAvailability       *LogSetting `yaml:"NnssfNSSAIAvailability"`
	NnssfNSSelection             *LogSetting `yaml:"NnssfNSSelection"`
	NsmfEventExposure            *LogSetting `yaml:"NsmfEventExposure"`
	NsmfPDUSession               *LogSetting `yaml:"NsmfPDUSession"`
	NudmEventExposure            *LogSetting `yaml:"NudmEventExposure"`
	NudmParameterProvision       *LogSetting `yaml:"NudmParameterProvision"`
	NudmSubscriberDataManagement *LogSetting `yaml:"NudmSubscriberDataManagement"`
	NudmUEAuthentication         *LogSetting `yaml:"NudmUEAuthentication"`
	NudmUEContextManagement      *LogSetting `yaml:"NudmUEContextManagement"`
	NudrDataRepository           *LogSetting `yaml:"NudrDataRepository"`
	PathUtil                     *LogSetting `yaml:"PathUtil"`
	PFCP                         *LogSetting `yaml:"PFCP"`
}

type LogSetting struct {
	DebugLevel   string `yaml:"debugLevel"`
	ReportCaller bool   `yaml:"ReportCaller"`
}
