package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"log"
)

func GetAllProcessingHashes() []string {
	processingHashSearchKey := fmt.Sprintf("ProcessingHash:%s:%s", "*", "*")
	processingHashes := RedisSearchKeys(processingHashSearchKey)
	return processingHashes
}

func GetAllProcessingItems(_processingHashKey string) []Request {
	fmt.Println(_processingHashKey)
	keyItems := strings.Split(_processingHashKey, ":")

	company := keyItems[1]
	tenant := keyItems[2]
	strHash := RedisHashGetAll(_processingHashKey)

	processingReqObjs := make([]Request, 0)

	for queueId, sessionId := range strHash {
		fmt.Println("queueId:", queueId, "sessionId:", sessionId)
		requestKey := fmt.Sprintf("Request:%s:%s:%s", tenant, company, sessionId)
		strReqObj := RedisGet(requestKey)
		fmt.Println(strReqObj)

		if strReqObj == "" {
			fmt.Println("Start SetNextProcessingItem")
			tenantInt, _ := strconv.Atoi(tenant)
			companyInt, _ := strconv.Atoi(company)
			SetNextProcessingItem(tenantInt, companyInt, _processingHashKey, queueId, sessionId, "")
		} else {
			var reqObj Request
			json.Unmarshal([]byte(strReqObj), &reqObj)

			if reqObj.SessionId == "" {

				fmt.Println("Critical issue request object found empty ---> set next item "+ queueId + "value " + sessionId)

				tenantInt, _ := strconv.Atoi(tenant)
				companyInt, _ := strconv.Atoi(company)
				SetNextProcessingItem(tenantInt, companyInt, _processingHashKey, queueId, sessionId, "")

			}else {

				processingReqObjs = AppendIfMissingReq(processingReqObjs, reqObj)
			}
		}
	}
	return processingReqObjs
}

func GetRejectedQueueId(_queueId string) string {
	rejectQueueId := fmt.Sprintf("%s:REJECTED", _queueId)
	return rejectQueueId
}

func SetNextProcessingItem(tenant, company int, _processingHash, _queueId, currentSession, requestState string) {
	eSession := RedisHashGetValue(_processingHash, _queueId)

	fmt.Println("Item in "+_processingHash+"set next processing item in queue "+_queueId+ " with session "+ currentSession +" has now in hash "+eSession)
	if eSession != "" && eSession == currentSession {
		rejectedQueueId := GetRejectedQueueId(_queueId)
		nextRejectedQueueItem := RedisListLpop(rejectedQueueId)

		if nextRejectedQueueItem == "" {
			nextQueueItem := RedisListLpop(_queueId)
			if nextQueueItem == "" {
				removeHResult := RedisRemoveHashField(_processingHash, _queueId)
				if removeHResult {
					fmt.Println("Remove HashField Success.." + _processingHash + "::" + _queueId)
				} else {
					fmt.Println("Remove HashField Failed.." + _processingHash + "::" + _queueId)
				}
			} else {
				setHResult := RedisHashSetField(_processingHash, _queueId, nextQueueItem)
				if setHResult {
					fmt.Println("Set HashField Success.." + _processingHash + "::" + _queueId + "::" + nextQueueItem)
				} else {
					fmt.Println("Set HashField Failed.." + _processingHash + "::" + _queueId + "::" + nextQueueItem)
				}
			}
		} else {
			setHResult := RedisHashSetField(_processingHash, _queueId, nextRejectedQueueItem)
			if setHResult {
				fmt.Println("Set HashField Success.." + _processingHash + "::" + _queueId + "::" + nextRejectedQueueItem)
			} else {
				fmt.Println("Set HashField Failed.." + _processingHash + "::" + _queueId + "::" + nextRejectedQueueItem)
			}
		}
	} else {

		fmt.Println("session Mismatched, " + requestState + " ignore setNextItem")
		/*there is a new session added to the hash,
		now the item should route on next processing
		process next item will run through status and remove if the status is not queued
		there is a possibility to lost the item if status changes has failed.
		recheck all queue status set methods for concurrency and async operations.
		*/


	}

	defer func() {
		//ReleasetLock(setNextLock, u1)l
	}()
}

func ContinueArdsProcess(_request Request) bool {
	if _request.ReqHandlingAlgo == "QUEUE" && _request.HandlingResource != "No matching resources at the moment" {
		req, _ := json.Marshal(_request)
		authToken := fmt.Sprintf("Bearer %s", accessToken)
		internalAuthToken := fmt.Sprintf("%d:%d", _request.Tenant, _request.Company)
		ardsUrl := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/ARDS/continueprocess", CreateHost(_request.LbIp, _request.LbPort))
		if Post(ardsUrl, string(req[:]), authToken, internalAuthToken) {
			fmt.Println("Continue Ards Process Success")
			return true
		} else {
			fmt.Println("Continue Ards Process Failed")
			return false
		}
	} else {
		return false
	}
}

func GetRequestState(_company, _tenant int, _sessionId string) string {
	reqStateKey := fmt.Sprintf("RequestState:%d:%d:%s", _tenant, _company, _sessionId)
	reqState := RedisGet(reqStateKey)
	return reqState
}

