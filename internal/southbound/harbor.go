// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HarborOCI struct {
	harborHost string
	oidcURL    string
	username   string
	token      string
}

const (
	HarborConfigurationURL = "/api/v2.0/configurations"
	HarborRobotsURL        = "/api/v2.0/robots"
	HarborProjectsURL      = "/api/v2.0/projects"
	HarborPingURL          = "/api/v2.0/ping"
	AddHeaders             = true
	NoHeaders              = false
)

type K8s interface {
	ReadSecret(ctx context.Context, name string) (map[string][]byte, error)
}

func NewK8s(namespace string) (K8s, error) {
	return NewK8sClient(namespace)
}

var K8sFactory = NewK8s

func HarborProjectName(org string, displayName string) string {
	return fmt.Sprintf(`catalog-apps-%s-%s`, org, displayName)
}

func readHarborAdminCredentials(ctx context.Context, harborNamespace string, harborAdminCredential string) (username, password string, err error) {
	k8sClient, err := K8sFactory(harborNamespace)
	if err != nil {
		return "", "", err
	}
	data, err := k8sClient.ReadSecret(ctx, harborAdminCredential)
	if err != nil {
		return "", "", err
	}

	credString, ok := data["credential"]
	if !ok {
		return "", "", fmt.Errorf("no credential found in secret")
	}

	creds := strings.Split(string(credString), ":")
	return creds[0], creds[1], nil
}

func newHarbor(ctx context.Context, harborHost string, oidcURL string, harborNamespace string, harborAdminCredential string) (*HarborOCI, error) {
	u, p, err := readHarborAdminCredentials(ctx, harborNamespace, harborAdminCredential)
	if err != nil {
		return nil, err
	}
	harbor := &HarborOCI{
		harborHost: harborHost,
		oidcURL:    oidcURL,
		username:   u,
		token:      p,
	}
	return harbor, nil
}

func NewHarborOCI(ctx context.Context, harborHost string, oidcURL string, harborNamespace string, harborAdminCredential string) (*HarborOCI, error) {
	return newHarbor(ctx, harborHost, oidcURL, harborNamespace, harborAdminCredential)
}

func (h *HarborOCI) doHarborREST(
	ctx context.Context,
	method string,
	endpoint string,
	body io.Reader,
	addHeaders bool,
) (*http.Response, error) {
	c := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	if addHeaders {
		req.SetBasicAuth(h.username, h.token)
		req.Header.Add("content-type", "application/json")
		req.Header.Add("accept", "application/json")
	}
	log.Infof("Harbor REST request method %s base URL %s", method, req.URL.String())

	resp, err := c.Do(req)
	if err != nil {
		log.Infof("Harbor REST call failed with error %s", err.Error())
	} else {
		log.Infof("Harbor REST call succeeded %s", resp.Status)
	}
	return resp, err
}

type ConfigurationAttributes struct {
	AuthMode        string `json:"auth_mode"`
	OidcName        string `json:"oidc_name"`
	OidcEndpoint    string `json:"oidc_endpoint"`
	OidcVerifyCert  bool   `json:"oidc_verify_cert"`
	OidcClientID    string `json:"oidc_client_id"`
	OidcScope       string `json:"oidc_scope"`
	OidcAutoOnboard bool   `json:"oidc_auto_onboard"`
	OidcUserClaim   string `json:"oidc_user_claim"`
	OidcGroupsClaim string `json:"oidc_groups_claim"`
	OidcAdminGroup  string `json:"oidc_admin_group"`
}

func (h *HarborOCI) Configurations(ctx context.Context) error {
	URL := h.harborHost + HarborConfigurationURL
	configAttrs := ConfigurationAttributes{
		AuthMode:        "oidc_auth",
		OidcName:        "Open Edge IAM",
		OidcEndpoint:    h.oidcURL + `/realms/master`,
		OidcVerifyCert:  false,
		OidcClientID:    "registry-client",
		OidcScope:       "openid,profile,offline_access,email",
		OidcAutoOnboard: true,
		OidcUserClaim:   "preferred_username",
		OidcGroupsClaim: "groups",
		OidcAdminGroup:  "service-admin-group",
	}
	configBody, err := json.Marshal(configAttrs)
	if err != nil {
		return err
	}
	resp, err := h.doHarborREST(ctx, http.MethodPut, URL, bytes.NewReader(configBody), AddHeaders)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		responseJSON := string(responseBody)
		return fmt.Errorf("%s", responseJSON)
	}

	return err
}

