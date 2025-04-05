package pkg

type Slot struct {
	Id    int    `json:"id"`
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type Reserve struct {
	ProceedingId string `json:"proceedingId"`
	SlotId       int    `json:"slotId"`
	Name         string `json:"name"`
	LastName     string `json:"lastName"`
	DateOfBirth  string `json:"dateOfBirth"`
}
