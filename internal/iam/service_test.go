package iam

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	userdb "github.com/Gkemhcs/kavach-backend/internal/auth/gen"
	"github.com/Gkemhcs/kavach-backend/internal/authz"
	groupsdb "github.com/Gkemhcs/kavach-backend/internal/groups/gen"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//go:embed test_data/*.json
var testDataFS embed.FS

// TestData represents the structure of our test data files
type TestData struct {
	TestCases []TestCase `json:"test_cases"`
}

// TestCase represents a single test case
type TestCase struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Expected    ExpectedResult         `json:"expected"`
	MockSetup   MockSetup              `json:"mock_setup"`
}

// ExpectedResult represents the expected outcome of a test
type ExpectedResult struct {
	Success bool        `json:"success"`
	Error   interface{} `json:"error"`
	// Additional fields for specific test types
	Organizations []map[string]interface{} `json:"organizations,omitempty"`
	SecretGroups  []map[string]interface{} `json:"secret_groups,omitempty"`
	Environments  []map[string]interface{} `json:"environments,omitempty"`
	RoleBinding   interface{}              `json:"role_binding,omitempty"`
}

// MockSetup represents the mock configuration for a test
type MockSetup struct {
	UserResolver      MockConfig `json:"user_resolver,omitempty"`
	UserGroupResolver MockConfig `json:"user_group_resolver,omitempty"`
	IamRepo           MockConfig `json:"iam_repo,omitempty"`
	PolicyEnforcer    MockConfig `json:"policy_enforcer,omitempty"`
}

// MockConfig represents a single mock configuration
type MockConfig struct {
	Method string                 `json:"method"`
	Return map[string]interface{} `json:"return"`
}

// IamServiceTestSuite provides a test suite for the IAM service
type IamServiceTestSuite struct {
	suite.Suite
	service            *IamService
	mockRepo           *MockIamRepository
	mockUserResolver   *MockUserResolver
	mockGroupResolver  *MockUserGroupResolver
	mockPolicyEnforcer *MockPolicyEnforcer
	logger             *logrus.Logger
	ctx                context.Context
}

// MockUserResolver is a mock implementation of types.UserResolver
type MockUserResolver struct {
	mock.Mock
}

// GetUserInfoByGithubUserName mocks the GetUserInfoByGithubUserName method
func (m *MockUserResolver) GetUserInfoByGithubUserName(ctx context.Context, username string) (*userdb.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userdb.User), args.Error(1)
}

// MockUserGroupResolver is a mock implementation of types.UserGroupResolver
type MockUserGroupResolver struct {
	mock.Mock
}

// GetUserGroupByName mocks the GetUserGroupByName method
func (m *MockUserGroupResolver) GetUserGroupByName(ctx context.Context, groupName, orgID string) (*groupsdb.UserGroup, error) {
	args := m.Called(ctx, groupName, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*groupsdb.UserGroup), args.Error(1)
}

// MockPolicyEnforcer is a mock implementation of PolicyEnforcer using authz.MockEnforcer
type MockPolicyEnforcer struct {
	*authz.MockEnforcer
}

// SetupSuite initializes the test suite
func (suite *IamServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.DebugLevel)
	suite.logger.SetOutput(io.Discard) // Discard logs during tests to keep output clean
}

// SetupTest initializes each test
func (suite *IamServiceTestSuite) SetupTest() {
	// Create new mock instances for each test to ensure isolation
	suite.mockRepo = &MockIamRepository{}
	suite.mockUserResolver = &MockUserResolver{}
	suite.mockGroupResolver = &MockUserGroupResolver{}
	suite.mockPolicyEnforcer = &MockPolicyEnforcer{
		MockEnforcer: &authz.MockEnforcer{},
	}

	suite.service = NewIamService(
		suite.mockRepo,
		suite.mockUserResolver,
		suite.mockGroupResolver,
		suite.logger,
		suite.mockPolicyEnforcer,
	)
}

// TearDownTest cleans up after each test
func (suite *IamServiceTestSuite) TearDownTest() {
	// Assert all mock expectations were met
	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockUserResolver.AssertExpectations(suite.T())
	suite.mockGroupResolver.AssertExpectations(suite.T())
	suite.mockPolicyEnforcer.MockEnforcer.AssertExpectations(suite.T())

	// Reset all mocks to ensure clean state for next test
	suite.mockRepo.ExpectedCalls = nil
	suite.mockUserResolver.ExpectedCalls = nil
	suite.mockGroupResolver.ExpectedCalls = nil
	suite.mockPolicyEnforcer.MockEnforcer.ExpectedCalls = nil
}

