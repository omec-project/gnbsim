// SPDX-FileCopyrightText: 2024 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"log"
	"sync/atomic"
	"time"
)

const (
	MSG_OUT        = 0x0001
	MSG_IN         = 0x0002
	REG_PROC_START = 0x0003 // trans0
	REG_REQ_OUT    = 0x0004 // trans1
	AUTH_REQ_IN    = 0x0005 // trans1
	AUTH_RSP_OUT   = 0x0006 // trans2
	SECM_CMD_IN    = 0x0007 // trans2
	SECM_CMP_OUT   = 0x0008 // trans3
	ICS_REQ_IN     = 0x0009 // trans3
	REG_COMP_OUT   = 0x000a // trans0
	REG_PROC_END   = 0x000b // trans0

	PDU_PROC_START     = 0x0011
	PDU_SESS_REQ_OUT   = 0x0012
	PDU_SESS_ACC_IN    = 0x0013
	PDU_SESS_RES_SETUP = 0x0014
	PDU_SESS_PROC_END  = 0x0015

	SVC_PROC_START = 0x0021
	SVC_REQ_OUT    = 0x0022
	SVC_ACCEPT_IN  = 0x0023
	SVC_PROC_END   = 0x0024

	UE_CTX_REL_OUT     = 0x0031
	UE_CTX_CMD_IN      = 0x0032
	UE_CTX_REL_CMP_OUT = 0x0033

	DEREG_REQ_OUT = 0x0041
	DEREG_ACC_IN  = 0x0042
)

type Registration struct {
	RegReqOutTime    time.Time
	AuthReqInTime    time.Time
	AuthRspOutTime   time.Time
	SecMCmdInTime    time.Time
	SecCmdCmpOutTime time.Time
	ICSReqInTime     time.Time
	RegProcTime      int64
	RegReqAuthReq    int64
	AuthRspSecMReq   int64
	SecModeRspICReq  int64
}

type PduSessEst struct {
	PduSessReqOutTime time.Time
	PduSessAcceptIn   time.Time
	PduSessProcTime   int64
	PduSessReqAccept  int64
}

type Deregistration struct {
	DeregReqOutTime        time.Time
	DeregAccInTime         time.Time
	DeregistrationProcTime int64
	DregReqAccTime         int64
}

type ServiceReq struct {
	ServiceReqOutTime  time.Time
	ServiceAccInTime   time.Time
	ServiceReqProcTime int64
	ServReqAccTime     int64
}

type CtxRelease struct {
	CtxRelReqOutTime   time.Time
	CtxRelCmdInTime    time.Time
	CtxReleaseProcTime int64
	CtxRelReqCmdTime   int64
}

type UeStats struct {
	Reg     []Registration // Historical. After completion move CReg here
	Pdu     []PduSessEst
	Svc     []ServiceReq
	Ctxrel  []CtxRelease
	Dreg    []Deregistration
	CSvc    ServiceReq
	CPdu    PduSessEst
	CCtxrel CtxRelease
	CDreg   Deregistration
	Supi    string
	CReg    Registration // Current
}

type StatisticsEvent struct {
	T     time.Time
	Supi  string
	EType int64
	Id    uint64
}

var (
	ReadChan        chan *StatisticsEvent
	Counter         atomic.Uint64
	NilCounter      atomic.Uint64
	UeStatsTable    map[string]*UeStats
	StatsTransTable map[uint64]*StatisticsEvent
)

func init() {
	// create channel
	ReadChan = make(chan *StatisticsEvent, 100000)
	go readStats()
	UeStatsTable = make(map[string]*UeStats)
	StatsTransTable = make(map[uint64]*StatisticsEvent)
}

func GetId() uint64 {
	c := Counter.Add(1)
	return c
}

func LogStats(m *StatisticsEvent) {
	ReadChan <- m
}

// when Request or response is received on socket
func RecvdMessage(m *StatisticsEvent) {
	ReadChan <- m
}

func SentMessage(m *StatisticsEvent) {
	ReadChan <- m
}

func addTrans(m *StatisticsEvent) {
	StatsTransTable[m.Id] = m
}

func popTrans(id uint64) *StatisticsEvent {
	t, found := StatsTransTable[id]
	if found == false {
		log.Println("No transaction found for Id:", id)
		return nil
	}
	delete(StatsTransTable, id)
	return t
}

func getUe(supi string) *UeStats {
	log.Println("Find the UE ", supi)
	ue, found := UeStatsTable[supi]
	if found == false {
		ue := UeStats{Supi: supi}
		UeStatsTable[supi] = &ue
		return &ue
	}
	return ue
}

