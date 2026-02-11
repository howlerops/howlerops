package rag

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

func newSilentVectorLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func TestNewVectorStoreUnsupportedType(t *testing.T) {
	_, err := NewVectorStore(&VectorStoreConfig{Type: "unknown"}, newSilentVectorLogger())
	if err == nil {
		t.Fatalf("expected error for unsupported store type")
	}
}

func TestNewVectorStoreMissingMySQLConfig(t *testing.T) {
	_, err := NewVectorStore(&VectorStoreConfig{Type: "mysql"}, newSilentVectorLogger())
	if err == nil {
		t.Fatalf("expected error when MySQL config missing")
	}
}

func TestNewVectorStoreDefaultSQLiteConfig(t *testing.T) {
	cfg := &VectorStoreConfig{}
	store, err := NewVectorStore(cfg, newSilentVectorLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatalf("expected store instance")
	}
	if cfg.SQLiteConfig == nil {
		t.Fatalf("expected default sqlite config to be populated")
	}
}