// loadTestData loads test cases from embedded JSON files
func (suite *IamServiceTestSuite) loadTestData(filename string) *TestData {
	data, err := testDataFS.ReadFile("test_data/" + filename)
	require.NoError(suite.T(), err, "Failed to read embedded test data file: %s", filename)

	var testData TestData
	err = json.Unmarshal(data, &testData)
	require.NoError(suite.T(), err, "Failed to unmarshal test data from: %s", filename)

	return &testData
}

// TestGrantRoleBindingWithData runs GrantRoleBinding tests using data-driven approach
func (suite *IamServiceTestSuite) TestGrantRoleBindingWithData() {
	testData := suite.loadTestData("grant_role_binding_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupGrantRoleBindingMocks(tc.MockSetup)

			req := suite.buildGrantRoleBindingRequest(tc.Input)
			err := suite.service.GrantRoleBinding(suite.ctx, req)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					require.Contains(suite.T(), err.Error(), fmt.Sprintf("%v", tc.Expected.Error))
				}
				// Validate that downstream methods were NOT called in error scenarios
				suite.validateNegativeMocks(tc.Name)
			}
		})
	}
}

// TestRevokeRoleBindingWithData runs RevokeRoleBinding tests using data-driven approach
func (suite *IamServiceTestSuite) TestRevokeRoleBindingWithData() {
	testData := suite.loadTestData("revoke_role_binding_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupRevokeRoleBindingMocks(tc.MockSetup)

			req := suite.buildRevokeRoleBindingRequest(tc.Input)
			err := suite.service.RevokeRoleBinding(suite.ctx, req)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					require.Contains(suite.T(), err.Error(), fmt.Sprintf("%v", tc.Expected.Error))
				}
				// Validate that downstream methods were NOT called in error scenarios
				suite.validateNegativeMocks(tc.Name)
			}
		})
	}
}

