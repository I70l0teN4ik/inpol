package pkg

type Slot struct {
	Id    int    `json:"id"`
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type ReservationQueue struct {
	Localization string `json:"localization"`
	Prefix       string `json:"prefix"`
	ID           string `json:"id"`
	Polish       string `json:"polish"`
	English      string `json:"english"`
	Russian      string `json:"russian"`
	Ukrainian    string `json:"ukrainian"`
}

type Reserve struct {
	ProceedingId string `json:"proceedingId"`
	SlotId       int    `json:"slotId"`
	Name         string `json:"name"`
	LastName     string `json:"lastName"`
	DateOfBirth  string `json:"dateOfBirth"`
}
