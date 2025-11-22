package main

import (
	"fmt"

	"github.com/jbeck018/howlerops/backend-go/pkg/storage"
	"github.com/jbeck018/howlerops/services"
)

func (a *App) ensureReportService() (*services.ReportService, error) {
	if a.reportService == nil {
		return nil, fmt.Errorf("report service not initialised")
	}
	return a.reportService, nil
}

// ListReports returns summaries for all saved reports.
func (a *App) ListReports() ([]storage.ReportSummary, error) {
	service, err := a.ensureReportService()
	if err != nil {
		return nil, err
	}
	return service.ListReports()
}

// GetReport loads a full report definition by ID.
func (a *App) GetReport(id string) (*storage.Report, error) {
	service, err := a.ensureReportService()
	if err != nil {
		return nil, err
	}
	return service.GetReport(id)
}

// SaveReport creates or updates a report definition.
func (a *App) SaveReport(report storage.Report) (*storage.Report, error) {
	service, err := a.ensureReportService()
	if err != nil {
		return nil, err
	}
	return service.SaveReport(&report)
}

// DeleteReport removes a report definition.
func (a *App) DeleteReport(id string) error {
	service, err := a.ensureReportService()
	if err != nil {
		return err
	}
	return service.DeleteReport(id)
}

// RunReport executes all (or selected) components within a report.
func (a *App) RunReport(req services.ReportRunRequest) (*services.ReportRunResponse, error) {
	service, err := a.ensureReportService()
	if err != nil {
		return nil, err
	}
	return service.RunReport(&req)
}
