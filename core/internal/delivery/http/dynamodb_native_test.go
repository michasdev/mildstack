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

	for _, target := range []string{"ListTables", "CreateTable", "DescribeTable", "DeleteTable", "GetItem", "PutItem", "DeleteItem"} {
		spec, ok := handler.registry[target]
		if !ok {
			t.Fatalf("expected target %q to be registered", target)
		}
		if !spec.supported {
			t.Fatalf("expected target %q to be supported", target)
		}
	}

	for _, target := range []string{"UpdateItem", "Query", "Scan", "BatchGetItem", "TransactWriteItems"} {
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

	putItem := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.PutItem",
		body: `{
			"TableName":"mildstack-archive",
			"Item":{
				"id":{"S":"item#1"},
				"title":{"S":"archive item"},
				"version":{"N":"1"}
			}
		}`,
	})
	if got, want := putItem.code, http.StatusOK; got != want {
		t.Fatalf("unexpected put item status: got %d want %d", got, want)
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
	if got, want := getItemResponse.Item["title"].S, "archive item"; got != want {
		t.Fatalf("unexpected title: got %q want %q", got, want)
	}
	if got, want := getItemResponse.Item["version"].N, "1"; got != want {
		t.Fatalf("unexpected version: got %q want %q", got, want)
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

	describeTable := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DescribeTable",
		body: `{
			"TableName":"mildstack-archive"
		}`,
	})
	assertDynamoError(t, describeTable, http.StatusBadRequest, "ResourceNotFoundException")

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
	var describeTableResponse describeTableResponse
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

	unsupported := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.UpdateItem",
		body:   `{}`,
	})
	assertDynamoError(t, unsupported, http.StatusNotFound, "UnknownOperationException")

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
	assertDynamoError(t, creatingDelete, http.StatusBadRequest, "ResourceInUseException")

	missingItem := doDynamoDBRequest(t, engine, dynamoRequest{
		target: "DynamoDB_20120810.DeleteItem",
		body: `{
			"TableName":"mildstack-records",
			"Key":{"id":{"S":"missing"}}
		}`,
	})
	assertDynamoError(t, missingItem, http.StatusBadRequest, "ResourceNotFoundException")

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
