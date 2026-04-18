package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/application"
)

func TestDynamoDBTargetRegistryDistinguishesSupportedAndUnsupportedOperations(t *testing.T) {
	t.Helper()

	handler := newDynamoDBNativeHandler(application.New())

	for _, target := range []string{"ListTables", "CreateTable", "DescribeTable", "DeleteTable", "GetItem", "PutItem", "UpdateItem", "Query", "Scan", "DeleteItem"} {
		spec, ok := handler.registry[target]
		if !ok {
			t.Fatalf("expected target %q to be registered", target)
		}
		if !spec.supported {
			t.Fatalf("expected target %q to be supported", target)
		}
	}

	for _, target := range []string{"BatchGetItem", "TransactWriteItems"} {
		spec, ok := handler.registry[target]
		if !ok {
			t.Fatalf("expected target %q to be registered", target)
		}
		if spec.supported {
			t.Fatalf("expected target %q to be marked unsupported", target)
		}
	}
}

func TestDynamoDBNativeRoutesHandleSupportedLocalSubset(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterDynamoDBNativeRoutes(engine, application.New())

	listTables := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.ListTables",
		body:   `{}`,
	})
	if got, want := listTables.code, http.StatusOK; got != want {
		t.Fatalf("unexpected list tables status: got %d want %d", got, want)
	}
	if got, want := listTables.contentType, dynamoDBJSONContentType; got != want {
		t.Fatalf("unexpected content type: got %q want %q", got, want)
	}
	var listTablesResponse listTablesResponse
	decodeResponse(t, listTables.body, &listTablesResponse)
	if got, want := len(listTablesResponse.TableNames), 1; got != want {
		t.Fatalf("unexpected table count: got %d want %d", got, want)
	}
	if got, want := listTablesResponse.TableNames[0], "mildstack-records"; got != want {
		t.Fatalf("unexpected seeded table name: got %q want %q", got, want)
	}

	createTable := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.CreateTable",
		body: `{
			"TableName":"mildstack-archive",
			"KeySchema":[{"AttributeName":"id","KeyType":"HASH"}],
			"AttributeDefinitions":[{"AttributeName":"id","AttributeType":"S"}],
			"BillingMode":"PAY_PER_REQUEST"
		}`,
	})
	if got, want := createTable.code, http.StatusOK; got != want {
		t.Fatalf("unexpected create table status: got %d want %d", got, want)
	}
	var createTableResponse createTableResponse
	decodeResponse(t, createTable.body, &createTableResponse)
	if got, want := createTableResponse.TableDescription.TableName, "mildstack-archive"; got != want {
		t.Fatalf("unexpected created table name: got %q want %q", got, want)
	}
	if got, want := createTableResponse.TableDescription.TableStatus, "CREATING"; got != want {
		t.Fatalf("unexpected table status: got %q want %q", got, want)
	}
	if got := createTableResponse.TableDescription.CreationDateTime; got <= 0 {
		t.Fatalf("expected create table creation date time to be populated, got %d", got)
	}

	describeTable := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DescribeTable",
		body: `{
			"TableName":"mildstack-archive"
		}`,
	})
	if got, want := describeTable.code, http.StatusOK; got != want {
		t.Fatalf("unexpected creating describe table status: got %d want %d", got, want)
	}
	var describeTableResponse describeTableResponse
	decodeResponse(t, describeTable.body, &describeTableResponse)
	if got, want := describeTableResponse.Table.TableStatus, "CREATING"; got != want {
		t.Fatalf("unexpected creating table status: got %q want %q", got, want)
	}

	putItem := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.PutItem",
		body: `{
			"TableName":"mildstack-archive",
			"Item":{
				"id":{"S":"item#1"},
				"title":{"S":"archive item"},
				"version":{"N":"1"},
				"obsolete":{"S":"remove me"}
			}
		}`,
	})
	if got, want := putItem.code, http.StatusOK; got != want {
		t.Fatalf("unexpected put item status: got %d want %d", got, want)
	}

	updateItem := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.UpdateItem",
		body: `{
			"TableName":"mildstack-archive",
			"Key":{"id":{"S":"item#1"}},
			"UpdateExpression":"SET title = :title ADD version :inc REMOVE obsolete",
			"ExpressionAttributeValues":{
				":title":{"S":"updated archive item"},
				":inc":{"N":"1"}
			},
			"ReturnValues":"ALL_NEW"
		}`,
	})
	if got, want := updateItem.code, http.StatusOK; got != want {
		t.Fatalf("unexpected update item status: got %d want %d", got, want)
	}
	var updateItemResponse updateItemResponse
	decodeResponse(t, updateItem.body, &updateItemResponse)
	if got, want := updateItemResponse.Attributes["title"].S, "updated archive item"; got != want {
		t.Fatalf("unexpected updated title: got %q want %q", got, want)
	}
	if got, want := updateItemResponse.Attributes["version"].N, "2"; got != want {
		t.Fatalf("unexpected updated version: got %q want %q", got, want)
	}
	if _, ok := updateItemResponse.Attributes["obsolete"]; ok {
		t.Fatal("expected obsolete attribute to be removed from update response")
	}

	getItem := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.GetItem",
		body: `{
			"TableName":"mildstack-archive",
			"Key":{"id":{"S":"item#1"}}
		}`,
	})
	if got, want := getItem.code, http.StatusOK; got != want {
		t.Fatalf("unexpected get item status: got %d want %d", got, want)
	}
	var getItemResponse getItemResponse
	decodeResponse(t, getItem.body, &getItemResponse)
	if got, want := getItemResponse.Item["title"].S, "updated archive item"; got != want {
		t.Fatalf("unexpected title: got %q want %q", got, want)
	}
	if got, want := getItemResponse.Item["version"].N, "2"; got != want {
		t.Fatalf("unexpected version: got %q want %q", got, want)
	}
	if _, ok := getItemResponse.Item["obsolete"]; ok {
		t.Fatal("expected obsolete attribute to be absent after update")
	}

	deleteItem := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DeleteItem",
		body: `{
			"TableName":"mildstack-archive",
			"Key":{"id":{"S":"item#1"}}
		}`,
	})
	if got, want := deleteItem.code, http.StatusOK; got != want {
		t.Fatalf("unexpected delete item status: got %d want %d", got, want)
	}
	var deleteItemResponse deleteItemResponse
	decodeResponse(t, deleteItem.body, &deleteItemResponse)

	time.Sleep(250 * time.Millisecond)

	describeTable = doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DescribeTable",
		body: `{
			"TableName":"mildstack-archive"
		}`,
	})
	if got, want := describeTable.code, http.StatusOK; got != want {
		t.Fatalf("unexpected active describe table status: got %d want %d", got, want)
	}
	decodeResponse(t, describeTable.body, &describeTableResponse)
	if got, want := describeTableResponse.Table.TableStatus, "ACTIVE"; got != want {
		t.Fatalf("unexpected active table status: got %q want %q", got, want)
	}

	deleteTable := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DeleteTable",
		body: `{
			"TableName":"mildstack-archive"
		}`,
	})
	if got, want := deleteTable.code, http.StatusOK; got != want {
		t.Fatalf("unexpected delete table status: got %d want %d", got, want)
	}
	var deleteTableResponse deleteTableResponse
	decodeResponse(t, deleteTable.body, &deleteTableResponse)
	if got, want := deleteTableResponse.TableDescription.TableStatus, "DELETING"; got != want {
		t.Fatalf("unexpected deleting table status: got %q want %q", got, want)
	}

	postDeleteList := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.ListTables",
		body:   `{}`,
	})
	if got, want := postDeleteList.code, http.StatusOK; got != want {
		t.Fatalf("unexpected post-delete list tables status: got %d want %d", got, want)
	}
	decodeResponse(t, postDeleteList.body, &listTablesResponse)
	for _, tableName := range listTablesResponse.TableNames {
		if tableName == "mildstack-archive" {
			t.Fatalf("expected deleted table to disappear from ListTables, got %v", listTablesResponse.TableNames)
		}
	}
}

