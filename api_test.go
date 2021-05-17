package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	SimpleContentRequest             = httptest.NewRequest("GET", "/?offset=0&count=5", nil)
	SimpleContentRequestWithBigCount = httptest.NewRequest("GET", "/?offset=0&count=30", nil)
	OffsetContentRequest             = httptest.NewRequest("GET", "/?offset=5&count=5", nil)
)

func runRequest(t *testing.T, srv http.Handler, r *http.Request) (content []*ContentItem) {
	response := httptest.NewRecorder()
	srv.ServeHTTP(response, r)

	if response.Code != 200 {
		t.Fatalf("Response code is %d, want 200", response.Code)
		return
	}

	decoder := json.NewDecoder(response.Body)
	err := decoder.Decode(&content)
	if err != nil {
		t.Fatalf("couldn't decode Response json: %v", err)
	}

	return content
}

func TestResponseCount(t *testing.T) {
	content := runRequest(t, app, SimpleContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

}

func TestResponseOrder(t *testing.T) {
	content := runRequest(t, app, SimpleContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

	for i, item := range content {
		if Provider(item.Source) != DefaultConfig[i%len(DefaultConfig)].Type {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, DefaultConfig[i].Type,
			)
		}
	}
}

func TestOffsetResponseOrder(t *testing.T) {
	content := runRequest(t, app, OffsetContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

	for j, item := range content {
		i := j + 5
		if Provider(item.Source) != DefaultConfig[i%len(DefaultConfig)].Type {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, DefaultConfig[i].Type,
			)
		}
	}
}

// Test when we have count > len(DefaultConfig)
func TestResponseOrderWithBigCount(t *testing.T) {
	content := runRequest(t, app, SimpleContentRequestWithBigCount)

	if len(content) != 30 {
		t.Fatalf("Got %d items back, want 30", len(content))
	}

	for i, item := range content {
		if Provider(item.Source) != DefaultConfig[i%len(DefaultConfig)].Type {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, DefaultConfig[i].Type,
			)
		}
	}
}

// Test when Provider 2 and Provider 3 return an error and so we should get content only from Provider 1,
// because fallback fails.
func TestResponseOrderWithErrorAndNoFallback(t *testing.T) {
	appCustom := App{
		ContentProvider: NewContentProviderService(
			map[Provider]Client{
				Provider1: SampleContentProvider{Source: Provider1},
				Provider2: &ClientMock{
					GetContentFn: func(userIP string, count int) ([]*ContentItem, error) {
						return []*ContentItem{}, errors.New("some error from provider 2")
					},
				},
				Provider3: &ClientMock{
					GetContentFn: func(userIP string, count int) ([]*ContentItem, error) {
						return []*ContentItem{}, errors.New("some error from provider 3")
					},
				},
			},
			DefaultConfig,
		),
	}

	content := runRequest(t, appCustom, SimpleContentRequest)

	if len(content) != 2 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

	for i, item := range content {
		if Provider(item.Source) != DefaultConfig[i%len(DefaultConfig)].Type {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, DefaultConfig[i].Type,
			)
		}
	}
}

// Test when Provider 2 returns an error and we should use it Fallback.
func TestResponseOrderWithErrorAndWeUseFallback(t *testing.T) {
	expectedSourcesOrder := []string{"1", "1", "3", "3", "1"}

	appCustom := App{
		ContentProvider: NewContentProviderService(
			map[Provider]Client{
				Provider1: SampleContentProvider{Source: Provider1},
				Provider2: &ClientMock{
					GetContentFn: func(userIP string, count int) ([]*ContentItem, error) {
						return []*ContentItem{}, errors.New("some error from provider 2")
					},
				},
				Provider3: SampleContentProvider{Source: Provider3},
			},
			DefaultConfig,
		),
	}

	content := runRequest(t, appCustom, SimpleContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

	for i, item := range content {
		if item.Source != expectedSourcesOrder[i] {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, expectedSourcesOrder[i],
			)
		}
	}
}

// Test when our first Provider fails without a fallback.
func TestResponseWhenFirstProviderFailsWithoutFallback(t *testing.T) {
	appCustom := App{
		ContentProvider: NewContentProviderService(
			map[Provider]Client{
				Provider1: &ClientMock{
					GetContentFn: func(userIP string, count int) ([]*ContentItem, error) {
						return []*ContentItem{}, errors.New("some error from provider 1")
					},
				},
				Provider2: SampleContentProvider{Source: Provider2},
				Provider3: SampleContentProvider{Source: Provider3},
			},
			[]ContentConfig{
				config4, config2, config3, config2, config3,
			},
		),
	}

	content := runRequest(t, appCustom, SimpleContentRequest)

	if len(content) != 0 {
		t.Fatalf("Got %d items back, want 0", len(content))
	}
}