// TestListAccessibleResourcesWithData runs list accessible resources tests using data-driven approach
func (suite *IamServiceTestSuite) TestListAccessibleResourcesWithData() {
	testData := suite.loadTestData("list_accessible_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupListAccessibleMocks(tc.MockSetup)

			switch {
			case tc.Input["user_id"] != nil && tc.Input["org_id"] == nil && tc.Input["group_id"] == nil:
				// Test ListAccessibleOrganizations
				userID := tc.Input["user_id"].(string)
				result, err := suite.service.ListAccessibleOrganizations(suite.ctx, userID)

				if tc.Expected.Success {
					require.NoError(suite.T(), err, "Expected success but got error: %v", err)
					if tc.Expected.Organizations != nil {
						assert.Len(suite.T(), result, len(tc.Expected.Organizations), "Expected %d organizations, got %d", len(tc.Expected.Organizations), len(result))
						// Validate each organization's fields
						for i, expectedOrg := range tc.Expected.Organizations {
							if i < len(result) {
								assert.Equal(suite.T(), expectedOrg["organization_id"].(string), result[i].ID.String(), "Organization ID mismatch at index %d", i)
								assert.Equal(suite.T(), expectedOrg["organization_name"].(string), result[i].OrgName, "Organization name mismatch at index %d", i)
								assert.Equal(suite.T(), expectedOrg["role"].(string), string(result[i].Role), "Organization role mismatch at index %d", i)
							}
						}
					}
				} else {
					require.Error(suite.T(), err, "Expected error but got success")
				}

			case tc.Input["user_id"] != nil && tc.Input["org_id"] != nil && tc.Input["group_id"] == nil:
				// Test ListAccessibleSecretGroups
				userID := tc.Input["user_id"].(string)
				orgID := tc.Input["org_id"].(string)
				result, err := suite.service.ListAccessibleSecretGroups(suite.ctx, userID, orgID)

				if tc.Expected.Success {
					require.NoError(suite.T(), err, "Expected success but got error: %v", err)
					if tc.Expected.SecretGroups != nil {
						assert.Len(suite.T(), result, len(tc.Expected.SecretGroups), "Expected %d secret groups, got %d", len(tc.Expected.SecretGroups), len(result))
						// Validate each secret group's fields
						for i, expectedGroup := range tc.Expected.SecretGroups {
							if i < len(result) {
								assert.Equal(suite.T(), expectedGroup["secret_group_id"].(string), result[i].ID.UUID.String(), "Secret group ID mismatch at index %d", i)
								assert.Equal(suite.T(), expectedGroup["secret_group_name"].(string), result[i].Name, "Secret group name mismatch at index %d", i)
								assert.Equal(suite.T(), expectedGroup["role"].(string), string(result[i].Role), "Secret group role mismatch at index %d", i)
							}
						}
					}
				} else {
					require.Error(suite.T(), err, "Expected error but got success")
				}

			case tc.Input["user_id"] != nil && tc.Input["org_id"] != nil && tc.Input["group_id"] != nil:
				// Test ListAccessibleEnvironments
				userID := tc.Input["user_id"].(string)
				orgID := tc.Input["org_id"].(string)
				groupID := tc.Input["group_id"].(string)
				result, err := suite.service.ListAccessibleEnvironments(suite.ctx, userID, orgID, groupID)

				if tc.Expected.Success {
					require.NoError(suite.T(), err, "Expected success but got error: %v", err)
					if tc.Expected.Environments != nil {
						assert.Len(suite.T(), result, len(tc.Expected.Environments), "Expected %d environments, got %d", len(tc.Expected.Environments), len(result))
						// Validate each environment's fields
						for i, expectedEnv := range tc.Expected.Environments {
							if i < len(result) {
								assert.Equal(suite.T(), expectedEnv["environment_id"].(string), result[i].ID.UUID.String(), "Environment ID mismatch at index %d", i)
								assert.Equal(suite.T(), expectedEnv["environment_name"].(string), result[i].Name, "Environment name mismatch at index %d", i)
								assert.Equal(suite.T(), expectedEnv["role"].(string), string(result[i].Role), "Environment role mismatch at index %d", i)
							}
						}
					}
				} else {
					require.Error(suite.T(), err, "Expected error but got success")
				}
			}
		})
	}
}

// TestDeleteRoleBindingWithData runs DeleteRoleBinding tests using data-driven approach
func (suite *IamServiceTestSuite) TestDeleteRoleBindingWithData() {
	testData := suite.loadTestData("delete_role_binding_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupDeleteRoleBindingMocks(tc.MockSetup)

			req := suite.buildDeleteRoleBindingRequest(tc.Input)
			err := suite.service.DeleteRoleBinding(suite.ctx, req)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					require.Contains(suite.T(), err.Error(), fmt.Sprintf("%v", tc.Expected.Error))
				}
				// Validate that downstream methods were NOT called in error scenarios
				suite.validateNegativeMocks(tc.Name)
			}
		})
	}
}

// TestCreateRoleBindingWithData runs CreateRoleBinding tests using data-driven approach
func (suite *IamServiceTestSuite) TestCreateRoleBindingWithData() {
	testData := suite.loadTestData("create_role_binding_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupCreateRoleBindingMocks(tc.MockSetup)

			req := suite.buildCreateRoleBindingRequest(tc.Input)
			result, err := suite.service.CreateRoleBinding(suite.ctx, req)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected role binding result but got nil")
				if tc.Expected.RoleBinding != nil {
					expectedBinding := tc.Expected.RoleBinding.(map[string]interface{})
					// Use assert for multiple field checks to get better diagnostics
					assert.Equal(suite.T(), expectedBinding["id"].(string), result.ID.String(), "Role binding ID mismatch")
					assert.Equal(suite.T(), expectedBinding["role"].(string), string(result.Role), "Role binding role mismatch")
					assert.Equal(suite.T(), expectedBinding["resource_type"].(string), string(result.ResourceType), "Role binding resource type mismatch")
					assert.Equal(suite.T(), expectedBinding["user_id"].(string), result.UserID.UUID.String(), "Role binding user ID mismatch")
					assert.Equal(suite.T(), expectedBinding["resource_id"].(string), result.ResourceID.String(), "Role binding resource ID mismatch")
					assert.Equal(suite.T(), expectedBinding["organization_id"].(string), result.OrganizationID.String(), "Role binding organization ID mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					require.Contains(suite.T(), err.Error(), fmt.Sprintf("%v", tc.Expected.Error))
				}
				// Validate that downstream methods were NOT called in error scenarios
				suite.validateNegativeMocks(tc.Name)
			}
		})
	}
}

