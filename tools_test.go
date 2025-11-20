package toolkit

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s1, s2 := testTools.RandomString(10), testTools.RandomString(10)
	if len(s1) != 10 || len(s2) != 10 {
		t.Error("the length of the random string is wrong")
	}
	if s1 == s2 {
		t.Error("the random strings are not random ")
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
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer writer.Close()
			defer wg.Done()

			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			file, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			img, _, err := image.Decode(file)
			if err != nil {
				t.Error(err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()
		r := httptest.NewRequest("POST", "/", pr)
		r.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(r, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}
		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: Expected file to exists %s", e.name, err.Error())
			}

			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}
		if e.errorExpected && err == nil {
			t.Errorf("%s: Error expected but none received", e.name)

		}
		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// set up a pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		/// create the form data field 'file'
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
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

	uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	// clean up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))
}

func TestTools_CreateDirIfNotExists(t *testing.T) {
	var testTool Tools

	err := testTool.CreateDirIfNotExists("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	err = testTool.CreateDirIfNotExists("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "Correct slug", s: "i-am-slug", expected: "i-am-slug", errorExpected: false},
	{name: "A single word", s: "Tomato", expected: "tomato", errorExpected: false},
	{name: "Two words without slug", s: "Two words", expected: "two-words", errorExpected: false},
	{name: "Two words with slugs at the beginning and end", s: "-Two words-", expected: "two-words", errorExpected: false},
	{name: "Special characters", s: "$$$Two$$ words@!-", expected: "two-words", errorExpected: false},
	{name: "Empty string", s: "", expected: "", errorExpected: true},
	{name: "Polish char", s: "ćóć", expected: "", errorExpected: true},
	{name: "Polish char and Roman char", s: "ćóćci", expected: "ci", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools

	for _, test := range slugTests {
		slug, err := testTool.Slugify(test.s)
		if err != nil && !test.errorExpected {
			t.Error(err)
		}
		if slug != test.expected && test.errorExpected {
			t.Errorf("The test (%s) failed: expected: %v, received: %v", test.name, test.expected, slug)
		}
		if err == nil && test.errorExpected {
			t.Errorf("The test (%s) passed even though it should failed: expected: %v, received: %v", test.name, test.expected, slug)
		}
	}

}

// DownloadStaticFile it downloads a file and tries to force the browser to avoid displaying it in the browser window by setting content disposition. It also allow specification
// of the display name
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, p, file, displayName string) {
	fp := path.Join(p, file)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))
	http.ServeFile(w, r, fp)
}

func TestTools_DownloadStaticFiles(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("get", "/", nil)

	var testTool Tools

	testTool.DownloadStaticFile(rr, req, "./testdata", "hotend.png", "bambu.png")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "190914" {
		t.Error("Wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"bambu.png\"" {
		t.Error("Wrong content disposition")
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
	{name: "Good json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly formatted json", json: `{"foo": }`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo":1 }`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files type", json: `{"foo":"1" }{"alpha": "beta"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error in json", json: `{"foo":1" }`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field in json", json: `{"foooo":"1" }`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown fields in json", json: `{"foooo":"1" }`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack:"1" }`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 5, allowUnknown: true},
	{name: "not json", json: `Hello, its mario`, errorExpected: true, maxSize: 1024, allowUnknown: true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools

	for _, e := range jsonTests {
		testTool.MaxJSONSize = e.maxSize
		testTool.AllowUnknownFields = e.allowUnknown

		// declare variable to read the decoded json into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error", err)
		}
		defer req.Body.Close()
		// create a recorder
		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", e.name)
		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but one received, %s", e.name, err.Error())

		}

	}

}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}
