package main

import (
	"encoding/json"
	"fmt"
)

func MultipleHandling(ardsLbIp, ardsLbPort, ServerType, RequestType, sessionId string, selectedResources SelectedResource, nuOfResRequested, reqCompany, reqTenant int) (handlingResult string, handlingResource []string) {
	return SelectMultipleHandlingResource(ardsLbIp, ardsLbPort, ServerType, RequestType, sessionId, selectedResources, nuOfResRequested, reqCompany, reqTenant)
}

func SelectMultipleHandlingResource(ardsLbIp, ardsLbPort, ServerType, RequestType, sessionId string, selectedResources SelectedResource, nuOfResRequested, reqCompany, reqTenant int) (handlingResult string, handlingResource []string) {
	selectedResList := make([]string, 0)
	selectedResKeyList := make([]string, 0)
	resourceIds := append(selectedResources.Priority, selectedResources.Threshold...)
	fmt.Println("///////////////////////////////////////selectedResources/////////////////////////////////////////////////")
	fmt.Println("Priority:: ", selectedResources.Priority)
	fmt.Println("Threshold:: ", selectedResources.Threshold)
	fmt.Println("ResourceIds:: ", resourceIds)

	resources := RedisMGet(resourceIds)

	for _, strResObj := range resources {

		fmt.Println(strResObj)

		var resObj Resource
		json.Unmarshal([]byte(strResObj), &resObj)

		resourceKey := fmt.Sprintf("Resource:%d:%d:%d", resObj.Tenant, resObj.Company, resObj.ResourceId)
		conInfo, cErr := GetConcurrencyInfo(resObj.Company, resObj.Tenant, resObj.ResourceId, RequestType)

		fmt.Println("conInfo.RejectCount:: ", conInfo.RejectCount)
		fmt.Println("conInfo.IsRejectCountExceeded:: ", conInfo.IsRejectCountExceeded)

		if cErr == nil {
			metaData, mErr := GetReqMetaData(reqCompany, reqTenant, ServerType, RequestType)
			fmt.Println("metaData.MaxRejectCount:: ", metaData.MaxRejectCount)

			if mErr == nil {
				resState, resMode, sErr := GetResourceState(resObj.Company, resObj.Tenant, resObj.ResourceId)
				if sErr == nil {

					if resState == "Available" && resMode == "Inbound" && conInfo.RejectCount < metaData.MaxRejectCount && conInfo.IsRejectCountExceeded == false {
						ClearSlotOnMaxRecerved(ardsLbIp, ardsLbPort, ServerType, RequestType, sessionId, resObj)

						var slotSearchTags = make([]string, 8)

						slotSearchTags[0] = fmt.Sprintf("Tag:SlotInfo:company_%d", resObj.Company)
						slotSearchTags[1] = fmt.Sprintf("Tag:SlotInfo:tenant_%d", resObj.Tenant)
						slotSearchTags[4] = fmt.Sprintf("Tag:SlotInfo:handlingType_%s", RequestType)
						slotSearchTags[5] = fmt.Sprintf("Tag:SlotInfo:state_%s", "Available")
						slotSearchTags[6] = fmt.Sprintf("Tag:SlotInfo:resourceId_%d", resObj.ResourceId)
						slotSearchTags[7] = fmt.Sprintf("Tag:SlotInfo:objType_%s", "CSlotInfo")


						fmt.Println("slotSearchTags: ", slotSearchTags)
						availableSlotKeys := RedisSInter(slotSearchTags)
						fmt.Println("availableSlotKeys: ", availableSlotKeys)

						availableSlots := RedisMGet(availableSlotKeys)

						for _, strSlotObj := range availableSlots {
							fmt.Println(strSlotObj)

							var slotObj CSlotInfo
							json.Unmarshal([]byte(strSlotObj), &slotObj)

							slotObj.State = "Reserved"
							slotObj.SessionId = sessionId
							slotObj.OtherInfo = "Inbound"
							slotObj.MaxReservedTime = metaData.MaxReservedTime
							slotObj.MaxAfterWorkTime = metaData.MaxAfterWorkTime
							slotObj.MaxFreezeTime = metaData.MaxFreezeTime
							slotObj.TempMaxRejectCount = metaData.MaxRejectCount

							if ReserveSlot(ardsLbIp, ardsLbPort, slotObj) == true {
								fmt.Println("Return resource Data:", conInfo.RefInfo)
								selectedResList = AppendIfMissingString(selectedResList, conInfo.RefInfo)
								selectedResKeyList = AppendIfMissingString(selectedResKeyList, resourceKey)
								if len(selectedResList) == nuOfResRequested {
									selectedResListString, _ := json.Marshal(selectedResList)
									handlingResult = string(selectedResListString)
									handlingResource = selectedResKeyList
								}
							}
						}
					}
				}
			}
		}

	}
	handlingResult = "No matching resources at the moment"
	handlingResource = make([]string, 0)
	return
}