func TestDynamoDBNativeRoutesLeaveRuntimeEndpointsUntouched(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterDynamoDBNativeRoutes(engine, application.New())
	engine.POST("/api/v1/runtime/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "runtime")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runtime/ping", strings.NewReader(`{"ignored":true}`))
	request.Header.Set("Content-Type", dynamoDBJSONContentType)
	request.Header.Set("X-Amz-Target", "DynamoDB_20120810.ListTables")

	engine.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected runtime status: got %d want %d", got, want)
	}
	if got, want := recorder.Body.String(), "runtime"; got != want {
		t.Fatalf("unexpected runtime response body: got %q want %q", got, want)
	}
}

func TestDynamoDBNativeRoutesPaginateListTablesDeterministically(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterDynamoDBNativeRoutes(engine, application.New())

	for _, name := range []string{"mildstack-alpha", "mildstack-zeta"} {
		response := doDynamoDBRequest(t, engine, dynamoRequest{
			target: "DynamoDB_20120810.CreateTable",
			body: `{
				"TableName":"` + name + `",
				"KeySchema":[{"AttributeName":"id","KeyType":"HASH"}],
				"AttributeDefinitions":[{"AttributeName":"id","AttributeType":"S"}]
			}`,
		})
		if got, want := response.code, http.StatusOK; got != want {
			t.Fatalf("unexpected create table status for %s: got %d want %d", name, got, want)
		}
	}

	firstPage := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.ListTables",
		body: `{
			"Limit":2
		}`,
	})
	if got, want := firstPage.code, http.StatusOK; got != want {
		t.Fatalf("unexpected first page status: got %d want %d", got, want)
	}
	var firstPageResponse listTablesResponse
	decodeResponse(t, firstPage.body, &firstPageResponse)
	if got, want := firstPageResponse.TableNames, []string{"mildstack-alpha", "mildstack-records"}; !equalStringSlices(got, want) {
		t.Fatalf("unexpected first page tables: got %v want %v", got, want)
	}
	if got, want := firstPageResponse.LastEvaluatedTableName, "mildstack-records"; got != want {
		t.Fatalf("unexpected first page cursor: got %q want %q", got, want)
	}

	secondPage := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.ListTables",
		body: `{
			"Limit":2,
			"ExclusiveStartTableName":"mildstack-records"
		}`,
	})
	if got, want := secondPage.code, http.StatusOK; got != want {
		t.Fatalf("unexpected second page status: got %d want %d", got, want)
	}
	var secondPageResponse listTablesResponse
	decodeResponse(t, secondPage.body, &secondPageResponse)
	if got, want := secondPageResponse.TableNames, []string{"mildstack-zeta"}; !equalStringSlices(got, want) {
		t.Fatalf("unexpected second page tables: got %v want %v", got, want)
	}
	if got, want := secondPageResponse.LastEvaluatedTableName, ""; got != want {
		t.Fatalf("unexpected second page cursor: got %q want %q", got, want)
	}
}

