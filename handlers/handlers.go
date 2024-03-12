package handlers

import "magpanel/database"

type HandlersStruct struct {
	dataBase *database.DatabaseStruct
}

func NewHandlers(dataBase *database.DatabaseStruct) *HandlersStruct {
	return &HandlersStruct{dataBase: dataBase}
}