type CreateProjectAttributes struct {
	ProjectName  string `json:"project_name"`
	Public       bool   `json:"public"`
	StorageLimit int    `json:"storage_limit"`
}

func (h *HarborOCI) CreateProject(ctx context.Context, org string, displayName string) error {
	URL := h.harborHost + HarborProjectsURL
	projectAttrs := CreateProjectAttributes{
		ProjectName:  HarborProjectName(org, displayName),
		Public:       false,
		StorageLimit: 0,
	}
	projectBody, err := json.Marshal(projectAttrs)
	if err != nil {
		return err
	}

	resp, err := h.doHarborREST(ctx, http.MethodPost, URL, bytes.NewReader(projectBody), AddHeaders)
	if err != nil {
		return err
	}
	if !(resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict) {
		responseBody, _ := io.ReadAll(resp.Body)
		responseJSON := string(responseBody)
		return fmt.Errorf("%s", responseJSON)
	}
	return nil
}

type MemberGroup struct {
	GroupName string `json:"group_name"`
}

type MembersAttributes struct {
	RoleID      int         `json:"role_id"`
	MemberGroup MemberGroup `json:"member_group"`
}

func (h *HarborOCI) SetMemberPermissions(ctx context.Context, roleID int, org string, displayName string, groupName string) error {
	URL := fmt.Sprintf("%s/api/v2.0/projects/%s/members", h.harborHost, HarborProjectName(org, displayName))
	membersAttrs := MembersAttributes{
		RoleID:      roleID,
		MemberGroup: MemberGroup{GroupName: groupName},
	}

	projectBody, err := json.Marshal(membersAttrs)
	if err != nil {
		return err
	}

	resp, err := h.doHarborREST(ctx, http.MethodPost, URL, bytes.NewReader(projectBody), AddHeaders)
	if err != nil {
		return err
	}
	if !(resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict) {
		responseBody, _ := io.ReadAll(resp.Body)
		responseJSON := string(responseBody)
		return fmt.Errorf("%s", responseJSON)
	}
	return nil
}

type HarborProject struct {
	ProjectID int `json:"project_id"`
}

func (h *HarborOCI) GetProjectID(ctx context.Context, org string, displayName string) (int, error) {
	URL := h.harborHost + "/api/v2.0/projects/" + HarborProjectName(org, displayName)

	projectResults := HarborProject{}
	resp, err := h.doHarborREST(ctx, http.MethodGet, URL, nil, AddHeaders)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		responseJSON := string(responseBody)
		return 0, fmt.Errorf("%s", responseJSON)
	}

	err = json.NewDecoder(resp.Body).Decode(&projectResults)
	if err != nil {
		return 0, err
	}

	return projectResults.ProjectID, nil
}

type RobotAccess struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
}
type RobotPermissions struct {
	Kind      string        `json:"kind"`
	Namespace string        `json:"namespace"`
	Access    []RobotAccess `json:"access"`
}

type CreateRobotAttributes struct {
	Disable     bool               `json:"disable"`
	Name        string             `json:"name"`
	Level       string             `json:"level"`
	Duration    int                `json:"duration"`
	Permissions []RobotPermissions `json:"permissions"`
}

type CreateRobotResponse struct {
	CreationTime time.Time `json:"creation_time"`
	ExpiresAt    int       `json:"expires_at"`
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Secret       string    `json:"secret"`
}

func addAccess(resource string, actions []string, permissions *RobotPermissions) {
	for _, action := range actions {
		permissions.Access = append(permissions.Access, RobotAccess{
			Resource: resource,
			Action:   action,
		})
	}
}

