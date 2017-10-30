package main

import (
	"encoding/json"
	"fmt"
)

func IsAttributeAvailable(reqAttributeInfo []ReqAttributeData, resAttributeInfo []ResAttributeData, reqType string) (isAttrAvailable, isThreshold bool) {

	isAttrAvailable = true
	isThreshold = false

	reqAttributeAvailability := make(map[string]bool)
	reqAttributes := make(map[string]string)

	for _, reqAtt := range reqAttributeInfo {
		if len(reqAtt.AttributeCode) > 0 {
			attCode := reqAtt.AttributeCode[0]
			reqAttributeAvailability[attCode] = false
			reqAttributes[attCode] = attCode
		}
	}

	for _, resAtt := range resAttributeInfo {
		if resAtt.Attribute == reqAttributes[resAtt.Attribute] && resAtt.HandlingType == reqType && !reqAttributeAvailability[resAtt.Attribute] {
			if resAtt.Percentage > 0 {
				reqAttributeAvailability[resAtt.Attribute] = true

				if resAtt.Percentage > 0 && resAtt.Percentage <= 25 {
					isThreshold = true
				}
			}
		}
	}

	fmt.Println("Check Attribute Availability:: ", reqAttributeAvailability)

	for _, availability := range reqAttributeAvailability {
		isAttrAvailable = isAttrAvailable && availability
	}

	fmt.Println("Check Attribute Availability Return:: isAttrAvailable: ", isAttrAvailable, " isThreshold: ", isThreshold)

	return
}

func GetConcurrencyInfo(_company, _tenant, _resId int, _category string) (ciObj ConcurrencyInfo, err error) {
	key := fmt.Sprintf("ConcurrencyInfo:%d:%d:%s:%s", _tenant, _company, _resId, _category)
	fmt.Println(key)
	var strCiObj string
	strCiObj, err = RedisGet_v1(key)
	fmt.Println(strCiObj)

	json.Unmarshal([]byte(strCiObj), &ciObj)

	return
}

func SelectResources(_requests []Request, _selectionAlgo string) []SelectionResult {
	var selectionResult []SelectionResult

	switch _selectionAlgo {
	case "BASIC":
		selectionResult = BasicSelection(_requests)
		break
	case "BASICTHRESHOLD":
		selectionResult = BasicThresholdSelection(_requests)
		break
	case "WEIGHTBASE":
		selectionResult = WeightBaseSelection(_requests)
		break
	case "LOCATIONBASE":
		selectionResult = LocationBaseSelection(_requests)
		break
	default:
		break
	}

	return selectionResult
}
