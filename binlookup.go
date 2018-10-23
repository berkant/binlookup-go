// Package binlookup is the Go port of github.com/paylike/binlookup
// to look up any BIN/IIN via lookup.binlist.net.
package binlookup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/pkg/errors"
)

// Client is the default HTTP client used by the package.
var Client = &http.Client{Timeout: 10 * time.Second}

// StatusCodeError is an error returned by `Search` in the event
// of a HTTP status code, other than `http.StatusOK`, sent by upstream.
type StatusCodeError int

func (s StatusCodeError) Error() string {
	return fmt.Sprintf("%d %v", s, http.StatusText(int(s)))
}

// Number is a placeholder for the `number` JSON object in `BIN`.
type Number struct {
	Length int
	Luhn   bool
}

// Country is a placeholder for the `country` JSON object in `BIN`.
type Country struct {
	Numeric,
	Name,
	Emoji,
	Currency string

	Short string `json:"alpha2"`

	Lat  float64 `json:"latitude"`
	Long float64 `json:"longitude"`
}

// Bank is a placeholder for the `bank` JSON object in `BIN`.
type Bank struct {
	Name, URL, Phone, City string
}

// BIN is the placeholder to host the deserialized JSON payload
// returned by upstream.
type BIN struct {
	Number              Number
	Scheme, Type, Brand string
	Prepaid             bool
	Country             Country
	Bank                Bank
}

// Search makes a BIN lookup request to Upstream.
//
// An error is returned when:
// 	- The bin parameter given to the function is incorrect in format.
// 	- HTTP request fails.
// 	- HTTP status code is not equal to 200, otherwise known as http.StatusOK.
// 	- The unmarshaling of the returned raw JSON payload fails.
//
// Since this function is dependent on a 3rd party service, the most flexible way
// to handle status codes would be returning a special error, which is StatusCodeError
// in this case.
// This is because there are many and many status codes that can be returned by a Web service.
// Thus, by returning the status code as an error, it's being made possible for clients
// to handle them on their own.
//
// Possible status codes that may be returned by upstream are (according to https://binlist.net/):
// 	- 400, http.StatusBadRequest: Returned when given BIN is incorrect in format. Since it's being checked initially, this status code isn't possible.
//	- 429, http.StatusTooManyRequests: Returned in possible throttling. See the link above to check the toleration.
//	- 200, http.StatusOK: In case of 200, the error would already be nil.
//	- 404, http.StatusNotFound: This is returned when BIN isn't present in the DB which upstream queries.
//	- And there may happen many more if the service is upset.
//
// These codes can be extracted by asserting StatusCodeError type over
// the error returned by Cause function of https://github.com/pkg/errors.
func Search(bin string) (b *BIN, err error) {
	ok := regexp.MustCompile(`^[1-9]\d{3,15}$`).MatchString(bin)
	if !ok {
		err = errors.New("BIN must be fully numerical, first digit must be in range of 1-9, and the next digits must be 3-15 characters long.")
		return
	}

	resp, err := Client.Get(fmt.Sprintf("https://lookup.binlist.net/%v", bin))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	switch s := resp.StatusCode; s {
	case http.StatusOK:
		break
	default:
		err = errors.Wrap(StatusCodeError(s), "Failed Due to Status Code Error")
		return
	}

	if err = json.NewDecoder(resp.Body).Decode(&b); err != nil {
		err = errors.WithMessage(err, "JSON Unmarshaling Failed")
		return
	}

	return
}
