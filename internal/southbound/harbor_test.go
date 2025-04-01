// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Suite of harbor southbound tests
type HarborTestSuite struct {
	suite.Suite
	ctx        context.Context
	cancel     context.CancelFunc
	testServer *TestHarborServer
}

func (s *HarborTestSuite) SetupSuite() {
}

func (s *HarborTestSuite) TearDownSuite() {
}

func (s *HarborTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 1*time.Minute)
	K8sFactory = NewTestK8s
	testServer := TestHarborServer{}
	s.testServer = testServer.Start()
	testServer.WithConfigurationHandler(configurationHandler).
		WithProjectHandler(projectHandler).
		WithRobotsHandler(robotsHandler).
		WithProjectsGetRobotsHandler(projectsRobotsGetHandler).
		WithProjectsDeleteRobotsHandler(projectsRobotsDeleteHandler).
		WithPermissionsHandler(permissionsHandler).
		WithPingHandler(pingHandler)
}

func (s *HarborTestSuite) TearDownTest() {
	s.cancel()
}

func TestHarbor(t *testing.T) {
	suite.Run(t, &HarborTestSuite{})
}

type TestHarborServer struct {
	ConfigurationHandler        func(w http.ResponseWriter, r *http.Request)
	ProjectHandler              func(w http.ResponseWriter, r *http.Request)
	RobotsHandler               func(w http.ResponseWriter, r *http.Request)
	ProjectsRobotsGetHandler    func(w http.ResponseWriter, r *http.Request)
	ProjectsRobotsDeleteHandler func(w http.ResponseWriter, r *http.Request)
	ProjectsPermissionsHandler  func(w http.ResponseWriter, r *http.Request)
	PingHandler                 func(w http.ResponseWriter, r *http.Request)
	Server                      *httptest.Server
}

func (t *TestHarborServer) WithConfigurationHandler(configurationHandler func(w http.ResponseWriter, r *http.Request)) *TestHarborServer {
	t.ConfigurationHandler = configurationHandler
	return t
}

func (t *TestHarborServer) WithProjectHandler(projectHandler func(w http.ResponseWriter, r *http.Request)) *TestHarborServer {
	t.ProjectHandler = projectHandler
	return t
}

func (t *TestHarborServer) WithRobotsHandler(robotsHandler func(w http.ResponseWriter, r *http.Request)) *TestHarborServer {
	t.RobotsHandler = robotsHandler
	return t
}

func (t *TestHarborServer) WithProjectsGetRobotsHandler(projectsRobotsGetHandler func(w http.ResponseWriter, r *http.Request)) *TestHarborServer {
	t.ProjectsRobotsGetHandler = projectsRobotsGetHandler
	return t
}

func (t *TestHarborServer) WithProjectsDeleteRobotsHandler(projectsRobotsDeleteHandler func(w http.ResponseWriter, r *http.Request)) *TestHarborServer {
	t.ProjectsRobotsDeleteHandler = projectsRobotsDeleteHandler
	return t
}

func (t *TestHarborServer) WithPermissionsHandler(projectsPermissionsHandler func(w http.ResponseWriter, r *http.Request)) *TestHarborServer {
	t.ProjectsPermissionsHandler = projectsPermissionsHandler
	return t
}

func (t *TestHarborServer) WithPingHandler(pingHandler func(w http.ResponseWriter, r *http.Request)) *TestHarborServer {
	t.PingHandler = pingHandler
	return t
}

func configurationHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	body := string(b)
	_ = body
	if r.Method == http.MethodPut &&
		strings.Contains(body, `"auth_mode":"oidc_auth"`) &&
		strings.Contains(body, `"oidc_client_id":"registry-client"`) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func permissionsHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	body := string(b)
	if r.Method == http.MethodPost &&
		strings.Contains(body, `"role_id":`) {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func projectHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	body := string(b)
	if r.Method == http.MethodPost &&
		strings.Contains(body, `"project_name":"catalog-apps-org-new-project"`) {
		w.WriteHeader(http.StatusCreated)
	} else if r.Method == http.MethodDelete &&
		strings.Contains(r.URL.Path, `catalog-apps-org-new-project`) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

var errorOnPing = false

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && !errorOnPing {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

var mockRobots = map[string]CreateRobotAttributes{}
var mockRobotIDs = map[string]int{}
var nextRobotID = 0

func robotsHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	b, _ := io.ReadAll(r.Body)
	body := string(b)
	switch r.Method {
	case http.MethodPost:
		robotAttrs := CreateRobotAttributes{}
		err = json.Unmarshal([]byte(body), &robotAttrs)
		if err == nil {
			//	robotName = fmt.Sprintf(`robot$%s-catalog-apps+%s`, projectID, robotName)
			robotAttrs.Name = fmt.Sprintf(`robot$%s+%s`, robotAttrs.Permissions[0].Namespace, robotAttrs.Name)
			mockRobots[robotAttrs.Name] = robotAttrs
			mockRobotIDs[robotAttrs.Name] = nextRobotID
			nextRobotID++
			w.WriteHeader(http.StatusCreated)
			resp := CreateRobotResponse{
				Name:   robotAttrs.Name,
				Secret: "super-sekret-shhh",
			}
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	case http.MethodGet:
		robotsResults := []HarborRobot{}
		for _, robot := range mockRobots {
			robotsResults = append(robotsResults, HarborRobot{Name: robot.Name, ID: mockRobotIDs[robot.Name]})
		}
		_ = json.NewEncoder(w).Encode(robotsResults)
	}
}

func projectsRobotsGetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		robotsResults := []HarborRobot{}
		i := 0
		for _, robot := range mockRobots {
			robotsResults = append(robotsResults, HarborRobot{Name: robot.Name, ID: i})
			i++
		}
		_ = json.NewEncoder(w).Encode(robotsResults)
	}
}

