package data

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/mongoapi"
	"github.com/free5gc/webconsole/backend/WebUI"
	"github.com/free5gc/webconsole/backend/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Default configuration - no longer needed for direct DB access
var (
	defaultTenantId = "" // Will be set to admin tenant ID
)

// Initialize admin tenant and get tenant ID
func InitializeAdminTenant() error {
	tenantData, err := mongoapi.RestfulAPIGetOne("tenantData", bson.M{"tenantName": "admin"})
	if err != nil {
		return fmt.Errorf("failed to get admin tenant: %v", err)
	}
	if len(tenantData) == 0 {
		return fmt.Errorf("admin tenant not found in database")
	}

	defaultTenantId = tenantData["tenantId"].(string)
	return nil
}

// NextIMSI function copied from nextimsi.go
func NextIMSI(imsi string) (string, error) {
	// Check if IMSI has the correct prefix
	if !strings.HasPrefix(imsi, "imsi-") {
		return "", fmt.Errorf("invalid IMSI format: must start with 'imsi-'")
	}

	// Remove the "imsi-" prefix
	imsiNumber := strings.TrimPrefix(imsi, "imsi-")

	// Check if we have at least 15 digits (5 + 10)
	if len(imsiNumber) < 15 {
		return "", fmt.Errorf("invalid IMSI format: must have at least 15 digits after 'imsi-'")
	}

	// Split into slide ID (first 5 digits) and subscriber number (last 10 digits)
	slideID := imsiNumber[:5]
	subscriberNumber := imsiNumber[len(imsiNumber)-10:]

	// Convert the last 10 digits to integer
	num, err := strconv.ParseInt(subscriberNumber, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid subscriber number format: %v", err)
	}

	// Increment by 1
	nextNum := num + 1

	// Handle overflow (if we exceed 10 digits)
	if nextNum > 9999999999 {
		return "", fmt.Errorf("subscriber number overflow: cannot increment beyond 9999999999")
	}

	// Format back to 10 digits with leading zeros
	nextSubscriberNumber := fmt.Sprintf("%010d", nextNum)

	// Reconstruct the IMSI
	// Handle case where there might be middle digits between slideID and subscriber number
	var middlePart string
	if len(imsiNumber) > 15 {
		middlePart = imsiNumber[5 : len(imsiNumber)-10]
	}

	nextIMSI := fmt.Sprintf("imsi-%s%s%s", slideID, middlePart, nextSubscriberNumber)

	return nextIMSI, nil
}
func PostSub(subsData *WebUI.SubsData, ueId string, servingPlmnId string) (err error) {
	filterUeIdOnly := bson.M{"ueId": ueId}
	filter := bson.M{"ueId": ueId, "servingPlmnId": servingPlmnId}
	subsData.UeId = ueId

	webAuthSubsBsonM := WebUI.ToBsonM(subsData.WebAuthenticationSubscription)
	webAuthSubsBsonM["ueId"] = ueId

	authSubs, errModel := WebUI.WebAuthSubToModels(subsData.WebAuthenticationSubscription)
	if errModel != nil {
		logger.ProcLog.Errorf("WebAuthSubToModels err: %+v", errModel)
		err = errModel
		return
	}
	authSubsBsonM := WebUI.ToBsonM(authSubs)
	authSubsBsonM["ueId"] = ueId

	webAuthSubsBsonM["tenantId"] = defaultTenantId
	authSubsBsonM["tenantId"] = defaultTenantId

	amDataBsonM := WebUI.ToBsonM(subsData.AccessAndMobilitySubscriptionData)
	amDataBsonM["ueId"] = ueId
	amDataBsonM["servingPlmnId"] = servingPlmnId
	amDataBsonM["tenantId"] = defaultTenantId

	// Replace all data with new one
	if err = mongoapi.RestfulAPIDeleteMany(smDataColl, filter); err != nil {
		logger.ProcLog.Errorf("PutSubscriberByID err: %+v", err)
		return
	}
	for _, data := range subsData.SessionManagementSubscriptionData {
		smDataBsonM := WebUI.ToBsonM(data)
		smDataBsonM["ueId"] = ueId
		smDataBsonM["servingPlmnId"] = servingPlmnId
		filterSmData := bson.M{"ueId": ueId, "servingPlmnId": servingPlmnId, "snssai": data.SingleNssai}
		if _, err = mongoapi.RestfulAPIPutOne(smDataColl, filterSmData, smDataBsonM); err != nil {
			logger.ProcLog.Errorf("PutSubscriberByID err: %+v", err)
			return
		}
	}

	for key, SnssaiData := range subsData.SmPolicyData.SmPolicySnssaiData {
		tmpSmPolicyDnnData := make(map[string]models.SmPolicyDnnData)
		for dnnKey, dnn := range SnssaiData.SmPolicyDnnData {
			escapedDnn := strings.ReplaceAll(dnnKey, ".", "_")
			tmpSmPolicyDnnData[escapedDnn] = dnn
		}
		SnssaiData.SmPolicyDnnData = tmpSmPolicyDnnData
		subsData.SmPolicyData.SmPolicySnssaiData[key] = SnssaiData
	}

	smfSelSubsBsonM := WebUI.ToBsonM(subsData.SmfSelectionSubscriptionData)
	smfSelSubsBsonM["ueId"] = ueId
	smfSelSubsBsonM["servingPlmnId"] = servingPlmnId
	amPolicyDataBsonM := WebUI.ToBsonM(subsData.AmPolicyData)
	amPolicyDataBsonM["ueId"] = ueId
	smPolicyDataBsonM := WebUI.ToBsonM(subsData.SmPolicyData)
	smPolicyDataBsonM["ueId"] = ueId

	if len(subsData.FlowRules) == 0 {
		logger.ProcLog.Infoln("No Flow Rule")
	} else {
		flowRulesBsonA := make([]any, 0, len(subsData.FlowRules))
		for _, flowRule := range subsData.FlowRules {
			flowRuleBsonM := WebUI.ToBsonM(flowRule)
			flowRuleBsonM["ueId"] = ueId
			flowRuleBsonM["servingPlmnId"] = servingPlmnId
			flowRulesBsonA = append(flowRulesBsonA, flowRuleBsonM)
		}
		if err = mongoapi.RestfulAPIPostMany(flowRuleDataColl, filter, flowRulesBsonA); err != nil {
			logger.ProcLog.Errorf("PutSubscriberByID err: %+v", err)
			return
		}
	}

	if len(subsData.QosFlows) == 0 {
		logger.ProcLog.Infoln("No QoS Flow")
	} else {
		qosFlowBsonA := make([]any, 0, len(subsData.QosFlows))
		for _, qosFlow := range subsData.QosFlows {
			qosFlowBsonM := WebUI.ToBsonM(qosFlow)
			qosFlowBsonM["ueId"] = ueId
			qosFlowBsonM["servingPlmnId"] = servingPlmnId
			qosFlowBsonA = append(qosFlowBsonA, qosFlowBsonM)
		}
		if err = mongoapi.RestfulAPIPostMany(qosFlowDataColl, filter, qosFlowBsonA); err != nil {
			logger.ProcLog.Errorf("PostSubscriberByID err: %+v", err)
			return
		}
	}

	if len(subsData.ChargingDatas) == 0 {
		logger.ProcLog.Infoln("No Charging Data")
	} else {
		for _, chargingData := range subsData.ChargingDatas {
			var previousChargingData WebUI.ChargingData
			var chargingFilter primitive.M

			chargingDataBsonM := WebUI.ToBsonM(chargingData)
			// Clear quota for offline charging flow
			if chargingData.ChargingMethod == WebUI.ChargingOffline {
				chargingDataBsonM["quota"] = "0"
			}

			if chargingData.Dnn != "" && chargingData.Filter != "" {
				// Flow-level charging
				chargingFilter = bson.M{
					"ueId": ueId, "servingPlmnId": servingPlmnId,
					"snssai": chargingData.Snssai,
					"dnn":    chargingData.Dnn,
					"qosRef": chargingData.QosRef,
					"filter": chargingData.Filter,
				}
			} else {
				// Slice-level charging
				chargingFilter = bson.M{
					"ueId": ueId, "servingPlmnId": servingPlmnId,
					"snssai": chargingData.Snssai,
					"qosRef": chargingData.QosRef,
					"dnn":    "",
					"filter": "",
				}
				chargingDataBsonM["dnn"] = ""
				chargingDataBsonM["filter"] = ""
			}
			var previousChargingDataInterface map[string]any
			previousChargingDataInterface, err = mongoapi.RestfulAPIGetOne(chargingDataColl, chargingFilter)
			if err != nil {
				logger.ProcLog.Errorf("PostSubscriberByID err: %+v", err)
				return
			}
			err = json.Unmarshal(WebUI.MapToByte(previousChargingDataInterface), &previousChargingData)
			if err != nil {
				logger.ProcLog.Error(err)
				return
			}

			ratingGroup := previousChargingDataInterface["ratingGroup"]
			if ratingGroup != nil {
				rg := ratingGroup.(int32)
				chargingDataBsonM["ratingGroup"] = rg
				if previousChargingData.Quota != chargingData.Quota {
					WebUI.SendRechargeNotification(ueId, rg)
				}
			}

			chargingDataBsonM["ueId"] = ueId
			chargingDataBsonM["servingPlmnId"] = servingPlmnId

			if _, err_put := mongoapi.RestfulAPIPutOne(chargingDataColl, chargingFilter, chargingDataBsonM); err_put != nil {
				logger.ProcLog.Errorf("PostSubscriberByID err: %+v", err_put)
				err = err_put
				return
			}
		}
	}

	if _, err = mongoapi.RestfulAPIPutOne(authWebSubsDataColl, filterUeIdOnly, webAuthSubsBsonM); err != nil {
		logger.ProcLog.Errorf("PutSubscriberByID err: %+v", err)
		return
	}
	if _, err = mongoapi.RestfulAPIPutOne(authSubsDataColl, filterUeIdOnly, authSubsBsonM); err != nil {
		logger.ProcLog.Errorf("PutSubscriberByID err: %+v", err)
		return
	}
	if _, err = mongoapi.RestfulAPIPutOne(amDataColl, filter, amDataBsonM); err != nil {
		logger.ProcLog.Errorf("PutSubscriberByID err: %+v", err)
		return
	}
	if _, err = mongoapi.RestfulAPIPutOne(smfSelDataColl, filter, smfSelSubsBsonM); err != nil {
		logger.ProcLog.Errorf("PutSubscriberByID err: %+v", err)
		return
	}
	if _, err = mongoapi.RestfulAPIPutOne(amPolicyDataColl, filterUeIdOnly, amPolicyDataBsonM); err != nil {
		logger.ProcLog.Errorf("PutSubscriberByID err: %+v", err)
		return
	}
	if _, err = mongoapi.RestfulAPIPutOne(smPolicyDataColl, filterUeIdOnly, smPolicyDataBsonM); err != nil {
		logger.ProcLog.Errorf("PutSubscriberByID err: %+v", err)
		return
	}
	return err
}
