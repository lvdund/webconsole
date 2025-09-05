package data

import (
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/webconsole/backend/WebUI"
)

// MongoDB collection names
const (
	authSubsDataColl    = "subscriptionData.authenticationData.authenticationSubscription"
	authWebSubsDataColl = "subscriptionData.authenticationData.webAuthenticationSubscription"
	amDataColl          = "subscriptionData.provisionedData.amData"
	smDataColl          = "subscriptionData.provisionedData.smData"
	smfSelDataColl      = "subscriptionData.provisionedData.smfSelectionSubscriptionData"
	amPolicyDataColl    = "policyData.ues.amData"
	smPolicyDataColl    = "policyData.ues.smData"
	flowRuleDataColl    = "policyData.ues.flowRule"
	qosFlowDataColl     = "policyData.ues.qosFlow"
	chargingDataColl    = "policyData.ues.chargingData"
	identityDataColl    = "subscriptionData.identityData"
)

var SubsData WebUI.SubsData = WebUI.SubsData{
	// UeId:   "imsi-208930000000002",

	PlmnID: "20893",
	WebAuthenticationSubscription: WebUI.WebAuthenticationSubscription{
		AuthenticationMethod: "5G_AKA",
		PermanentKey: &WebUI.PermanentKey{
			PermanentKeyValue:   "8baf473f2f8fd09487cccbd7097c6862",
			EncryptionKey:       0,
			EncryptionAlgorithm: 0,
		},
		SequenceNumber: "000000000023",
		Milenage: &WebUI.Milenage{
			&WebUI.Op{
				OpValue:             "8e27b6af0e692e750f32667a3b14605d",
				EncryptionKey:       0,
				EncryptionAlgorithm: 0,
			},
		},
		Opc: &WebUI.Opc{
			OpcValue:            "",
			EncryptionKey:       0,
			EncryptionAlgorithm: 0,
		},
		AuthenticationManagementField: "8000",
	},

	AccessAndMobilitySubscriptionData: models.AccessAndMobilitySubscriptionData{
		Gpsis: []string{"msisdn-"},
		SubscribedUeAmbr: &models.AmbrRm{
			Uplink:   "1 Gbps",
			Downlink: "2 Gbps",
		},
		Nssai: &models.Nssai{
			DefaultSingleNssais: []models.Snssai{
				{Sst: 1, Sd: "010203"},
			},
			SingleNssais: []models.Snssai{
				{Sst: 1, Sd: "112233"},
			},
		},
	},

	SessionManagementSubscriptionData: []models.SessionManagementSubscriptionData{
		{
			SingleNssai: &models.Snssai{
				Sst: 1,
				Sd:  "010203",
			},
			DnnConfigurations: map[string]models.DnnConfiguration{
				"internet": models.DnnConfiguration{
					PduSessionTypes: &models.PduSessionTypes{
						DefaultSessionType:  models.PduSessionType_IPV4,
						AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4},
					},
					SscModes: &models.SscModes{
						DefaultSscMode:  models.SscMode__1,
						AllowedSscModes: []models.SscMode{models.SscMode__2, models.SscMode__3},
					},
					Var5gQosProfile: &models.SubscribedDefaultQos{
						Var5qi: 9,
						Arp: &models.Arp{
							PriorityLevel: 8,
						},
						PriorityLevel: 8,
					},
					SessionAmbr: &models.Ambr{
						Uplink:   "1000 Mbps",
						Downlink: "1000 Mbps",
					},
				},
			},
		},
		{
			SingleNssai: &models.Snssai{
				Sst: 1,
				Sd:  "112233",
			},
			DnnConfigurations: map[string]models.DnnConfiguration{
				"internet": {
					PduSessionTypes: &models.PduSessionTypes{
						DefaultSessionType:  models.PduSessionType_IPV4,
						AllowedSessionTypes: []models.PduSessionType{models.PduSessionType_IPV4},
					},
					SscModes: &models.SscModes{
						DefaultSscMode:  models.SscMode__1,
						AllowedSscModes: []models.SscMode{models.SscMode__2, models.SscMode__3},
					},
					Var5gQosProfile: &models.SubscribedDefaultQos{
						Var5qi: 9,
						Arp: &models.Arp{
							PriorityLevel: 8,
						},
						PriorityLevel: 8,
					},
					SessionAmbr: &models.Ambr{
						Uplink:   "1000 Mbps",
						Downlink: "1000 Mbps",
					},
				},
			},
		},
	},

	SmfSelectionSubscriptionData: models.SmfSelectionSubscriptionData{
		SubscribedSnssaiInfos: map[string]models.SnssaiInfo{
			"01010203": models.SnssaiInfo{
				DnnInfos: []models.DnnInfo{models.DnnInfo{Dnn: "internet"}},
			},
			"01112233": models.SnssaiInfo{
				DnnInfos: []models.DnnInfo{models.DnnInfo{Dnn: "internet"}},
			},
		},
	},

	AmPolicyData: models.AmPolicyData{
		SubscCats: []string{"free5gc"},
	},

	SmPolicyData: models.SmPolicyData{
		SmPolicySnssaiData: map[string]models.SmPolicySnssaiData{
			"01010203": models.SmPolicySnssaiData{
				Snssai: &models.Snssai{Sst: 1, Sd: "010203"},
				SmPolicyDnnData: map[string]models.SmPolicyDnnData{"internet": models.SmPolicyDnnData{
					Dnn: "internet",
				}},
			},
			"01112233": models.SmPolicySnssaiData{
				Snssai: &models.Snssai{Sst: 1, Sd: "112233"},
				SmPolicyDnnData: map[string]models.SmPolicyDnnData{"internet": models.SmPolicyDnnData{
					Dnn: "internet",
				}},
			},
		},
	},

	FlowRules: []WebUI.FlowRule{
		{
			Filter:     "1.1.1.1/32",
			Precedence: 128,
			Snssai:     "01010203",
			Dnn:        "internet",
			QosRef:     1,
		},
		{
			Filter:     "1.1.1.1/32",
			Precedence: 127,
			Snssai:     "01112233",
			Dnn:        "internet",
			QosRef:     2,
		},
	},

	QosFlows: []WebUI.QosFlow{
		{
			Snssai: "01010203",
			Dnn:    "internet",
			QosRef: 1,
			Var5QI: 8,
			MBRUL:  "208 Mbps",
			MBRDL:  "208 Mbps",
			GBRUL:  "108 Mbps",
			GBRDL:  "108 Mbps",
		},
		{
			Snssai: "01112233",
			Dnn:    "internet",
			QosRef: 2,
			Var5QI: 7,
			MBRUL:  "407 Mbps",
			MBRDL:  "407 Mbps",
			GBRUL:  "207 Mbps",
			GBRDL:  "207 Mbps",
		},
	},

	ChargingDatas: []WebUI.ChargingData{
		{
			Snssai:         "01010203",
			Dnn:            "",
			Filter:         "",
			ChargingMethod: "Offline",
			Quota:          "100000",
			UnitCost:       "1",
		},
		{
			Snssai:         "01010203",
			Dnn:            "internet",
			QosRef:         1,
			Filter:         "1.1.1.1/32",
			ChargingMethod: "Offline",
			Quota:          "100000",
			UnitCost:       "1",
		},
		{
			Snssai:         "01112233",
			Dnn:            "",
			Filter:         "",
			ChargingMethod: "Online",
			Quota:          "100000",
			UnitCost:       "1",
		},
		{
			Snssai:         "01112233",
			Dnn:            "internet",
			QosRef:         2,
			Filter:         "1.1.1.1/32",
			ChargingMethod: "Online",
			Quota:          "5000",
			UnitCost:       "1",
		},
	}}
