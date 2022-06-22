package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/datasources"
)

// Gets public dashboard via access token
func (dr *DashboardServiceImpl) GetPublicDashboard(ctx context.Context, accessToken string) (*models.Dashboard, error) {
	pubdash, d, err := dr.dashboardStore.GetPublicDashboard(ctx, accessToken)

	if err != nil {
		return nil, err
	}

	if pubdash == nil || d == nil {
		return nil, models.ErrPublicDashboardNotFound
	}

	if !pubdash.IsEnabled {
		return nil, models.ErrPublicDashboardNotFound
	}

	ts := pubdash.BuildTimeSettings(d)
	d.Data.SetPath([]string{"time", "from"}, ts.From)
	d.Data.SetPath([]string{"time", "to"}, ts.To)

	return d, nil
}

// GetPublicDashboardConfig is a helper method to retrieve the public dashboard configuration for a given dashboard from the database
func (dr *DashboardServiceImpl) GetPublicDashboardConfig(ctx context.Context, orgId int64, dashboardUid string) (*models.PublicDashboard, error) {
	pdc, err := dr.dashboardStore.GetPublicDashboardConfig(ctx, orgId, dashboardUid)
	if err != nil {
		return nil, err
	}

	return pdc, nil
}

// SavePublicDashboardConfig is a helper method to persist the sharing config
// to the database. It handles validations for sharing config and persistence
func (dr *DashboardServiceImpl) SavePublicDashboardConfig(ctx context.Context, dto *dashboards.SavePublicDashboardConfigDTO) (*models.PublicDashboard, error) {
	if len(dto.DashboardUid) == 0 {
		return nil, models.ErrDashboardIdentifierNotSet
	}

	if dto.PublicDashboard.Uid == "" {
		return dr.savePublicDashboardConfig(ctx, dto)
	}

	return dr.updatePublicDashboardConfig(ctx, dto)
}

func (dr *DashboardServiceImpl) savePublicDashboardConfig(ctx context.Context, dto *dashboards.SavePublicDashboardConfigDTO) (*models.PublicDashboard, error) {
	uid, err := dr.dashboardStore.GenerateNewPublicDashboardUid(ctx)
	if err != nil {
		return nil, err
	}

	accessToken, err := GenerateAccessToken()
	if err != nil {
		return nil, err
	}

	cmd := models.SavePublicDashboardConfigCommand{
		DashboardUid: dto.DashboardUid,
		OrgId:        dto.OrgId,
		PublicDashboard: models.PublicDashboard{
			Uid:          uid,
			DashboardUid: dto.DashboardUid,
			OrgId:        dto.OrgId,
			IsEnabled:    dto.PublicDashboard.IsEnabled,
			TimeSettings: dto.PublicDashboard.TimeSettings,
			CreatedBy:    dto.UserId,
			CreatedAt:    time.Now(),
			AccessToken:  accessToken,
		},
	}

	return dr.dashboardStore.SavePublicDashboardConfig(ctx, cmd)
}

func (dr *DashboardServiceImpl) updatePublicDashboardConfig(ctx context.Context, dto *dashboards.SavePublicDashboardConfigDTO) (*models.PublicDashboard, error) {
	cmd := models.SavePublicDashboardConfigCommand{
		PublicDashboard: models.PublicDashboard{
			Uid:          dto.PublicDashboard.Uid,
			IsEnabled:    dto.PublicDashboard.IsEnabled,
			TimeSettings: dto.PublicDashboard.TimeSettings,
			UpdatedBy:    dto.UserId,
			UpdatedAt:    time.Now(),
		},
	}

	return dr.dashboardStore.UpdatePublicDashboardConfig(ctx, cmd)
}

// BuildPublicDashboardMetricRequest merges public dashboard parameters with
// dashboard and returns a metrics request to be sent to query backend
func (dr *DashboardServiceImpl) BuildPublicDashboardMetricRequest(ctx context.Context, accessToken string, panelId int64) (dtos.MetricRequest, *models.SignedInUser, error) {
	var metricRequest dtos.MetricRequest
	var anonymousUser *models.SignedInUser

	publicDashboard, dashboard, err := dr.dashboardStore.GetPublicDashboard(ctx, accessToken)
	if err != nil {
		return metricRequest, anonymousUser, err
	}
	if publicDashboard == nil || dashboard == nil {
		return metricRequest, anonymousUser, models.ErrPublicDashboardNotFound
	}
	if !publicDashboard.IsEnabled {
		return metricRequest, anonymousUser, models.ErrPublicDashboardNotFound
	}

	// get queries
	queriesByPanel := models.GetQueriesFromDashboard(dashboard.Data)
	if _, ok := queriesByPanel[panelId]; !ok {
		return dtos.MetricRequest{}, anonymousUser, models.ErrPublicDashboardPanelNotFound
	}
	metricRequest.Queries = queriesByPanel[panelId]

	// get time range
	ts := publicDashboard.BuildTimeSettings(dashboard)
	metricRequest.From = ts.From
	metricRequest.To = ts.To

	// Build user with permissions on datasources
	var uids []string
	for _, query := range metricRequest.Queries {
		uids = append(uids, query.Get("datasource").Get("uid").MustString())
	}
	anonymousUser = &models.SignedInUser{OrgId: dashboard.OrgId, Permissions: make(map[int64]map[string][]string)}
	permissions := make(map[string][]string)
	datasourceScope := fmt.Sprintf("datasources:uid:%s", strings.Join(uids, ","))
	permissions[datasources.ActionQuery] = []string{datasourceScope}
	permissions[datasources.ActionRead] = []string{datasourceScope}
	anonymousUser.Permissions[dashboard.OrgId] = permissions

	return metricRequest, anonymousUser, nil
}

// generates a uuid formatted without dashes to use as access token
func GenerateAccessToken() (string, error) {
	token, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", token), nil
}
