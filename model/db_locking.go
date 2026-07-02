package model

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func txForUpdate(tx *gorm.DB) *gorm.DB {
	if tx == nil || tx.Dialector == nil {
		return tx
	}
	if strings.EqualFold(tx.Dialector.Name(), "sqlite") {
		return tx
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}