// We shouldn't wait for expected providers if we got an error, just finish them and return a result.
func TestDontWaitForOtherProvidersIfWeGotError(t *testing.T) {
	start := time.Now()

	appCustom := App{
		ContentProvider: NewContentProviderService(
			map[Provider]Client{
				Provider1: &ClientMock{
					GetContentFn: func(userIP string, count int) ([]*ContentItem, error) {
						return []*ContentItem{}, errors.New("some error from provider 1")
					},
				},
				Provider2: &ClientMock{
					GetContentFn: func(userIP string, count int) ([]*ContentItem, error) {
						return []*ContentItem{}, errors.New("some error from provider 2")
					},
				},
				Provider3: &ClientMock{
					GetContentFn: func(userIP string, count int) ([]*ContentItem, error) {
						// let's imitate some pending request to provider
						time.Sleep(5 * time.Second)
						return []*ContentItem{{
							Source: "3",
						}}, nil
					},
				},
			},
			DefaultConfig,
			WithTimeOut(500*time.Millisecond),
		),
	}

	content := runRequest(t, appCustom, SimpleContentRequest)

	if len(content) != 0 {
		t.Fatalf("Got %d items back, want 0", len(content))
	}

	execTime := time.Since(start)
	if execTime > time.Second {
		t.Fatalf("test time should be less then 1 second, got: %s", execTime)
	}
}

// Test that after timeout we return result.
func TestTimeoutToProviders(t *testing.T) {
	start := time.Now()

	appCustom := App{
		ContentProvider: NewContentProviderService(
			map[Provider]Client{
				Provider1: SampleContentProvider{Source: Provider1},
				Provider2: SampleContentProvider{Source: Provider2},
				Provider3: &ClientMock{
					GetContentFn: func(userIP string, count int) ([]*ContentItem, error) {
						// let's imitate some pending request to provider
						time.Sleep(5 * time.Second)
						return []*ContentItem{{
							Source: "3",
						}}, nil
					},
				},
			},
			DefaultConfig,
			WithTimeOut(500*time.Millisecond),
		),
	}

	content := runRequest(t, appCustom, SimpleContentRequest)
	if len(content) != 4 {
		t.Fatalf("Got %d items back, want 4", len(content))
	}

	execTime := time.Since(start)
	if execTime > time.Second {
		t.Fatalf("test time should be less then 1 second, got: %s", execTime)
	}
}

func TestValidateParams(t *testing.T) {
	testCases := []struct {
		name               string
		url                string
		expectedStatusCode int
		compareBody        bool
		expectedBody       string
	}{
		{
			name:               "success request without params",
			url:                "/",
			expectedStatusCode: http.StatusOK,
			compareBody:        false,
		},
		{
			name:               "success request only with offset param",
			url:                "/?offset=5",
			expectedStatusCode: http.StatusOK,
			compareBody:        false,
		},
		{
			name:               "failed request with bad offset param",
			url:                "/?offset=string",
			expectedStatusCode: http.StatusBadRequest,
			compareBody:        true,
			expectedBody:       `{"error":{"code":"validation","message":"bad offset param: strconv.ParseInt: parsing \"string\": invalid syntax"}}` + "\n",
		},
		{
			name:               "failed request with negative offset param",
			url:                "/?offset=-1",
			expectedStatusCode: http.StatusBadRequest,
			compareBody:        true,
			expectedBody:       `{"error":{"code":"validation","message":"offset should be bigger than 0"}}` + "\n",
		},
		{
			name:               "success request only with count param",
			url:                "/?count=5",
			expectedStatusCode: http.StatusOK,
			compareBody:        false,
		},
		{
			name:               "failed request with bad count param",
			url:                "/?count=string",
			expectedStatusCode: http.StatusBadRequest,
			compareBody:        true,
			expectedBody:       `{"error":{"code":"validation","message":"bad count param: strconv.ParseInt: parsing \"string\": invalid syntax"}}` + "\n",
		},
		{
			name:               "failed request with too big count param",
			url:                "/?count=999",
			expectedStatusCode: http.StatusBadRequest,
			compareBody:        true,
			expectedBody:       `{"error":{"code":"validation","message":"count should be less than 500"}}` + "\n",
		},
		{
			name:               "failed request with negative count param",
			url:                "/?count=-1",
			expectedStatusCode: http.StatusBadRequest,
			compareBody:        true,
			expectedBody:       `{"error":{"code":"validation","message":"count should be bigger than 0"}}` + "\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", tc.url, nil)

			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)

			if response.Code != tc.expectedStatusCode {
				t.Errorf("Response code is %d, want 200", response.Code)
			}

			if tc.compareBody {
				body, _ := ioutil.ReadAll(response.Body)
				if string(body) != tc.expectedBody {
					t.Errorf("Body is %s, want %s", string(body), tc.expectedBody)
				}
			}
		})
	}
}
