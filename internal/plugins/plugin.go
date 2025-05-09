// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"
	"fmt"
	nexushook "github.com/open-edge-platform/app-orch-tenant-controller/internal/nexus"
	"github.com/open-edge-platform/orch-library/go/dazl"
)

var log = dazl.GetPackageLogger()

type Event struct {
	EventType    string
	Organization string
	Name         string
	UUID         string
	Project      nexushook.NexusProjectInterface
}

type PluginData *map[string]string

type Plugin interface {
	Name() string
	Initialize(context.Context, PluginData) error
	CreateEvent(context.Context, Event, PluginData) error
	DeleteEvent(context.Context, Event, PluginData) error
}

var plugins = []Plugin{}

func Initialize(ctx context.Context) error {
	data := &map[string]string{}
	for _, plugin := range plugins {
		log.Infof("Initializing plugin %s", plugin.Name())
		err := plugin.Initialize(ctx, data)
		log.Infof("Done initializing plugin %s, result %v", plugin.Name(), err)
		if err != nil {
			return err
		}
	}
	log.Infof("Done initializing plugins")
	return nil
}

func Dispatch(ctx context.Context, event Event, hook *nexushook.Hook) error {
	data := &map[string]string{}
	var err error
	for _, plugin := range plugins {
		log.Infof("Sending event %v to %s", event, plugin.Name())
		if hook != nil {
			err = hook.SetWatcherStatusInProgress(event.Project, fmt.Sprintf("Processing project %s with %s", event.EventType, plugin.Name()))
		}
		if err != nil {
			return err
		}
		if event.EventType == "create" {
			err = plugin.CreateEvent(ctx, event, data)
		} else if event.EventType == "delete" {
			err = plugin.DeleteEvent(ctx, event, data)
		} else {
			err = fmt.Errorf("unknown event type: %s", event.EventType)
		}
		if err != nil {
			log.Infof("Error processing event %v by %s, error is %v", event, plugin.Name(), err)
		} else {
			log.Infof("Successfully processed event %v by %s", event, plugin.Name())
		}
		if err != nil {
			return err
		}
	}
	log.Infof("Done dispatching event: %v", event)
	return nil
}

func Register(plugin Plugin) {
	plugins = append(plugins, plugin)
}