func TestDynamoDBNativeRoutesReturnAWSShapedErrors(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterDynamoDBNativeRoutes(engine, application.New())

	malformed := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "ListTables",
		body:   `{}`,
	})
	assertDynamoError(t, malformed, http.StatusBadRequest, "ValidationException")

	unsupportedQuery := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.Query",
		body: `{
			"TableName":"mildstack-records",
			"KeyConditionExpression":"id = :id",
			"IndexName":"mildstack-index",
			"ExpressionAttributeValues":{
				":id":{"S":"example#1"}
			}
		}`,
	})
	assertDynamoError(t, unsupportedQuery, http.StatusBadRequest, "ValidationException")

	unsupportedScan := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.Scan",
		body: `{
			"TableName":"mildstack-records",
			"ProjectionExpression":"title",
			"Segment":1,
			"TotalSegments":2
		}`,
	})
	assertDynamoError(t, unsupportedScan, http.StatusBadRequest, "ValidationException")

	updateValidation := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.UpdateItem",
		body:   `{}`,
	})
	assertDynamoError(t, updateValidation, http.StatusBadRequest, "ValidationException")

	duplicateCreate := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.CreateTable",
		body: `{
			"TableName":"mildstack-duplicate",
			"KeySchema":[{"AttributeName":"id","KeyType":"HASH"}],
			"AttributeDefinitions":[{"AttributeName":"id","AttributeType":"S"}]
		}`,
	})
	if got, want := duplicateCreate.code, http.StatusOK; got != want {
		t.Fatalf("unexpected first create status: got %d want %d", got, want)
	}
	duplicateCreate = doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.CreateTable",
		body: `{
			"TableName":"mildstack-duplicate",
			"KeySchema":[{"AttributeName":"id","KeyType":"HASH"}],
			"AttributeDefinitions":[{"AttributeName":"id","AttributeType":"S"}]
		}`,
	})
	assertDynamoError(t, duplicateCreate, http.StatusBadRequest, "ResourceInUseException")

	creatingDelete := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DeleteTable",
		body: `{
			"TableName":"mildstack-duplicate"
		}`,
	})
	if got, want := creatingDelete.code, http.StatusOK; got != want {
		t.Fatalf("unexpected delete-on-creating status: got %d want %d", got, want)
	}
	var creatingDeleteResponse deleteTableResponse
	decodeResponse(t, creatingDelete.body, &creatingDeleteResponse)
	if got, want := creatingDeleteResponse.TableDescription.TableStatus, "DELETING"; got != want {
		t.Fatalf("unexpected delete-on-creating table status: got %q want %q", got, want)
	}

	missingItem := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DeleteItem",
		body: `{
			"TableName":"mildstack-records",
			"Key":{"id":{"S":"missing"}}
		}`,
	})
	assertDynamoError(t, missingItem, http.StatusBadRequest, "ResourceNotFoundException")

	conditionalFailure := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.UpdateItem",
		body: `{
			"TableName":"mildstack-records",
			"Key":{"id":{"S":"missing"}},
			"UpdateExpression":"SET title = :title",
			"ConditionExpression":"attribute_exists(id)",
			"ExpressionAttributeValues":{
				":title":{"S":"updated"}
			}
		}`,
	})
	assertDynamoError(t, conditionalFailure, http.StatusBadRequest, "ConditionalCheckFailedException")

	missingTable := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.GetItem",
		body: `{
			"TableName":"missing-table",
			"Key":{"id":{"S":"item#1"}}
		}`,
	})
	assertDynamoError(t, missingTable, http.StatusBadRequest, "ResourceNotFoundException")

	missingDescribe := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DescribeTable",
		body: `{
			"TableName":"missing-table"
		}`,
	})
	assertDynamoError(t, missingDescribe, http.StatusBadRequest, "ResourceNotFoundException")

	missingDelete := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DeleteTable",
		body: `{
			"TableName":"missing-table"
		}`,
	})
	assertDynamoError(t, missingDelete, http.StatusBadRequest, "ResourceNotFoundException")
}

