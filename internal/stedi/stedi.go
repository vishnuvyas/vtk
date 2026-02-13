// Package stedi - tools for communicating with stedi eligibility engine
// and claims warehouse
package stedi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type StediClient struct {
	apiKey string
	url    string
	// include provider info here.
	providerName string
	npi          string
	client       *http.Client
}

type StediDate time.Time

func (d StediDate) MarshalJSON() ([]byte, error) {
	formatted := time.Time(d).Format("20060102")
	return json.Marshal(formatted)
}

func (d *StediDate) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = s[1 : len(s)-1]
	t, err := time.Parse("20060102", s)
	if err != nil {
		return err
	}
	*d = StediDate(t)
	return nil
}

type StediSubscriber struct {
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
	DateOfBirth StediDate `json:"dateOfBirth"`
	MemberID    string    `json:"memberId"`
}

func NewStediClient(providerName, npi, apiKey string) *StediClient {
	StediURL := "https://healthcare.us.stedi.com/2024-04-01/change/medicalnetwork/eligibility/v3"
	return &StediClient{
		apiKey:       apiKey,
		url:          StediURL,
		providerName: providerName,
		npi:          npi,
		client:       &http.Client{},
	}
}

func (s *StediClient) RealtimeEligibility(ctx context.Context, stediPayerId string, subscriber StediSubscriber) (string, error) {
	// do the actual API call here.
	// construct the message according to documentation
	message := struct {
		ExternalPatientID string `json:"externalPatientId"`
		Provider          struct {
			NPI              string `json:"npi"`
			OrganizationName string `json:"organiztationName"`
		}
		Subscriber   StediSubscriber `json:"subscriber"`
		StediPayerID string          `json:"tradingPartnerServiceId"`
	}{
		ExternalPatientID: "patient_uuid",
		Provider: struct {
			NPI              string "json:\"npi\""
			OrganizationName string "json:\"organiztationName\""
		}{
			NPI:              s.npi,
			OrganizationName: s.providerName,
		},
		Subscriber:   subscriber,
		StediPayerID: stediPayerId,
	}
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		slog.Error("Unable to marshall request", "err", err)
		return "", err
	}
	// setup the request here along with the authorization headers
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewBuffer(jsonMessage))
	req.Header.Set("Authorization", s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		slog.Error("error creating request", "err", err)
		return "", err
	}

	// do the request
	resp, err := s.client.Do(req)
	if err != nil {
		slog.Error("error while execuiting api call", "err", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		slog.Error("API returned non 200 response", "statusCode", resp.StatusCode, "status", resp.Status)
		return "", fmt.Errorf("non 200 response from the api: %d (%s)", resp.StatusCode, resp.Status)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("error reading body", "err", err)
		return "", err
	}
	// print the resonse
	fmt.Println(string(bodyBytes))
	return "", nil
}