func readStats() {
	for {
		select {
		case m := <-ReadChan:
			switch m.EType {
			case REG_PROC_START:
				log.Println("*******Received Event : REG_PROC_START: ", m)
			case REG_PROC_END:
				log.Println("*******Received Event : REG_PROC_END: ", m)
			case REG_REQ_OUT:
				log.Println("*******Received Event : REG_REQ_OUT: ", m)
				addTrans(m)
			case AUTH_REQ_IN:
				log.Println("*******Received Event : AUTH_REQ_IN : ", m)
				t := popTrans(m.Id) // remove MSG in trans but use the time msg was received
				m.T = t.T
				ue := getUe(m.Supi)
				ue.CReg.AuthReqInTime = m.T
				x := m.T.Sub(ue.CReg.RegReqOutTime)
				ue.CReg.RegReqAuthReq = x.Nanoseconds()
				log.Println("*************************************Time between Reg Req & Auth Req ", ue.CReg.RegReqAuthReq)
			case AUTH_RSP_OUT:
				log.Println("*******Received Event : AUTH_RSP_OUT: ", m)
				addTrans(m)
			case SECM_CMD_IN:
				log.Println("*******Received Event : SECM_CMD_IN: ", m)
				t := popTrans(m.Id) // remove MSG in trans but use the time msg was received
				m.T = t.T
				ue := getUe(m.Supi)
				ue.CReg.SecMCmdInTime = m.T
				x := m.T.Sub(ue.CReg.AuthRspOutTime)
				ue.CReg.AuthRspSecMReq = x.Nanoseconds()
				log.Println("**************************************Time between Auth Rsp and Sec M Req ", ue.CReg.AuthRspSecMReq)
			case SECM_CMP_OUT:
				log.Println("*******Received Event : SECM_CMP_OUT: ", m)
				addTrans(m)
			case ICS_REQ_IN:
				log.Println("*******Received Event : ICS_REQ_IN: ", m)
				t := popTrans(m.Id) // remove MSG in trans but use the time msg was received
				m.T = t.T
				ue := getUe(m.Supi)
				ue.CReg.ICSReqInTime = m.T
				x := m.T.Sub(ue.CReg.SecCmdCmpOutTime)
				ue.CReg.SecModeRspICReq = x.Nanoseconds()
				log.Println("**************************************Time between Sec Mod Cmd & ICSReq ", ue.CReg.SecModeRspICReq)
			case REG_COMP_OUT:
				log.Println("*******Received Event : REG_COMP_OUT: ", m)
				addTrans(m)
			case PDU_SESS_REQ_OUT:
				log.Println("*******Received Event PDU_SESS_REQ_OUT: ", m)
				addTrans(m)
			case PDU_SESS_ACC_IN:
				log.Println("*******Received Event : PDU_SESS_ACC_IN: ", m)
				t := popTrans(m.Id) // remove MSG in trans but use the time msg was received
				m.T = t.T
				ue := getUe(m.Supi)
				ue.CPdu.PduSessAcceptIn = m.T
				x := m.T.Sub(ue.CPdu.PduSessReqOutTime)
				ue.CPdu.PduSessReqAccept = x.Nanoseconds()
				log.Println("**************************************Time between PDU Sess Req & Accept ", ue.CPdu.PduSessReqAccept)
				ue.CPdu.PduSessProcTime = ue.CPdu.PduSessReqAccept
				ue.Pdu = append(ue.Pdu, ue.CPdu)
				ue.CPdu = PduSessEst{}
				//			case PDU_SESS_RES_SETUP:
				//				log.Println("*******Received Event PDU_SESS_RES_SETUP: ", m)
				//				addTrans(m)
			case UE_CTX_REL_OUT:
				log.Println("*******Received UE_CTX_REL_OUT ", m)
				addTrans(m)
			case UE_CTX_CMD_IN:
				log.Println("*******Received Event : UE_CTX_CMD_IN: ", m)
				t := popTrans(m.Id) // remove MSG in trans but use the time msg was received
				m.T = t.T
				ue := getUe(m.Supi)
				ue.CCtxrel.CtxRelCmdInTime = m.T
				if ue.CCtxrel.CtxRelReqOutTime.IsZero() == false {
					x := m.T.Sub(ue.CCtxrel.CtxRelReqOutTime)
					ue.CCtxrel.CtxRelReqCmdTime = x.Nanoseconds()
					ue.CCtxrel.CtxReleaseProcTime = ue.CCtxrel.CtxRelReqCmdTime
					ue.Ctxrel = append(ue.Ctxrel, ue.CCtxrel)
					log.Println("**************************************Time between Ctx Rel Req & Cmd ", ue.CCtxrel.CtxRelReqCmdTime)
					ue.CCtxrel = CtxRelease{}
				}
			case DEREG_REQ_OUT:
				log.Println("*******Received DEREG_REQ_OUT ", m)
				addTrans(m)
			case DEREG_ACC_IN:
				log.Println("*******Received Event : DEREG_ACC_IN : ", m)
				t := popTrans(m.Id) // remove MSG in trans but use the time msg was received
				m.T = t.T
				ue := getUe(m.Supi)
				ue.CDreg.DeregAccInTime = m.T
				x := m.T.Sub(ue.CDreg.DeregReqOutTime)
				ue.CDreg.DregReqAccTime = x.Nanoseconds()
				ue.CDreg.DeregistrationProcTime = ue.CDreg.DregReqAccTime
				ue.Dreg = append(ue.Dreg, ue.CDreg)
				log.Println("**************************************Time between Dereg Req & Accept ", ue.CDreg.DregReqAccTime)
				ue.CDreg = Deregistration{}
			case SVC_REQ_OUT:
				log.Println("*******Received SVC_REQ_OUT", m)
				addTrans(m)
			case SVC_ACCEPT_IN:
				log.Println("*******Received Event : SVC_ACCEPT_IN: ", m)
				t := popTrans(m.Id) // remove MSG in trans but use the time msg was received
				m.T = t.T
				ue := getUe(m.Supi)
				ue.CSvc.ServiceAccInTime = m.T
				x := m.T.Sub(ue.CSvc.ServiceReqOutTime)
				ue.CSvc.ServReqAccTime = x.Nanoseconds()
				ue.CSvc.ServiceReqProcTime = ue.CSvc.ServReqAccTime
				ue.Svc = append(ue.Svc, ue.CSvc)
				log.Println("**************************************Time between Service Req & Accept ", ue.CSvc.ServReqAccTime)
				ue.CSvc = ServiceReq{}
			case MSG_OUT:
				log.Println("*******Received Event : MSG_OUT: ", m)
				if m.Id != 0 {
					t := popTrans(m.Id) // Don't add new Event event in table
					if t != nil {
						t.T = m.T
						ue := getUe(t.Supi)
						switch t.EType {
						case REG_REQ_OUT:
							ue.CReg.RegReqOutTime = t.T
						case AUTH_RSP_OUT:
							ue.CReg.AuthRspOutTime = t.T
						case SECM_CMP_OUT:
							ue.CReg.SecCmdCmpOutTime = t.T
						case REG_COMP_OUT:
							ue.CReg.RegProcTime = ue.CReg.RegReqAuthReq + ue.CReg.AuthRspSecMReq + ue.CReg.SecModeRspICReq
							ue.Reg = append(ue.Reg, ue.CReg) // push the history
							ue.CReg = Registration{}
						case PDU_SESS_REQ_OUT:
							ue.CPdu.PduSessReqOutTime = t.T
							//						case PDU_SESS_RES_SETUP:
							//							ue.CPdu.PduSessProcTime = ue.CPdu.PduSessReqAccept
							//							ue.Pdu = append(ue.Pdu, ue.CPdu)
						case UE_CTX_REL_OUT:
							ue.CCtxrel.CtxRelReqOutTime = t.T
						case UE_CTX_REL_CMP_OUT:
							if ue.CCtxrel.CtxRelReqCmdTime != 0 {
								ue.CCtxrel.CtxReleaseProcTime = ue.CCtxrel.CtxRelReqCmdTime
								ue.Ctxrel = append(ue.Ctxrel, ue.CCtxrel)
								ue.CCtxrel = CtxRelease{}
							}
						case SVC_REQ_OUT:
							ue.CSvc.ServiceReqOutTime = t.T
						case DEREG_REQ_OUT:
							ue.CDreg.DeregReqOutTime = t.T
						}
					}
				}
			case MSG_IN:
				log.Println("*******Received Event : MSG_IN: ", m)
				addTrans(m)
			}
		}
	}
}

