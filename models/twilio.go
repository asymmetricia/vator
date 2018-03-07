package models

import (
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/pdbogen/vator/log"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var log = Log

type Twilio struct {
	Sid             string
	AuthToken       string `json:"auth_token"`
	Status          string
	SubResourceUris map[string]string `json:"subresource_uris"`
}

type IncomingPhoneNumbersResponse struct {
	IncomingPhoneNumbers []PhoneNumber `json:"incoming_phone_numbers"`
}

type PhoneNumber struct {
	Sid         string
	PhoneNumber string `json:"phone_number"`
}

func (t *Twilio) SendSms(to, msg string) error {
	from, err := t.PhoneNumber()
	if err != nil {
		return fmt.Errorf("sending to %q: %s", to, err)
	}
	if from == "" {
		return errors.New("no error, but twilio phone number was blank")
	}
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", from)
	data.Set("Body", msg)
	rdr := strings.NewReader(data.Encode())
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.Sid),
		rdr)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-type", "application/x-www-form-urlencoded")

	if err != nil {
		return fmt.Errorf("building request to send to %q: %s", to, err)
	}

	req.SetBasicAuth(t.Sid, t.AuthToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request to send to %q: %s", to, err)
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Warning("failed reading request body: %s", err)
		}
		return fmt.Errorf("non-2XX %d sending request to send to %q: %q", res.StatusCode, to, string(body))
	}
	return nil
}

func (t *Twilio) PhoneNumber() (string, error) {
	uri, ok := t.SubResourceUris["incoming_phone_numbers"]
	if !ok {
		return "", errors.New("subresources URIs did not contain incoming_phone_numbers")
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.twilio.com/%s", uri), nil)
	if err != nil {
		return "", fmt.Errorf("building API request for incoming_phone_numbers: %s", err)
	}
	req.SetBasicAuth(t.Sid, t.AuthToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending API request for incoming_phone_numbers: %s", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading response body for incoming_phone_numbers: %s", err)
	}

	if res.StatusCode/100 != 2 {
		return "", fmt.Errorf("non-2XX %d response for incoming_phone_numbers: %s", res.StatusCode, string(body))
	}

	var resObj IncomingPhoneNumbersResponse

	if err := json.Unmarshal(body, &resObj); err != nil {
		return "", fmt.Errorf("parsing response body %q for incoming_phone_numbers: %s", string(body), err)
	}

	if len(resObj.IncomingPhoneNumbers) == 0 {
		return "", fmt.Errorf("no phone numbers")
	}
	return resObj.IncomingPhoneNumbers[0].PhoneNumber, nil
}

func NewTwilio(twSid, twToken string) (*Twilio, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s.json", twSid), nil)
	if err != nil {
		return nil, fmt.Errorf("building API request: %s", err)
	}
	req.SetBasicAuth(twSid, twToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending API request: %s", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading response body: %s", err)
	}
	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("non-2XX %d sending API request: %s", res.StatusCode, string(body))
	}
	ret := &Twilio{}
	if err := json.Unmarshal(body, ret); err != nil {
		return nil, fmt.Errorf("parsing json %q: %s", string(body), err)
	}

	if ret.Status != "active" {
		return nil, fmt.Errorf("twilio account status is %q, not active", ret.Status)
	}

	if _, err := ret.PhoneNumber(); err != nil {
		return nil, fmt.Errorf("twilio account has no phone numbers: %s", err)
	}

	return ret, nil
}
