package main

import "time"

// CleanProduct er den rensede, silver-lignende model.
type CleanProduct struct {
	ProductName string
	Color       string
	Department  string
	Price       int64
	Campaign    string
	IsActive    bool
}

func transform(raw []RawProduct) []CleanProduct {
	now := time.Now()
	var clean []CleanProduct

	for _, r := range raw {
		if r.Price <= 0 {
			continue // dropper ugyldige priser
		}

		// Tjek det faktiske datoformat i jeres data og juster layout-strengen.
		since, errS := time.Parse(time.RFC3339, r.DateSoldSince)
		until, errU := time.Parse(time.RFC3339, r.DateSoldUntil)
		active := errS == nil && errU == nil && now.After(since) && now.Before(until)

		clean = append(clean, CleanProduct{
			ProductName: r.ProductName,
			Color:       r.Color,
			Department:  r.Department,
			Price:       int64(r.Price),
			Campaign:    r.Campaign,
			IsActive:    active,
		})
	}

	return clean
}
