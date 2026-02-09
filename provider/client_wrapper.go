package inwx

import (
	"fmt"
	"time"

	inwx "github.com/nrdcg/goinwx"
)

const zonesCacheTTL = 5 * time.Minute

type ClientWrapper struct {
	client         *inwx.Client
	zonesCache     []string
	zonesCacheTime time.Time
}

type AbstractClientWrapper interface {
	login() (*inwx.LoginResponse, error)
	logout() error
	getRecords(domain string) (*[]inwx.NameserverRecord, error)
	getZones() (*[]string, error)
	createRecord(request *inwx.NameserverRecordRequest) error
	updateRecord(recID string, request *inwx.NameserverRecordRequest) error
	deleteRecord(recID string) error
}

func (w *ClientWrapper) login() (*inwx.LoginResponse, error) {
	return w.client.Account.Login()
}

func (w *ClientWrapper) logout() error {
	return w.client.Account.Logout()
}

func (w *ClientWrapper) getRecords(domain string) (*[]inwx.NameserverRecord, error) {
	zone, err := w.client.Nameservers.Info(&inwx.NameserverInfoRequest{Domain: domain})
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve records for zone %s: %w", domain, err)
	}
	return &zone.Records, nil
}

func (w *ClientWrapper) getZones() (*[]string, error) {
	if w.zonesCache != nil && time.Since(w.zonesCacheTime) < zonesCacheTTL {
		zones := w.zonesCache
		return &zones, nil
	}

	zones := []string{}
	page := 1
	for {
		response, err := w.client.Nameservers.ListWithParams(&inwx.NameserverListRequest{
			Domain:    "*",
			Page:      page,
			PageLimit: 100,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list nameserver zones (page %d): %w", page, err)
		}
		for _, domain := range response.Domains {
			zones = append(zones, domain.Domain)
		}
		if len(response.Domains) == 0 || len(zones) >= response.Count {
			break
		}
		page++
	}

	w.zonesCache = zones
	w.zonesCacheTime = time.Now()

	return &zones, nil
}

func (w *ClientWrapper) createRecord(request *inwx.NameserverRecordRequest) error {
	_, err := w.client.Nameservers.CreateRecord(request)
	return err
}

func (w *ClientWrapper) updateRecord(recID string, request *inwx.NameserverRecordRequest) error {
	return w.client.Nameservers.UpdateRecord(recID, request)
}

func (w *ClientWrapper) deleteRecord(recID string) error {
	return w.client.Nameservers.DeleteRecord(recID)
}
