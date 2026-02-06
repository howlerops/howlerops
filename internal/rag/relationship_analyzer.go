package rag

import "github.com/sirupsen/logrus"

// RelationshipAnalyzer constructs foreign-key graphs; a thin placeholder for now.
type RelationshipAnalyzer struct{ logger *logrus.Logger }

func NewRelationshipAnalyzer(logger *logrus.Logger) *RelationshipAnalyzer {
	return &RelationshipAnalyzer{logger: logger}
}