// setupGrantRoleBindingMocks sets up mocks for GrantRoleBinding tests
func (suite *IamServiceTestSuite) setupGrantRoleBindingMocks(mockSetup MockSetup) {
	if mockSetup.UserResolver.Method != "" {
		suite.setupUserResolverMock(mockSetup.UserResolver)
	}

	if mockSetup.UserGroupResolver.Method != "" {
		suite.setupUserGroupResolverMock(mockSetup.UserGroupResolver)
	}

	if mockSetup.IamRepo.Method != "" {
		suite.setupIamRepoMock(mockSetup.IamRepo)
	}

	if mockSetup.PolicyEnforcer.Method != "" {
		suite.setupPolicyEnforcerMock(mockSetup.PolicyEnforcer)
	}

}

// setupRevokeRoleBindingMocks sets up mocks for RevokeRoleBinding tests
func (suite *IamServiceTestSuite) setupRevokeRoleBindingMocks(mockSetup MockSetup) {
	if mockSetup.UserResolver.Method != "" {
		suite.setupUserResolverMock(mockSetup.UserResolver)
	}

	if mockSetup.UserGroupResolver.Method != "" {
		suite.setupUserGroupResolverMock(mockSetup.UserGroupResolver)
	}

	if mockSetup.IamRepo.Method != "" {
		suite.setupIamRepoMock(mockSetup.IamRepo)
	}

	if mockSetup.PolicyEnforcer.Method != "" {
		suite.setupPolicyEnforcerMock(mockSetup.PolicyEnforcer)
	}

}

// setupListAccessibleMocks sets up mocks for list accessible resources tests
func (suite *IamServiceTestSuite) setupListAccessibleMocks(mockSetup MockSetup) {
	if mockSetup.IamRepo.Method != "" {
		suite.setupIamRepoMock(mockSetup.IamRepo)
	}

}

// setupDeleteRoleBindingMocks sets up mocks for DeleteRoleBinding tests
func (suite *IamServiceTestSuite) setupDeleteRoleBindingMocks(mockSetup MockSetup) {
	if mockSetup.IamRepo.Method != "" {
		suite.setupIamRepoMock(mockSetup.IamRepo)
	}

}

// setupCreateRoleBindingMocks sets up mocks for CreateRoleBinding tests
func (suite *IamServiceTestSuite) setupCreateRoleBindingMocks(mockSetup MockSetup) {
	if mockSetup.IamRepo.Method != "" {
		suite.setupIamRepoMock(mockSetup.IamRepo)
	}
}

// validateNegativeMocks validates that certain methods were NOT called when they shouldn't be
func (suite *IamServiceTestSuite) validateNegativeMocks(testCase string) {
	// GrantRoleBinding error cases - these should NOT call downstream methods
	if testCase == "user_not_found_error" || testCase == "group_not_found_error" {
		// When user/group is not found, IAM repo and policy enforcer should NOT be called
		suite.mockRepo.AssertNotCalled(suite.T(), "GrantRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "GrantRole")
	}

	// GrantRoleBinding duplicate error - should NOT call policy enforcer
	if testCase == "duplicate_role_binding_error" {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "GrantRole")
	}

	// GrantRoleBinding database error - should NOT call policy enforcer
	if testCase == "database_error" {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "GrantRole")
	}

	// RevokeRoleBinding error cases - these should NOT call downstream methods
	if testCase == "user_not_found_error" || testCase == "group_not_found_error" {
		// When user/group is not found, IAM repo and policy enforcer should NOT be called
		suite.mockRepo.AssertNotCalled(suite.T(), "RevokeRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RevokeRoleCascade")
	}

	// RevokeRoleBinding database error - should NOT call policy enforcer
	if testCase == "database_error" {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RevokeRoleCascade")
	}

	// DeleteRoleBinding error cases
	if testCase == "role_binding_not_found_error" || testCase == "database_error" {
		// When role binding not found or database error, no additional calls should be made
		suite.mockRepo.AssertNotCalled(suite.T(), "DeleteRoleBinding")
	}

	// CreateRoleBinding error cases
	if testCase == "duplicate_role_binding_error" || testCase == "database_error" ||
		testCase == "invalid_user_id" || testCase == "invalid_resource_type" {
		// When any error occurs, no additional calls should be made
		suite.mockRepo.AssertNotCalled(suite.T(), "CreateRoleBinding")
	}
}

