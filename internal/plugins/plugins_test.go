// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

// Suite of plugins tests
type PluginsTestSuite struct {
	suite.Suite
}

func (s *PluginsTestSuite) SetupSuite() {
}

func (s *PluginsTestSuite) TearDownSuite() {
}

func (s *PluginsTestSuite) SetupTest() {
}

func (s *PluginsTestSuite) TearDownTest() {
}

func TestPlugins(t *testing.T) {
	suite.Run(t, &PluginsTestSuite{})
}
