package main

import (
	"encoding/json"
	"fmt"
)

func SingleHandling(ardsLbIp, ardsLbPort, serverType, requestType, sessionId string, selectedResources SelectedResource, reqCompany, reqTenant int) (handlingResult, handlingResource string) {
	return SelectHandlingResource(ardsLbIp, ardsLbPort, serverType, requestType, sessionId, selectedResources, reqCompany, reqTenant)
}

func SelectHandlingResource(ardsLbIp, ardsLbPort, serverType, requestType, sessionId string, selectedResources SelectedResource, reqCompany, reqTenant int) (handlingResult, handlingResource string) {
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

		fmt.Println("Start GetConcurrencyInfo")
		conInfo, cErr := GetConcurrencyInfo(resObj.Company, resObj.Tenant, resObj.ResourceId, requestType)
		fmt.Println("End GetConcurrencyInfo")
		fmt.Println("Start GetReqMetaData")
		metaData, mErr := GetReqMetaData(reqCompany, reqTenant, serverType, requestType)
		fmt.Println("End GetReqMetaData")
		fmt.Println("Start GetResourceState")
		resState, resMode, sErr := GetResourceState(resObj.Company, resObj.Tenant, resObj.ResourceId)
		fmt.Println("Start GetResourceState")

		fmt.Println("conInfo.RejectCount:: ", conInfo.RejectCount)
		fmt.Println("conInfo.IsRejectCountExceeded:: ", conInfo.IsRejectCountExceeded)
		fmt.Println("metaData.MaxRejectCount:: ", metaData.MaxRejectCount)

		if cErr == nil {

			if mErr == nil {

				if sErr == nil {

					if resState == "Available" && resMode == "Inbound" && conInfo.RejectCount < metaData.MaxRejectCount && conInfo.IsRejectCountExceeded == false {
						fmt.Println("===========================================Start====================================================")
						ClearSlotOnMaxRecerved(ardsLbIp, ardsLbPort, serverType, requestType, sessionId, resObj)

						var slotSearchTags = make([]string, 8)

						slotSearchTags[0] = fmt.Sprintf("Tag:SlotInfo:company_%d", resObj.Company)
						slotSearchTags[1] = fmt.Sprintf("Tag:SlotInfo:tenant_%d", resObj.Tenant)
						slotSearchTags[4] = fmt.Sprintf("Tag:SlotInfo:handlingType_%s", requestType)
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
								fmt.Println("Return resource Data:", resObj.OtherInfo)
								handlingResult = conInfo.RefInfo
								handlingResource = resourceKey
								return
							}
						}
					}
				}
			}
		}

	}
	handlingResult = "No matching resources at the moment"
	handlingResource = ""
	return
}
