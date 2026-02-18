// SPDX-FileCopyrightText: 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"strings"
	"testing"

	"github.com/omec-project/gnbsim/common"
	"github.com/omec-project/openapi/models"
)

func TestProfileInit_ValidatesDnnForPduSessionProfiles(t *testing.T) {
	tests := []struct {
		sNssai      *models.Snssai
		name        string
		profileType string
		dnn         string
		errorMsg    string
		expectError bool
	}{
		{
			name:        "pdusessest profile without dnn should fail",
			profileType: PDU_SESS_EST,
			dnn:         "",
			sNssai:      &models.Snssai{Sst: 1, Sd: "010203"},
			expectError: true,
			errorMsg:    "dnn is required",
		},
		{
			name:        "pdusessest profile without sNssai should fail",
			profileType: PDU_SESS_EST,
			dnn:         "internet",
			sNssai:      nil,
			expectError: true,
			errorMsg:    "sNssai is required",
		},
		{
			name:        "pdusessest profile with sst=0 should fail",
			profileType: PDU_SESS_EST,
			dnn:         "internet",
			sNssai:      &models.Snssai{Sst: 0, Sd: "010203"},
			expectError: true,
			errorMsg:    "sNssai.sst is required",
		},
		{
			name:        "pdusessest profile with valid dnn and sNssai should pass",
			profileType: PDU_SESS_EST,
			dnn:         "internet",
			sNssai:      &models.Snssai{Sst: 1, Sd: "010203"},
			expectError: false,
		},
		{
			name:        "register profile without dnn should pass",
			profileType: REGISTER,
			dnn:         "",
			sNssai:      nil,
			expectError: false,
		},
		{
			name:        "deregister profile without dnn should fail",
			profileType: DEREGISTER,
			dnn:         "",
			sNssai:      &models.Snssai{Sst: 1, Sd: "010203"},
			expectError: true,
			errorMsg:    "dnn is required",
		},
		{
			name:        "anrelease profile without sNssai should fail",
			profileType: AN_RELEASE,
			dnn:         "internet",
			sNssai:      nil,
			expectError: true,
			errorMsg:    "sNssai is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &Profile{
				ProfileType: tt.profileType,
				Name:        "test-profile",
				GnbName:     "gnb1",
				StartImsi:   "208930100007487",
				Key:         "5122250214c33e723a5dd523fc145fc0",
				Opc:         "981d464c7c52eb6e5036234984ad0bcf",
				SeqNum:      "16f3b3f70fc2",
				Dnn:         tt.dnn,
				SNssai:      tt.sNssai,
				UeCount:     1,
				Plmn: &models.PlmnId{
					Mcc: "208",
					Mnc: "93",
				},
			}

			err := profile.Init()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', but got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}

func TestProfileInit_ValidatesDnnForCustomProfiles(t *testing.T) {
	tests := []struct {
		pIterations map[string]*PIterations
		sNssai      *models.Snssai
		name        string
		dnn         string
		errorMsg    string
		expectError bool
	}{
		{
			name: "custom profile with PDU session without dnn should fail",
			pIterations: map[string]*PIterations{
				"iteration1": {
					Name: "iteration1",
					ProcMap: map[int]common.ProcedureType{
						1: common.REGISTRATION_PROCEDURE,
						2: common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
					},
				},
			},
			dnn:         "",
			sNssai:      &models.Snssai{Sst: 1, Sd: "010203"},
			expectError: true,
			errorMsg:    "dnn is required",
		},
		{
			name: "custom profile with PDU session without sNssai should fail",
			pIterations: map[string]*PIterations{
				"iteration1": {
					Name: "iteration1",
					ProcMap: map[int]common.ProcedureType{
						1: common.REGISTRATION_PROCEDURE,
						2: common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
					},
				},
			},
			dnn:         "internet",
			sNssai:      nil,
			expectError: true,
			errorMsg:    "sNssai is required",
		},
		{
			name: "custom profile with PDU session with sst=0 should fail",
			pIterations: map[string]*PIterations{
				"iteration1": {
					Name: "iteration1",
					ProcMap: map[int]common.ProcedureType{
						1: common.REGISTRATION_PROCEDURE,
						2: common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
					},
				},
			},
			dnn:         "internet",
			sNssai:      &models.Snssai{Sst: 0, Sd: "010203"},
			expectError: true,
			errorMsg:    "sNssai.sst is required",
		},
		{
			name: "custom profile with PDU session with valid dnn and sNssai should pass",
			pIterations: map[string]*PIterations{
				"iteration1": {
					Name: "iteration1",
					ProcMap: map[int]common.ProcedureType{
						1: common.REGISTRATION_PROCEDURE,
						2: common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
					},
				},
			},
			dnn:         "internet",
			sNssai:      &models.Snssai{Sst: 1, Sd: "010203"},
			expectError: false,
		},
		{
			name: "custom profile without PDU session and without dnn should pass",
			pIterations: map[string]*PIterations{
				"iteration1": {
					Name: "iteration1",
					ProcMap: map[int]common.ProcedureType{
						1: common.REGISTRATION_PROCEDURE,
						2: common.UE_INITIATED_DEREGISTRATION_PROCEDURE,
					},
				},
			},
			dnn:         "",
			sNssai:      nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &Profile{
				ProfileType: CUSTOM_PROCEDURE,
				Name:        "test-custom-profile",
				GnbName:     "gnb1",
				StartImsi:   "208930100007487",
				Key:         "5122250214c33e723a5dd523fc145fc0",
				Opc:         "981d464c7c52eb6e5036234984ad0bcf",
				SeqNum:      "16f3b3f70fc2",
				Dnn:         tt.dnn,
				SNssai:      tt.sNssai,
				UeCount:     1,
				PIterations: tt.pIterations,
				Plmn: &models.PlmnId{
					Mcc: "208",
					Mnc: "93",
				},
			}

			err := profile.Init()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', but got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}

func TestRequiresPduSession(t *testing.T) {
	tests := []struct {
		name        string
		profileType string
		expected    bool
	}{
		{
			name:        "register profile does not require PDU session",
			profileType: REGISTER,
			expected:    false,
		},
		{
			name:        "pdusessest profile requires PDU session",
			profileType: PDU_SESS_EST,
			expected:    true,
		},
		{
			name:        "deregister profile requires PDU session",
			profileType: DEREGISTER,
			expected:    true,
		},
		{
			name:        "anrelease profile requires PDU session",
			profileType: AN_RELEASE,
			expected:    true,
		},
		{
			name:        "uetriggservicereq profile requires PDU session",
			profileType: UE_TRIGG_SERVICE_REQ,
			expected:    true,
		},
		{
			name:        "nwtriggeruedereg profile requires PDU session",
			profileType: NW_TRIGG_UE_DEREG,
			expected:    true,
		},
		{
			name:        "uereqpdusessrelease profile requires PDU session",
			profileType: UE_REQ_PDU_SESS_RELEASE,
			expected:    true,
		},
		{
			name:        "nwreqpdusessrelease profile requires PDU session",
			profileType: NW_REQ_PDU_SESS_RELEASE,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &Profile{
				ProfileType: tt.profileType,
				Name:        "test-profile",
				GnbName:     "gnb1",
				StartImsi:   "208930100007487",
				Key:         "5122250214c33e723a5dd523fc145fc0",
				Opc:         "981d464c7c52eb6e5036234984ad0bcf",
				SeqNum:      "16f3b3f70fc2",
				UeCount:     1,
			}

			// Initialize procedures for the profile type
			if err := initProcedureList(profile); err != nil {
				t.Fatalf("Failed to initialize procedure list: %v", err)
			}

			result := requiresPduSession(profile)

			if result != tt.expected {
				t.Errorf("Expected requiresPduSession to return %v, but got %v", tt.expected, result)
			}
		})
	}
}

func TestRequiresPduSessionCustomProfile(t *testing.T) {
	tests := []struct {
		pIterations map[string]*PIterations
		name        string
		expected    bool
	}{
		{
			name: "custom profile without PDU session",
			pIterations: map[string]*PIterations{
				"iteration1": {
					Name: "iteration1",
					ProcMap: map[int]common.ProcedureType{
						1: common.REGISTRATION_PROCEDURE,
						2: common.UE_INITIATED_DEREGISTRATION_PROCEDURE,
					},
				},
			},
			expected: false,
		},
		{
			name: "custom profile with PDU session in iterations",
			pIterations: map[string]*PIterations{
				"iteration1": {
					Name: "iteration1",
					ProcMap: map[int]common.ProcedureType{
						1: common.REGISTRATION_PROCEDURE,
						2: common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
						3: common.USER_DATA_PKT_GENERATION_PROCEDURE,
					},
				},
			},
			expected: true,
		},
		{
			name: "custom profile with multiple iterations, one with PDU session",
			pIterations: map[string]*PIterations{
				"iteration1": {
					Name: "iteration1",
					ProcMap: map[int]common.ProcedureType{
						1: common.REGISTRATION_PROCEDURE,
					},
					NextItr: "iteration2",
				},
				"iteration2": {
					Name: "iteration2",
					ProcMap: map[int]common.ProcedureType{
						1: common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
						2: common.USER_DATA_PKT_GENERATION_PROCEDURE,
					},
					NextItr: "quit",
				},
			},
			expected: true,
		},
		{
			name: "custom profile with multiple iterations, none with PDU session",
			pIterations: map[string]*PIterations{
				"iteration1": {
					Name: "iteration1",
					ProcMap: map[int]common.ProcedureType{
						1: common.REGISTRATION_PROCEDURE,
					},
					NextItr: "iteration2",
				},
				"iteration2": {
					Name: "iteration2",
					ProcMap: map[int]common.ProcedureType{
						1: common.UE_INITIATED_DEREGISTRATION_PROCEDURE,
					},
					NextItr: "quit",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &Profile{
				ProfileType: CUSTOM_PROCEDURE,
				Name:        "test-custom-profile",
				GnbName:     "gnb1",
				StartImsi:   "208930100007487",
				Key:         "5122250214c33e723a5dd523fc145fc0",
				Opc:         "981d464c7c52eb6e5036234984ad0bcf",
				SeqNum:      "16f3b3f70fc2",
				UeCount:     1,
				PIterations: tt.pIterations,
			}

			// Initialize procedures for custom profile
			if err := initProcedureList(profile); err != nil {
				t.Fatalf("Failed to initialize procedure list: %v", err)
			}

			result := requiresPduSession(profile)

			if result != tt.expected {
				t.Errorf("Expected requiresPduSession to return %v, but got %v", tt.expected, result)
			}
		})
	}
}