func TestDynamoDBNativeRoutesHandleQueryAndScanSubset(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterDynamoDBNativeRoutes(engine, application.New())

	createTable := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.CreateTable",
		body: `{
			"TableName":"mildstack-reads",
			"KeySchema":[
				{"AttributeName":"id","KeyType":"HASH"},
				{"AttributeName":"sk","KeyType":"RANGE"}
			],
			"AttributeDefinitions":[
				{"AttributeName":"id","AttributeType":"S"},
				{"AttributeName":"sk","AttributeType":"S"}
			],
			"BillingMode":"PAY_PER_REQUEST"
		}`,
	})
	if got, want := createTable.code, http.StatusOK; got != want {
		t.Fatalf("unexpected create table status: got %d want %d", got, want)
	}

	for _, item := range []struct {
		key   string
		sk    string
		title string
	}{
		{key: "item#1", sk: "001", title: "skip-one"},
		{key: "item#2", sk: "002", title: "keep-two"},
		{key: "item#3", sk: "003", title: "keep-three"},
	} {
		response := doDynamoDBRequest(t, engine, dynamoRequest{
			target: "DynamoDB_20120810.PutItem",
			body: `{
				"TableName":"mildstack-reads",
				"Item":{
					"id":{"S":"series#1"},
					"sk":{"S":"` + item.sk + `"},
					"title":{"S":"` + item.title + `"},
					"row_id":{"S":"` + item.key + `"}
				}
			}`,
		})
		if got, want := response.code, http.StatusOK; got != want {
			t.Fatalf("unexpected put item status for %s: got %d want %d", item.sk, got, want)
		}
	}

	query := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.Query",
		body: `{
			"TableName":"mildstack-reads",
			"KeyConditionExpression":"id = :id AND sk BETWEEN :start AND :end",
			"ExpressionAttributeValues":{
				":id":{"S":"series#1"},
				":start":{"S":"001"},
				":end":{"S":"003"}
			},
			"ScanIndexForward":false,
			"Limit":2
		}`,
	})
	if got, want := query.code, http.StatusOK; got != want {
		t.Fatalf("unexpected query status: got %d want %d", got, want)
	}
	var queryResponse queryResponse
	decodeResponse(t, query.body, &queryResponse)
	if got, want := queryResponse.Count, 2; got != want {
		t.Fatalf("unexpected query count: got %d want %d", got, want)
	}
	if got, want := queryResponse.ScannedCount, 2; got != want {
		t.Fatalf("unexpected query scanned count: got %d want %d", got, want)
	}
	if got, want := queryResponse.Items[0]["sk"].S, "003"; got != want {
		t.Fatalf("unexpected first query item: got %q want %q", got, want)
	}
	if got, want := queryResponse.Items[1]["sk"].S, "002"; got != want {
		t.Fatalf("unexpected second query item: got %q want %q", got, want)
	}
	if got, want := queryResponse.LastEvaluatedKey["sk"].S, "002"; got != want {
		t.Fatalf("unexpected query last evaluated key: got %q want %q", got, want)
	}

	queryBeginsWith := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.Query",
		body: `{
			"TableName":"mildstack-reads",
			"KeyConditionExpression":"id = :id AND begins_with(sk, :prefix)",
			"ExpressionAttributeValues":{
				":id":{"S":"series#1"},
				":prefix":{"S":"00"}
			}
		}`,
	})
	if got, want := queryBeginsWith.code, http.StatusOK; got != want {
		t.Fatalf("unexpected begins_with query status: got %d want %d", got, want)
	}
	decodeResponse(t, queryBeginsWith.body, &queryResponse)
	if got, want := len(queryResponse.Items), 3; got != want {
		t.Fatalf("unexpected begins_with query item count: got %d want %d", got, want)
	}
	if got, want := queryResponse.Items[0]["sk"].S, "001"; got != want {
		t.Fatalf("unexpected begins_with first item: got %q want %q", got, want)
	}
	if got, want := queryResponse.Items[2]["sk"].S, "003"; got != want {
		t.Fatalf("unexpected begins_with third item: got %q want %q", got, want)
	}

	scanPage1 := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.Scan",
		body: `{
			"TableName":"mildstack-reads",
			"FilterExpression":"begins_with(title, :prefix)",
			"ExpressionAttributeValues":{
				":prefix":{"S":"keep"}
			},
			"Limit":1
		}`,
	})
	if got, want := scanPage1.code, http.StatusOK; got != want {
		t.Fatalf("unexpected scan page 1 status: got %d want %d", got, want)
	}
	var scanResponse scanResponse
	decodeResponse(t, scanPage1.body, &scanResponse)
	if got, want := scanResponse.Count, 0; got != want {
		t.Fatalf("unexpected scan page 1 count: got %d want %d", got, want)
	}
	if got, want := scanResponse.ScannedCount, 1; got != want {
		t.Fatalf("unexpected scan page 1 scanned count: got %d want %d", got, want)
	}
	if len(scanResponse.Items) != 0 {
		t.Fatalf("expected first scan page to be empty, got %#v", scanResponse.Items)
	}
	if got, want := scanResponse.LastEvaluatedKey["sk"].S, "001"; got != want {
		t.Fatalf("unexpected scan page 1 cursor: got %q want %q", got, want)
	}

	scanPage2 := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.Scan",
		body: `{
			"TableName":"mildstack-reads",
			"FilterExpression":"begins_with(title, :prefix)",
			"ExpressionAttributeValues":{
				":prefix":{"S":"keep"}
			},
			"Limit":1,
			"ExclusiveStartKey":{
				"id":{"S":"series#1"},
				"sk":{"S":"001"}
			}
		}`,
	})
	if got, want := scanPage2.code, http.StatusOK; got != want {
		t.Fatalf("unexpected scan page 2 status: got %d want %d", got, want)
	}
	decodeResponse(t, scanPage2.body, &scanResponse)
	if got, want := scanResponse.Count, 1; got != want {
		t.Fatalf("unexpected scan page 2 count: got %d want %d", got, want)
	}
	if got, want := scanResponse.Items[0]["title"].S, "keep-two"; got != want {
		t.Fatalf("unexpected scan page 2 item: got %q want %q", got, want)
	}
}

