package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONToRemote(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("OK")),
			Header:     make(http.Header),
		}
	})
	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"
	var testTools Tools
	_, _, err := testTools.PushJSONToRemote("http://example", foo, client)
	if err != nil {
		t.Error("failed to call remote url:", err)
	}
}

func TestTools_Randomstring(t *testing.T) {

	var testTools Tools

	s := testTools.RandomString(10)

	if len(s) != 10 {
		t.Error("wrong length random string returned")
	}

}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// set up a pipe avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}
			// create form data filed 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}

		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		// var testTools = Tools{
		// 	AllowedTypes: e.allowedTypes,
		// }
		var testTools Tools
		testTools.AllowedTypes = e.allowedTypes

		uploadFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}
		if !e.errorExpected {
			var testFileName string = fmt.Sprintf("./testdata/uploads/%s", uploadFiles[0].NewFileName)
			if _, errA := os.Stat(testFileName); os.IsNotExist(errA) {
				t.Errorf("%s: ecpected file to exist: %s", e.name, err.Error())
			}

			// clean up test uploads
			_ = os.Remove(testFileName)
		}

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected but none received", e.name)
		}
		wg.Wait()

	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// set up a pipe avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}
		// create form data filed 'file'
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}

	}()

	// read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	var testFileName string = fmt.Sprintf("./testdata/uploads/%s", uploadFiles.NewFileName)
	if _, errA := os.Stat(testFileName); os.IsNotExist(errA) {
		t.Errorf("ecpected file to exist: %s", err.Error())
	}

	_ = os.Remove(testFileName)

}

func TestTools_CreateDirIfNotExist(t *testing.T) {

	var testTools Tools
	path := "./testdata/myDir"
	err := testTools.CreateDirIfNotExist(path)
	if err != nil {
		t.Error(err)
	}
	err = testTools.CreateDirIfNotExist(path)
	if err != nil {
		t.Error(err)
	}
	os.Remove(path)
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	expectedError bool
}{
	{name: "valid string", s: "time to get 123", expected: "time-to-get-123", expectedError: false},
	{name: "empty string", s: "", expected: "", expectedError: true},
	{name: "complex string", s: "Moon, THIS, &fish +time to get 123", expected: "moon-this-fish-time-to-get-123", expectedError: false},
	{name: "chinese string", s: "歡迎你", expected: "", expectedError: true},
	{name: "chinese string and roman characters", s: "hello world 歡迎你", expected: "hello-world", expectedError: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools

	for _, e := range slugTests {

		slug, err := testTool.Slugify(e.s)
		if err != nil && !e.expectedError {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}
		if !e.expectedError && slug != e.expected {
			t.Errorf("%s: wrong slug returned; expected %s but get %s", e.name, e.expected, slug)
		}

	}
}

func TestTools_DownloadStaticFile(t *testing.T) {

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTool Tools

	testTool.DownloadStaticFile(rr, req, "./testdata/pic.png", "haha.png")
	res := rr.Result()
	defer res.Body.Close()
	cl := res.Header["Content-Length"][0]
	if cl != "12325" {
		t.Error("wrong content length of", cl)
	}

	cp := res.Header["Content-Disposition"][0]
	if cp != "attachment; filename=\"haha.png\"" {
		t.Error("wrong content disposition:", cp)
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}

}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "good json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly formatted json", json: `{"foo": }`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": "bar"}{"name": "haha"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error in json", json: `{"foo": "bar"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown filed in json", json: `{"fooo": "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown filed in json", json: `{"fooo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{"jack": "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 2, allowUnknown: false},
	{name: "not json", json: `hello world`, errorExpected: true, maxSize: 1024, allowUnknown: false},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools
	for _, e := range jsonTests {
		testTool.MaxJSONSize = e.maxSize
		testTool.AllowUnknownFileds = e.allowUnknown

		// declare a variable to read the decoded json into
		var decodedJson struct {
			Foo string `json:"foo"`
		}

		// create request
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))

		if err != nil {
			t.Log(err)
		}
		defer req.Body.Close()
		// create a recorder
		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJson)

		if e.errorExpected && err == nil {
			t.Errorf("%s: expected error, but receive none", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: not expected error, but receive one: %s", e.name, err.Error())
		}
	}
}

func TestTools_WriteJson(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()

	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")
	err := testTools.WriteJson(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write json: %v", err)
	}

}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("some error happened"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}
	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error(err)
	}

	if !payload.Error {
		t.Error("error set to false in json, but it should be true")
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503, but got %d", rr.Code)
	}
}
