package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func SingleResourceAlgo(ardsLbIp, ardsLbPort, serverType, requestType, sessionId string, selectedResources SelectedResource, reqCompany, reqTenant int) (handlingResult, handlingResource string) {
	handlingResult, handlingResource = SingleHandling(ardsLbIp, ardsLbPort, serverType, requestType, sessionId, selectedResources, reqCompany, reqTenant)
	return

}

func ReserveSlot(ardsLbIp, ardsLbPort string, slotInfo CSlotInfo) bool {
	url := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/ARDS/resource/%s/concurrencyslot", CreateHost(ardsLbIp, ardsLbPort), slotInfo.ResourceId)
	fmt.Println("URL:>", url)

	slotInfoJson, _ := json.Marshal(slotInfo)
	fmt.Println("request Data:: ", string(slotInfoJson))
	var jsonStr = []byte(slotInfoJson)
	authToken := fmt.Sprintf("Bearer %s", accessToken)
	internalAuthToken := fmt.Sprintf("%d:%d", slotInfo.Tenant, slotInfo.Company)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", authToken)
	req.Header.Set("companyinfo", internalAuthToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	result := string(body)
	fmt.Println("response Body:", result)

	var resConv updateCsReult
	json.Unmarshal(body, &resConv)

	if resConv.IsSuccess == true {
		fmt.Println("Return true")
		return true
	}

	fmt.Println("Return false")
	return false
}

func ClearSlotOnMaxRecerved(ardsLbIp, ardsLbPort, serverType, requestType, sessionId string, resObj Resource) {
	var slotSearchTags = make([]string, 6)

	slotSearchTags[0] = fmt.Sprintf("Tag:SlotInfo:company_%d", resObj.Company)
	slotSearchTags[1] = fmt.Sprintf("Tag:SlotInfo:tenant_%d", resObj.Tenant)
	slotSearchTags[2] = fmt.Sprintf("Tag:SlotInfo:handlingType_%s", requestType)
	slotSearchTags[3] = fmt.Sprintf("Tag:SlotInfo:state_%s", "Reserved")
	slotSearchTags[4] = fmt.Sprintf("Tag:SlotInfo:resourceId_%d", resObj.ResourceId)
	slotSearchTags[5] = fmt.Sprintf("Tag:SlotInfo:objType_%s", "CSlotInfo")

	fmt.Println("slotSearchTags: ", slotSearchTags)
	reservedSlotKeys := RedisSInter(slotSearchTags)
	fmt.Println("reservedSlotKeys: ", reservedSlotKeys)

	reservedSlots := RedisMGet(reservedSlotKeys)

	for _, slot := range reservedSlots {

		fmt.Println(slot)

		var slotObj CSlotInfo
		json.Unmarshal([]byte(slot), &slotObj)

		fmt.Println("Datetime Info" + slotObj.LastReservedTime)
		t, _ := time.Parse(layout, slotObj.LastReservedTime)
		t1 := int(time.Now().Sub(t).Seconds())
		t2 := slotObj.MaxReservedTime
		fmt.Println(fmt.Sprintf("Time Info T1: %d", t1))
		fmt.Println(fmt.Sprintf("Time Info T2: %d", t2))
		if t1 > t2 {
			slotObj.State = "Available"
			slotObj.OtherInfo = "ClearReserved"

			ReserveSlot(ardsLbIp, ardsLbPort, slotObj)
		}
	}
}

func GetReqMetaData(_company, _tenent int, _serverType, _requestType string) (metaObj ReqMetaData, err error) {
	key := fmt.Sprintf("ReqMETA:%d:%d:%s:%s", _tenent, _company, _serverType, _requestType)
	fmt.Println(key)
	var strMetaObj string
	strMetaObj, err = RedisGet_v1(key)

	fmt.Println(strMetaObj)
	json.Unmarshal([]byte(strMetaObj), &metaObj)

	return
}

func GetResourceState(_company, _tenant, _resId int) (state string, mode string, err error) {
	key := fmt.Sprintf("ResourceState:%d:%d:%d", _tenant, _company, _resId)
	fmt.Println(key)
	var strResStateObj string

	strResStateObj, err = RedisGet_v1(key)

	fmt.Println(strResStateObj)

	var resStatus ResourceStatus
	json.Unmarshal([]byte(strResStateObj), &resStatus)
	state = resStatus.State
	mode = resStatus.Mode
	return
}

func HandlingResources(Company, Tenant, ResourceCount int, ArdsLbIp, ArdsLbPort, SessionId, ServerType, RequestType, HandlingAlgo, OtherInfo string, selectedResources SelectedResource) (handlingResult string, handlingResource []string) {

	handlingResult = ""
	handlingResource = make([]string, 0)

	switch HandlingAlgo {
	case "SINGLE":
		var singleHandlingResource string
		handlingResult, singleHandlingResource = SingleResourceAlgo(ArdsLbIp, ArdsLbPort, ServerType, RequestType, SessionId, selectedResources, Company, Tenant)
		handlingResource = append(handlingResource, singleHandlingResource)
	case "MULTIPLE":
		fmt.Println("ReqOtherInfo:", OtherInfo)
		resCount := ResourceCount
		fmt.Println("GetRequestedResCount:", resCount)
		handlingResult, handlingResource = MultipleHandling(ArdsLbIp, ArdsLbPort, ServerType, RequestType, SessionId, selectedResources, resCount, Company, Tenant)
	default:
		handlingResult = ""
		handlingResource = make([]string, 0)
	}

	return
}