package main

import (
	"fmt"

	"github.com/jbeck018/howlerops/pkg/storage"
	"github.com/jbeck018/howlerops/services"
)

// WailsReportService is a Wails-compatible service that wraps services.ReportService
type WailsReportService struct {
	deps          *SharedDeps
	reportService *services.ReportService
}

// NewWailsReportService creates a new WailsReportService instance
func NewWailsReportService(deps *SharedDeps, rs *services.ReportService) *WailsReportService {
	return &WailsReportService{
		deps:          deps,
		reportService: rs,
	}
}

// ensureReportService checks if the report service is initialized
func (s *WailsReportService) ensureReportService() error {
	if s.reportService == nil {
		return fmt.Errorf("report service not initialised")
	}
	return nil
}

// ListReports returns summaries for all saved reports
func (s *WailsReportService) ListReports() ([]storage.ReportSummary, error) {
	if err := s.ensureReportService(); err != nil {
		return nil, err
	}
	return s.reportService.ListReports()
}

// GetReport loads a full report definition by ID
func (s *WailsReportService) GetReport(id string) (*storage.Report, error) {
	if err := s.ensureReportService(); err != nil {
		return nil, err
	}
	return s.reportService.GetReport(id)
}

// SaveReport creates or updates a report definition
func (s *WailsReportService) SaveReport(report storage.Report) (*storage.Report, error) {
	if err := s.ensureReportService(); err != nil {
		return nil, err
	}
	return s.reportService.SaveReport(&report)
}

// DeleteReport removes a report definition
func (s *WailsReportService) DeleteReport(id string) error {
	if err := s.ensureReportService(); err != nil {
		return err
	}
	return s.reportService.DeleteReport(id)
}

// RunReport executes all (or selected) components within a report
func (s *WailsReportService) RunReport(req services.ReportRunRequest) (*services.ReportRunResponse, error) {
	if err := s.ensureReportService(); err != nil {
		return nil, err
	}
	return s.reportService.RunReport(&req)
}
