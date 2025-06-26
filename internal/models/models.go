package models

func Models() []interface{} {
	return []interface{}{
		&Matter{},
		&MatterActivity{},
		&MatterActivityCategory{},
		&MatterRelated{},
		&MatterStatus{},
	}
}
