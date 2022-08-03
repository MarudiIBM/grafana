package store

import (
	"context"

	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/plugins/manager/registry"
)

var _ plugins.Store = (*Service)(nil)
var _ plugins.RendererManager = (*Service)(nil)
var _ plugins.SecretsPluginManager = (*Service)(nil)

type Service struct {
	pluginRegistry registry.Service
}

func ProvideService(pluginRegistry registry.Service) *Service {
	return &Service{
		pluginRegistry: pluginRegistry,
	}
}

func (s *Service) Plugin(ctx context.Context, pluginID string) (plugins.PluginDTO, bool) {
	p, exists := s.plugin(ctx, pluginID)
	if !exists {
		return plugins.PluginDTO{}, false
	}

	return p.ToDTO(), true
}

func (s *Service) Plugins(ctx context.Context, pluginTypes ...plugins.Type) []plugins.PluginDTO {
	// if no types passed, assume all
	if len(pluginTypes) == 0 {
		pluginTypes = plugins.PluginTypes
	}

	var requestedTypes = make(map[plugins.Type]struct{})
	for _, pt := range pluginTypes {
		requestedTypes[pt] = struct{}{}
	}

	pluginsList := make([]plugins.PluginDTO, 0)
	for _, p := range s.availablePlugins(ctx) {
		if _, exists := requestedTypes[p.Type]; exists {
			pluginsList = append(pluginsList, p.ToDTO())
		}
	}
	return pluginsList
}

func (s *Service) Renderer() *plugins.Plugin {
	for _, p := range s.availablePlugins(context.TODO()) {
		if p.IsRenderer() {
			return p
		}
	}

	return nil
}

func (s *Service) SecretsManager() *plugins.Plugin {
	for _, p := range s.availablePlugins(context.TODO()) {
		if p.IsSecretsManager() {
			return p
		}
	}

	return nil
}

// plugin finds a plugin with `pluginID` from the registry that is not decommissioned
func (s *Service) plugin(ctx context.Context, pluginID string) (*plugins.Plugin, bool) {
	p, exists := s.pluginRegistry.Plugin(ctx, pluginID)
	if !exists {
		return nil, false
	}

	if p.IsDecommissioned() {
		return nil, false
	}

	return p, true
}

// availablePlugins returns all non-decommissioned plugins from the registry
func (s *Service) availablePlugins(ctx context.Context) []*plugins.Plugin {
	var res []*plugins.Plugin
	for _, p := range s.pluginRegistry.Plugins(ctx) {
		if !p.IsDecommissioned() {
			res = append(res, p)
		}
	}
	return res
}

func (s *Service) Routes() []*plugins.StaticRoute {
	staticRoutes := make([]*plugins.StaticRoute, 0)

	for _, p := range s.availablePlugins(context.TODO()) {
		if p.StaticRoute() != nil {
			staticRoutes = append(staticRoutes, p.StaticRoute())
		}
	}
	return staticRoutes
}