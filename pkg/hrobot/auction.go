// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hrobot

import (
	"context"
	"time"
)

// AuctionService provides access to server market/auction related functions in the Hetzner Robot API.
type AuctionService struct {
	client *Client
}

// NewAuctionService creates a new AuctionService.
func NewAuctionService(client *Client) *AuctionService {
	return &AuctionService{client: client}
}

// AuctionServer represents a server available on the Hetzner auction market.
type AuctionServer struct {
	ID              uint32         `json:"id"`
	Name            string         `json:"name"`
	Description     []string       `json:"description"`
	Traffic         string         `json:"traffic"`
	Distributions   []string       `json:"dist"`
	Languages       []string       `json:"lang"`
	Datacenter      *string        `json:"datacenter"`
	CPU             string         `json:"cpu"`
	CPUBenchmark    uint32         `json:"cpu_benchmark"`
	MemorySize      float64        `json:"memory_size"`      // in GB
	HDDSize         float64        `json:"hdd_size"`         // in GB
	HDDText         string         `json:"hdd_text"`         // human-readable HDD description
	HDDCount        uint8          `json:"hdd_count"`        // number of primary HDDs
	Price           StringFloat    `json:"price"`            // monthly price net
	PriceVAT        StringFloat    `json:"price_vat"`        // monthly price gross
	PriceSetup      StringFloat    `json:"price_setup"`      // setup price net
	PriceSetupVAT   StringFloat    `json:"price_setup_vat"`  // setup price gross
	PriceHourly     StringFloat    `json:"price_hourly"`     // hourly price net
	PriceHourlyVAT  StringFloat    `json:"price_hourly_vat"` // hourly price gross
	FixedPrice      bool           `json:"fixed_price"`
	NextReduce      int64          `json:"next_reduce"`      // seconds until next price reduction
	NextReduceDate  string         `json:"next_reduce_date"` // timestamp of next price reduction
	OrderableAddons []AuctionAddon `json:"orderable_addons"`
}

// AuctionAddon represents an addon that can be purchased with an auction server.
type AuctionAddon struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Location *string             `json:"location"`
	Min      uint32              `json:"min"`
	Max      uint32              `json:"max"`
	Prices   []AuctionAddonPrice `json:"price"`
}

// AuctionAddonPrice represents the price for an addon in a specific location.
type AuctionAddonPrice struct {
	Location        string  `json:"location"`
	Price           float64 `json:"price"`
	PriceSetup      float64 `json:"price_setup"`
	PriceMonthly    float64 `json:"price_monthly"`
	PriceMonthlyVAT float64 `json:"price_monthly_vat"`
	PriceSetupVAT   float64 `json:"price_setup_vat"`
}

// NextReduceTime returns the next price reduction time as a time.Time.
func (a *AuctionServer) NextReduceTime() *time.Time {
	if a.NextReduceDate == "" {
		return nil
	}

	// Parse the timestamp in Berlin timezone
	t, err := time.Parse("2006-01-02 15:04:05", a.NextReduceDate)
	if err != nil {
		return nil
	}

	// Load Berlin timezone
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return nil
	}

	// Set the location
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
	return &t
}

// List retrieves all servers available on the auction market.
//
// GET /order/server_market/product
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-order-server-market-product
func (a *AuctionService) List(ctx context.Context) ([]AuctionServer, error) {
	path := "/order/server_market/product"
	var result []AuctionServer
	if err := a.client.GetWrappedList(ctx, path, "product", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Get retrieves a specific auction server by ID.
//
// GET /order/server_market/product/{id}
//
// See: https://robot.hetzner.com/doc/webservice/en.html#get-order-server-market-product-id
func (a *AuctionService) Get(ctx context.Context, id uint32) (*AuctionServer, error) {
	path := "/order/server_market/product/" + string(rune(id))
	var result AuctionServer
	if err := a.client.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
