package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

func CalculateWeight(reqAttributeInfo []ReqAttributeData, resAttributeInfo []ResAttributeData) float64 {
	var calculatedWeight float64
	calculatedWeight = 0.00
	for _, reqAtt := range reqAttributeInfo {
		if len(reqAtt.AttributeCode) > 0 {
			attCode := reqAtt.AttributeCode[0]

			for _, resAtt := range resAttributeInfo {
				if attCode == resAtt.Attribute && resAtt.HandlingType == reqAtt.HandlingType {

					reqAttPrecentage, _ := strconv.ParseFloat(reqAtt.WeightPrecentage, 64)
					fmt.Println("**********reqAttPrecentage:", reqAttPrecentage)
					fmt.Println("**********resAttPrecentage:", resAtt.Percentage)
					reqWeight := reqAttPrecentage / 100.00
					resAttWeight := resAtt.Percentage / 100.00
					calculatedWeight = calculatedWeight + (reqWeight * resAttWeight)
				}
			}
		}
	}
	return calculatedWeight
}

func WeightBaseSelection(_requests []Request) (result []SelectionResult) {

	var selectedResources = make([]SelectionResult, len(_requests))

	for i, reqObj := range _requests {

		selectedResources[i].Request = reqObj.SessionId

		var resourceWeightInfo = make([]WeightBaseResourceInfo, 0)
		var matchingResources = make([]string, 0)
		if len(reqObj.AttributeInfo) > 0 {
			var resourceSearchTags = make([]string, 3)

			resourceSearchTags[0] = fmt.Sprintf("Tag:Resource:company_%d", reqObj.Company)
			resourceSearchTags[1] = fmt.Sprintf("Tag:Resource:tenant_%d", reqObj.Tenant)
			resourceSearchTags[2] = fmt.Sprintf("Tag:Resource:objType_%s", "Resource")

			for _, value := range reqObj.AttributeInfo {
				for _, att := range value.AttributeCode {
					resourceSearchTags = append(resourceSearchTags, fmt.Sprintf("Tag:Resource:%s:attribute_%s", reqObj.RequestType, att))
				}
			}


			fmt.Println("resourceSearchTags: ", resourceSearchTags)
			searchResourceKeys := RedisSInter(resourceSearchTags)
			fmt.Println("searchResourceKeys: ", searchResourceKeys)

			searchResources := RedisMGet(searchResourceKeys)

			for _, resource := range searchResources {
				fmt.Println(resource)

				var resObj Resource
				json.Unmarshal([]byte(resource), &resObj)

				if resObj.ResourceName != "" {
					concInfo, err := GetConcurrencyInfo(resObj.Company, resObj.Tenant, resObj.ResourceId, reqObj.RequestType)
					calcWeight := CalculateWeight(reqObj.AttributeInfo, resObj.ResourceAttributeInfo)
					resKey := fmt.Sprintf("Resource:%d:%d:%d", resObj.Tenant, resObj.Company, resObj.ResourceId)
					var tempWeightInfo WeightBaseResourceInfo
					tempWeightInfo.ResourceId = resKey
					tempWeightInfo.Weight = calcWeight
					if err != nil {
						fmt.Println("Error in GetConcurrencyInfo")
						tempWeightInfo.LastConnectedTime = ""
					} else {
						if concInfo.LastConnectedTime == "" {
							tempWeightInfo.LastConnectedTime = ""
						} else {
							tempWeightInfo.LastConnectedTime = concInfo.LastConnectedTime
						}
					}
					resourceWeightInfo = append(resourceWeightInfo, tempWeightInfo)
				}
			}

			sort.Sort(ByWaitingTime(resourceWeightInfo))
			for _, res := range resourceWeightInfo {
				matchingResources = AppendIfMissingString(matchingResources, res.ResourceId)
				logWeight := fmt.Sprintf("###################################### %s --------- %f --------%s", res.ResourceId, res.Weight,res.LastConnectedTime)
				fmt.Println(logWeight)
			}

		}
		selectedResources[i].Resources.Priority = matchingResources
	}
	return selectedResources
}
