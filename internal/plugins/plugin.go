// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // Internal package
package plugins

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/orch-library/go/dazl"
)

var log = dazl.GetPackageLogger()

type Event struct {
	EventType    string
	Organization string
	Name         string
	UUID         string
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

func Dispatch(ctx context.Context, event Event, data PluginData) error {
	if data == nil {
		data = &map[string]string{}
	}
	for _, plugin := range plugins {
		log.Infof("Sending event %v to %s", event, plugin.Name())
		var err error
		if event.EventType == "create" {
			err = plugin.CreateEvent(ctx, event, data)
		} else if event.EventType == "delete" {
			err = plugin.DeleteEvent(ctx, event, data)
		} else {
			err = fmt.Errorf("unknown event type: %s", event.EventType)
		}
		if err != nil {
			log.Infof("Error processing event %v by %s, error is %v", event, plugin.Name(), err)
			return err
		}
		log.Infof("Successfully processed event %v by %s", event, plugin.Name())
	}
	log.Infof("Done dispatching event: %v", event)
	return nil
}

func Register(plugin Plugin) {
	plugins = append(plugins, plugin)
}

func RemoveAllPlugins() {
	plugins = []Plugin{}
}