// setupUserResolverMock sets up user resolver mock
func (suite *IamServiceTestSuite) setupUserResolverMock(config MockConfig) {
	if config.Method == "GetUserInfoByGithubUserName" {
		if config.Return["error"] != nil {
			suite.mockUserResolver.On("GetUserInfoByGithubUserName", suite.ctx, mock.AnythingOfType("string")).
				Return(nil, errors.New(config.Return["error"].(string))).Once()
		} else {
			user := &userdb.User{
				ID:         uuid.MustParse(config.Return["user"].(map[string]interface{})["id"].(string)),
				Provider:   "github",
				ProviderID: config.Return["user"].(map[string]interface{})["github_username"].(string),
				Email:      sql.NullString{String: config.Return["user"].(map[string]interface{})["email"].(string), Valid: true},
			}
			suite.mockUserResolver.On("GetUserInfoByGithubUserName", suite.ctx, mock.AnythingOfType("string")).
				Return(user, nil).Once()
		}
	}
}

// setupUserGroupResolverMock sets up user group resolver mock
func (suite *IamServiceTestSuite) setupUserGroupResolverMock(config MockConfig) {
	if config.Method == "GetUserGroupByName" {
		if config.Return["error"] != nil {
			suite.mockGroupResolver.On("GetUserGroupByName", suite.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(nil, errors.New(config.Return["error"].(string))).Once()
		} else {
			group := &groupsdb.UserGroup{
				ID:             uuid.MustParse(config.Return["group"].(map[string]interface{})["id"].(string)),
				Name:           config.Return["group"].(map[string]interface{})["name"].(string),
				OrganizationID: uuid.MustParse(config.Return["group"].(map[string]interface{})["organization_id"].(string)),
			}
			suite.mockGroupResolver.On("GetUserGroupByName", suite.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(group, nil).Once()
		}
	}
}

// setupIamRepoMock sets up IAM repository mock
func (suite *IamServiceTestSuite) setupIamRepoMock(config MockConfig) {
	switch config.Method {
	case "GrantRoleBinding":
		if config.Return["error"] != nil {
			suite.mockRepo.On("GrantRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.GrantRoleBindingParams")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockRepo.On("GrantRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.GrantRoleBindingParams")).
				Return(nil).Once()
		}
	case "RevokeRoleBinding":
		if config.Return["error"] != nil {
			// When there's an error, we need to return a nil sql.Result and the error
			suite.mockRepo.On("RevokeRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.RevokeRoleBindingParams")).
				Return(nil, errors.New(config.Return["error"].(string))).Once()
		} else {
			mockResult := &MockSqlResult{}
			rowsAffected := int64(config.Return["result"].(map[string]interface{})["rows_affected"].(float64))
			mockResult.On("RowsAffected").Return(rowsAffected, nil).Once()
			suite.mockRepo.On("RevokeRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.RevokeRoleBindingParams")).
				Return(mockResult, nil).Once()
		}
	case "ListAccessibleOrganizations":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListAccessibleOrganizations", suite.ctx, mock.AnythingOfType("uuid.NullUUID")).
				Return(nil, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock organizations from test data
			orgs := []iam_db.ListAccessibleOrganizationsRow{}
			if config.Return["organizations"] != nil {
				for _, org := range config.Return["organizations"].([]interface{}) {
					orgMap := org.(map[string]interface{})
					orgs = append(orgs, iam_db.ListAccessibleOrganizationsRow{
						ID:      uuid.MustParse(orgMap["organization_id"].(string)),
						OrgName: orgMap["organization_name"].(string),
						Role:    iam_db.UserRole(orgMap["role"].(string)),
					})
				}
			}
			suite.mockRepo.On("ListAccessibleOrganizations", suite.ctx, mock.AnythingOfType("uuid.NullUUID")).
				Return(orgs, nil).Once()
		}
	case "ListAccessibleSecretGroups":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListAccessibleSecretGroups", suite.ctx, mock.AnythingOfType("iam_db.ListAccessibleSecretGroupsParams")).
				Return(nil, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock secret groups from test data
			groups := []iam_db.ListAccessibleSecretGroupsRow{}
			if config.Return["secret_groups"] != nil {
				for _, group := range config.Return["secret_groups"].([]interface{}) {
					groupMap := group.(map[string]interface{})
					groups = append(groups, iam_db.ListAccessibleSecretGroupsRow{
						ID:   uuid.NullUUID{UUID: uuid.MustParse(groupMap["secret_group_id"].(string)), Valid: true},
						Name: groupMap["secret_group_name"].(string),
						Role: iam_db.UserRole(groupMap["role"].(string)),
					})
				}
			}
			suite.mockRepo.On("ListAccessibleSecretGroups", suite.ctx, mock.AnythingOfType("iam_db.ListAccessibleSecretGroupsParams")).
				Return(groups, nil).Once()
		}
	case "ListAccessibleEnvironments":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListAccessibleEnvironments", suite.ctx, mock.AnythingOfType("iam_db.ListAccessibleEnvironmentsParams")).
				Return(nil, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock environments from test data
			environments := []iam_db.ListAccessibleEnvironmentsRow{}
			if config.Return["environments"] != nil {
				for _, env := range config.Return["environments"].([]interface{}) {
					envMap := env.(map[string]interface{})
					environments = append(environments, iam_db.ListAccessibleEnvironmentsRow{
						ID:   uuid.NullUUID{UUID: uuid.MustParse(envMap["environment_id"].(string)), Valid: true},
						Name: envMap["environment_name"].(string),
						Role: iam_db.UserRole(envMap["role"].(string)),
					})
				}
			}
			suite.mockRepo.On("ListAccessibleEnvironments", suite.ctx, mock.AnythingOfType("iam_db.ListAccessibleEnvironmentsParams")).
				Return(environments, nil).Once()
		}
	case "DeleteRoleBinding":
		if config.Return["error"] != nil {
			suite.mockRepo.On("DeleteRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.DeleteRoleBindingParams")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockRepo.On("DeleteRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.DeleteRoleBindingParams")).
				Return(nil).Once()
		}
	case "CreateRoleBinding":
		if config.Return["error"] != nil {
			suite.mockRepo.On("CreateRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.CreateRoleBindingParams")).
				Return(iam_db.RoleBinding{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock role binding from test data
			bindingData := config.Return["role_binding"].(map[string]interface{})
			binding := iam_db.RoleBinding{
				ID:             uuid.MustParse(bindingData["id"].(string)),
				UserID:         uuid.NullUUID{UUID: uuid.MustParse(bindingData["user_id"].(string)), Valid: true},
				Role:           iam_db.UserRole(bindingData["role"].(string)),
				ResourceType:   iam_db.ResourceType(bindingData["resource_type"].(string)),
				ResourceID:     uuid.MustParse(bindingData["resource_id"].(string)),
				OrganizationID: uuid.MustParse(bindingData["organization_id"].(string)),
			}

			// Handle optional fields
			if bindingData["secret_group_id"] != nil {
				binding.SecretGroupID = uuid.NullUUID{
					UUID:  uuid.MustParse(bindingData["secret_group_id"].(string)),
					Valid: true,
				}
			}
			if bindingData["environment_id"] != nil {
				binding.EnvironmentID = uuid.NullUUID{
					UUID:  uuid.MustParse(bindingData["environment_id"].(string)),
					Valid: true,
				}
			}

			suite.mockRepo.On("CreateRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.CreateRoleBindingParams")).
				Return(binding, nil).Once()
		}
	}
}

