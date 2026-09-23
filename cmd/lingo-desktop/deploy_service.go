package main

import (
	"context"
	"fmt"

	"lingo-translate/pkg/app"
	"lingo-translate/pkg/merger"
	"lingo-translate/pkg/plugins/chanomhub"
)

type DeployService struct {
	session *Session
}

func NewDeployService(session *Session) *DeployService {
	return &DeployService{session: session}
}

// DeployLayer installs non-destructive JS translation layer (RPGM only)
func (d *DeployService) DeployLayer(gamePath, langName string) error {
	ws, err := d.session.GetWorkspace()
	if err != nil {
		return err
	}
	return ws.DeployLayer(gamePath, langName)
}

// ExportCopy patches files into a destination copy folder
func (d *DeployService) ExportCopy(gamePath, destPath string) error {
	ws, err := d.session.GetWorkspace()
	if err != nil {
		return err
	}
	return ws.ExportCopy(context.Background(), gamePath, destPath)
}

// Merge reconciles current translations with an updated version of the game
func (d *DeployService) Merge(newGamePath string) (*merger.MergeStats, error) {
	ws, err := d.session.GetWorkspace()
	if err != nil {
		return nil, err
	}
	return ws.MergeNewVersion(context.Background(), newGamePath)
}

// Publish archives and publishes mod to Chanomhub
func (d *DeployService) Publish(opts app.PublishOptions) (*chanomhub.PublishResult, error) {
	if opts.GameDir == "" {
		ws, err := d.session.GetWorkspace()
		if err == nil && ws.Project() != nil {
			opts.GameDir = ws.Project().SourcePath
		}
	}
	if opts.GameDir == "" {
		return nil, fmt.Errorf("game directory is required")
	}
	return app.Publish(context.Background(), opts)
}
