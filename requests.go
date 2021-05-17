package main

import (
	"fmt"
	"net/http"
	"strconv"
)

const (
	defaultCount = 5
	// maxCount is an upped bound of content we can request.
	maxCount = 500
)

type getContentRequest struct {
	// count represents the number of items desired.
	count int
	// offset represents the number of items previously requested. Content should be offset by this number.
	offset int
}

// getContent returns some content from different providers based on params: count and offset.
func (a App) getContent(w http.ResponseWriter, req *http.Request) {
	var params = getContentRequest{
		count: defaultCount,
	}

	urlParams := req.URL.Query()

	if urlParams.Get("offset") != "" {
		offset, err := strconv.ParseInt(urlParams.Get("offset"), 10, 64)
		if err != nil {
			encodeErrorResp(
				Error{Code: ErrValidation, Message: fmt.Sprintf("bad offset param: %v", err)},
				w,
			)
			return
		}

		if offset < 0 {
			encodeErrorResp(
				Error{Code: ErrValidation, Message: "offset should be bigger than 0"},
				w,
			)
			return
		}

		params.offset = int(offset)
	}

	if urlParams.Get("count") != "" {
		count, err := strconv.ParseInt(urlParams.Get("count"), 10, 64)
		if err != nil {
			encodeErrorResp(
				Error{Code: ErrValidation, Message: fmt.Sprintf("bad count param: %v", err)},
				w,
			)
			return
		}

		if count < 0 {
			encodeErrorResp(
				Error{Code: ErrValidation, Message: "count should be bigger than 0"},
				w,
			)
			return
		}

		if count > maxCount {
			encodeErrorResp(
				Error{Code: ErrValidation, Message: fmt.Sprintf("count should be less than %d", maxCount)},
				w,
			)
			return
		}

		params.count = int(count)
	}

	content, err := a.ContentProvider.Fetch(req.Context(), a.parseUserIP(req), params.count, params.offset)
	if err != nil {
		encodeErrorResp(err, w)
	}

	encodeJSONResponse(w, content)
	return
}