func DumpStats() {
	log.Println("Dump all metrics")
	for _, ue := range UeStatsTable {
		for _, s := range ue.Reg {
			log.Printf("UE: %s Total Reg Time = %d RegReqAuthReq: %d  AuthRspSecMReq: %d SecModeRspICReq: %d ", ue.Supi, s.RegProcTime, s.RegReqAuthReq, s.AuthRspSecMReq, s.SecModeRspICReq)
		}
		for _, s := range ue.Pdu {
			log.Printf("UE: %s Total Pdu Est Time = %d PduSessReqAccept: %d  ", ue.Supi, s.PduSessProcTime, s.PduSessReqAccept)
		}
		for _, s := range ue.Ctxrel {
			log.Printf("UE: %s Total Ctx Release Time = %d CtxRelReqCmdTime: %d  ", ue.Supi, s.CtxReleaseProcTime, s.CtxRelReqCmdTime)
		}
		for _, s := range ue.Svc {
			log.Printf("UE: %s Total Service Req Time = %d ServReqAccTime: %d  ", ue.Supi, s.ServiceReqProcTime, s.ServReqAccTime)
		}
		for _, s := range ue.Dreg {
			log.Printf("UE: %s Total Deregistration Time = %d DregReqAccTime: %d  ", ue.Supi, s.DeregistrationProcTime, s.DregReqAccTime)
		}
	}
	for k1, v1 := range StatsTransTable {
		log.Println("k1 ", k1, " v1: ", v1)
	}
}