// setupPolicyEnforcerMock sets up policy enforcer mock
func (suite *IamServiceTestSuite) setupPolicyEnforcerMock(config MockConfig) {
	switch config.Method {
	case "GrantRole":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("GrantRole", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("GrantRole", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(nil).Once()
		}
	case "RevokeRoleCascade":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("RevokeRoleCascade", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("RevokeRoleCascade", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(nil).Once()
		}
	}
}

// buildGrantRoleBindingRequest builds a GrantRoleBindingRequest from test input
func (suite *IamServiceTestSuite) buildGrantRoleBindingRequest(input map[string]interface{}) GrantRoleBindingRequest {
	req := GrantRoleBindingRequest{
		UserName:       input["user_name"].(string),
		GroupName:      input["group_name"].(string),
		Role:           input["role"].(string),
		ResourceType:   input["resource_type"].(string),
		ResourceID:     uuid.MustParse(input["resource_id"].(string)),
		OrganizationID: uuid.MustParse(input["organization_id"].(string)),
	}

	if input["secret_group_id"] != nil {
		sgData := input["secret_group_id"].(map[string]interface{})
		if sgData["uuid"] != nil {
			req.SecretGroupID = uuid.NullUUID{
				UUID:  uuid.MustParse(sgData["uuid"].(string)),
				Valid: sgData["valid"].(bool),
			}
		}
	}

	if input["environment_id"] != nil {
		envData := input["environment_id"].(map[string]interface{})
		if envData["uuid"] != nil {
			req.EnvironmentID = uuid.NullUUID{
				UUID:  uuid.MustParse(envData["uuid"].(string)),
				Valid: envData["valid"].(bool),
			}
		}
	}

	return req
}

// buildRevokeRoleBindingRequest builds a RevokeRoleBindingRequest from test input
func (suite *IamServiceTestSuite) buildRevokeRoleBindingRequest(input map[string]interface{}) RevokeRoleBindingRequest {
	req := RevokeRoleBindingRequest{
		UserName:       input["user_name"].(string),
		GroupName:      input["group_name"].(string),
		Role:           input["role"].(string),
		ResourceType:   input["resource_type"].(string),
		ResourceID:     uuid.MustParse(input["resource_id"].(string)),
		OrganizationID: uuid.MustParse(input["organization_id"].(string)),
	}

	if input["secret_group_id"] != nil {
		sgData := input["secret_group_id"].(map[string]interface{})
		if sgData["uuid"] != nil {
			req.SecretGroupID = uuid.NullUUID{
				UUID:  uuid.MustParse(sgData["uuid"].(string)),
				Valid: sgData["valid"].(bool),
			}
		}
	}

	if input["environment_id"] != nil {
		envData := input["environment_id"].(map[string]interface{})
		if envData["uuid"] != nil {
			req.EnvironmentID = uuid.NullUUID{
				UUID:  uuid.MustParse(envData["uuid"].(string)),
				Valid: envData["valid"].(bool),
			}
		}
	}

	return req
}

// buildDeleteRoleBindingRequest builds a DeleteRoleBindingRequest from test input
func (suite *IamServiceTestSuite) buildDeleteRoleBindingRequest(input map[string]interface{}) DeleteRoleBindingRequest {
	req := DeleteRoleBindingRequest{
		ResourceType: input["resource_type"].(string),
		ResourceID:   uuid.MustParse(input["resource_id"].(string)),
	}

	return req
}

// buildCreateRoleBindingRequest builds a CreateRoleBindingRequest from test input
func (suite *IamServiceTestSuite) buildCreateRoleBindingRequest(input map[string]interface{}) CreateRoleBindingRequest {
	req := CreateRoleBindingRequest{
		UserID:         uuid.MustParse(input["user_id"].(string)),
		Role:           input["role"].(string),
		ResourceType:   input["resource_type"].(string),
		ResourceID:     uuid.MustParse(input["resource_id"].(string)),
		OrganizationID: uuid.MustParse(input["organization_id"].(string)),
	}

	if input["secret_group_id"] != nil {
		sgData := input["secret_group_id"].(map[string]interface{})
		if sgData["uuid"] != nil {
			req.SecretGroupID = uuid.NullUUID{
				UUID:  uuid.MustParse(sgData["uuid"].(string)),
				Valid: sgData["valid"].(bool),
			}
		}
	}

	if input["environment_id"] != nil {
		envData := input["environment_id"].(map[string]interface{})
		if envData["uuid"] != nil {
			req.EnvironmentID = uuid.NullUUID{
				UUID:  uuid.MustParse(envData["uuid"].(string)),
				Valid: envData["valid"].(bool),
			}
		}
	}

	return req
}

// TestIamServiceTestSuite runs the test suite
func TestIamServiceTestSuite(t *testing.T) {
	suite.Run(t, new(IamServiceTestSuite))
}
