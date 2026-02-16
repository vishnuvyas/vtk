// Package stedi - tools for communicating with stedi eligibility engine
// and claims warehouse
package stedi

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
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
	FirstName         string    `json:"firstName"`
	LastName          string    `json:"lastName"`
	DateOfBirth       StediDate `json:"dateOfBirth"`
	MemberID          string    `json:"memberId"`
	ExternalPatientID string    `json:"-"`
	payerName         string
	planName          string
}

type ExtendedSubscriber struct {
	Subscriber   StediSubscriber
	StediPayerID string
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

func LoadSubscriberInfoCSV(filename string) ([]ExtendedSubscriber, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open csv %q: %w", filename, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	// r.Comma = ',' // set if needed

	// Read header row
	header, err := r.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("csv %q: empty file", filename)
		}
		return nil, fmt.Errorf("read header from %q: %w", filename, err)
	}

	// Normalize headers and build index
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[norm(h)] = i
	}

	required := []string{"firstname", "lastname", "dateofbirth", "memberid", "stedipayerid", "externalpatientid", "payername", "planname"}
	for _, k := range required {
		if _, ok := idx[k]; !ok {
			return nil, fmt.Errorf("csv %q: missing required header %q", filename, k)
		}
	}

	get := func(rec []string, key string) (string, bool) {
		i := idx[key]
		if i < 0 || i >= len(rec) {
			return "", false
		}
		val := strings.TrimSpace(rec[i])
		return val, (val != "")
	}

	var subs []ExtendedSubscriber
	line := 1 // header is line 1
	skippedDOB := 0
	allSkipped := 0

	for {
		rec, err := r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("read csv %q at line %d: %w", filename, line+1, err)
		}
		line++

		firstName, ok1 := get(rec, "firstname")
		lastName, ok2 := get(rec, "lastname")
		payerID, ok3 := get(rec, "stedipayerid")
		memberID, ok4 := get(rec, "memberid")
		dobStr, ok5 := get(rec, "dateofbirth")
		externalPatientID, ok6 := get(rec, "externalPatientId")
		payerName, ok7 := get(rec, "payername")
		planName, ok8 := get(rec, "planname")
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 || !ok7 || !ok8 {
			// row shorter than header (or malformed)
			allSkipped++
			slog.Warn("Skipping short/malformed row", "line", line, "len", len(rec))
			continue
		}

		dob, err := time.Parse("20060102", dobStr)
		if err != nil {
			skippedDOB++
			slog.Warn("Skipping row due to dob parse", "line", line, "dob", dobStr, "err", err)
			continue
		}

		subs = append(subs, ExtendedSubscriber{
			StediPayerID: payerID,
			Subscriber: StediSubscriber{
				FirstName:         firstName,
				LastName:          lastName,
				MemberID:          memberID,
				DateOfBirth:       StediDate(dob),
				ExternalPatientID: externalPatientID,
				payerName:         payerName,
				planName:          planName,
			},
		})
	}

	if skippedDOB > 0 || allSkipped > 0 {
		slog.Warn("CSV load completed with skipped rows", "file", filename, "skippedDOB", skippedDOB, "otherSkipped", allSkipped)
	}
	return subs, nil
}

func (s *StediClient) RealtimeEligibility(ctx context.Context, stediPayerID string, subscriber StediSubscriber) (string, error) {
	// do the actual API call here.
	// construct the message according to documentation
	patientKey := fmt.Sprintf("%s-%s-%s", subscriber.FirstName, subscriber.LastName, subscriber.DateOfBirth)
	const namespaceStr = "6ba7b810-98ed-11da-adc0-2cd803534e97"
	namespaceUUID := uuid.MustParse(namespaceStr)
	deterministicUUID := uuid.NewSHA1(namespaceUUID, []byte(patientKey))

	if subscriber.ExternalPatientID == "" {
		subscriber.ExternalPatientID = deterministicUUID.String()
	}

	message := struct {
		ExternalPatientID string `json:"externalPatientId"`
		Encounter         struct {
			ServiceTypeCodes []string `json:"serviceTypeCodes"`
		} `json:"encounter"`
		Provider struct {
			NPI              string `json:"npi"`
			OrganizationName string `json:"organizationName"`
		} `json:"provider"`
		Subscriber   StediSubscriber `json:"subscriber"`
		StediPayerID string          `json:"tradingPartnerServiceId"`
	}{
		ExternalPatientID: subscriber.ExternalPatientID,
		Encounter: struct {
			ServiceTypeCodes []string "json:\"serviceTypeCodes\""
		}{
			ServiceTypeCodes: []string{"30"},
		},
		Provider: struct {
			NPI              string "json:\"npi\""
			OrganizationName string "json:\"organizationName\""
		}{
			NPI:              s.npi,
			OrganizationName: s.providerName,
		},
		Subscriber:   subscriber,
		StediPayerID: stediPayerID,
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
	bodyBytes, err := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		slog.Error("API returned non 200 response", "statusCode", resp.StatusCode, "status", resp.Status, "body", string(bodyBytes))
		return "", fmt.Errorf("non 200 response from the api: %d (%s)", resp.StatusCode, resp.Status)
	}
	if err != nil {
		slog.Error("error reading body", "err", err)
		return "", err
	}

	var respMessage map[string]any
	json.Unmarshal(bodyBytes, &respMessage)
	respMessage["_payer"] = subscriber.payerName
	respMessage["_planName"] = subscriber.planName
	respMessage["_patientUuid"] = subscriber.ExternalPatientID
	respMessage["_firstName"] = subscriber.FirstName
	respMessage["_lastName"] = subscriber.LastName
	respMessage["_dateOfBirth"] = time.Time(subscriber.DateOfBirth).Local().Format("20060201")

	respBytes, err := json.Marshal(respMessage)
	if err != nil {
		return "", fmt.Errorf("unable to marshall enriched response: %v", err)
	}

	// print the resonse
	return string(respBytes), nil
}