func projectsRobotsDeleteHandler(w http.ResponseWriter, r *http.Request) {
	URLSegments := strings.Split(r.URL.Path, "/")
	robotID := URLSegments[len(URLSegments)-1]

	for robotName := range mockRobots {
		if strconv.Itoa(mockRobotIDs[robotName]) == robotID {
			delete(mockRobots, robotName)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func (t *TestHarborServer) Start() *TestHarborServer {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == HarborConfigurationURL {
			t.ConfigurationHandler(w, r)
		} else if r.URL.Path == HarborRobotsURL {
			t.RobotsHandler(w, r)
		} else if strings.Contains(r.URL.Path, "robots") && r.Method == http.MethodGet {
			t.ProjectsRobotsGetHandler(w, r)
		} else if strings.Contains(r.URL.Path, "robots") && r.Method == http.MethodDelete {
			t.ProjectsRobotsDeleteHandler(w, r)
		} else if strings.Contains(r.URL.Path, "/members") {
			t.ProjectsPermissionsHandler(w, r)
		} else if strings.Contains(r.URL.Path, HarborProjectsURL) {
			t.ProjectHandler(w, r)
		} else if strings.Contains(r.URL.Path, "ping") && r.Method == http.MethodGet {
			t.PingHandler(w, r)
		}

	}))
	t.Server = server
	return t
}

type testK8s struct {
}

func (k *testK8s) ReadSecret(_ context.Context, _ string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	result["credential"] = []byte(`admin:admin`)
	return result, nil
}

func NewTestK8s(_ string) (K8s, error) {
	c := &testK8s{}
	result := make(map[string]string)
	result["credential"] = `admin:admin`
	return c, nil
}

func (s *HarborTestSuite) TestHarborConfigurations() {
	var err error

	h, err := newHarbor(s.ctx, s.testServer.Server.URL, "OIDC", "harbor", "credential")
	s.NoError(err)

	err = h.Configurations(s.ctx)
	s.NoError(err)
}

func (s *HarborTestSuite) TestHarborCreateProject() {
	var err error

	h, err := newHarbor(s.ctx, s.testServer.Server.URL, "OIDC", "harbor", "credential")
	s.NoError(err)

	err = h.CreateProject(s.ctx, "org", "new-project")
	s.NoError(err)
}

func (s *HarborTestSuite) TestHarborCreateRobot() {
	var err error

	h, err := newHarbor(s.ctx, s.testServer.Server.URL, "OIDC", "harbor", "credential")
	s.NoError(err)

	name, secret, err := h.CreateRobot(s.ctx, "new-robot", "org", "new-project")
	s.NoError(err)
	s.Equal("robot$catalog-apps-org-new-project+new-robot", name)
	s.Equal("super-sekret-shhh", secret)

	s.Len(mockRobots, 1)
	s.Equal(mockRobots[name].Name, name)

	robot, err := h.GetRobot(s.ctx, "org", "new-project", "new-robot")
	s.NoError(err)
	s.NotNil(robot)
	s.Equal("robot$catalog-apps-org-new-project+new-robot", robot.Name)

	err = h.DeleteRobot(s.ctx, "org", "new-project", robot.ID)
	s.NoError(err)
	s.Len(mockRobots, 0)

	robot, err = h.GetRobot(s.ctx, "org", "new-project", "new-robot")
	s.Error(err)
	s.Nil(robot)
}

func (s *HarborTestSuite) TestHarborPermissions() {
	var err error

	h, err := newHarbor(s.ctx, s.testServer.Server.URL, "OIDC", "harbor", "credential")
	s.NoError(err)

	err = h.SetMemberPermissions(s.ctx, 3, "org", "new-project", "new-project")
	s.NoError(err)
}

func (s *HarborTestSuite) TestHarborDeleteProject() {
	var err error

	h, err := newHarbor(s.ctx, s.testServer.Server.URL, "OIDC", "harbor", "credential")
	s.NoError(err)

	err = h.CreateProject(s.ctx, "org", "new-project")
	s.NoError(err)
	err = h.DeleteProject(s.ctx, "org", "new-project")
	s.NoError(err)
	err = h.DeleteProject(s.ctx, "org", "nobody-home")
	s.Error(err)
	s.Contains(err.Error(), "error deleting project org-nobody-home")
}

func (s *HarborTestSuite) TestHarborPing() {
	var err error

	h, err := newHarbor(s.ctx, s.testServer.Server.URL, "OIDC", "harbor", "credential")
	s.NoError(err)

	err = h.Ping(s.ctx)
	s.NoError(err)

	errorOnPing = true
	err = h.Ping(s.ctx)
	s.Error(err)
}
