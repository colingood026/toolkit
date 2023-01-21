package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

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