func SetRequestState(_company, _tenant int, _sessionId, _newState string) string {
	reqStateKey := fmt.Sprintf("RequestState:%d:%d:%s", _tenant, _company, _sessionId)
	reqState := RedisSet(reqStateKey, _newState)
	return reqState
}

func ContinueProcessing(_request Request, _selectedResources SelectedResource) (continueProcessingResult bool, handlingResource []string) {
	fmt.Println("ReqOtherInfo:", _request.OtherInfo)
	var result string
	result, handlingResource = HandlingResources(_request.Company, _request.Tenant, _request.ResourceCount, _request.LbIp, _request.LbPort, _request.SessionId, _request.ServerType, _request.RequestType, _request.HandlingAlgo, _request.OtherInfo, _selectedResources)
	_request.HandlingResource = result
	continueProcessingResult = ContinueArdsProcess(_request)
	return
}

func AcquireProcessingHashLock(hashId, uuid string) bool {
	lockKey := fmt.Sprintf("ProcessingHashLock:%s", hashId)
	if RedisSetNx(lockKey, uuid, 60) == true {

		fmt.Println("lockKey: ", lockKey)
		return true

	} else {

		return false

	}
}

func ReleasetLock(hashId, uuid string) {
	lockKey := fmt.Sprintf("ProcessingHashLock:%s", hashId)

	if RedisRemoveRLock(lockKey, uuid) == true {
		fmt.Println("Release lock ", lockKey, "success.")
	} else {
		fmt.Println("Release lock ", lockKey, "failed.")
	}
	return
}

func ExecuteRequestHash(_processingHashKey, uuid string) {
	defer func() {
		ReleasetLock(_processingHashKey, uuid)
	}()

	if RedisCheckKeyExist(_processingHashKey) {
		processingItems := GetAllProcessingItems(_processingHashKey)
		if len(processingItems) > 0 {

			defaultRequest := processingItems[0]

			sort.Sort(ByReqPriority(processingItems))

			selectedResourcesForHash := SelectResources(processingItems, defaultRequest.SelectionAlgo)
			pickedResources := make([]string, 0)

			for _, longestWItem := range processingItems {

				fmt.Println("Execute processing hash item::", longestWItem.Priority)

				if longestWItem.SessionId != "" {
					requestState := GetRequestState(longestWItem.Company, longestWItem.Tenant, longestWItem.SessionId)
					if requestState == "QUEUED" {

						log.Println("pickedResources: ", pickedResources)

						resourceForRequest, isExist := GetSelectedResourceForRequest(selectedResourcesForHash, longestWItem.SessionId, pickedResources)

						log.Println("resourceForRequest: ", resourceForRequest)

						if isExist {
							continueProcessingResult, handlingResource := ContinueProcessing(longestWItem, resourceForRequest)
							if continueProcessingResult{
								log.Println("handlingResource: ", handlingResource)
								pickedResources = append(pickedResources, handlingResource...)
								fmt.Println("Continue ARDS Process Success")
							}
						}else {
							fmt.Println("Request not found in Selected Resource Data")
						}
					} else {
						fmt.Println("State of the queue item" + longestWItem.SessionId + "is not queued ->" + requestState)
						SetNextProcessingItem(longestWItem.Tenant, longestWItem.Company, _processingHashKey, longestWItem.QueueId, longestWItem.SessionId, requestState)
					}
				} else {
					fmt.Println("No Session Found")
				}
			}
		} else {
			fmt.Println("No Processing Items Found")
		}
	} else {
		fmt.Println("No Processing Hash Found")
	}
}

func ExecuteRequestHashWithMsgQueue(_processingHashKey, uuid string) {
	defer func() {

		ReleasetLock(_processingHashKey, uuid)

	}()
	for RedisCheckKeyExist(_processingHashKey) {

		processingItems := GetAllProcessingItems(_processingHashKey)

		if len(processingItems) > 0 {

			defaultRequest := processingItems[0]

			sort.Sort(ByReqPriority(processingItems))

			selectedResourcesForHash := SelectResources(processingItems, defaultRequest.SelectionAlgo)
			pickedResources := make([]string, 0)

			for _, longestWItem := range processingItems {

				fmt.Println("Execute processing hash item::", longestWItem.Priority)

				if longestWItem.SessionId != "" {
					requestState := GetRequestState(longestWItem.Company, longestWItem.Tenant, longestWItem.SessionId)
					if requestState == "QUEUED" {

						resourceForRequest, isExist := GetSelectedResourceForRequest(selectedResourcesForHash, longestWItem.SessionId, pickedResources)
						if isExist {
							continueProcessingResult, handlingResource := ContinueProcessing(longestWItem, resourceForRequest)
							if continueProcessingResult{
								pickedResources = append(pickedResources, handlingResource...)
								fmt.Println("Continue ARDS Process Success")
							}
						}else {
							fmt.Println("Request not found in Selected Resource Data")
						}
					} else {

						SetNextProcessingItem(longestWItem.Tenant, longestWItem.Company, _processingHashKey, longestWItem.QueueId, longestWItem.SessionId, requestState)
					}
				} else {

					fmt.Println("No Session Found")
				}
			}
		} else {

			fmt.Println("No Processing Items Found")
		}
	}
}