type dynamoRequest struct {
	target string
	body   string
}

type capturedResponse struct {
	code        int
	contentType string
	body        string
}

func doDynamoDBRequest(t *testing.T, engine *gin.Engine, request dynamoRequest) capturedResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	httpRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(request.body))
	httpRequest.Header.Set("Content-Type", dynamoDBJSONContentType)
	httpRequest.Header.Set("X-Amz-Target", request.target)

	engine.ServeHTTP(recorder, httpRequest)

	return capturedResponse{
		code:        recorder.Code,
		contentType: recorder.Header().Get("Content-Type"),
		body:        recorder.Body.String(),
	}
}

func decodeResponse(t *testing.T, body string, into any) {
	t.Helper()

	if err := json.Unmarshal([]byte(body), into); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, body)
	}
}

func assertDynamoError(t *testing.T, response capturedResponse, wantStatus int, wantType string) {
	t.Helper()

	if got, want := response.code, wantStatus; got != want {
		t.Fatalf("unexpected status: got %d want %d\nbody: %s", got, want, response.body)
	}
	if got, want := response.contentType, dynamoDBJSONContentType; got != want {
		t.Fatalf("unexpected content type: got %q want %q", got, want)
	}

	var payload dynamoDBErrorResponse
	decodeResponse(t, response.body, &payload)
	if !strings.Contains(payload.Type, wantType) {
		t.Fatalf("unexpected error type: got %q want substring %q", payload.Type, wantType)
	}
}

func equalStringSlices(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
