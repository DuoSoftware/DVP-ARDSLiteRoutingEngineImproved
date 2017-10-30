package main

import (
	"encoding/json"
	"fmt"
	"sort"
)

func BasicThresholdSelection(_requests []Request) (result []SelectionResult) {

	var selectedResources = make([]SelectionResult, len(_requests))

	for i, reqObj := range _requests {

		selectedResources[i].Request = reqObj.SessionId

		var matchingResources = make([]string, 0)
		var matchingThresholdResources = make([]string, 0)

		if len(reqObj.AttributeInfo) > 0 {

			var resourceConcInfo = make([]ConcurrencyInfo, 0)
			var resourceThresholdConcInfo = make([]ConcurrencyInfo, 0)

			var resourceSearchTags = make([]string, 3)

			resourceSearchTags[0] = fmt.Sprintf("Tag:Resource:company_%d", reqObj.Company)
			resourceSearchTags[1] = fmt.Sprintf("Tag:Resource:tenant_%d", reqObj.Tenant)
			resourceSearchTags[2] = fmt.Sprintf("Tag:Resource:objType_%s", "Resource")

			for _, value := range reqObj.AttributeInfo {
				for _, att := range value.AttributeCode {
					resourceSearchTags = append(resourceSearchTags, fmt.Sprintf("Tag:Resource:%s:attribute_%s", reqObj.RequestType, att))
				}
			}


			fmt.Println(resourceSearchTags)
			searchResourceKeys := RedisSInter(resourceSearchTags)
			fmt.Println("searchResourceKeys: ", searchResourceKeys)

			searchResources := RedisMGet(searchResourceKeys)

			for _, resource := range searchResources {
				fmt.Println(resource)

				var resObj Resource
				json.Unmarshal([]byte(resource), &resObj)

				_attAvailable, _isThreshold := IsAttributeAvailable(reqObj.AttributeInfo, resObj.ResourceAttributeInfo, reqObj.RequestType)

				if resObj.ResourceName != "" && _attAvailable {
					concInfo, err := GetConcurrencyInfo(resObj.Company, resObj.Tenant, resObj.ResourceId, reqObj.RequestType)
					if err != nil {
						fmt.Println("Error in GetConcurrencyInfo")
					} else {
						if _isThreshold {
							resourceThresholdConcInfo = append(resourceThresholdConcInfo, concInfo)
						} else {
							resourceConcInfo = append(resourceConcInfo, concInfo)
						}
					}
				}
			}

			sort.Sort(timeSlice(resourceConcInfo))
			sort.Sort(timeSlice(resourceThresholdConcInfo))

			for _, res := range resourceConcInfo {
				resKey := fmt.Sprintf("Resource:%d:%d:%d", res.Tenant, res.Company, res.ResourceId)
				matchingResources = AppendIfMissingString(matchingResources, resKey)
				fmt.Println(resKey)
			}

			for _, res := range resourceThresholdConcInfo {
				resKey := fmt.Sprintf("Resource:%d:%d:%d", res.Tenant, res.Company, res.ResourceId)
				matchingThresholdResources = AppendIfMissingString(matchingThresholdResources, resKey)
				fmt.Println(resKey)
			}

		}
		selectedResources[i].Resources.Priority = matchingResources
		selectedResources[i].Resources.Threshold = matchingThresholdResources
	}
	return selectedResources

}
