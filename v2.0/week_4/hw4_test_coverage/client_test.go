package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type headers struct {
	AccessToken string
}

type requestParams struct {
	Limit      int
	Offset     int    // Можно учесть после сортировки
	Query      string // подстрока в 1 из полей
	OrderField string
	// -1 по убыванию, 0 как встретилось, 1 по возрастанию
	OrderBy int
	Headers headers
}

type XmlUser struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type XmlUsers struct {
	XMLName xml.Name  `xml:"root"`
	Users   []XmlUser `xml:"row"`
}

func toInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func parseRequest(r *http.Request) requestParams {
	return requestParams{
		Limit:      toInt(r.FormValue("limit")),
		Offset:     toInt(r.FormValue("offset")),
		Query:      r.FormValue("query"),
		OrderField: r.FormValue("order_field"),
		OrderBy:    toInt(r.FormValue("order_by")),
		Headers:    headers{AccessToken: r.Header.Get("AccessToken")},
	}
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	req := parseRequest(r)
	if len(req.Headers.AccessToken) == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var orderField string
	switch req.OrderField {
	case "Id":
		orderField = "Id"
	case "Age":
		orderField = "Age"
	case "", "Name":
		orderField = "Name"
	default:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"Error":"ErrorBadOrderField"}`))
		return
	}

	file, err := os.ReadFile("dataset.xml")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var allUsers XmlUsers
	err = xml.Unmarshal(file, &allUsers)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resultUsers := make([]User, 0, len(allUsers.Users))
	for _, user := range allUsers.Users {
		resultUser := User{
			Id:     user.Id,
			Name:   user.FirstName + user.LastName,
			Age:    user.Age,
			About:  user.About,
			Gender: user.Gender,
		}
		if len(req.Query) == 0 || strings.Contains(resultUser.Name, req.Query) || strings.Contains(resultUser.About, req.Query) {
			resultUsers = append(resultUsers, resultUser)
		}
	}

	if req.OrderBy != 0 {
		sort.Slice(resultUsers, func(i, j int) bool {
			var f, s int
			if req.OrderBy > 0 {
				f = i
				s = j
			} else {
				f = j
				s = i
			}
			if orderField == "Id" {
				return resultUsers[f].Id < resultUsers[s].Id
			}
			if orderField == "Age" {
				return resultUsers[f].Age < resultUsers[s].Age
			}
			return resultUsers[f].Name < resultUsers[s].Name
		})
	}

	response, err := json.Marshal(resultUsers[min(len(resultUsers), req.Offset):min(len(resultUsers), req.Offset+req.Limit)])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func TestFindUsers_Ok(t *testing.T) {
	// Arrange
	client := getHTTPClient("token", SearchServer)

	testCases := []SearchRequest{
		{Limit: 1, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
		{Limit: 100, Offset: 0, Query: "", OrderField: "", OrderBy: 0},
		{Limit: 25, Offset: 25, Query: "", OrderField: "", OrderBy: 0},
		{Limit: 25, Offset: 0, Query: "Boyd", OrderField: "Id", OrderBy: 1},
	}

	for _, req := range testCases {
		// Act
		resp, err := client.FindUsers(req)

		// Assert
		assertSuccess(t, resp, err)
	}
}

func TestFindUsers_ErrorBadOrderField(t *testing.T) {
	// Arrange
	req := SearchRequest{Limit: 1, Offset: 1, OrderField: "abc"}
	client := getHTTPClient("token", SearchServer)

	// Act
	resp, err := client.FindUsers(req)

	// Assert
	assertError(t, resp, err)
}

func TestFindUsers_BadJsonInResponse(t *testing.T) {
	// Arrange
	req := SearchRequest{Limit: 1, Offset: 1}
	client := getHTTPClient("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{"))
	}))

	// Act
	resp, err := client.FindUsers(req)

	// Assert
	assertError(t, resp, err)
}

func TestFindUsers_ResponseBadRequestError(t *testing.T) {
	testCases := []struct {
		badJSON bool
	}{
		{true},
		{false},
	}

	for _, tc := range testCases {
		// Arrange
		req := SearchRequest{Limit: 1, Offset: 1}
		client := getHTTPClient("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			response := `{"Error": "Error"`
			if !tc.badJSON {
				response += "}"
			}
			w.Write([]byte(response))
		}))

		// Act
		resp, err := client.FindUsers(req)

		// Assert
		assertError(t, resp, err)
	}
}

func TestFindUsers_ResponseInternalServerError(t *testing.T) {
	// Arrange
	req := SearchRequest{Limit: 1, Offset: 1}
	client := getHTTPClient("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	// Act
	resp, err := client.FindUsers(req)

	// Assert
	assertError(t, resp, err)
}

func TestFindUsers_ResponseStatusUnauthorized(t *testing.T) {
	// Arrange
	req := SearchRequest{Limit: 1, Offset: 1}
	client := getHTTPClient("", SearchServer)

	// Act
	resp, err := client.FindUsers(req)

	// Assert
	assertError(t, resp, err)
}

func TestFindUsers_RequestTimeoutError(t *testing.T) {
	// Arrange
	req := SearchRequest{Limit: 1, Offset: 1}
	client := getHTTPClient("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))

	// Act
	resp, err := client.FindUsers(req)

	// Assert
	assertError(t, resp, err)
}

func TestFindUsers_RequestError(t *testing.T) {
	// Arrange
	req := SearchRequest{Limit: 1, Offset: 1}
	var url string
	client := &SearchClient{AccessToken: "", URL: url}

	// Act
	resp, err := client.FindUsers(req)

	// Assert
	assertError(t, resp, err)
}

func TestFindUsers_LimitLessThanZero(t *testing.T) {
	// Arrange
	req := SearchRequest{Limit: -1}
	client := getHTTPClient("", SearchServer)

	// Act
	resp, err := client.FindUsers(req)

	// Assert
	assertError(t, resp, err)
}

func TestFindUsers_OffsetLessThanZero(t *testing.T) {
	// Arrange
	req := SearchRequest{Limit: 26, Offset: -1}
	client := getHTTPClient("", SearchServer)

	// Act
	resp, err := client.FindUsers(req)

	// Assert
	assertError(t, resp, err)
}

func getHTTPServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handler))
}

func getHTTPClient(accessToken string, handler http.HandlerFunc) *SearchClient {
	server := getHTTPServer(handler)
	return &SearchClient{AccessToken: accessToken, URL: server.URL}
}

func assertError(t *testing.T, resp *SearchResponse, err error) {
	if err == nil {
		t.Error("Expected not nil error")
	}
	if resp != nil {
		t.Errorf("Expected nil response, got: %v", resp)
	}
}

func assertSuccess(t *testing.T, resp *SearchResponse, err error) {
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
	if resp == nil {
		t.Error("Expected not nil response")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
