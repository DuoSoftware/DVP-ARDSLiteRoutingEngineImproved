package main

import (
	"encoding/json"
	"fmt"
)

func LocationBaseSelection(_requests []Request) (result []SelectionResult) {
	fmt.Println("-----------Start Location base----------------")

	var selectedResources = make([]SelectionResult, len(_requests))

	for i, reqObj := range _requests {

		selectedResources[i].Request = reqObj.SessionId

		var matchingResources = make([]string, 0)
		if reqObj.OtherInfo != "" {

			var locationObj ReqLocationData
			json.Unmarshal([]byte(reqObj.OtherInfo), &locationObj)

			fmt.Println("reqOtherInfo:: ", locationObj)

			if locationObj != (ReqLocationData{}) {
				fmt.Println("Start Get locations")
				locationResult := RedisGeoRadius(reqObj.Tenant, reqObj.Company, locationObj)
				fmt.Println("locations:: ", locationResult)

				subReplys, _ := locationResult.Array()
				for _, lor := range subReplys {

					resourceLocInfo, _ := lor.List()

					if len(resourceLocInfo) > 1 {
						issMapKey := fmt.Sprintf("ResourceIssMap:%d:%d:%s", reqObj.Tenant, reqObj.Company, resourceLocInfo[0])
						fmt.Println("start map iss: ", issMapKey)
						resourceKey := RedisGet(issMapKey)
						fmt.Println("resourceKey: ", resourceKey)
						if resourceKey != "" {

							strResObj := RedisGet(resourceKey)
							fmt.Println(strResObj)

							var resObj Resource
							json.Unmarshal([]byte(strResObj), &resObj)

							if resObj.ResourceName != "" {
								resKey := fmt.Sprintf("Resource:%d:%d:%d", resObj.Tenant, resObj.Company, resObj.ResourceId)
								if len(reqObj.AttributeInfo) > 0 {
									_attAvailable, _ := IsAttributeAvailable(reqObj.AttributeInfo, resObj.ResourceAttributeInfo, reqObj.RequestType)
									if _attAvailable {
										matchingResources = AppendIfMissingString(matchingResources, resKey)
									}
								} else {
									matchingResources = AppendIfMissingString(matchingResources, resKey)
								}
							}
						}
					}
				}

				selectedResources[i].Resources.Priority = matchingResources
			}

		}
	}

	return selectedResources

}