func (h *HarborOCI) CreateRobot(ctx context.Context, robotName string, org string, displayName string) (string, string, error) {
	repositoryActions := []string{"list", "pull", "push", "delete"}
	artifactActions := []string{"read", "list", "delete"}
	artifactLabelActions := []string{"create", "delete"}
	tagActions := []string{"create", "delete", "list"}
	scanActions := []string{"create", "stop"}

	URL := h.harborHost + HarborRobotsURL
	robotAttrs := CreateRobotAttributes{}
	robotAttrs.Name = robotName
	robotAttrs.Level = "project"
	robotAttrs.Duration = -1
	permission := &RobotPermissions{
		Kind:      "project",
		Namespace: HarborProjectName(org, displayName),
		Access:    make([]RobotAccess, 0),
	}
	addAccess("repository", repositoryActions, permission)
	addAccess("artifact", artifactActions, permission)
	addAccess("artifact-label", artifactLabelActions, permission)
	addAccess("tag", tagActions, permission)
	addAccess("scan", scanActions, permission)
	robotAttrs.Permissions = append(robotAttrs.Permissions, *permission)

	robotBody, err := json.Marshal(robotAttrs)
	if err != nil {
		return "", "", err
	}

	resp, err := h.doHarborREST(ctx, http.MethodPost, URL, bytes.NewReader(robotBody), AddHeaders)
	if err != nil {
		return "", "", err
	}
	createRobotResponseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		responseJSON := string(responseBody)
		return "", "", fmt.Errorf("%s", responseJSON)
	}
	createRobotResponse := &CreateRobotResponse{}
	err = json.Unmarshal(createRobotResponseBody, createRobotResponse)
	if err != nil {
		return "", "", err
	}

	return createRobotResponse.Name, createRobotResponse.Secret, err
}

type HarborRobot struct {
	CreationTime time.Time `json:"creation_time"`
	Disable      bool      `json:"disable"`
	Duration     int       `json:"duration"`
	Editable     bool      `json:"editable"`
	ExpiresAt    int       `json:"expires_at"`
	ID           int       `json:"id"`
	Level        string    `json:"level"`
	Name         string    `json:"name"`
	Permissions  []struct {
		Access []struct {
			Action   string `json:"action"`
			Resource string `json:"resource"`
		} `json:"access"`
		Kind      string `json:"kind"`
		Namespace string `json:"namespace"`
	} `json:"permissions"`
	UpdateTime time.Time `json:"update_time"`
}

func (h *HarborOCI) GetRobot(ctx context.Context, org string, displayName string, robotName string, projectID int) (*HarborRobot, error) {
	robotName = fmt.Sprintf(`robot$%s+%s`, HarborProjectName(org, displayName), robotName)
	URL := h.harborHost + "/api/v2.0/robots?q=Level=project,ProjectID=" + fmt.Sprintf("%d", projectID)

	robotsResults := []HarborRobot{}
	resp, err := h.doHarborREST(ctx, http.MethodGet, URL, nil, AddHeaders)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		responseJSON := string(responseBody)
		return nil, fmt.Errorf("%s", responseJSON)
	}

	err = json.NewDecoder(resp.Body).Decode(&robotsResults)
	if err != nil {
		return nil, err
	}

	for i, robot := range robotsResults {
		log.Infof("Checking robot[%d] %v for %s", i, robot, robotName)
		if robot.Name == robotName {
			return &robot, err
		}
	}

	return nil, fmt.Errorf("harbor robot %s not found", robotName)
}

func (h *HarborOCI) DeleteRobot(ctx context.Context, org string, displayName string, robotID int) error {
	_ = org
	_ = displayName
	URL := fmt.Sprintf("%s/api/v2.0/robots/%d", h.harborHost, robotID)
	resp, err := h.doHarborREST(ctx, http.MethodDelete, URL, nil, AddHeaders)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		responseBody, _ := io.ReadAll(resp.Body)
		responseJSON := string(responseBody)
		return fmt.Errorf("%s", responseJSON)
	}

	return err
}

func (h *HarborOCI) DeleteProject(ctx context.Context, org string, displayName string) error {
	URL := fmt.Sprintf("%s%s/%s", h.harborHost, HarborProjectsURL, HarborProjectName(org, displayName))
	resp, err := h.doHarborREST(ctx, http.MethodDelete, URL, nil, AddHeaders)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		responseBody, _ := io.ReadAll(resp.Body)
		responseJSON := string(responseBody)
		return fmt.Errorf("error deleting project %s-%s: code %d message %s", org, displayName, resp.StatusCode, responseJSON)
	}

	return err
}

func (h *HarborOCI) Ping(ctx context.Context) error {
	URL := h.harborHost + HarborPingURL
	resp, err := h.doHarborREST(ctx, http.MethodGet, URL, nil, NoHeaders)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		responseJSON := string(responseBody)
		return fmt.Errorf("%s", responseJSON)
	}

	return nil
}
