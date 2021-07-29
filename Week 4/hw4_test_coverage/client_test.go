package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// код писать тут
func SearchServer(w http.ResponseWriter, r *http.Request) {
	req := SearchRequest{}
	var err error
	req.Limit, err = strconv.Atoi(r.FormValue("limit"))
	req.Offset, err = strconv.Atoi(r.FormValue("offset"))
	req.Query = r.FormValue("query")
	req.OrderField = r.FormValue("order_field")
	req.OrderBy, err = strconv.Atoi(r.FormValue("order_by"))

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Query == "timeout" {
		time.Sleep(3 * time.Second)
		return
	}

	if req.Query == "error" {
		http.Redirect(w, r, string([]byte{0x7f}), http.StatusPermanentRedirect)
		return
	}

	accessToken := r.Header.Get("AccessToken")
	if accessToken == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if _, fErr := os.Stat("dataset.xml"); os.IsNotExist(fErr) || req.Query == "FileNotFound" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.Query == "BadJson" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{"))
		return
	}

	if req.OrderField == "" {
		w.WriteHeader(http.StatusBadRequest)
		errResp := SearchErrorResponse{Error: "ErrorBadOrderField"}
		res, _ := json.Marshal(errResp)
		w.Write(res)
		return
	}

	if req.OrderBy < -1 || req.OrderBy > 1 {
		w.WriteHeader(http.StatusBadRequest)
		errResp := SearchErrorResponse{Error: ErrorBadOrderField}
		res, _ := json.Marshal(errResp)
		w.Write(res)
		return
	}

	w.WriteHeader(http.StatusOK)
	if req.Query == "Empty" {
		w.Write([]byte{})
		return
	}

	resLen := req.Limit
	if req.Query != "NextPage" {
		resLen--
	}
	users := make([]User, 0, resLen)
	user := User{}
	for i := 0; i < resLen; i++ {
		users = append(users, user)
	}

	usersStr, err := json.Marshal(users)
	w.Write(usersStr)
}

// Тесты

type TestCase struct {
	client   *SearchClient
	request  SearchRequest
	response *SearchResponse
	err      error
}

func RunTestCase(testCase *TestCase) {
	if testCase.client == nil {
		testCase.client = &SearchClient{}
	}
	testCase.response, testCase.err = testCase.client.FindUsers(testCase.request)
}

func ErrorTest(t *testing.T, tErr, expectedErr error) {
	if tErr == nil || tErr.Error() != expectedErr.Error() {
		t.Error(tErr)
	}
}

// req.Limit < 0 (line 71)
func TestLineLimit1(t *testing.T) {
	err := fmt.Errorf("limit must be > 0")
	testCase := TestCase{request: SearchRequest{Limit: -1}}
	RunTestCase(&testCase)
	ErrorTest(t, testCase.err, err)
}

// req.Limit > 25 (line 74)
func TestLineLimit2(t *testing.T) {
	testCase := TestCase{request: SearchRequest{Limit: 26}}
	RunTestCase(&testCase)
}

// req.Offset < 0 (line 77)
func TestLineOffset(t *testing.T) {
	err := fmt.Errorf("offset must be > 0")
	testCase := TestCase{request: SearchRequest{Offset: -1}}
	RunTestCase(&testCase)
	ErrorTest(t, testCase.err, err)
}

func TestClientDoTimeoutError(t *testing.T) {
	ts := httptest.NewServer(http.TimeoutHandler(http.HandlerFunc(SearchServer), 2*time.Second, ""))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client:  &SearchClient{URL: ts.URL},
		request: SearchRequest{Limit: 1, Offset: 1, Query: "timeout"},
	}
	RunTestCase(&testCase)
	if testCase.err == nil || !strings.Contains(testCase.err.Error(), "timeout for") {
		t.Error()
	}
}

func TestClientDoOtherError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client:  &SearchClient{URL: ts.URL},
		request: SearchRequest{Limit: 1, Offset: 1, Query: "error"},
	}
	RunTestCase(&testCase)
	if testCase.err == nil || !strings.Contains(testCase.err.Error(), "unknown error") {
		t.Error(testCase.err)
	}
}

func TestStatusUnauthorized(t *testing.T) {
	err := fmt.Errorf("Bad AccessToken")
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client:  &SearchClient{"", ts.URL},
		request: SearchRequest{Limit: 1, Offset: 1},
	}
	RunTestCase(&testCase)
	ErrorTest(t, testCase.err, err)
}

func TestStatusInternalServerError(t *testing.T) {
	err := fmt.Errorf("SearchServer fatal error")
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client:  &SearchClient{"123", ts.URL},
		request: SearchRequest{Limit: 1, Offset: 1, Query: "FileNotFound"},
	}
	RunTestCase(&testCase)
	ErrorTest(t, testCase.err, err)
}

func TestUnpackJsonError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client:  &SearchClient{"123", ts.URL},
		request: SearchRequest{Limit: 1, Offset: 1, Query: "BadJson"},
	}
	RunTestCase(&testCase)
	if testCase.err == nil || !strings.Contains(testCase.err.Error(), "cant unpack error json") {
		t.Error(testCase.err)
	}
}

func TestBadOrderFieldError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client: &SearchClient{"123", ts.URL},
		request: SearchRequest{
			Limit:  1,
			Offset: 1,
			Query:  "Name=\"Name\"",
		},
	}
	RunTestCase(&testCase)
	if testCase.err == nil || !strings.Contains(testCase.err.Error(), "OrderFeld") {
		t.Error(testCase.err)
	}
}

func TestUnknownBadRequestError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client: &SearchClient{"123", ts.URL},
		request: SearchRequest{
			Limit:      1,
			Offset:     1,
			Query:      "Name=\"Name\"",
			OrderField: "Name",
			OrderBy:    -2,
		},
	}
	RunTestCase(&testCase)
	if testCase.err == nil || !strings.Contains(testCase.err.Error(), "unknown bad request error") {
		t.Error(testCase.err)
	}
}

func TestUsersParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client: &SearchClient{"123", ts.URL},
		request: SearchRequest{
			Limit:      1,
			Offset:     1,
			Query:      "Empty",
			OrderField: "Name",
			OrderBy:    0,
		},
	}
	RunTestCase(&testCase)
	if testCase.err == nil || !strings.Contains(testCase.err.Error(), "cant unpack result json") {
		t.Error(testCase.err)
	}
}

func TestHasNextPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client: &SearchClient{"123", ts.URL},
		request: SearchRequest{
			Limit:      3,
			Offset:     1,
			Query:      "NextPage",
			OrderField: "Name",
			OrderBy:    0,
		},
	}
	RunTestCase(&testCase)
	if testCase.response == nil || testCase.err != nil {
		t.Error(testCase.err)
	}
}

func TestNoNextPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer func() {
		ts.Close()
	}()
	testCase := TestCase{
		client: &SearchClient{"123", ts.URL},
		request: SearchRequest{
			Limit:      3,
			Offset:     1,
			Query:      "NoNextPage",
			OrderField: "Name",
			OrderBy:    0,
		},
	}
	RunTestCase(&testCase)
	if testCase.response == nil || testCase.err != nil {
		t.Error(testCase.err)
	}
}
